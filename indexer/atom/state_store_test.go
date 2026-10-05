package atom

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func openAtomStateTestDB(t *testing.T) common.KVDB {
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

func seedAtomState(t *testing.T, kv common.KVDB) {
	t.Helper()
	wb := kv.NewWriteBatch()
	defer wb.Close()

	status := &Status{Version: DB_VERSION, Height: 100, TickerCount: 1, MintCount: 1}
	ticker := &Ticker{
		Id:           0,
		Name:         "atom",
		DisplayName:  "ATOM",
		MintedAmount: 1000,
		MintedTimes:  1,
		HolderCount:  1,
	}
	balance := &UtxoBalance{
		UtxoId:     99,
		AddressId:  7,
		Outpoint:   "tx:0",
		AtomicalId: "atomicali0",
		Ticker:     "atom",
		Amount:     1000,
	}
	mint := &MintInfo{
		Id:         0,
		AtomicalId: balance.AtomicalId,
		Ticker:     "atom",
		AddressId:  7,
		UtxoId:     99,
		Amount:     1000,
	}

	writes := []struct {
		key   []byte
		value any
	}{
		{[]byte(DB_STATUS_KEY), status},
		{[]byte(GetTickerKey("atom")), ticker},
		{[]byte(GetTickerIdKey(0)), "atom"},
		{[]byte(GetUtxoBalanceKey(99, balance.AtomicalId)), balance},
		{[]byte(GetTickerUtxoKey("atom", 99, balance.AtomicalId)), int64(1000)},
		{[]byte(GetHolderAssetKey(7, "atom")), int64(1000)},
		{[]byte(GetTickerHolderKey("atom", 7)), int64(1000)},
		{[]byte(GetMintHistoryKey("atom", 0)), mint},
		{[]byte(GetAddressMintHistoryKey("atom", 7, 0)), mint},
	}
	for _, write := range writes {
		if err := db.SetDB(write.key, write.value, wb); err != nil {
			t.Fatalf("seed %q: %v", write.key, err)
		}
	}
	if err := wb.Flush(); err != nil {
		t.Fatalf("flush seed state: %v", err)
	}
}

func TestAtomInitKeepsDurableStateInBadger(t *testing.T) {
	kv := openAtomStateTestDB(t)
	seedAtomState(t, kv)

	indexer := NewIndexer(kv, &chaincfg.TestNet4Params)
	indexer.Init(nil)
	if len(indexer.tickerMap) != 1 {
		t.Fatalf("ticker metadata count=%d, want 1", len(indexer.tickerMap))
	}
	if len(indexer.utxoBalances) != 0 || len(indexer.holderBalances) != 0 ||
		len(indexer.tickerHolders) != 0 || len(indexer.tickerUtxos) != 0 || len(indexer.mintHistory) != 0 {
		t.Fatalf("Init loaded durable state into memory")
	}
}

func TestAtomRPCReadsDoNotPopulateProcessingCache(t *testing.T) {
	kv := openAtomStateTestDB(t)
	seedAtomState(t, kv)
	indexer := NewIndexer(kv, &chaincfg.TestNet4Params)
	indexer.Init(nil)

	balances := indexer.GetUtxoBalances(99)
	if len(balances) != 1 || balances[0].Amount != 1000 {
		t.Fatalf("UTXO balances=%v", balances)
	}
	if got := indexer.GetAddressAssets(7)["atom"]; got != 1000 {
		t.Fatalf("address amount=%d", got)
	}
	if got := indexer.GetHoldersWithTick("atom")[7]; got == nil || got.Int64() != 1000 {
		t.Fatalf("ticker holder=%v", got)
	}
	if got, total := indexer.GetMintHistoryWithAddress(7, "atom", 0, 10); total != 1 || len(got) != 1 {
		t.Fatalf("address mint history len=%d total=%d", len(got), total)
	}
	if len(indexer.utxoBalances) != 0 || len(indexer.holderBalances) != 0 || len(indexer.mintHistory) != 0 {
		t.Fatalf("RPC reads populated processing cache")
	}
}

func TestAtomProcessingLoadsOnlyTouchedUtxoAndUpdateDBReleasesIt(t *testing.T) {
	kv := openAtomStateTestDB(t)
	seedAtomState(t, kv)
	indexer := NewIndexer(kv, &chaincfg.TestNet4Params)
	indexer.Init(nil)

	indexer.mutex.Lock()
	items := indexer.ensureUtxoBalancesLoadedLocked(99)
	indexer.mutex.Unlock()
	if len(items) != 1 || len(indexer.utxoBalances) != 1 {
		t.Fatalf("lazy processing state=%v", indexer.utxoBalances)
	}

	indexer.UpdateDB()
	if len(indexer.utxoBalances) != 0 || len(indexer.holderBalances) != 0 ||
		len(indexer.tickerHolders) != 0 || len(indexer.tickerUtxos) != 0 || len(indexer.mintHistory) != 0 {
		t.Fatalf("UpdateDB retained lazy processing state")
	}
}
