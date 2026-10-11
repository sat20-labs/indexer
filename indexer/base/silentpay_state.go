package base

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/sat20-labs/indexer/common"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
)

type spLiveChange struct {
	atHeight int
	bitmap   []byte // nil means delete; every mutation installs an owned slice
}

// silentPayState is Base-owned state, not another independently progressing
// indexer. Pending Archive records use the same compact immutable bytes as disk.
// Only mutable Live deltas are copied by commit clones. No historical source
// transactions are retained in a block-wide decoded cache.
type silentPayState struct {
	db          common.KVDB
	network     string
	startHeight int
	blocks      map[int]*spPendingBlock
	live        map[string]spLiveChange
	views       *spViewPool
	poisoned    error
}

func newSilentPayState(kv common.KVDB, params *chaincfg.Params) *silentPayState {
	return &silentPayState{
		db: kv, network: params.Name, startHeight: spStartHeight(params),
		blocks: make(map[int]*spPendingBlock), live: make(map[string]spLiveChange), views: newSPViewPool(),
	}
}

func (s *silentPayState) schemaBytes() []byte {
	return []byte(fmt.Sprintf("%d:%s", SPSchemaVersion, s.network))
}

func (s *silentPayState) validateInitialState(stats *SyncStats) error {
	if _, ok := s.db.(common.SnapshotDB); !ok {
		return fmt.Errorf("silentpay requires durable snapshot DB")
	}
	if _, err := s.db.Read(spPendingKey); err == nil {
		return ErrSPIncompleteCommit
	} else if !errors.Is(err, common.ErrKeyNotFound) {
		return err
	}
	schema, err := s.db.Read(spSchemaKey)
	if errors.Is(err, common.ErrKeyNotFound) && stats.SyncHeight < 0 {
		return nil
	}
	if err != nil || !bytes.Equal(schema, s.schemaBytes()) {
		return fmt.Errorf("%w: rebuild Base with SilentPay schema", ErrSPNotReady)
	}
	if stats.SyncHeight >= s.startHeight {
		raw, err := s.db.Read(spBlockKey(stats.SyncHeight))
		if err != nil {
			return fmt.Errorf("%w: missing committed SP block", ErrSPCorrupt)
		}
		hash, _, err := spDecodeBlock(raw)
		if err != nil || hash != stats.SyncBlockHash {
			return fmt.Errorf("%w: committed anchor mismatch", ErrSPCorrupt)
		}
	}
	return nil
}

func (s *silentPayState) clone() *silentPayState {
	if s == nil {
		return nil
	}
	copyState := &silentPayState{
		db: s.db, network: s.network, startHeight: s.startHeight, views: s.views, poisoned: s.poisoned,
		blocks: make(map[int]*spPendingBlock, len(s.blocks)), live: make(map[string]spLiveChange, len(s.live)),
	}
	for height, block := range s.blocks {
		// The records and directory never change after process publishes them.
		// Each clone owns only its map; deleting a map entry does not change
		// any other clone or require copying thousands of output keys.
		copyState.blocks[height] = block
	}
	for key, change := range s.live {
		copyState.live[key] = spLiveChange{atHeight: change.atHeight, bitmap: append([]byte(nil), change.bitmap...)}
	}
	return copyState
}

func (s *silentPayState) subtract(frozen *silentPayState) {
	if s == nil || frozen == nil {
		return
	}
	for height := range frozen.blocks {
		delete(s.blocks, height)
	}
	for key, old := range frozen.live {
		current, ok := s.live[key]
		if ok && current.atHeight == old.atHeight && bytes.Equal(current.bitmap, old.bitmap) {
			delete(s.live, key)
		}
	}
}

// Used by public-record tests. Production indexing goes directly from Base
// transactions to compact records and never retains this expanded DTO.
func spCandidate(tx *common.Transaction, index uint32) (*SPTransaction, error) {
	raw, _, err := spCandidateRecord(tx, index)
	if err != nil || raw == nil {
		return nil, err
	}
	candidate, err := spDecodeTransaction(raw, index)
	return &candidate, err
}

