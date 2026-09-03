package base

import (
	"strings"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/sat20-labs/indexer/common"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
)

func TestSyncStatsClonePreservesAllPersistedFields(t *testing.T) {
	source := &SyncStats{
		ChainTip:       200,
		SyncHeight:     180,
		SyncBlockHash:  "block-hash",
		ReorgsDetected: []int{12, 34},
		AllUtxoCount:   500,
		AddressCount:   400,
		UtxoCount:      300,
		TotalSats:      250,
		BurnedSats:     75,
	}

	clone := source.Clone()
	if clone.BurnedSats != source.BurnedSats {
		t.Fatalf("BurnedSats = %d, want %d", clone.BurnedSats, source.BurnedSats)
	}
	clone.ReorgsDetected[0] = 99
	if source.ReorgsDetected[0] != 12 {
		t.Fatal("clone shares ReorgsDetected with source")
	}
}

func TestBaseIndexerCloneCopiesAddressDeltas(t *testing.T) {
	source := NewBaseIndexer(nil, &chaincfg.TestNet4Params, 0, 100)
	source.utxoIndex = common.NewUTXOIndex()
	source.addressValueMap["OP_RETURN"] = &common.AddressValueV2{
		AddressId:   41,
		AddressType: int(txscript.NullDataTy),
		Op:          1,
		UtxoCount:   2,
		Utxos: map[uint64]int64{
			1001: 25,
			1002: 50,
		},
	}
	source.addressUtxoDeleted[41] = map[uint64]bool{900: true}

	clone := source.Clone(true)
	got := clone.addressValueMap["OP_RETURN"]
	if got == nil || got.AddressId != 41 || got.AddressType != int(txscript.NullDataTy) {
		t.Fatalf("cloned address metadata = %#v", got)
	}
	if got.Op != 1 {
		t.Fatalf("clone Op = %d, want snapshot value 1", got.Op)
	}
	if got.UtxoCount != 2 {
		t.Fatalf("clone UtxoCount=%d, want 2", got.UtxoCount)
	}
	if source.addressValueMap["OP_RETURN"].Op != 0 {
		t.Fatalf("source Op = %d, want stored marker 0", source.addressValueMap["OP_RETURN"].Op)
	}

	got.Utxos[1001] = 999
	delete(got.Utxos, 1002)
	delete(clone.addressUtxoDeleted[41], 900)
	if source.addressValueMap["OP_RETURN"].Utxos[1001] != 25 {
		t.Fatal("clone shares the source UTXO addition map")
	}
	if !source.addressUtxoDeleted[41][900] {
		t.Fatal("clone shares the source UTXO deletion map")
	}
}

