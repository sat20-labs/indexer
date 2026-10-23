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
		if err := wb.Put(spPositionKey(spLivePrefix, i/13, uint32(i%13+1)), []byte{byte(i % 251), 1}); err != nil {
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
	if err := s.db.Delete(spPositionKey(spLivePrefix, 0, 1)); err != nil {
		t.Fatal(err)
	}
	if err := s.db.Write(spPositionKey(spLivePrefix, 10000, 1), []byte{1}); err != nil {
		t.Fatal(err)
	}

	var total, previousHeight int
	var previousIndex uint32
	livePager := spAuditPager{view: view, prefix: spLivePrefix}
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
		liveKey, liveValue, err := livePager.next(context.Background())
		if err != nil || !bytes.Equal(liveKey, spPositionKey(spLivePrefix, h, index)) || !bytes.Equal(value, liveValue) {
			t.Fatalf("interleaved Live pager changed snapshot or overwrote Archive bytes: %v", err)
		}
		total++
		previousHeight, previousIndex = h, index
		return nil
	})
	if err != nil || total != count {
		t.Fatalf("paged DB audit visited %d/%d records: %v", total, count, err)
	}
	if key, _, err := livePager.next(context.Background()); err != nil || key != nil {
		t.Fatalf("unexpected trailing Live: %x, %v", key, err)
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

type spAuditScanView struct {
	common.DBReadView
	scans   int
	failAt  int
	failure error
}

func (v *spAuditScanView) Scan(options common.ScanOptions, visit func([]byte, []byte) error) error {
	v.scans++
	if err := v.DBReadView.Scan(options, visit); err != nil {
		return err
	}
	if v.scans == v.failAt {
		return v.failure
	}
	return nil
}

func TestSilentPayDBAuditBytePagesAndScanErrors(t *testing.T) {
	s := newSPTestState(t)
	wb := s.db.NewWriteBatch()
	const count = 7
	for i := 0; i < count; i++ {
		value := bytes.Repeat([]byte{byte(i + 1)}, 2<<20)
		if err := wb.Put(spPositionKey(spArchivePrefix, 0, uint32(i+1)), value); err != nil {
			t.Fatal(err)
		}
	}
	if err := wb.Flush(); err != nil {
		t.Fatal(err)
	}
	wb.Close()
	view, err := s.db.(common.SnapshotDB).NewReadView()
	if err != nil {
		t.Fatal(err)
	}
	defer view.Close()
	counted := &spAuditScanView{DBReadView: view}
	visited := 0
	err = spAuditEach(context.Background(), counted, spArchivePrefix, func(key, value []byte) error {
		visited++
		if !bytes.Equal(key, spPositionKey(spArchivePrefix, 0, uint32(visited))) || len(value) != 2<<20 {
			t.Fatal("byte-bounded paging lost or reordered a row")
		}
		for _, b := range value {
			if b != byte(visited) {
				t.Fatal("page buffer corrupted record bytes")
			}
		}
		return nil
	})
	if err != nil || visited != count || counted.scans != 2 {
		t.Fatalf("byte pages: visited=%d scans=%d err=%v", visited, counted.scans, err)
	}

	failure := errors.New("injected iterator failure after copied rows")
	counted.scans, counted.failAt, counted.failure = 0, 2, failure
	visited = 0
	err = spAuditEach(context.Background(), counted, spArchivePrefix, func(_, _ []byte) error { visited++; return nil })
	if !errors.Is(err, failure) || visited != 4 {
		t.Fatalf("scan failure swallowed or partial page published: visited=%d err=%v", visited, err)
	}
}
