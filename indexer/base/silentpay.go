package base

import (
	"context"
	"errors"
	"fmt"

	"github.com/sat20-labs/indexer/common"
)

// SilentPay is a public derived index. Client errors and expiring snapshots
// remain ordinary RPC errors; corruption or storage failures are fatal to the
// Indexer, as they are in the rest of BaseIndexer. Persistence failures are
// guarded by sp/commit-pending before they reach BaseIndexer.UpdateDB.
func panicOnSilentPayDataFailure(operation string, err error) {
	switch {
	case err == nil,
		errors.Is(err, ErrSPInvalidRequest),
		errors.Is(err, ErrSPSnapshotExpired),
		errors.Is(err, ErrSPSnapshotLimit),
		errors.Is(err, ErrSPNotReady),
		errors.Is(err, ErrSPNotInitialized),
		errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded):
		return
	default:
		panic(fmt.Errorf("BaseIndexer SilentPay %s: %w", operation, err))
	}
}

func (b *BaseIndexer) initSilentPay() error {
	if b.silentpay != nil {
		b.silentpay.views.close()
		b.silentpay = nil
	}
	state := newSilentPayState(b.db, b.chaincfgParam)
	if err := state.validateInitialState(b.stats); err != nil {
		state.views.close()
		if errors.Is(err, ErrSPNotInitialized) {
			common.Log.Warnf("SilentPay is not initialized for existing Base DB at height %d; other indexing continues without SP", b.stats.SyncHeight)
			return nil
		}
		return err
	}
	b.silentpay = state
	return nil
}

func (b *BaseIndexer) processSilentPay(block *common.Block) error {
	if b.silentpay == nil {
		return nil // existing Base without SP history continues to index normally
	}
	if block == nil || block.Height != b.lastHeight+1 {
		return fmt.Errorf("%w: non-contiguous Base/SP processing", ErrSPCorrupt)
	}
	return b.silentpay.process(block)
}

func (b *BaseIndexer) beginSilentPayCommit() error {
	// Isolated legacy unit fixtures can deliberately bypass Init. Production
	// block processing always requires an initialized SilentPay component.
	if b.silentpay == nil {
		return nil
	}
	if err := b.silentpay.checkPendingRange(b.stats.SyncHeight, b.lastHeight, b.lastHash); err != nil {
		return err
	}
	return b.silentpay.beginCommit(b.lastHeight, b.lastHash)
}

func (b *BaseIndexer) writeSilentPay(wb common.WriteBatch) error {
	if b.silentpay == nil {
		return nil
	}
	return b.silentpay.writeTo(wb)
}

func (b *BaseIndexer) finishSilentPayCommit() error {
	if b.silentpay == nil {
		return nil
	}
	return b.silentpay.finishCommit()
}

// CloseSilentPayViews runs after RPC readers drain and before the Base DB
// closes, including reorg, reload and CheckSelf shutdown paths.
func (b *BaseIndexer) CloseSilentPayViews() {
	if b != nil && b.silentpay != nil {
		b.silentpay.views.close()
	}
}

func (b *BaseIndexer) CheckSilentPay(ctx context.Context) error {
	if b.silentpay == nil {
		return ErrSPNotInitialized
	}
	err := b.silentpay.audit(ctx, SPAnchor{Height: b.stats.SyncHeight, BlockHash: b.stats.SyncBlockHash})
	// Database corruption or cancelled audits are still fatal. The routine
	// intentionally never consults Bitcoin Core; independent T recomputation
	// belongs to E2E, not this local consistency check.
	if err != nil {
		panic(fmt.Errorf("BaseIndexer SilentPay CheckSelf failed: %w", err))
	}
	return nil
}

func (b *BaseIndexer) SilentPayStatus(ctx context.Context) (SPStatus, error) {
	if b.silentpay == nil {
		return SPStatus{}, ErrSPNotInitialized
	}
	result, err := b.silentpay.status(ctx)
	panicOnSilentPayDataFailure("status", err)
	return result, err
}

func (b *BaseIndexer) SilentPayCreateSnapshot(ctx context.Context) (SPSnapshot, error) {
	if b.silentpay == nil {
		return SPSnapshot{}, ErrSPNotInitialized
	}
	result, err := b.silentpay.createSnapshot(ctx)
	panicOnSilentPayDataFailure("create snapshot", err)
	return result, err
}

func (b *BaseIndexer) SilentPayScan(ctx context.Context, req SPScanRequest) (SPScanPage, error) {
	if b.silentpay == nil {
		return SPScanPage{}, ErrSPNotInitialized
	}
	result, err := b.silentpay.scan(ctx, req)
	panicOnSilentPayDataFailure("scan", err)
	return result, err
}

func (b *BaseIndexer) SilentPayLookup(ctx context.Context, positions []SPPosition) ([]SPPositionResult, error) {
	if b.silentpay == nil {
		return nil, ErrSPNotInitialized
	}
	result, err := b.silentpay.lookup(ctx, positions)
	panicOnSilentPayDataFailure("lookup", err)
	return result, err
}

// SilentPayOutpoints reconciles at the explicitly requested immutable snapshot.
// It never substitutes current bitcoind/mempool state for that snapshot.
func (b *BaseIndexer) SilentPayOutpoints(ctx context.Context, snapshotID string, positions []SPPosition) (SPAnchor, []SPPositionResult, error) {
	if b.silentpay == nil {
		return SPAnchor{}, nil, ErrSPNotInitialized
	}
	anchor, result, err := b.silentpay.snapshotOutpoints(ctx, snapshotID, positions)
	panicOnSilentPayDataFailure("outpoints", err)
	return anchor, result, err
}
