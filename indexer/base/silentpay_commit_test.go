package base

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/indexer/common"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
)

var errSPInjectedIO = errors.New("injected SilentPay persistence failure")

// spFailureDB deliberately commits each batch entry separately. This models
// a backend that splits a large WriteBatch into independently committed parts.
// Tests must not assume that Flush failure rolls back already written entries.
type spFailureDB struct {
	common.KVDB
	phase     string
	syncCalls int
	putCalls  int
	injected bool
}

func (d *spFailureDB) NewReadView() (common.DBReadView, error) {
	return d.KVDB.(common.SnapshotDB).NewReadView()
}

func (d *spFailureDB) Sync() error {
	d.syncCalls++
	if (d.phase == "fence_sync" && d.syncCalls == 1) ||
		(d.phase == "data_sync" && d.syncCalls == 2) {
		d.injected = true
		return errSPInjectedIO
	}
	return d.KVDB.(common.SnapshotDB).Sync()
}

func (d *spFailureDB) Delete(key []byte) error {
	if d.phase == "fence_delete" && string(key) == string(spPendingKey) {
		d.injected = true
		return errSPInjectedIO
	}
	return d.KVDB.Delete(key)
}

func (d *spFailureDB) NewWriteBatch() common.WriteBatch {
	return &spSplitFailureBatch{db: d}
}

type spSplitFailureBatch struct {
	db *spFailureDB
}

func (b *spSplitFailureBatch) Put(key, value []byte) error {
	b.db.putCalls++
	if b.db.phase == "split_batch_put" && b.db.putCalls == 2 {
		b.db.injected = true
		return errSPInjectedIO
	}
	return b.db.KVDB.Write(key, value)
}

func (b *spSplitFailureBatch) Delete(key []byte) error {
	return b.db.KVDB.Delete(key)
}

func (b *spSplitFailureBatch) Flush() error {
	if b.db.phase == "batch_flush" {
		b.db.injected = true
		return errSPInjectedIO
	}
	return b.db.KVDB.(common.SnapshotDB).Sync()
}

func (b *spSplitFailureBatch) Close() {}

func attemptSPTestCommit(s *silentPayState, anchor SPAnchor) error {
	if err := s.beginCommit(anchor.Height, anchor.BlockHash); err != nil {
		return err
	}
	wb := s.db.NewWriteBatch()
	defer wb.Close()
	if err := s.writeTo(wb); err != nil {
		return err
	}
	if err := indexdb.SetDB([]byte(SyncStatsKey), &SyncStats{
		SyncHeight: anchor.Height, SyncBlockHash: anchor.BlockHash,
	}, wb); err != nil {
		return err
	}
	if err := wb.Flush(); err != nil {
		return err
	}
	return s.finishCommit()
}

func TestSilentPayPersistenceFailuresNeverPublishPartialState(t *testing.T) {
	for _, phase := range []string{"fence_sync", "split_batch_put", "batch_flush", "data_sync", "fence_delete"} {
		t.Run(phase, func(t *testing.T) {
			s := newSPTestState(t)
			first := spTestCandidate(0, 11)
			if err := s.process(first); err != nil {
				t.Fatal(err)
			}
			committed := SPAnchor{Height: 0, BlockHash: first.Hash}
			spCommitState(t, s, committed)
			oldView, oldAnchor, err := s.freshView(context.Background())
			if err != nil || oldAnchor != committed {
				t.Fatalf("baseline view: %+v %v", oldAnchor, err)
			}
			defer oldView.Close()

			second := spTestCandidate(1, 12)
			if err := s.process(second); err != nil {
				t.Fatal(err)
			}
			underlying := s.db
			failing := &spFailureDB{KVDB: underlying, phase: phase}
			s.db = failing
			if err := attemptSPTestCommit(s, SPAnchor{Height: 1, BlockHash: second.Hash}); !errors.Is(err, errSPInjectedIO) {
				t.Fatalf("failure was not exercised at %s: %v", phase, err)
			}
			if _, err := underlying.Read(spPendingKey); err != nil {
				t.Fatalf("partial write lost its durable fence: %v", err)
			}
			if phase == "split_batch_put" {
				if _, err := underlying.Read(spBlockKey(1)); err != nil {
					t.Fatalf("test did not persist the first sub-batch: %v", err)
				}
				if _, err := underlying.Read(spPositionKey(spArchivePrefix, 1, 1)); !errors.Is(err, common.ErrKeyNotFound) {
					t.Fatalf("second sub-batch unexpectedly persisted: %v", err)
				}
			}
			if view, _, err := s.freshView(context.Background()); !errors.Is(err, ErrSPIncompleteCommit) {
				if view != nil {
					view.Close()
				}
				t.Fatalf("partial write became queryable: %v", err)
			}
			// Even when the new SyncStats happened to be written before failure,
			// a newly initialized component must reject the marked database.
			reloaded := newSilentPayState(underlying, &chaincfg.RegressionNetParams)
			defer reloaded.views.close()
			if err := reloaded.validateInitialState(&SyncStats{SyncHeight: 1, SyncBlockHash: second.Hash}); !errors.Is(err, ErrSPIncompleteCommit) {
				t.Fatalf("restart trusted a partial commit: %v", err)
			}
			if anchor, err := spReadAnchor(oldView, s.network); err != nil || anchor != committed {
				t.Fatalf("previously pinned complete view changed: %+v %v", anchor, err)
			}
			if _, err := oldView.Read(spBlockKey(1)); !errors.Is(err, common.ErrKeyNotFound) {
				t.Fatalf("old view exposed part of failed update: %v", err)
			}
		})
	}
}

