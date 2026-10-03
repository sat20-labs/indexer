package base

import (
	"testing"

	"github.com/btcsuite/btcd/txscript"
	"github.com/sat20-labs/indexer/common"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
)

func TestBaseSnapshotFlushMarksSpentLiveOutputPersisted(t *testing.T) {
	kv := openBaseTestDB(t)
	const (
		address   = "UQ==" // Base64 of the nonstandard OP_TRUE script.
		addressID = uint64(7)
		utxoID    = uint64(1001)
		value     = int64(25)
	)

	live := newBaseForUpdate(kv)
	addPendingBaseUtxo(live, address, addressID, utxoID, value, 1)

	var output *common.TxOutputV2
	for _, item := range live.utxoIndex.Index {
		output = item
		break
	}
	if output == nil {
		t.Fatal("pending output not found")
	}
	output.OutValue.PkScript = []byte{txscript.OP_TRUE}
	output.AddressType = int(txscript.NonStandardTy)
	live.addressValueMap[address].AddressType = output.AddressType

	// Delayed DB buffering clones the pending state before later blocks are
	// processed. TxOutputV2 pointers are intentionally shared so a successful
	// snapshot flush can mark the corresponding live output durable.
	snapshot := live.Clone(true)

	// Simulate a later block spending the output after the snapshot was taken
	// but before that snapshot was flushed.
	delete(live.utxoIndex.Index, output.OutPointStr)
	live.delUTXOs = append(live.delUTXOs, output)
	live.inputUtxo(&common.TxInput{TxOutputV2: *output})

	// Match performUpdateDBInBuffer: subtract before flushing the snapshot.
	live.Subtract(snapshot)

	snapshot.UpdateDB()
	if !output.IsPersisted() {
		t.Fatal("snapshot flush did not mark the shared live output persisted")
	}

	// This is the same stats hand-off performed by the delayed DB-buffer path.
	live.SetSyncStats(snapshot.GetSyncStats())
	if len(live.delUTXOs) != 1 || !live.delUTXOs[0].IsPersisted() {
		t.Fatalf("live persisted deletion was lost: %#v", live.delUTXOs)
	}
	if !live.addressUtxoDeleted[addressID][utxoID] {
		t.Fatal("address UTXO deletion was lost after subtracting snapshot")
	}

	candidates := live.UpdateDB()
	var stats SyncStats
	if err := indexdb.GetValueFromDB([]byte(SyncStatsKey), &stats, kv); err != nil {
		t.Fatalf("read sync stats: %v", err)
	}
	if stats.UtxoCount != 0 {
		t.Fatalf("UtxoCount=%d, want 0 after snapshot-created output is later spent", stats.UtxoCount)
	}
	if _, err := kv.Read(indexdb.GetAddressValueDBKeyCompact(addressID, utxoID)); err != common.ErrKeyNotFound {
		t.Fatalf("spent address UTXO still exists, read err=%v", err)
	}
	if _, err := kv.Read(indexdb.GetAddressUtxoCountKey(addressID)); err != common.ErrKeyNotFound {
		t.Fatalf("empty address retains a count key, read err=%v", err)
	}
	if candidates[address] != addressID {
		t.Fatalf("empty address missing from cleanup candidates: %v", candidates)
	}
	live.CleanEmptyAddress(nil, candidates)
	for _, key := range [][]byte{indexdb.GetAddressDBKeyV2(address), indexdb.GetAddressIdKey(addressID)} {
		if _, err := kv.Read(key); err != common.ErrKeyNotFound {
			t.Fatalf("empty address retains metadata %s, read err=%v", key, err)
		}
	}
}
