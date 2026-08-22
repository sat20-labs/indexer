//go:build !pebble

package db

import (
	"bytes"
	"testing"

	"github.com/sat20-labs/indexer/common"
)

func TestBadgerReadBatchGetRefReturnsOwnedBytes(t *testing.T) {
	path := t.TempDir()
	raw, err := openBadgerDB(path, 1)
	if err != nil {
		t.Fatalf("open Badger: %v", err)
	}
	database := &badgerDB{path: path, db: raw}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("close Badger: %v", err)
		}
	})

	key := []byte("owned-value")
	want := []byte("persistent-value")
	if err := database.Write(key, want); err != nil {
		t.Fatalf("write value: %v", err)
	}

	err = database.View(func(txn common.ReadBatch) error {
		got, err := txn.GetRef(key)
		if err != nil {
			return err
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("GetRef = %q, want %q", got, want)
		}

		got[0] = 'X'
		reread, err := txn.Get(key)
		if err != nil {
			return err
		}
		if !bytes.Equal(reread, want) {
			t.Fatalf("mutating GetRef result changed transaction value: %q", reread)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view: %v", err)
	}

	after, err := database.Read(key)
	if err != nil {
		t.Fatalf("read after View: %v", err)
	}
	if !bytes.Equal(after, want) {
		t.Fatalf("stored value changed: %q", after)
	}
}
