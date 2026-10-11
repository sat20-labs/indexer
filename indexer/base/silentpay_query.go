package base

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/sat20-labs/indexer/common"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
)

func (s *silentPayState) freshView(ctx context.Context) (common.DBReadView, SPAnchor, error) {
	if err := ctx.Err(); err != nil {
		return nil, SPAnchor{}, err
	}
	capability, ok := s.db.(common.SnapshotDB)
	if !ok {
		return nil, SPAnchor{}, ErrSPNotReady
	}
	view, err := capability.NewReadView()
	if err != nil {
		return nil, SPAnchor{}, err
	}
	anchor, err := spReadAnchor(view, s.network)
	if err == nil && anchor.Height >= s.startHeight {
		raw, e := view.Read(spBlockKey(anchor.Height))
		if e != nil {
			err = fmt.Errorf("%w: committed block missing: %v", ErrSPCorrupt, e)
		} else {
			hash, _, e := spDecodeBlock(raw)
			if e != nil || hash != anchor.BlockHash {
				err = fmt.Errorf("%w: committed block hash", ErrSPCorrupt)
			}
		}
	}
	if err != nil {
		view.Close()
		return nil, SPAnchor{}, err
	}
	return view, anchor, nil
}

func spReadBlockHeader(view common.DBReadView, height int) (string, []uint32, error) {
	raw, err := view.Read(spBlockKey(height))
	if err != nil {
		return "", nil, fmt.Errorf("%w: missing block %d: %v", ErrSPCorrupt, height, err)
	}
	return spDecodeBlock(raw)
}

func spReadTransaction(view common.DBReadView, height int, index uint32) (SPTransaction, error) {
	raw, err := view.Read(spPositionKey(spArchivePrefix, height, index))
	if err != nil {
		return SPTransaction{}, fmt.Errorf("%w: missing tx %d/%d: %v", ErrSPCorrupt, height, index, err)
	}
	return spDecodeTransaction(raw, index)
}

func (s *silentPayState) status(ctx context.Context) (SPStatus, error) {
	result := SPStatus{
		Network: s.network,
		SchemaVersion: SPSchemaVersion,
		CoverageStartHeight: s.startHeight,
		BaseAnchor: SPAnchor{Height: -1},
		SupportedModes: []string{"archive", "live"},
	}
	view, anchor, err := s.freshView(ctx)
	if err != nil { return result, err }
	defer view.Close()
	result.BaseAnchor = anchor
	result.DataReady = true
	// Keep /sp/status strictly local. An external GetRawTx probe without
	// context cancellation used to hold IndexerMgr.rpcAdmission across a
	// potentially long bitcoind call. Actual transaction availability is
	// checked by the existing rawtx/batch API at scan-hit time.
	return result, ctx.Err()
}

func (s *silentPayState) createSnapshot(ctx context.Context) (SPSnapshot, error) {
	view, anchor, err := s.freshView(ctx)
	if err != nil {
		return SPSnapshot{}, err
	}
	if err = ctx.Err(); err != nil {
		view.Close()
		return SPSnapshot{}, err
	}
	return s.views.add(view, anchor, s.startHeight)
}

func (s *silentPayState) scan(ctx context.Context, req SPScanRequest) (SPScanPage, error) {
	if req.Limit == 0 {
		req.Limit = 100
	}
	if req.Limit < 0 || req.Limit > spMaxPageTransactions || req.FromHeight < 0 || req.ToHeight < -1 || req.FromHeight > int(^uint32(0))-1 {
		return SPScanPage{}, ErrSPInvalidRequest
	}
	if req.Mode == "live" {
		return s.scanLive(ctx, req)
	}
	if req.Mode != "archive" || req.SnapshotID != "" {
		return SPScanPage{}, ErrSPInvalidRequest
	}
	return s.scanArchive(ctx, req)
}

