package exotic

import (
	"reflect"
	"testing"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func openExoticStateTestDB(t *testing.T) common.KVDB {
	t.Helper()
	kv := db.NewKVDBWithCache(t.TempDir(), 1)
	if kv == nil {
		t.Fatal("open test database")
	}
	t.Cleanup(func() {
		if err := kv.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})
	return kv
}

func newExoticStateTestIndexer(kv common.KVDB) *ExoticIndexer {
	return &ExoticIndexer{
		db:                   kv,
		status:               &Status{Version: DB_VERSION},
		tickerMap:            make(map[string]*TickInfo),
		holderInfo:           make(map[uint64]*HolderInfo),
		utxoMap:              make(map[string]map[uint64]int64),
		utxoDeleted:          make(map[string]map[uint64]bool),
		holderBalanceTouched: make(map[string]int64),
		holderActionList:     make([]*HolderAction, 0),
		tickerAdded:          make(map[string]*common.Ticker),
	}
}

func TestExoticInitDoesNotLoadAllTickerUtxos(t *testing.T) {
	kv := openExoticStateTestDB(t)
	wb := kv.NewWriteBatch()
	for id := uint64(1); id <= 5; id++ {
		amount := int64(id)
		if err := db.SetDB([]byte(GetTickerUtxoKey("rare", id)), &amount, wb); err != nil {
			wb.Close()
			t.Fatalf("seed ticker UTXO: %v", err)
		}
	}
	if err := wb.Flush(); err != nil {
		wb.Close()
		t.Fatalf("flush ticker UTXOs: %v", err)
	}
	wb.Close()

	indexer := NewExoticIndexer(kv)
	indexer.Init(nil)
	if len(indexer.utxoMap) != 0 {
		t.Fatalf("Init loaded full ticker UTXO state: %v", indexer.utxoMap)
	}
	if len(indexer.holderInfo) != 0 {
		t.Fatalf("Init loaded holder state: %v", indexer.holderInfo)
	}
}

func TestExoticHolderAggregatePersistsAndDeletes(t *testing.T) {
	kv := openExoticStateTestDB(t)
	indexer := newExoticStateTestIndexer(kv)

	indexer.adjustTickerHolderAmount("rare", 7, 25)
	if got := indexer.getTickerHolderAmounts("rare"); !reflect.DeepEqual(got, map[uint64]int64{7: 25}) {
		t.Fatalf("pending holders=%v", got)
	}
	indexer.UpdateDB()
	if got := indexer.getTickerHolderAmounts("rare"); !reflect.DeepEqual(got, map[uint64]int64{7: 25}) {
		t.Fatalf("persisted holders=%v", got)
	}

	indexer.adjustTickerHolderAmount("rare", 7, -25)
	indexer.UpdateDB()
	if got := indexer.getTickerHolderAmounts("rare"); len(got) != 0 {
		t.Fatalf("deleted holders=%v", got)
	}
	if _, err := kv.Read(GetTickerHolderKey("rare", 7)); err != common.ErrKeyNotFound {
		t.Fatalf("zero aggregate was not deleted: %v", err)
	}
}

func TestExoticTickerUtxoOverlay(t *testing.T) {
	kv := openExoticStateTestDB(t)
	wb := kv.NewWriteBatch()
	amount := int64(10)
	if err := db.SetDB([]byte(GetTickerUtxoKey("rare", 1)), &amount, wb); err != nil {
		wb.Close()
		t.Fatalf("seed ticker UTXO: %v", err)
	}
	if err := wb.Flush(); err != nil {
		wb.Close()
		t.Fatalf("flush ticker UTXO: %v", err)
	}
	wb.Close()

	indexer := newExoticStateTestIndexer(kv)
	indexer.utxoMap["rare"] = map[uint64]int64{2: 20}
	indexer.utxoDeleted["rare"] = map[uint64]bool{1: true}
	if got := indexer.getTickerUtxos("rare"); !reflect.DeepEqual(got, map[uint64]int64{2: 20}) {
		t.Fatalf("ticker UTXO overlay=%v", got)
	}
}

func TestExoticRPCReadDoesNotPopulateProcessingCache(t *testing.T) {
	kv := openExoticStateTestDB(t)
	wb := kv.NewWriteBatch()
	holder := &HolderInfo{
		AddressId: 7,
		Tickers: map[string]*common.AssetAbbrInfo{
			"rare": {
				BindingSat: 1,
				Offsets: common.AssetOffsets{
					&common.OffsetRange{Start: 0, End: 1},
				},
			},
		},
	}
	if err := db.SetDB([]byte(GetHolderInfoKey(99)), holder, wb); err != nil {
		wb.Close()
		t.Fatalf("seed holder: %v", err)
	}
	if err := wb.Flush(); err != nil {
		wb.Close()
		t.Fatalf("flush holder: %v", err)
	}
	wb.Close()

	indexer := newExoticStateTestIndexer(kv)
	assets := indexer.GetAssetsWithUtxo(99)
	rare := assets["rare"]
	if rare.Size() != 1 {
		t.Fatalf("assets=%v", assets)
	}
	if len(indexer.holderInfo) != 0 {
		t.Fatalf("RPC read populated processing cache: %v", indexer.holderInfo)
	}
}

func TestExoticPendingDeleteHidesPersistedHolderFromRPC(t *testing.T) {
	kv := openExoticStateTestDB(t)
	wb := kv.NewWriteBatch()
	holder := &HolderInfo{
		AddressId: 7,
		Tickers: map[string]*common.AssetAbbrInfo{
			"rare": {
				BindingSat: 1,
				Offsets: common.AssetOffsets{
					&common.OffsetRange{Start: 0, End: 1},
				},
			},
		},
	}
	if err := db.SetDB([]byte(GetHolderInfoKey(99)), holder, wb); err != nil {
		wb.Close()
		t.Fatalf("seed holder: %v", err)
	}
	if err := wb.Flush(); err != nil {
		wb.Close()
		t.Fatalf("flush holder: %v", err)
	}
	wb.Close()

	indexer := newExoticStateTestIndexer(kv)
	indexer.utxoDeleted["rare"] = map[uint64]bool{99: true}
	if assets := indexer.GetAssetsWithUtxo(99); assets != nil {
		t.Fatalf("pending-deleted UTXO remained visible: %v", assets)
	}
}
