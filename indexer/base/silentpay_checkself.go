package base

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/sat20-labs/indexer/common"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
)

// spAuditPager copies bounded pages out of one pinned view. Offsets into a
// reusable byte buffer avoid two allocations per row. Returned slices remain
// valid until the next call to next; no database iterator stays open between
// calls, so another pager or a Base point read can safely use the same view.
type spAuditPager struct {
	view                 common.DBReadView
	prefix, cursor, data []byte
	rows                 []spAuditRow
	index                int
	done                 bool
}

type spAuditRow struct{ key, value, end int }

func (p *spAuditPager) next(ctx context.Context) ([]byte, []byte, error) {
	const maxRecords = 1024
	const maxPageBytes = 8 << 20
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if p.index == len(p.rows) {
		if p.done {
			return nil, nil, nil
		}
		if p.rows == nil {
			p.rows = make([]spAuditRow, 0, maxRecords)
		}
		p.rows, p.data, p.index = p.rows[:0], p.data[:0], 0
		err := p.view.Scan(common.ScanOptions{
			Prefix: p.prefix, Start: p.cursor, StartInclusive: false,
		}, func(key, value []byte) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if p.data == nil {
				// Estimate the first page from its first row, capped by the
				// existing byte budget, to avoid repeated initial buffer growth.
				p.data = make([]byte, 0, min(maxRecords*(len(key)+len(value)), maxPageBytes))
			}
			row := spAuditRow{key: len(p.data)}
			p.data = append(p.data, key...)
			row.value = len(p.data)
			p.data = append(p.data, value...)
			row.end = len(p.data)
			p.rows = append(p.rows, row)
			if len(p.rows) >= maxRecords || len(p.data) >= maxPageBytes {
				return common.ErrStopScan
			}
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
		p.done = len(p.rows) < maxRecords && len(p.data) < maxPageBytes
		if len(p.rows) == 0 {
			return nil, nil, nil
		}
		last := p.rows[len(p.rows)-1]
		// Keep the seek key separate: the next page overwrites data.
		p.cursor = append(p.cursor[:0], p.data[last.key:last.value]...)
	}
	row := p.rows[p.index]
	p.index++
	return p.data[row.key:row.value:row.value], p.data[row.value:row.end:row.end], nil
}

// The callback borrows page bytes only for its duration, after Scan releases
// its lock. Associated reads and another paged scan use the same snapshot.
func spAuditEach(ctx context.Context, view common.DBReadView, prefix []byte,
	visit func(key, value []byte) error) error {
	pager := spAuditPager{view: view, prefix: prefix}
	for {
		key, value, err := pager.next(ctx)
		if err != nil {
			return err
		}
		if key == nil {
			return nil
		}
		if err := visit(key, value); err != nil {
			return err
		}
	}
}

// Preserve GetUTXODBKey's exact bytes: prefix + display-order txid bytes +
// decimal vout. Archive stores the hash in internal order. Reserve suffix
// space once per transaction; callers append each vout to this same prefix.
func spAuditUTXOPrefix(hash []byte) []byte {
	key := make([]byte, len(common.DB_KEY_UTXO)+32, len(common.DB_KEY_UTXO)+32+10)
	copy(key, common.DB_KEY_UTXO)
	for i := 0; i < 32; i++ {
		key[len(common.DB_KEY_UTXO)+i] = hash[31-i]
	}
	return key
}

// spAuditCandidate derives its expected Live bitmap exclusively from the
// committed Base UTXO data. Full audits additionally compare address scripts
// and validate T's curve membership. It does not trust the existing SP bitmap,
// nor query bitcoind. Without the original witness/prevouts stored in Base,
// the BIP352 tweak cannot be independently recomputed here.
func spAuditCandidate(view common.DBReadView, height int, txIndex uint32,
	raw, live []byte, hasLive, full bool) (uint64, uint64, error) {

	readHeader := spRecordHeaderStructure
	if full {
		readHeader = spRecordHeader
	}
	reader, count, err := readHeader(raw)
	if err != nil {
		return 0, 0, err
	}
	if height < 0 || height > 0x1fffffff || txIndex == 0 || txIndex > 0x1ffff {
		return 0, 0, fmt.Errorf("%w: invalid SP creation position %d/%d", ErrSPCorrupt, height, txIndex)
	}
	utxoPrefix := spAuditUTXOPrefix(raw[1:33])
	bits := make([]byte, (count+7)/8)
	var liveOutputs uint64
	var previous uint32
	for i := 0; i < count; i++ {
		vout, key, e := spNextRecordOutput(raw, reader, i, previous)
		if e != nil || vout > 0x3ffff {
			return 0, 0, fmt.Errorf("%w: invalid SP output %d/%d/%d", ErrSPCorrupt, height, txIndex, i)
		}
		previous = vout
		utxoKey := strconv.AppendUint(utxoPrefix, uint64(vout), 10)
		encoded, e := view.Read(utxoKey)
		if errors.Is(e, common.ErrKeyNotFound) {
			continue
		}
		if e != nil {
			return 0, 0, e
		}
		var utxo common.UtxoValueInDB
		if e := indexdb.DecodeBytesWithProto3(encoded, &utxo); e != nil {
			return 0, 0, e
		}
		if utxo.UtxoId != common.ToUtxoId(height, int(txIndex), int(vout)) {
			outpoint, _ := indexdb.GetUtxoByDBKey(utxoKey)
			return 0, 0, fmt.Errorf("%w: Base UTXO ID %s", ErrSPCorrupt, outpoint)
		}
		if full {
			address, e := view.Read(indexdb.GetAddressIdKey(utxo.AddressId))
			if e != nil {
				return 0, 0, e
			}
			script, e := common.GetPkScriptFromAddress(string(address))
			if e != nil || !common.SPP2TROutput(script) || !bytes.Equal(script[2:], key) {
				outpoint, _ := indexdb.GetUtxoByDBKey(utxoKey)
				return 0, 0, fmt.Errorf("%w: Base P2TR output %s", ErrSPCorrupt, outpoint)
			}
		}
		bits[i/8] |= 1 << uint(i%8)
		liveOutputs++
	}
	if reader.Len() != 0 {
		return 0, 0, ErrSPCorrupt
	}
	if liveOutputs == 0 {
		if hasLive {
			return 0, 0, fmt.Errorf("%w: fully spent SP transaction remains Live %d/%d", ErrSPCorrupt, height, txIndex)
		}
		return 0, 0, nil
	}
	if !hasLive || !bytes.Equal(live, bits) {
		return 0, 0, fmt.Errorf("%w: SP Live/Base bitmap mismatch %d/%d", ErrSPCorrupt, height, txIndex)
	}
	return 1, liveOutputs, nil
}

// audit checks directory completeness and Live = Archive intersect Base UTXO
// at one committed database version. Full audits also validate T's curve
// membership and compare P2TR keys with Base address scripts. Both modes use
// NO Bitcoin RPC data and cannot independently recompute BIP352 T or find
// candidates jointly omitted from the directory and Archive.
func (s *silentPayState) audit(ctx context.Context, expectedAnchor SPAnchor, full bool) error {
	view, anchor, err := s.freshView(ctx)
	if err != nil {
		return err
	}
	defer view.Close()
	if anchor != expectedAnchor {
		return fmt.Errorf("%w: committed Base/SP anchor changed", ErrSPCorrupt)
	}

	start := s.startHeight
	// Empty pre-activation ranges must contain no derived SP records.
	if anchor.Height < start {
		for _, prefix := range [][]byte{spBlockPrefix, spArchivePrefix, spLivePrefix} {
			var found bool
			if err := view.Scan(common.ScanOptions{Prefix: prefix, KeysOnly: true, Limit: 1},
				func(_, _ []byte) error { found = true; return nil }); err != nil {
				return err
			}
			if found {
				return fmt.Errorf("%w: SP data before coverage start", ErrSPCorrupt)
			}
		}
		return nil
	}

	// One bool per indexed block, not one allocation for each Bitcoin UTXO.
	// The final directory pass detects missing zero-candidate block records
	// and candidates absent from the Archive.
	covered := make([]bool, anchor.Height-start+1)
	var previousHeight = -1
	var directoryIndices []uint32
	directoryCursor := 0
	var expectedLiveTx, expectedLiveOutputs, archiveTx uint64
	started := time.Now()
	mode := "core"
	if full {
		mode = "full"
	}
	nextProgress := start + 10000
	// Both prefixes have the same position ordering. Merge their bounded
	// pages instead of point-reading every Live row and then scanning it again.
	livePager := spAuditPager{view: view, prefix: spLivePrefix}
	liveKey, liveValue, err := livePager.next(ctx)
	if err != nil {
		return err
	}

	err = spAuditEach(ctx, view, spArchivePrefix, func(key, raw []byte) error {
		height, txIndex, e := spParsePosition(spArchivePrefix, key)
		if e != nil || height < start || height > anchor.Height {
			return fmt.Errorf("%w: archived SP position outside coverage", ErrSPCorrupt)
		}
		if height != previousHeight {
			if previousHeight >= start && directoryCursor != len(directoryIndices) {
				return fmt.Errorf("%w: SP Archive missing a directory record in block %d", ErrSPCorrupt, previousHeight)
			}
			header, e := view.Read(spBlockKey(height))
			if e != nil {
				return fmt.Errorf("%w: SP block directory missing at %d: %v", ErrSPCorrupt, height, e)
			}
			_, directoryIndices, e = spDecodeBlock(header)
			if e != nil || len(directoryIndices) == 0 {
				return fmt.Errorf("%w: invalid SP block directory %d", ErrSPCorrupt, height)
			}
			previousHeight, directoryCursor = height, 0
			covered[height-start] = true
			if height >= nextProgress {
				common.Log.Infof("SP DB CheckSelf scanning Archive at %d: tx=%d, live tx=%d, outputs=%d, elapsed=%v, mode=%s",
					height, archiveTx, expectedLiveTx, expectedLiveOutputs, time.Since(started), mode)
				nextProgress = height + 10000
			}
		}
		if directoryCursor == len(directoryIndices) || directoryIndices[directoryCursor] != txIndex {
			return fmt.Errorf("%w: orphan or unordered Archive row %d/%d", ErrSPCorrupt, height, txIndex)
		}
		directoryCursor++
		archiveTx++
		hasLive := false
		if liveKey != nil {
			liveHeight, liveIndex, e := spParsePosition(spLivePrefix, liveKey)
			if e != nil || liveHeight < start || liveHeight > anchor.Height || liveIndex == 0 {
				return fmt.Errorf("%w: invalid SP Live position", ErrSPCorrupt)
			}
			order := bytes.Compare(liveKey[len(spLivePrefix):], key[len(spArchivePrefix):])
			if order < 0 {
				return fmt.Errorf("%w: orphan SP Live row %d/%d", ErrSPCorrupt, liveHeight, liveIndex)
			}
			hasLive = order == 0
		}
		liveTx, liveOutputs, e := spAuditCandidate(view, height, txIndex, raw, liveValue, hasLive, full)
		if e != nil {
			return e
		}
		expectedLiveTx += liveTx
		expectedLiveOutputs += liveOutputs
		if hasLive {
			liveKey, liveValue, e = livePager.next(ctx)
			if e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if previousHeight >= start && directoryCursor != len(directoryIndices) {
		return fmt.Errorf("%w: missing trailing SP Archive row in block %d", ErrSPCorrupt, previousHeight)
	}
	if liveKey != nil {
		return fmt.Errorf("%w: trailing orphan SP Live row", ErrSPCorrupt)
	}

	nextHeight := start
	err = view.Scan(common.ScanOptions{Prefix: spBlockPrefix}, func(key, raw []byte) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if len(key) != len(spBlockPrefix)+4 || !bytes.HasPrefix(key, spBlockPrefix) {
			return ErrSPCorrupt
		}
		height := int(uint32(key[len(spBlockPrefix)])<<24 |
			uint32(key[len(spBlockPrefix)+1])<<16 |
			uint32(key[len(spBlockPrefix)+2])<<8 |
			uint32(key[len(spBlockPrefix)+3]))
		if height != nextHeight || height > anchor.Height {
			return fmt.Errorf("%w: non-contiguous SP block directory at %d", ErrSPCorrupt, nextHeight)
		}
		hash, indices, e := spDecodeBlock(raw)
		if e != nil {
			return e
		}
		if covered[height-start] != (len(indices) != 0) {
			return fmt.Errorf("%w: SP block %d missing Archive candidates", ErrSPCorrupt, height)
		}
		if height == anchor.Height && hash != anchor.BlockHash {
			return fmt.Errorf("%w: SP block tip differs from committed Base anchor", ErrSPCorrupt)
		}
		nextHeight++
		return nil
	})
	if err != nil {
		return err
	}
	if nextHeight != anchor.Height+1 {
		return fmt.Errorf("%w: SP block directory ends at %d, expected %d", ErrSPCorrupt, nextHeight-1, anchor.Height)
	}

	common.Log.Infof("SP DB CheckSelf passed through %d: Archive tx=%d Live tx=%d outputs=%d, elapsed=%v (mode=%s, no Bitcoin RPC)",
		anchor.Height, archiveTx, expectedLiveTx, expectedLiveOutputs, time.Since(started), mode)
	return nil
}
