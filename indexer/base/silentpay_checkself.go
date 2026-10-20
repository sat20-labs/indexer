package base

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/sat20-labs/indexer/common"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
)

// spAuditEach scans a single pinned database version in bounded pages. Read
// views serialize Read and Scan, so the callback runs after the Scan releases
// its iterator lock and can safely read Base UTXOs, addresses and SP rows.
// The local slice owns its bytes; no iterator-owned slice escapes its callback.
func spAuditEach(ctx context.Context, view common.DBReadView, prefix []byte,
	visit func(key, value []byte) error) error {

	const maxRecords = 1024
	const maxPageBytes = 8 << 20
	var cursor []byte
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		type row struct{ key, value []byte }
		page := make([]row, 0, maxRecords)
		size := 0
		err := view.Scan(common.ScanOptions{
			Prefix: prefix, Start: cursor, StartInclusive: false,
		}, func(key, value []byte) error {
			page = append(page, row{
				key: append([]byte(nil), key...), value: append([]byte(nil), value...),
			})
			size += len(key) + len(value)
			if len(page) >= maxRecords || size >= maxPageBytes {
				return common.ErrStopScan
			}
			return nil
		})
		if err != nil {
			return err
		}
		for _, item := range page {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := visit(item.key, item.value); err != nil {
				return err
			}
		}
		if len(page) == 0 || (len(page) < maxRecords && size < maxPageBytes) {
			return nil
		}
		cursor = page[len(page)-1].key
	}
}

// spAuditCandidate derives its expected Live bitmap exclusively from the
// committed Base UTXO/address data. It does not trust the existing SP bitmap,
// nor query bitcoind. Without the original witness/prevouts stored in Base,
// the BIP352 tweak cannot be independently recomputed here.
func spAuditCandidate(view common.DBReadView, height int, txIndex uint32,
	raw []byte) (uint64, uint64, error) {

	reader, count, err := spRecordHeader(raw)
	if err != nil {
		return 0, 0, err
	}
	if height < 0 || height > 0x1fffffff || txIndex == 0 || txIndex > 0x1ffff {
		return 0, 0, fmt.Errorf("%w: invalid SP creation position %d/%d", ErrSPCorrupt, height, txIndex)
	}
	var hash chainhash.Hash
	copy(hash[:], raw[1:33])
	txid := hash.String()
	bits := make([]byte, (count+7)/8)
	var liveOutputs uint64
	var previous uint32
	for i := 0; i < count; i++ {
		vout, key, e := spNextRecordOutput(raw, reader, i, previous)
		if e != nil || vout > 0x3ffff {
			return 0, 0, fmt.Errorf("%w: invalid SP output %d/%d/%d", ErrSPCorrupt, height, txIndex, i)
		}
		previous = vout
		outpoint := txid + ":" + strconv.FormatUint(uint64(vout), 10)
		encoded, e := view.Read(indexdb.GetUTXODBKey(outpoint))
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
			return 0, 0, fmt.Errorf("%w: Base UTXO ID %s", ErrSPCorrupt, outpoint)
		}
		address, e := view.Read(indexdb.GetAddressIdKey(utxo.AddressId))
		if e != nil {
			return 0, 0, e
		}
		script, e := common.GetPkScriptFromAddress(string(address))
		if e != nil || !common.SPP2TROutput(script) || !bytes.Equal(script[2:], key) {
			return 0, 0, fmt.Errorf("%w: Base P2TR output %s", ErrSPCorrupt, outpoint)
		}
		bits[i/8] |= 1 << uint(i%8)
		liveOutputs++
	}
	if reader.Len() != 0 {
		return 0, 0, ErrSPCorrupt
	}
	live, err := view.Read(spPositionKey(spLivePrefix, height, txIndex))
	if liveOutputs == 0 {
		if !errors.Is(err, common.ErrKeyNotFound) {
			return 0, 0, fmt.Errorf("%w: fully spent SP transaction remains Live %d/%d", ErrSPCorrupt, height, txIndex)
		}
		return 0, 0, nil
	}
	if err != nil || !bytes.Equal(live, bits) {
		return 0, 0, fmt.Errorf("%w: SP Live/Base bitmap mismatch %d/%d", ErrSPCorrupt, height, txIndex)
	}
	return 1, liveOutputs, nil
}

// audit performs a complete local consistency check at one committed Base/SP
// database version. It intentionally uses NO Bitcoin RPC data. The database
// does not retain all original input witness or historical prevout scripts,
// so it cannot independently rediscover candidates or recompute BIP352 T.
// Cryptographic agreement remains the responsibility of vectors and E2E.
func (s *silentPayState) audit(ctx context.Context, expectedAnchor SPAnchor) error {
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
	nextProgress := start + 10000

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
				common.Log.Infof("SP DB CheckSelf scanning Archive at %d: tx=%d, live tx=%d, outputs=%d, elapsed=%v",
					height, archiveTx, expectedLiveTx, expectedLiveOutputs, time.Since(started))
				nextProgress = height + 10000
			}
		}
		if directoryCursor == len(directoryIndices) || directoryIndices[directoryCursor] != txIndex {
			return fmt.Errorf("%w: orphan or unordered Archive row %d/%d", ErrSPCorrupt, height, txIndex)
		}
		directoryCursor++
		archiveTx++
		liveTx, liveOutputs, e := spAuditCandidate(view, height, txIndex, raw)
		if e != nil {
			return e
		}
		expectedLiveTx += liveTx
		expectedLiveOutputs += liveOutputs
		return nil
	})
	if err != nil {
		return err
	}
	if previousHeight >= start && directoryCursor != len(directoryIndices) {
		return fmt.Errorf("%w: missing trailing SP Archive row in block %d", ErrSPCorrupt, previousHeight)
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

	// Every expected live row was checked by exact position and bitmap above;
	// a full prefix count plus key validation catches orphan extra Live rows.
	var liveCount uint64
	err = view.Scan(common.ScanOptions{Prefix: spLivePrefix, KeysOnly: false}, func(key, value []byte) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		height, index, e := spParsePosition(spLivePrefix, key)
		if e != nil || height < start || height > anchor.Height || index == 0 || spBitmapEmpty(value) {
			return fmt.Errorf("%w: invalid or orphan SP Live position", ErrSPCorrupt)
		}
		liveCount++
		return nil
	})
	if err != nil {
		return err
	}
	if liveCount != expectedLiveTx {
		return fmt.Errorf("%w: unexpected SP Live rows %d != %d", ErrSPCorrupt, liveCount, expectedLiveTx)
	}
	common.Log.Infof("SP DB CheckSelf passed through %d: Archive tx=%d Live tx=%d outputs=%d, elapsed=%v (no Bitcoin RPC)",
		anchor.Height, archiveTx, expectedLiveTx, expectedLiveOutputs, time.Since(started))
	return nil
}
