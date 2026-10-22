package db

import (
	"errors"
	"testing"

	"github.com/sat20-labs/indexer/common"
)

func TestAtomicWriteBatchCommitAndAbort(t *testing.T) {
	database := NewKVDBWithOptions(t.TempDir(), OpenOptions{BlockCacheMB: 1, IndexCacheMB: 1})
	if database == nil {
		t.Fatal("database unavailable")
	}
	defer database.Close()
	atomic, ok := database.(common.AtomicWriteBatchDB)
	if !ok {
		t.Fatal("backend has no atomic transaction capability")
	}
	if err := database.Write([]byte("existing"), []byte("before")); err != nil {
		t.Fatal(err)
	}
	for _, commit := range []bool{false, true} {
		batch := atomic.NewAtomicWriteBatch()
		if err := batch.Put([]byte("ledger"), []byte("settled")); err != nil {
			t.Fatal(err)
		}
		if err := batch.Delete([]byte("existing")); err != nil {
			t.Fatal(err)
		}
		if _, err := database.Read([]byte("ledger")); !errors.Is(err, common.ErrKeyNotFound) {
			t.Fatalf("Put published before commit: %v", err)
		}
		if value, err := database.Read([]byte("existing")); err != nil || string(value) != "before" {
			t.Fatalf("Delete published before commit: %q, %v", value, err)
		}
		if commit {
			if err := batch.Flush(); err != nil {
				t.Fatal(err)
			}
		}
		batch.Close()
	}
	if value, err := database.Read([]byte("ledger")); err != nil || string(value) != "settled" {
		t.Fatalf("ledger not committed: %q, %v", value, err)
	}
	if _, err := database.Read([]byte("existing")); !errors.Is(err, common.ErrKeyNotFound) {
		t.Fatalf("delete not committed: %v", err)
	}
}

// Exercises the old bulk-writer's large-record fast path. An oversized atomic
// transaction may fail, but must never publish a preceding ledger or the large
// record during Put. Closing the candidate always leaves the database intact.
func TestAtomicWriteBatchLargeRecordDoesNotCommitDuringPut(t *testing.T) {
	database := NewKVDBWithOptions(t.TempDir(), OpenOptions{BlockCacheMB: 1, IndexCacheMB: 1})
	if database == nil {
		t.Fatal("database unavailable")
	}
	defer database.Close()
	atomic, ok := database.(common.AtomicWriteBatchDB)
	if !ok {
		t.Fatal("backend has no atomic transaction capability")
	}
	batch := atomic.NewAtomicWriteBatch()
	if err := batch.Put([]byte("ledger"), []byte("candidate")); err != nil {
		t.Fatal(err)
	}
	_ = batch.Put([]byte("proof"), make([]byte, 64<<20))
	for _, key := range []string{"ledger", "proof"} {
		if _, err := database.Read([]byte(key)); !errors.Is(err, common.ErrKeyNotFound) {
			t.Fatalf("Put published %s: %v", key, err)
		}
	}
	batch.Close()
	for _, key := range []string{"ledger", "proof"} {
		if _, err := database.Read([]byte(key)); !errors.Is(err, common.ErrKeyNotFound) {
			t.Fatalf("aborted transaction retained %s: %v", key, err)
		}
	}
}