// Returned bytes are immutable. Pending data is shared; persisted reads own
// only this source record. The caller must release it after processing a group.
func (s *silentPayState) transactionRecord(height int, index uint32) ([]byte, error) {
	if block, ok := s.blocks[height]; ok {
		i := sort.Search(len(block.indices), func(i int) bool { return block.indices[i] >= index })
		if i < len(block.indices) && block.indices[i] == index {
			return block.records[i], nil
		}
		return nil, common.ErrKeyNotFound
	}
	raw, err := s.db.Read(spPositionKey(spArchivePrefix, height, index))
	if errors.Is(err, common.ErrKeyNotFound) {
		// Missing Archive is a legitimate noncandidate only when the block's
		// candidate directory also excludes it. Do not hide missing data.
		header, e := s.db.Read(spBlockKey(height))
		if e != nil {
			return nil, fmt.Errorf("%w: missing SP block directory %d: %v", ErrSPCorrupt, height, e)
		}
		_, indices, e := spDecodeBlock(header)
		if e != nil {
			return nil, e
		}
		i := sort.Search(len(indices), func(i int) bool { return indices[i] >= index })
		if i < len(indices) && indices[i] == index {
			return nil, fmt.Errorf("%w: missing mandatory Archive %d/%d", ErrSPCorrupt, height, index)
		}
		return nil, common.ErrKeyNotFound
	}
	return raw, err
}

