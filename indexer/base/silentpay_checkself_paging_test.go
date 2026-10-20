package base

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/sat20-labs/indexer/common"
)

// Paging must keep a single pinned version and release Scan's iterator lock
// before reading associated SP and Base keys. In particular, this guards
// against deadlocks in both Pebble and Badger view implementations.
func TestSilentPayDBAuditPagesReadSamePinnedSnapshot(t *testing.T) {
	s := newSPTestState(t)
	wb := s.db.NewWriteBatch()
	const count = 2500
	for i := 0; i < count; i++ {
		key := spPositionKey(spArchivePrefix, i/13, uint32(i%13+1))
		if err := wb.Put(key, []byte{byte(i % 251), 1}); err != nil {
			t.Fatal(err)
		}
	}
	if err := wb.Flush(); err != nil {
		t.Fatal(err)
	}
	wb.Close()
	capability := s.db.(common.SnapshotDB)
	view, err := capability.NewReadView()
	if err != nil {
		t.Fatal(err)
	}
	defer view.Close()

	// Writes after pinning the snapshot must not affect this multi-page audit.
	// This is important when normal Base block commits continue concurrently.
	firstKey := spPositionKey(spArchivePrefix, 0, 1)
	if err := s.db.Delete(firstKey); err != nil {
		t.Fatal(err)
	}
	extraKey := spPositionKey(spArchivePrefix, 10000, 1)
	if err := s.db.Write(extraKey, []byte("later uncommitted SP record")); err != nil {
		t.Fatal(err)
	}

	var total, previousHeight int
	var previousIndex uint32
	err = spAuditEach(context.Background(), view, spArchivePrefix, func(key, value []byte) error {
		h, index, err := spParsePosition(spArchivePrefix, key)
		if err != nil {
			return err
		}
		if total > 0 && (h < previousHeight || (h == previousHeight && index <= previousIndex)) {
			t.Fatal("paged SP Archive iteration duplicated or reordered a record")
		}
		// This would deadlock if the current database Scan still held its
		// read-view mutex when the visitor was called.
		again, err := view.Read(key)
		if err != nil || !bytes.Equal(value, again) {
			t.Fatalf("paged read view returned inconsistent records: %v", err)
		}
		total++
		previousHeight, previousIndex = h, index
		return nil
	})
	if err != nil || total != count {
		t.Fatalf("paged DB audit visited %d/%d records: %v", total, count, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	visited := 0
	err = spAuditEach(ctx, view, spArchivePrefix, func(_, _ []byte) error {
		visited++
		if visited == 1200 {
			cancel()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) || visited != 1200 {
		t.Fatalf("DB audit ignored cancellation at %d: %v", visited, err)
	}
}
