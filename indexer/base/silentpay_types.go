package base

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

const SPSchemaVersion = 1

var (
	ErrSPNotReady         = errors.New("silentpay_not_ready")
	ErrSPNotInitialized  = errors.New("silentpay_not_initialized")
	ErrSPIncompleteCommit = errors.New("silentpay_incomplete_base_commit: rebuild from a complete baseline")
	ErrSPCorrupt          = errors.New("silentpay_corrupt_data")
	ErrSPSnapshotExpired  = errors.New("snapshot_expired")
	ErrSPInvalidRequest   = errors.New("invalid_silentpay_request")
	ErrSPSnapshotLimit    = errors.New("silentpay_snapshot_limit")
)

// Disk keys use fixed-width big-endian creation positions. Values are binary,
// not hex/JSON. A transaction and its T are stored once, never once per output.
var (
	spBlockPrefix   = []byte("sp/b/")
	spArchivePrefix = []byte("sp/a/")
	spLivePrefix    = []byte("sp/l/")
	spSchemaKey     = []byte("sp/schema")
	spPendingKey    = []byte("sp/commit-pending")
)

const spMaxOutputs = 100000 // structural allocation bound, NOT BIP352 K_max
const spMaxRecordBytes = 4 * 1024 * 1024
const spMaxPageBytes = 8 * 1024 * 1024
const spMaxPageTransactions = 500

type SPAnchor struct {
	Height    int    `json:"height" cbor:"height"`
	BlockHash string `json:"block_hash" cbor:"block_hash"`
}

type SPOutput struct {
	Vout      uint32 `json:"vout" cbor:"vout"`
	OutputKey []byte `json:"output_key" cbor:"output_key"`
}

// Public query representation only. Pending Archive and spending paths operate
// on compact records instead of retaining this expanded output-array structure.
type SPTransaction struct {
	TxIndex       uint32     `json:"tx_index" cbor:"tx_index"`
	Tweak         []byte     `json:"tweak" cbor:"tweak"`
	P2TROutputs   []SPOutput `json:"p2tr_outputs" cbor:"p2tr_outputs"`
	UnspentBitmap []byte     `json:"unspent_bitmap,omitempty" cbor:"unspent_bitmap,omitempty"`
	TxID          string     `json:"-" cbor:"-"` // retained for lookup, omitted during scans
}

type SPBlock struct {
	Height       int             `json:"height" cbor:"height"`
	BlockHash    string          `json:"block_hash" cbor:"block_hash"`
	Transactions []SPTransaction `json:"transactions" cbor:"transactions"`
}

type SPStatus struct {
	Network             string   `json:"network"`
	SchemaVersion       int      `json:"schema_version"`
	CoverageStartHeight int      `json:"coverage_start_height"`
	BaseAnchor          SPAnchor `json:"base_anchor"`
	DataReady           bool     `json:"data_ready"`
	SupportedModes      []string `json:"supported_modes"`
}

type SPSnapshot struct {
	SnapshotID          string   `json:"snapshot_id"`
	Anchor              SPAnchor `json:"anchor"`
	ExpiresAt           int64    `json:"expires_at"`
	CoverageStartHeight int      `json:"coverage_start_height"`
	SchemaVersion       int      `json:"schema_version"`
}

type SPScanRequest struct {
	Mode       string
	SnapshotID string
	FromHeight int
	ToHeight   int
	AnchorHash string
	Cursor     string
	Limit      int
}

type SPScanPage struct {
	SnapshotID string    `json:"snapshot_id,omitempty" cbor:"snapshot_id,omitempty"`
	Anchor     SPAnchor  `json:"anchor" cbor:"anchor"`
	Blocks     []SPBlock `json:"blocks" cbor:"blocks"`
	NextCursor string    `json:"next_cursor,omitempty" cbor:"next_cursor,omitempty"`
	Done       bool      `json:"done" cbor:"done"`
}

type SPPosition struct {
	Height       int    `json:"height"`
	BlockHash    string `json:"block_hash"`
	TxIndex      uint32 `json:"tx_index"`
	Vout         uint32 `json:"vout,omitempty"`
	ExpectedTxID string `json:"expected_txid,omitempty"`
}

type SPPositionResult struct {
	SPPosition
	TxID  string `json:"txid,omitempty"`
	State string `json:"state,omitempty"`
	Error string `json:"error,omitempty"`
}

func spStartHeight(params *chaincfg.Params) int {
	if params.Name == chaincfg.MainNetParams.Name {
		return 709632
	}
	// The Indexer maps its testnet setting to testnet4, where Taproot is
	// active from genesis. Regtest fixtures likewise use active Taproot.
	return 0
}

func spBlockKey(height int) []byte {
	key := make([]byte, len(spBlockPrefix)+4)
	copy(key, spBlockPrefix)
	binary.BigEndian.PutUint32(key[len(spBlockPrefix):], uint32(height))
	return key
}

func spPositionKey(prefix []byte, height int, txIndex uint32) []byte {
	key := make([]byte, len(prefix)+8)
	copy(key, prefix)
	binary.BigEndian.PutUint32(key[len(prefix):], uint32(height))
	binary.BigEndian.PutUint32(key[len(prefix)+4:], txIndex)
	return key
}