func (s *silentPayState) scanArchive(ctx context.Context, req SPScanRequest) (SPScanPage, error) {
	view, baseAnchor, err := s.freshView(ctx)
	if err != nil {
		return SPScanPage{}, err
	}
	defer view.Close()
	if req.ToHeight < 0 {
		req.ToHeight = baseAnchor.Height
	}
	if req.FromHeight < s.startHeight {
		req.FromHeight = s.startHeight
	}
	if req.ToHeight > baseAnchor.Height {
		return SPScanPage{}, ErrSPNotReady
	}
	page := SPScanPage{Blocks: make([]SPBlock, 0), Anchor: SPAnchor{Height: req.ToHeight}}
	if req.ToHeight < s.startHeight {
		// We have no SP block hash for this pre-coverage range. Returning the
		// current Base anchor would let the SDK skip unscanned SP history.
		return SPScanPage{}, ErrSPInvalidRequest
	}
	hash, _, err := spReadBlockHeader(view, req.ToHeight)
	if err != nil {
		return SPScanPage{}, err
	}
	if req.AnchorHash != "" && req.AnchorHash != hash {
		return SPScanPage{}, ErrSPSnapshotExpired
	}
	page.Anchor.BlockHash = hash
	cursor := spCursor{Mode: "archive", From: req.FromHeight, To: req.ToHeight, Anchor: hash, Height: req.FromHeight}
	if req.Cursor != "" {
		cursor, err = s.views.decodeCursor(req.Cursor, cursor)
		if err != nil {
			return SPScanPage{}, err
		}
	}
	if req.FromHeight > req.ToHeight+1 {
		return SPScanPage{}, ErrSPInvalidRequest
	}
	count, used, headers := 0, 0, 0
	for cursor.Height <= req.ToHeight {
		if err = ctx.Err(); err != nil {
			return SPScanPage{}, err
		}
		bh, indices, e := spReadBlockHeader(view, cursor.Height)
		if e != nil {
			return SPScanPage{}, e
		}
		if uint64(cursor.Index) > uint64(len(indices)) {
			return SPScanPage{}, ErrSPInvalidRequest
		}
		block := SPBlock{Height: cursor.Height, BlockHash: bh, Transactions: make([]SPTransaction, 0)}
		headers++
		used += 64
		for int(cursor.Index) < len(indices) {
			tx, e := spReadTransaction(view, cursor.Height, indices[cursor.Index])
			if e != nil {
				return SPScanPage{}, e
			}
			size := 100 + len(tx.P2TROutputs)*64
			if count >= req.Limit || (count > 0 && used+size > spMaxPageBytes) {
				if len(block.Transactions) > 0 {
					page.Blocks = append(page.Blocks, block)
				}
				page.NextCursor, e = s.views.encodeCursor(cursor)
				return page, e
			}
			block.Transactions = append(block.Transactions, tx)
			cursor.Index++
			count++
			used += size
		}
		page.Blocks = append(page.Blocks, block)
		cursor.Height++
		cursor.Index = 0
		if headers >= 1000 || count >= req.Limit || used >= spMaxPageBytes {
			break
		}
	}
	page.Done = cursor.Height > req.ToHeight
	if !page.Done {
		page.NextCursor, err = s.views.encodeCursor(cursor)
	}
	return page, err
}

type spLiveRow struct {
	height int
	index uint32
	bitmap []byte
}

func (s *silentPayState) scanLive(ctx context.Context, req SPScanRequest) (SPScanPage, error) {
	entry, release, err := s.views.acquire(req.SnapshotID)
	if err != nil {
		return SPScanPage{}, err
	}
	defer release()
	if err = ctx.Err(); err != nil {
		return SPScanPage{}, err
	}
	if req.ToHeight >= 0 && req.ToHeight != entry.meta.Anchor.Height {
		return SPScanPage{}, ErrSPInvalidRequest
	}
	req.ToHeight = entry.meta.Anchor.Height
	if req.AnchorHash != "" && req.AnchorHash != entry.meta.Anchor.BlockHash {
		return SPScanPage{}, ErrSPSnapshotExpired
	}
	if req.FromHeight < s.startHeight {
		req.FromHeight = s.startHeight
	}
	cursor := spCursor{Mode: "live", Snapshot: req.SnapshotID, From: req.FromHeight, To: req.ToHeight, Anchor: entry.meta.Anchor.BlockHash, Height: req.FromHeight}
	if req.Cursor != "" {
		cursor, err = s.views.decodeCursor(req.Cursor, cursor)
		if err != nil {
			return SPScanPage{}, err
		}
	}
	page := SPScanPage{SnapshotID: req.SnapshotID, Anchor: entry.meta.Anchor, Blocks: make([]SPBlock, 0)}
	rows := make([]spLiveRow, 0, req.Limit+1)
	// Do not reenter the read view from its Scan callback: adapters serialize
	// read-only transaction access. Only a bounded row page is copied here.
	err = entry.view.Scan(common.ScanOptions{Prefix: spLivePrefix, Start: spPositionKey(spLivePrefix, cursor.Height, cursor.Index), Limit: req.Limit+1}, func(key, value []byte) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		h, index, e := spParsePosition(spLivePrefix, key)
		if e != nil || h > req.ToHeight || h < s.startHeight || index == 0 {
			return ErrSPCorrupt
		}
		rows = append(rows, spLiveRow{height: h, index: index, bitmap: append([]byte(nil), value...)})
		return nil
	})
	if err != nil {
		return SPScanPage{}, err
	}
	used, consumed := 0, 0
	for _, row := range rows {
		if err = ctx.Err(); err != nil {
			return SPScanPage{}, err
		}
		if consumed == req.Limit {
			break
		}
		tx, e := spReadTransaction(entry.view, row.height, row.index)
		if e != nil {
			return SPScanPage{}, e
		}
		if e = spValidateBitmap(row.bitmap, len(tx.P2TROutputs), false); e != nil {
			return SPScanPage{}, e
		}
		size := 100 + len(tx.P2TROutputs)*64
		if consumed > 0 && used+size > spMaxPageBytes {
			break
		}
		hash, indices, e := spReadBlockHeader(entry.view, row.height)
		if e != nil {
			return SPScanPage{}, e
		}
		i := sort.Search(len(indices), func(i int) bool { return indices[i] >= row.index })
		if i == len(indices) || indices[i] != row.index {
			return SPScanPage{}, ErrSPCorrupt
		}
		tx.UnspentBitmap = append([]byte(nil), row.bitmap...)
		if len(page.Blocks) == 0 || page.Blocks[len(page.Blocks)-1].Height != row.height {
			page.Blocks = append(page.Blocks, SPBlock{Height: row.height, BlockHash: hash, Transactions: make([]SPTransaction, 0)})
		}
		last := len(page.Blocks)-1
		page.Blocks[last].Transactions = append(page.Blocks[last].Transactions, tx)
		cursor.Height = row.height
		cursor.Index = row.index
		consumed++
		used += size
	}
	page.Done = consumed == len(rows)
	if !page.Done {
		page.NextCursor, err = s.views.encodeCursor(cursor)
	}
	return page, err
}