// inputs are sorted by vout and all refer to one creation transaction. The KV
// interface reads a complete compact value, not byte ranges. Walk its deltas
// without expanding unrelated outputs, then release it before the next source.
func (s *silentPayState) consumeGroup(inputs []*common.TxInput, height int) error {
	first := inputs[0]
	raw, err := s.transactionRecord(first.OutHeight, uint32(first.OutTxIndex))
	if errors.Is(err, common.ErrKeyNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	r, count, err := spRecordHeader(raw)
	if err != nil {
		return err
	}
	var hash chainhash.Hash
	copy(hash[:], raw[1:33])
	txid := hash.String()
	for i, in := range inputs {
		if in.TxId != txid {
			return fmt.Errorf("%w: spending txid mismatch", ErrSPCorrupt)
		}
		if i > 0 && inputs[i-1].TxOutIndex == in.TxOutIndex {
			return fmt.Errorf("%w: duplicate confirmed spend", ErrSPCorrupt)
		}
	}
	key := string(spPositionKey(spLivePrefix, first.OutHeight, uint32(first.OutTxIndex)))
	change, ok := s.live[key]
	bits := change.bitmap
	if !ok {
		bits, err = s.db.Read([]byte(key))
		if err != nil {
			return fmt.Errorf("%w: candidate Live entry missing: %v", ErrSPCorrupt, err)
		}
	}
	if err := spValidateBitmap(bits, count, false); err != nil {
		return err
	}
	// One copy per source, not per input; a frozen commit's bitmap stays intact.
	bits = append([]byte(nil), bits...)
	wanted := 0
	var previous uint32
	for ordinal := 0; ordinal < count; ordinal++ {
		vout, outputKey, err := spNextRecordOutput(raw, r, ordinal, previous)
		if err != nil {
			return err
		}
		previous = vout
		if wanted == len(inputs) {
			continue // validate remaining encoding without retaining its outputs
		}
		in := inputs[wanted]
		if uint32(in.TxOutIndex) < vout {
			return fmt.Errorf("%w: spending output missing", ErrSPCorrupt)
		}
		if uint32(in.TxOutIndex) != vout {
			continue
		}
		if !bytes.Equal(outputKey, in.OutValue.PkScript[2:]) {
			return fmt.Errorf("%w: spending output mismatch", ErrSPCorrupt)
		}
		mask := byte(1 << uint(ordinal%8))
		if bits[ordinal/8]&mask == 0 {
			return fmt.Errorf("%w: duplicate confirmed spend", ErrSPCorrupt)
		}
		bits[ordinal/8] &^= mask
		wanted++
	}
	if r.Len() != 0 || wanted != len(inputs) {
		return fmt.Errorf("%w: spending output or record encoding", ErrSPCorrupt)
	}
	if spBitmapEmpty(bits) {
		bits = nil
	}
	s.live[key] = spLiveChange{atHeight: height, bitmap: bits}
	return nil
}

// Base has already resolved and validated the block's input UTXOs. Archive
// creation is independent of Live spending, so create immutable records in
// transaction order first, then group the current block's input references by
// source. This also handles same-block parent/child spends without caching all
// historical source outputs. No transaction/input owned by Base is reordered.
func (s *silentPayState) process(block *common.Block) (err error) {
	if s.poisoned != nil {
		return s.poisoned
	}
	defer func() {
		if err != nil {
			s.poisoned = err
		}
	}()
	if block == nil || block.Height < 0 || !spHashStringValid(block.Hash) {
		return ErrSPCorrupt
	}
	if block.Height < s.startHeight {
		return nil
	}
	if _, ok := s.blocks[block.Height]; ok {
		return fmt.Errorf("%w: duplicate pending block", ErrSPCorrupt)
	}
	stored := &spPendingBlock{hash: block.Hash}
	var spends []*common.TxInput
	for index, tx := range block.Transactions {
		if index == 0 {
			continue
		}
		if tx == nil {
			return ErrSPCorrupt
		}
		for _, in := range tx.Inputs {
			if in == nil {
				return ErrSPCorrupt
			}
			if !common.SPP2TROutput(in.OutValue.PkScript) || in.OutHeight < s.startHeight || in.OutTxIndex == 0 {
				continue
			}
			if in.OutTxIndex < 0 || uint64(in.OutTxIndex) > uint64(^uint32(0)) || in.TxOutIndex < 0 || uint64(in.TxOutIndex) > uint64(^uint32(0)) {
				return ErrSPCorrupt
			}
			spends = append(spends, in)
		}
		raw, count, e := spCandidateRecord(tx, uint32(index))
		if e != nil {
			return e
		}
		if raw == nil {
			continue
		}
		stored.indices = append(stored.indices, uint32(index))
		stored.records = append(stored.records, raw)
		key := string(spPositionKey(spLivePrefix, block.Height, uint32(index)))
		s.live[key] = spLiveChange{atHeight: block.Height, bitmap: spFullBitmap(count)}
	}
	// No code mutates this block after publication; commit clones may share it.
	s.blocks[block.Height] = stored
	sort.Slice(spends, func(i, j int) bool {
		a, b := spends[i], spends[j]
		if a.OutHeight != b.OutHeight {
			return a.OutHeight < b.OutHeight
		}
		if a.OutTxIndex != b.OutTxIndex {
			return a.OutTxIndex < b.OutTxIndex
		}
		return a.TxOutIndex < b.TxOutIndex
	})
	for start := 0; start < len(spends); {
		end := start + 1
		for end < len(spends) && spends[end].OutHeight == spends[start].OutHeight && spends[end].OutTxIndex == spends[start].OutTxIndex {
			end++
		}
		if err := s.consumeGroup(spends[start:end], block.Height); err != nil {
			return err
		}
		start = end
	}
	return nil
}

func (s *silentPayState) checkPendingRange(previous, height int, hash string) error {
	if s.poisoned != nil {
		return s.poisoned
	}
	start := previous + 1
	if start < s.startHeight {
		start = s.startHeight
	}
	if height < previous {
		return ErrSPCorrupt
	}
	expected := height - start + 1
	if expected < 0 {
		expected = 0
	}
	if len(s.blocks) != expected {
		return fmt.Errorf("%w: pending block coverage %d != %d", ErrSPCorrupt, len(s.blocks), expected)
	}
	for h := start; h <= height; h++ {
		if _, ok := s.blocks[h]; !ok {
			return fmt.Errorf("%w: missing SP block %d", ErrSPCorrupt, h)
		}
	}
	// Empty repeat flushes have no new pending block to validate.
	if expected > 0 && s.blocks[height].hash != hash {
		return ErrSPCorrupt
	}
	return nil
}

// Fence all Base writes: either backend can split a large WriteBatch. A failed
// commit retains the marker; production callers panic rather than repair/retry.
func (s *silentPayState) beginCommit(height int, hash string) error {
	if _, err := s.db.Read(spPendingKey); err == nil {
		return ErrSPIncompleteCommit
	} else if !errors.Is(err, common.ErrKeyNotFound) {
		return err
	}
	capability, ok := s.db.(common.SnapshotDB)
	if !ok {
		return ErrSPNotReady
	}
	h, err := chainhash.NewHashFromStr(hash)
	if err != nil {
		return err
	}
	marker := make([]byte, 40)
	binary.BigEndian.PutUint64(marker, uint64(height))
	copy(marker[8:], h[:])
	if err := s.db.Write(spPendingKey, marker); err != nil {
		return err
	}
	return capability.Sync()
}

func (s *silentPayState) writeTo(wb common.WriteBatch) error {
	if s.poisoned != nil {
		return s.poisoned
	}
	heights := make([]int, 0, len(s.blocks))
	for h := range s.blocks {
		heights = append(heights, h)
	}
	sort.Ints(heights)
	for _, h := range heights {
		block := s.blocks[h]
		if len(block.indices) != len(block.records) {
			return ErrSPCorrupt
		}
		header, err := spEncodeBlockDirectory(block.hash, block.indices)
		if err != nil {
			return err
		}
		if err := wb.Put(spBlockKey(h), header); err != nil {
			return err
		}
		for i, index := range block.indices {
			if err := wb.Put(spPositionKey(spArchivePrefix, h, index), block.records[i]); err != nil {
				return err
			}
		}
	}
	keys := make([]string, 0, len(s.live))
	for key := range s.live {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		change := s.live[key]
		var err error
		if change.bitmap == nil {
			err = wb.Delete([]byte(key))
		} else {
			err = wb.Put([]byte(key), change.bitmap)
		}
		if err != nil {
			return err
		}
	}
	return wb.Put(spSchemaKey, s.schemaBytes())
}

func (s *silentPayState) finishCommit() error {
	capability, ok := s.db.(common.SnapshotDB)
	if !ok {
		return ErrSPNotReady
	}
	if err := capability.Sync(); err != nil {
		return err
	}
	if err := s.db.Delete(spPendingKey); err != nil {
		return err
	}
	if err := capability.Sync(); err != nil {
		return err
	}
	// Drop references after durable completion. Do not clear shared blocks in
	// place: another compiling/commit clone can still own immutable records.
	s.blocks = make(map[int]*spPendingBlock)
	s.live = make(map[string]spLiveChange)
	return nil
}

func spReadAnchor(view common.DBReadView, network string) (SPAnchor, error) {
	anchor := SPAnchor{Height: -1}
	if _, err := view.Read(spPendingKey); err == nil {
		return anchor, ErrSPIncompleteCommit
	} else if !errors.Is(err, common.ErrKeyNotFound) {
		return anchor, err
	}
	raw, err := view.Read([]byte(SyncStatsKey))
	if errors.Is(err, common.ErrKeyNotFound) {
		return anchor, ErrSPNotReady // fresh database, no committed Base height
	}
	if err != nil {
		return anchor, err
	}
	var stats SyncStats
	if err := indexdb.DecodeBytes(raw, &stats); err != nil {
		return anchor, err
	}
	if stats.SyncHeight < 0 {
		return anchor, ErrSPNotReady
	}
	if !spHashStringValid(stats.SyncBlockHash) {
		return anchor, fmt.Errorf("%w: invalid committed Base anchor", ErrSPCorrupt)
	}
	schema, err := view.Read(spSchemaKey)
	if err != nil {
		if errors.Is(err, common.ErrKeyNotFound) {
			return anchor, fmt.Errorf("%w: committed SilentPay schema missing", ErrSPCorrupt)
		}
		return anchor, err
	}
	if !bytes.Equal(schema, []byte(fmt.Sprintf("%d:%s", SPSchemaVersion, network))) {
		return anchor, fmt.Errorf("%w: committed SilentPay schema mismatch", ErrSPCorrupt)
	}
	anchor.Height, anchor.BlockHash = stats.SyncHeight, stats.SyncBlockHash
	return anchor, nil
}

func spVoutString(vout uint32) string { return strconv.FormatUint(uint64(vout), 10) }