func spParsePosition(prefix, key []byte) (int, uint32, error) {
	if len(key) != len(prefix)+8 || !bytes.HasPrefix(key, prefix) {
		return 0, 0, ErrSPCorrupt
	}
	height := int(binary.BigEndian.Uint32(key[len(prefix):]))
	return height, binary.BigEndian.Uint32(key[len(prefix)+4:]), nil
}

func spEncodeBlock(block SPBlock) ([]byte, error) {
	indices := make([]uint32, len(block.Transactions))
	for i, tx := range block.Transactions {
		indices[i] = tx.TxIndex
	}
	return spEncodeBlockDirectory(block.BlockHash, indices)
}

func spDecodeBlock(raw []byte) (string, []uint32, error) {
	if len(raw) < 34 || len(raw) > spMaxRecordBytes || raw[0] != SPSchemaVersion {
		return "", nil, ErrSPCorrupt
	}
	var hash chainhash.Hash
	copy(hash[:], raw[1:33])
	r := bytes.NewReader(raw[33:])
	count, err := wire.ReadVarInt(r, 0)
	if err != nil || count > 100000 || count > uint64(r.Len()) {
		return "", nil, ErrSPCorrupt
	}
	indices := make([]uint32, 0, int(count))
	var previous uint64
	for i := uint64(0); i < count; i++ {
		delta, err := wire.ReadVarInt(r, 0)
		if err != nil || delta == 0 || delta > uint64(^uint32(0))-previous {
			return "", nil, ErrSPCorrupt
		}
		previous += delta
		indices = append(indices, uint32(previous))
	}
	if r.Len() != 0 {
		return "", nil, ErrSPCorrupt
	}
	return hash.String(), indices, nil
}

func spEncodeTransaction(tx SPTransaction) ([]byte, error) {
	if len(tx.Tweak) != 33 || len(tx.P2TROutputs) == 0 || len(tx.P2TROutputs) > spMaxOutputs {
		return nil, ErrSPCorrupt
	}
	if tx.Tweak[0] != 2 && tx.Tweak[0] != 3 {
		return nil, ErrSPCorrupt
	}
	if _, err := btcec.ParsePubKey(tx.Tweak); err != nil {
		return nil, ErrSPCorrupt
	}
	hash, err := chainhash.NewHashFromStr(tx.TxID)
	if err != nil || len(tx.TxID) != 64 {
		return nil, ErrSPCorrupt
	}
	buf := bytes.NewBuffer(append([]byte{SPSchemaVersion}, hash[:]...))
	buf.Write(tx.Tweak)
	if err := wire.WriteVarInt(buf, 0, uint64(len(tx.P2TROutputs))); err != nil {
		return nil, err
	}
	var previous uint32
	for i, out := range tx.P2TROutputs {
		if len(out.OutputKey) != 32 || (i > 0 && out.Vout <= previous) {
			return nil, ErrSPCorrupt
		}
		if err := wire.WriteVarInt(buf, 0, uint64(out.Vout-previous)); err != nil {
			return nil, err
		}
		buf.Write(out.OutputKey)
		previous = out.Vout
	}
	if buf.Len() > spMaxRecordBytes {
		return nil, ErrSPCorrupt
	}
	return buf.Bytes(), nil
}

// raw is a caller-owned record from DB.Read (or a newly encoded test record).
// Borrow immutable slices for T and output keys instead of allocating a separate
// 32-byte object for every output. No DB iterator/transaction-owned bytes escape.
func spDecodeTransaction(raw []byte, index uint32) (SPTransaction, error) {
	tx := SPTransaction{TxIndex: index}
	r, count, err := spRecordHeader(raw)
	if err != nil {
		return tx, err
	}
	var hash chainhash.Hash
	copy(hash[:], raw[1:33])
	tx.TxID = hash.String()
	tx.Tweak = raw[33:66:66]
	tx.P2TROutputs = make([]SPOutput, count)
	var previous uint32
	for i := 0; i < count; i++ {
		vout, key, err := spNextRecordOutput(raw, r, i, previous)
		if err != nil {
			return tx, err
		}
		tx.P2TROutputs[i] = SPOutput{Vout: vout, OutputKey: key}
		previous = vout
	}
	if r.Len() != 0 {
		return tx, ErrSPCorrupt
	}
	return tx, nil
}

func spFullBitmap(count int) []byte {
	bits := make([]byte, (count+7)/8)
	for i := 0; i < count; i++ {
		bits[i/8] |= 1 << uint(i%8)
	}
	return bits
}

func spValidateBitmap(bits []byte, count int, allowEmpty bool) error {
	if len(bits) != (count+7)/8 || count == 0 {
		return ErrSPCorrupt
	}
	if count%8 != 0 && bits[len(bits)-1] & ^byte((1<<uint(count%8))-1) != 0 {
		return ErrSPCorrupt
	}
	var set byte
	for _, b := range bits {
		set |= b
	}
	if set == 0 && !allowEmpty {
		return ErrSPCorrupt
	}
	return nil
}

func spBitmapEmpty(bits []byte) bool {
	for _, b := range bits {
		if b != 0 {
			return false
		}
	}
	return true
}

func spHashStringValid(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
