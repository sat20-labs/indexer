package base

import (
	"testing"

	indexdb "github.com/sat20-labs/indexer/indexer/db"
)

func TestScanPersistedAddressUtxosCountsZeroOnlyAddress(t *testing.T) {
	kv := openBaseTestDB(t)
	wb := kv.NewWriteBatch()
	defer wb.Close()

	entries := []struct {
		addressID uint64
		utxoID    uint64
		value     int64
	}{
		{addressID: 7, utxoID: 100, value: 0},
		{addressID: 8, utxoID: 200, value: 25},
		{addressID: 8, utxoID: 201, value: 0},
	}
	for _, entry := range entries {
		encoded, err := indexdb.EncodeAddressUtxoValue(entry.value)
		if err != nil {
			t.Fatal(err)
		}
		if err := wb.Put(indexdb.GetAddressValueDBKey(entry.addressID, entry.utxoID), encoded); err != nil {
			t.Fatal(err)
		}
	}
	if err := wb.Flush(); err != nil {
		t.Fatal(err)
	}

	summary, err := scanPersistedAddressUtxos(kv)
	if err != nil {
		t.Fatal(err)
	}
	if summary.AllUtxos != 3 || summary.NonZeroUtxos != 1 || summary.TotalSats != 25 {
		t.Fatalf("unexpected UTXO summary: %#v", summary)
	}
	if summary.AllAddresses != 2 {
		t.Fatalf("all-address count=%d, want 2", summary.AllAddresses)
	}
	if len(summary.NonZeroAddresses) != 1 || !summary.NonZeroAddresses[8] {
		t.Fatalf("non-zero addresses=%v, want only address 8", summary.NonZeroAddresses)
	}
}