func (s *silentPayState) lookup(ctx context.Context, positions []SPPosition) ([]SPPositionResult, error) {
	if len(positions) == 0 || len(positions) > 500 {
		return nil, ErrSPInvalidRequest
	}
	view, anchor, err := s.freshView(ctx)
	if err != nil {
		return nil, err
	}
	defer view.Close()
	return s.batchPositions(ctx, view, anchor, positions, false)
}

// snapshotOutpoints uses the same immutable view for the Archive record,
// Live bitmap and Base UTXO membership; it never consults the current tip.
func (s *silentPayState) snapshotOutpoints(ctx context.Context, snapshotID string, positions []SPPosition) (SPAnchor, []SPPositionResult, error) {
	if snapshotID == "" || len(positions) == 0 || len(positions) > 500 {
		return SPAnchor{}, nil, ErrSPInvalidRequest
	}
	entry, release, err := s.views.acquire(snapshotID)
	if err != nil {
		return SPAnchor{}, nil, err
	}
	defer release()
	result, err := s.batchPositions(ctx, entry.view, entry.meta.Anchor, positions, true)
	if err != nil {
		return SPAnchor{}, nil, err
	}
	return entry.meta.Anchor, result, nil
}

// batchPositions handles a bounded request (<=500) by creation transaction.
// The input slice and output order are unchanged. Each source's directory,
// compact Archive and (if needed) Live bitmap are read once, then released.
// Only the requested output ordinals are held, never all SPOutput objects.
func (s *silentPayState) batchPositions(ctx context.Context, view common.DBReadView, anchor SPAnchor, positions []SPPosition, outpoints bool) ([]SPPositionResult, error) {
	results := make([]SPPositionResult, len(positions))
	order := make([]int, len(positions))
	for i, p := range positions {
		results[i] = SPPositionResult{SPPosition: p}
		if outpoints {
			results[i].State = "unknown"
		}
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool {
		a, b := positions[order[i]], positions[order[j]]
		if a.Height != b.Height {
			return a.Height < b.Height
		}
		if a.TxIndex != b.TxIndex {
			return a.TxIndex < b.TxIndex
		}
		if a.Vout != b.Vout {
			return a.Vout < b.Vout
		}
		return order[i] < order[j]
	})

	for start := 0; start < len(order); {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		src := positions[order[start]]
		end := start + 1
		for end < len(order) {
			p := positions[order[end]]
			if p.Height != src.Height || p.TxIndex != src.TxIndex {
				break
			}
			end++
		}

		// Validate each request before resolving the common source. Invalid
		// positions retain their own error without hiding a valid sibling.
		eligible := make([]int, 0, end-start)
		for _, idx := range order[start:end] {
			p := positions[idx]
			if p.Height < s.startHeight || p.Height > anchor.Height || !spHashStringValid(p.BlockHash) {
				results[idx].Error = ErrSPInvalidRequest.Error()
				continue
			}
			eligible = append(eligible, idx)
		}
		start = end
		if len(eligible) == 0 {
			continue
		}

		bh, indices, err := spReadBlockHeader(view, src.Height)
		if err != nil {
			return nil, err
		}
		sameChain := make([]int, 0, len(eligible))
		for _, idx := range eligible {
			if positions[idx].BlockHash != bh {
				results[idx].Error = ErrSPSnapshotExpired.Error()
			} else {
				sameChain = append(sameChain, idx)
			}
		}
		if len(sameChain) == 0 {
			continue
		}
		j := sort.Search(len(indices), func(i int) bool { return indices[i] >= src.TxIndex })
		if j == len(indices) || indices[j] != src.TxIndex {
			for _, idx := range sameChain {
				results[idx].Error = fmt.Errorf("%w: not_candidate_transaction", ErrSPInvalidRequest).Error()
			}
			continue
		}

		raw, err := view.Read(spPositionKey(spArchivePrefix, src.Height, src.TxIndex))
		if err != nil {
			return nil, fmt.Errorf("%w: missing tx %d/%d: %v", ErrSPCorrupt, src.Height, src.TxIndex, err)
		}
		reader, count, err := spRecordHeader(raw)
		if err != nil {
			return nil, err
		}
		var txHash chainhash.Hash
		copy(txHash[:], raw[1:33])
		txid := txHash.String()
		matching := make([]int, 0, len(sameChain))
		for _, idx := range sameChain {
			if expected := positions[idx].ExpectedTxID; expected != "" && expected != txid {
				results[idx].Error = fmt.Errorf("%w: transaction_identity_mismatch", ErrSPInvalidRequest).Error()
			} else {
				matching = append(matching, idx)
			}
		}

		// Vout positions in matching remain sorted by the source's vout.
		// Walk the entire compact record once, including the tail, so
		// malformed data cannot be silently accepted by early matches.
		var ordinals []int
		if outpoints {
			ordinals = make([]int, len(matching))
			for i := range ordinals {
				ordinals[i] = -1
			}
		}
		nextWanted := 0
		var previous uint32
		for ordinal := 0; ordinal < count; ordinal++ {
			if ordinal&1023 == 0 {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
			}
			vout, _, err := spNextRecordOutput(raw, reader, ordinal, previous)
			if err != nil {
				return nil, err
			}
			previous = vout
			if outpoints {
				for nextWanted < len(matching) && positions[matching[nextWanted]].Vout <= vout {
					if positions[matching[nextWanted]].Vout == vout {
						ordinals[nextWanted] = ordinal
					}
					nextWanted++
				}
			}
		}
		if reader.Len() != 0 {
			return nil, ErrSPCorrupt
		}
		if !outpoints {
			for _, idx := range matching {
				results[idx].TxID = txid
			}
			continue
		}

		hasOutput := false
		for _, ordinal := range ordinals {
			if ordinal >= 0 {
				hasOutput = true
				break
			}
		}
		if !hasOutput {
			for _, idx := range matching {
				results[idx].Error = "not_candidate_output"
			}
			continue
		}

		bits, err := view.Read(spPositionKey(spLivePrefix, src.Height, src.TxIndex))
		if err == nil {
			if err := spValidateBitmap(bits, count, false); err != nil {
				return nil, err
			}
		} else if errors.Is(err, common.ErrKeyNotFound) {
			bits = nil // all candidate P2TR outputs are spent
		} else {
			return nil, err
		}

		// Duplicate vouts in one request share their result; no extra Base
		// lookup or allocation is needed. All other outputs remain untouched.
		var lastVout uint32
		var lastState string
		hasLast := false
		for i, idx := range matching {
			ordinal := ordinals[i]
			if ordinal < 0 {
				results[idx].Error = "not_candidate_output"
				continue
			}
			p := positions[idx]
			item := &results[idx]
			item.TxID = txid
			if hasLast && p.Vout == lastVout {
				item.State = lastState
				continue
			}

			utxoRaw, err := view.Read(indexdb.GetUTXODBKey(txid + ":" + spVoutString(p.Vout)))
			switch {
			case errors.Is(err, common.ErrKeyNotFound):
				item.State = "spent"
			case err != nil:
				return nil, err
			default:
				var utxo common.UtxoValueInDB
				if err := indexdb.DecodeBytesWithProto3(utxoRaw, &utxo); err != nil {
					return nil, err
				}
				if utxo.UtxoId != common.ToUtxoId(p.Height, int(p.TxIndex), int(p.Vout)) {
					return nil, ErrSPCorrupt
				}
				item.State = "unspent"
			}
			live := bits != nil && bits[ordinal/8]&(1<<uint(ordinal%8)) != 0
			if live != (item.State == "unspent") {
				return nil, fmt.Errorf("%w: Live/Base disagreement", ErrSPCorrupt)
			}
			lastVout, lastState, hasLast = p.Vout, item.State, true
		}
	}
	return results, nil
}

func spEqualTransactions(a, b SPTransaction) bool {
	x, e := spEncodeTransaction(a)
	if e != nil {
		return false
	}
	y, e := spEncodeTransaction(b)
	return e == nil && bytes.Equal(x, y)
}
