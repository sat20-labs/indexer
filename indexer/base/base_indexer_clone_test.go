package base

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/sat20-labs/indexer/common"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
	"google.golang.org/protobuf/proto"
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

func TestBaseIndexerClonePreservesAddressMetadata(t *testing.T) {
	source := NewBaseIndexer(nil, &chaincfg.TestNet4Params, 0, 100)
	source.utxoIndex = common.NewUTXOIndex()
	source.addressValueMap["OP_RETURN"] = &common.AddressValueV2{
		AddressId:   41,
		AddressType: int(txscript.NullDataTy),
		Op:          1,
		Utxos: map[uint64]int64{
			1001: 25,
			1002: 50,
		},
	}

	clone := source.Clone(true)
	got := clone.addressValueMap["OP_RETURN"]
	if got == nil {
		t.Fatal("cloned OP_RETURN address is missing")
	}
	if got.AddressId != 41 {
		t.Fatalf("AddressId = %d, want 41", got.AddressId)
	}
	if got.AddressType != int(txscript.NullDataTy) {
		t.Fatalf("AddressType = %d, want NullDataTy(%d)", got.AddressType, txscript.NullDataTy)
	}
	if got.Op != 1 {
		t.Fatalf("clone Op = %d, want snapshot value 1", got.Op)
	}
	if source.addressValueMap["OP_RETURN"].Op != 0 {
		t.Fatalf("source Op = %d, want stored marker 0", source.addressValueMap["OP_RETURN"].Op)
	}

	got.Utxos[1001] = 999
	delete(got.Utxos, 1002)
	if source.addressValueMap["OP_RETURN"].Utxos[1001] != 25 {
		t.Fatal("clone shares the source UTXO map")
	}
	if _, ok := source.addressValueMap["OP_RETURN"].Utxos[1002]; !ok {
		t.Fatal("deleting a cloned UTXO changed the source")
	}
}