func TestSilentPaySplitBatchSuccessfulCommitPublishesTogether(t *testing.T) {
	s := newSPTestState(t)
	first := spTestCandidate(0, 13)
	if err := s.process(first); err != nil {
		t.Fatal(err)
	}
	s.db = &spFailureDB{KVDB: s.db}
	anchor := SPAnchor{Height: 0, BlockHash: first.Hash}
	if err := attemptSPTestCommit(s, anchor); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Read(spPendingKey); !errors.Is(err, common.ErrKeyNotFound) {
		t.Fatalf("successful commit kept pending marker: %v", err)
	}
	view, actual, err := s.freshView(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer view.Close()
	if actual != anchor {
		t.Fatalf("anchor mismatch: %+v", actual)
	}
	if _, err := spReadTransaction(view, 0, 1); err != nil {
		t.Fatal(err)
	}
	if bits, err := view.Read(spPositionKey(spLivePrefix, 0, 1)); err != nil || len(bits) != 1 || bits[0] != 3 {
		t.Fatalf("incomplete Live after commit: %x %v", bits, err)
	}
	if len(s.blocks) != 0 || len(s.live) != 0 {
		t.Fatal("successful commit retained already persisted deltas")
	}
}


 // Base.UpdateDB must not return to its caller after a failed persist.
 // The durable pending marker is checked before any automatic retry/restart.
func TestSilentPayBaseUpdateDBPanicsAfterPersistenceFailure(t *testing.T) {
	for _, phase := range []string{"fence_sync", "split_batch_put", "batch_flush", "data_sync", "fence_delete"} {
		t.Run(phase, func(t *testing.T) {
			original := newSPTestState(t)
			failing := &spFailureDB{KVDB: original.db, phase: phase}
			b := NewBaseIndexer(failing, &chaincfg.RegressionNetParams, 0, 10)
			b.Init() // production initialization: UTXO index and SP state
			t.Cleanup(b.CloseSilentPayViews)
			block := spTestCandidate(0, 29)
			if err := b.silentpay.process(block); err != nil { t.Fatal(err) }
			b.lastHeight, b.lastHash = block.Height, block.Hash

			var panicValue any
			func() {
				defer func() { panicValue = recover() }()
				b.UpdateDB()
			}()
			if panicValue == nil {
				t.Fatal("BaseIndexer continued after persistence failure")
			}
			if !strings.Contains(fmt.Sprint(panicValue), errSPInjectedIO.Error()) {
				t.Fatalf("panic did not originate at injected failure: %v", panicValue)
			}
			if !failing.injected {
				t.Fatalf("injected phase %s was never reached", phase)
			}
			if _, err := failing.Read(spPendingKey); err != nil {
				t.Fatalf("failed commit did not retain durable fence: %v", err)
			}
		})
	}
}