func openBaseTestDB(t *testing.T) common.KVDB {
	t.Helper()
	kv := indexdb.NewKVDBWithCache(t.TempDir(), 1)
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

func newBaseForUpdate(kv common.KVDB) *BaseIndexer {
	indexer := NewBaseIndexer(kv, &chaincfg.TestNet4Params, 0, 100)
	indexer.stats = &SyncStats{ReorgsDetected: make([]int, 0)}
	indexer.lastHeight = 1
	indexer.lastHash = "snapshot-hash"
	indexer.utxoIndex = common.NewUTXOIndex()
	return indexer
}

func addPendingBaseUtxo(indexer *BaseIndexer, address string, addressID, utxoID uint64, value int64, op int) {
	out := common.NewTxOutputV2(value)
	out.UtxoId = utxoID
	out.OutPointStr = "0000000000000000000000000000000000000000000000000000000000000001:0"
	out.AddressId = addressID
	out.AddressType = int(txscript.WitnessV1TaprootTy)
	indexer.utxoIndex.Index[out.OutPointStr] = out
	indexer.addressValueMap[address] = &common.AddressValueV2{
		AddressId:   addressID,
		AddressType: int(txscript.WitnessV1TaprootTy),
		Op:          op,
		UtxoCount:   1,
		Utxos:       map[uint64]int64{utxoID: value},
	}
	indexer.idToAddressMap[addressID] = address
}

func TestBaseIndexerUpdateDBStoresAddressUtxosByPrefix(t *testing.T) {
	kv := openBaseTestDB(t)
	const (
		address   = "bc1ptestaddress"
		addressID = uint64(7)
		utxoID    = uint64(1001)
		value     = int64(25)
	)
	indexer := newBaseForUpdate(kv)
	addPendingBaseUtxo(indexer, address, addressID, utxoID, value, 1)

	indexer.UpdateDB()

	var meta common.AddressValueInDBV2
	if err := indexdb.GetValueFromDBWithProto3(indexdb.GetAddressDBKeyV2(address), kv, &meta); err != nil {
		t.Fatalf("read address metadata: %v", err)
	}
	if meta.AddressId != addressID || len(meta.Utxos) != 0 {
		t.Fatalf("metadata = %#v; want stable id and no embedded UTXOs", &meta)
	}

	raw, err := kv.Read(indexdb.GetAddressValueDBKey(addressID, utxoID))
	if err != nil {
		t.Fatalf("read address UTXO: %v", err)
	}
	got, err := indexdb.DecodeAddressUtxoValue(raw)
	if err != nil || got != value {
		t.Fatalf("address UTXO = %d, %v; want %d", got, err, value)
	}
	count, err := indexdb.GetAddressUtxoCountFromDB(kv, addressID)
	if err != nil || count != 1 {
		t.Fatalf("address UTXO count=%d,%v want 1", count, err)
	}
	if _, err := kv.Read(indexdb.GetAddressDBKey(address)); err != common.ErrKeyNotFound {
		t.Fatalf("legacy a-<address> mapping was written, err=%v", err)
	}

	utxos, err := indexer.loadAddressUtxos(kv, addressID)
	if err != nil {
		t.Fatalf("load address UTXOs: %v", err)
	}
	if len(utxos) != 1 || utxos[utxoID] != value {
		t.Fatalf("address UTXOs = %v", utxos)
	}
}

func TestBaseIndexerEmptyAddressBecomesCleanupCandidate(t *testing.T) {
	kv := openBaseTestDB(t)
	const (
		address   = "bc1ptestaddress"
		addressID = uint64(7)
		utxoID    = uint64(1001)
	)

	seed := newBaseForUpdate(kv)
	addPendingBaseUtxo(seed, address, addressID, utxoID, 25, 1)
	seed.UpdateDB()

	remove := newBaseForUpdate(kv)
	remove.addressValueMap[address] = &common.AddressValueV2{
		AddressId:   addressID,
		AddressType: int(txscript.WitnessV1TaprootTy),
		Op:          0,
		UtxoCount:   0,
		Utxos:       make(map[uint64]int64),
	}
	remove.addressUtxoDeleted[addressID] = map[uint64]bool{utxoID: true}
	candidates := remove.UpdateDB()
	if candidates[address] != addressID {
		t.Fatalf("cleanup candidates=%v, want %s:%d", candidates, address, addressID)
	}
	if _, err := kv.Read(indexdb.GetAddressUtxoCountKey(addressID)); err != common.ErrKeyNotFound {
		t.Fatalf("zero count key retained, err=%v", err)
	}

	remove.CleanEmptyAddress(candidates, candidates)
	if _, err := kv.Read(indexdb.GetAddressDBKeyV2(address)); err != common.ErrKeyNotFound {
		t.Fatalf("address metadata retained after cleanup, err=%v", err)
	}
	if _, err := kv.Read(indexdb.GetAddressIdKey(addressID)); err != common.ErrKeyNotFound {
		t.Fatalf("address id mapping retained after cleanup, err=%v", err)
	}
	if _, err := kv.Read(indexdb.GetAddressValueDBKey(addressID, utxoID)); err != common.ErrKeyNotFound {
		t.Fatalf("spent address UTXO retained, err=%v", err)
	}
}

func TestBaseIndexerProtocolCanPreserveEmptyAddressMetadata(t *testing.T) {
	kv := openBaseTestDB(t)
	const (
		address   = "bc1ppreserved"
		addressID = uint64(11)
	)
	indexer := newBaseForUpdate(kv)
	indexer.addressValueMap[address] = &common.AddressValueV2{
		AddressId: addressID, AddressType: int(txscript.WitnessV1TaprootTy), Op: 1,
		UtxoCount: 0, Utxos: make(map[uint64]int64),
	}
	candidates := indexer.UpdateDB()
	delete(candidates, address) // same contract as BRC20Indexer.CheckEmptyAddress
	indexer.CleanEmptyAddress(nil, candidates)

	var meta common.AddressValueInDBV2
	if err := indexdb.GetValueFromDBWithProto3(indexdb.GetAddressDBKeyV2(address), kv, &meta); err != nil {
		t.Fatalf("protocol-preserved metadata missing: %v", err)
	}
	if meta.AddressId != addressID {
		t.Fatalf("preserved address id=%d want %d", meta.AddressId, addressID)
	}
}

func TestProtectLiveAddressDeletionCandidates(t *testing.T) {
	indexer := NewBaseIndexer(nil, &chaincfg.TestNet4Params, 0, 10)
	const address = "bc1preused"
	indexer.addressValueMap[address] = &common.AddressValueV2{
		AddressId: 9, UtxoCount: 1, Utxos: map[uint64]int64{100: 20},
	}
	candidates := map[string]uint64{address: 9, "other": 10}
	indexer.ProtectLiveAddressDeletionCandidates(candidates)
	if _, ok := candidates[address]; ok {
		t.Fatal("live reused address was not protected")
	}
	if candidates["other"] != 10 {
		t.Fatalf("unrelated candidate changed: %v", candidates)
	}
}

func TestBaseIndexerSubtractKeepsOnlyNewAddressDeltas(t *testing.T) {
	source := NewBaseIndexer(nil, &chaincfg.MainNetParams, 0, 100)
	source.utxoIndex = common.NewUTXOIndex()
	source.addressValueMap["address"] = &common.AddressValueV2{
		AddressId: 7,
		Op:        1,
		UtxoCount: 1,
		Utxos:     map[uint64]int64{1001: 25},
	}
	source.addressUtxoDeleted[7] = map[uint64]bool{900: true}

	flushed := source.Clone(true)
	current := source.addressValueMap["address"]
	current.Utxos[1002] = 20
	current.UtxoCount = 2
	source.addressUtxoDeleted[7][901] = true

	source.Subtract(flushed)
	got := source.addressValueMap["address"]
	if got == nil || len(got.Utxos) != 1 || got.Utxos[1002] != 20 {
		t.Fatalf("pending additions = %#v", got)
	}
	if got.UtxoCount != 2 {
		t.Fatalf("logical UTXO count=%d, want 2", got.UtxoCount)
	}
	if source.addressUtxoDeleted[7][900] || !source.addressUtxoDeleted[7][901] {
		t.Fatalf("pending deletions = %v", source.addressUtxoDeleted[7])
	}
}

func TestBaseIndexerUpdateDBCountsOnlyPersistedDeletes(t *testing.T) {
	kv := openBaseTestDB(t)
	if err := kv.Write(indexdb.GetUtxoIdKey(1), []byte("persisted-utxo-key")); err != nil {
		t.Fatalf("seed persisted UTXO id: %v", err)
	}

	indexer := newBaseForUpdate(kv)
	indexer.stats.UtxoCount = 1
	persistedOutput := &common.TxOutputV2{
		TxOutput: common.TxOutput{
			UtxoId:      1,
			OutPointStr: strings.Repeat("0", 64) + ":0",
		},
	}
	persistedOutput.MarkPersisted()
	indexer.delUTXOs = []*common.TxOutputV2{
		persistedOutput,
		{
			TxOutput: common.TxOutput{
				UtxoId:      2,
				OutPointStr: strings.Repeat("1", 64) + ":0",
			},
		},
	}

	indexer.UpdateDB()

	var persisted SyncStats
	if err := indexdb.GetValueFromDB([]byte(SyncStatsKey), &persisted, kv); err != nil {
		t.Fatalf("read persisted stats: %v", err)
	}
	if persisted.UtxoCount != 0 {
		t.Fatalf("UtxoCount=%d, want 0; same-window UTXO delete must not decrement durable count", persisted.UtxoCount)
	}
}