func TestBaseIndexerSnapshotAppendsOPReturnAddressState(t *testing.T) {
	kv := indexdb.NewKVDBWithCache(t.TempDir(), 1)
	if kv == nil {
		t.Fatal("open test database")
	}
	t.Cleanup(func() {
		if err := kv.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	const (
		addressID uint64 = 7
		oldUtxoID uint64 = 1001
		newUtxoID uint64 = 1002
	)

	wb := kv.NewWriteBatch()
	existing := &common.AddressValueInDBV2{
		AddressId:   addressID,
		AddressType: int32(txscript.NullDataTy),
		Utxos: []*common.UtxoIdInDB{
			{UtxoId: oldUtxoID, Value: 25},
		},
	}
	if err := indexdb.SetDBWithProto3(
		indexdb.GetAddressDBKeyV2("OP_RETURN"), existing, wb,
	); err != nil {
		wb.Close()
		t.Fatalf("seed OP_RETURN address: %v", err)
	}
	if err := wb.Flush(); err != nil {
		wb.Close()
		t.Fatalf("flush seeded OP_RETURN address: %v", err)
	}
	wb.Close()

	source := NewBaseIndexer(kv, &chaincfg.TestNet4Params, 0, 100)
	source.stats = &SyncStats{ReorgsDetected: make([]int, 0)}
	source.lastHeight = 1
	source.lastHash = "snapshot-hash"
	source.utxoIndex = common.NewUTXOIndex()

	output := common.NewTxOutputV2(20)
	output.UtxoId = newUtxoID
	output.OutPointStr = "0000000000000000000000000000000000000000000000000000000000000001:0"
	output.AddressId = addressID
	output.AddressType = int(txscript.NullDataTy)
	source.utxoIndex.Index[output.OutPointStr] = output
	source.addressValueMap["OP_RETURN"] = &common.AddressValueV2{
		AddressId:   addressID,
		AddressType: int(txscript.NullDataTy),
		Op:          1,
		Utxos: map[uint64]int64{
			newUtxoID: 20,
		},
	}

	snapshot := source.Clone(true)
	snapshot.UpdateDB()

	var got common.AddressValueInDBV2
	if err := indexdb.GetValueFromDBWithProto3(
		indexdb.GetAddressDBKeyV2("OP_RETURN"), kv, &got,
	); err != nil {
		t.Fatalf("read persisted OP_RETURN address: %v", err)
	}
	if got.AddressType != int32(txscript.NullDataTy) {
		t.Fatalf("persisted AddressType = %d, want NullDataTy(%d)", got.AddressType, txscript.NullDataTy)
	}

	utxos := make(map[uint64]int64, len(got.Utxos))
	for _, item := range got.Utxos {
		utxos[item.UtxoId] = item.Value
	}
	if value, ok := utxos[oldUtxoID]; !ok || value != 25 {
		t.Fatalf("historical OP_RETURN UTXO = (%d, %v), want (25, true)", value, ok)
	}
	if value, ok := utxos[newUtxoID]; !ok || value != 20 {
		t.Fatalf("new OP_RETURN UTXO = (%d, %v), want (20, true)", value, ok)
	}
}

func TestBaseIndexerSubtractKeepsOnlyNewOPReturnDelta(t *testing.T) {
	source := NewBaseIndexer(nil, &chaincfg.MainNetParams, 0, 100)
	source.utxoIndex = common.NewUTXOIndex()
	source.addressValueMap["OP_RETURN"] = &common.AddressValueV2{
		AddressId:   7,
		AddressType: int(txscript.NullDataTy),
		Op:          1,
		Utxos: map[uint64]int64{
			1001: 25,
		},
	}

	flushed := source.Clone(true)
	current := source.addressValueMap["OP_RETURN"]
	current.Utxos[1002] = 20
	current.Op = 1

	source.Subtract(flushed)
	got := source.addressValueMap["OP_RETURN"]
	if got == nil {
		t.Fatal("new OP_RETURN delta was removed")
	}
	if len(got.Utxos) != 1 {
		t.Fatalf("pending OP_RETURN UTXOs = %v, want only the new delta", got.Utxos)
	}
	if value, ok := got.Utxos[1002]; !ok || value != 20 {
		t.Fatalf("new OP_RETURN UTXO = (%d, %v), want (20, true)", value, ok)
	}
	if _, ok := got.Utxos[1001]; ok {
		t.Fatal("flushed OP_RETURN UTXO remained in the live delta")
	}
}

func TestAppendAddressUtxosToBytesIsIdempotent(t *testing.T) {
	existing, err := proto.Marshal(&common.AddressValueInDBV2{
		AddressId:   7,
		AddressType: int32(txscript.NullDataTy),
		Utxos: []*common.UtxoIdInDB{
			{UtxoId: 1001, Value: 25},
		},
	})
	if err != nil {
		t.Fatalf("marshal existing address: %v", err)
	}

	updated, err := appendAddressUtxosToBytes(existing, map[uint64]int64{
		1001: 25,
		1002: 20,
	})
	if err != nil {
		t.Fatalf("append address UTXOs: %v", err)
	}

	var got common.AddressValueInDBV2
	if err := proto.Unmarshal(updated, &got); err != nil {
		t.Fatalf("unmarshal updated address: %v", err)
	}
	if len(got.Utxos) != 2 {
		t.Fatalf("persisted UTXO records = %d, want 2: %v", len(got.Utxos), got.Utxos)
	}
	counts := make(map[uint64]int)
	values := make(map[uint64]int64)
	for _, item := range got.Utxos {
		counts[item.UtxoId]++
		values[item.UtxoId] = item.Value
	}
	if counts[1001] != 1 || values[1001] != 25 {
		t.Fatalf("existing UTXO count/value = %d/%d, want 1/25", counts[1001], values[1001])
	}
	if counts[1002] != 1 || values[1002] != 20 {
		t.Fatalf("new UTXO count/value = %d/%d, want 1/20", counts[1002], values[1002])
	}

	if _, err := appendAddressUtxosToBytes(existing, map[uint64]int64{1001: 26}); err == nil {
		t.Fatal("value mismatch for an existing address UTXO was accepted")
	}
}

func TestBaseIndexerServiceSnapshotsDoNotRepeatOPReturnDelta(t *testing.T) {
	kv := indexdb.NewKVDBWithCache(t.TempDir(), 1)
	if kv == nil {
		t.Fatal("open test database")
	}
	t.Cleanup(func() {
		if err := kv.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	live := NewBaseIndexer(kv, &chaincfg.MainNetParams, 0, 100)
	live.utxoIndex = common.NewUTXOIndex()
	live.stats = &SyncStats{}
	live.addressValueMap["OP_RETURN"] = &common.AddressValueV2{
		AddressId:   7,
		AddressType: int(txscript.NullDataTy),
		Op:          1,
		Utxos: map[uint64]int64{
			1001: 25,
		},
	}

	first := live.Clone(true)
	current := live.addressValueMap["OP_RETURN"]
	current.Utxos[1002] = 20
	current.Op = 1

	// This is the service-mode order: remove the flushed snapshot from the
	// live buffer, persist it, then create/persist the next snapshot.
	live.Subtract(first)
	first.UpdateDB()
	second := live.Clone(true)
	second.UpdateDB()

	var got common.AddressValueInDBV2
	if err := indexdb.GetValueFromDBWithProto3(
		indexdb.GetAddressDBKeyV2("OP_RETURN"), kv, &got,
	); err != nil {
		t.Fatalf("read persisted OP_RETURN address: %v", err)
	}
	if len(got.Utxos) != 2 {
		t.Fatalf("persisted OP_RETURN records = %d, want 2: %v", len(got.Utxos), got.Utxos)
	}
	counts := make(map[uint64]int)
	var total int64
	for _, item := range got.Utxos {
		counts[item.UtxoId]++
		total += item.Value
	}
	if counts[1001] != 1 || counts[1002] != 1 {
		t.Fatalf("persisted OP_RETURN counts = %v, want one record per UTXO", counts)
	}
	if total != 45 {
		t.Fatalf("persisted OP_RETURN sats = %d, want 45", total)
	}
}
