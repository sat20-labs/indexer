package indexer

import (
	"context"

	base "github.com/sat20-labs/indexer/indexer/base"
)

// Public SilentPay reads use the same admission barrier as other index reads.
// The Base methods pin committed DB versions; no hot compilation maps escape.
func (b *IndexerMgr) SilentPayStatus(ctx context.Context) (base.SPStatus, error) {
	b.rpcEnter()
	defer b.rpcLeft()
	if b.base == nil || b.baseDB == nil {
		return base.SPStatus{}, base.ErrSPNotReady
	}
	return b.base.SilentPayStatus(ctx)
}

func (b *IndexerMgr) SilentPayCreateSnapshot(ctx context.Context) (base.SPSnapshot, error) {
	b.rpcEnter()
	defer b.rpcLeft()
	if b.base == nil || b.baseDB == nil {
		return base.SPSnapshot{}, base.ErrSPNotReady
	}
	return b.base.SilentPayCreateSnapshot(ctx)
}

func (b *IndexerMgr) SilentPayScan(ctx context.Context, request base.SPScanRequest) (base.SPScanPage, error) {
	b.rpcEnter()
	defer b.rpcLeft()
	if b.base == nil || b.baseDB == nil {
		return base.SPScanPage{}, base.ErrSPNotReady
	}
	return b.base.SilentPayScan(ctx, request)
}

func (b *IndexerMgr) SilentPayLookup(ctx context.Context, positions []base.SPPosition) ([]base.SPPositionResult, error) {
	b.rpcEnter()
	defer b.rpcLeft()
	if b.base == nil || b.baseDB == nil {
		return nil, base.ErrSPNotReady
	}
	return b.base.SilentPayLookup(ctx, positions)
}

func (b *IndexerMgr) SilentPayOutpoints(ctx context.Context, snapshotID string, positions []base.SPPosition) (base.SPAnchor, []base.SPPositionResult, error) {
	b.rpcEnter()
	defer b.rpcLeft()
	if b.base == nil || b.baseDB == nil {
		return base.SPAnchor{}, nil, base.ErrSPNotReady
	}
	return b.base.SilentPayOutpoints(ctx, snapshotID, positions)
}
