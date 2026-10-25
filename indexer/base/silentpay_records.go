package base

import (
	"bytes"
	"fmt"
	"io"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
)

// A pending block is immutable after process publishes it. Compiling and
// commit clones share its compact records, not expanded public SPTransaction
// objects. The on-disk v1 encoding is also the in-memory Archive encoding.
type spPendingBlock struct {
	hash    string
	indices []uint32
	records [][]byte
}

func spEncodeBlockDirectory(hashText string, indices []uint32) ([]byte, error) {
	hash, err := chainhash.NewHashFromStr(hashText)
	if err != nil || len(hashText) != 64 {
		return nil, fmt.Errorf("%w: block hash", ErrSPCorrupt)
	}
	buf := bytes.NewBuffer(make([]byte, 0, 33+wire.VarIntSerializeSize(uint64(len(indices)))+len(indices)*3))
	buf.WriteByte(SPSchemaVersion)
	buf.Write(hash[:])
	if err := wire.WriteVarInt(buf, 0, uint64(len(indices))); err != nil {
		return nil, err
	}
	var previous uint32
	for _, index := range indices {
		if index == 0 || index <= previous {
			return nil, fmt.Errorf("%w: transaction order", ErrSPCorrupt)
		}
		if err := wire.WriteVarInt(buf, 0, uint64(index-previous)); err != nil {
			return nil, err
		}
		previous = index
	}
	return buf.Bytes(), nil
}

// spCandidateRecord encodes directly from Base's existing transaction. It does
// not allocate one SPOutput/key slice per output just to serialize it again.
func spCandidateRecord(tx *common.Transaction, index uint32) ([]byte, int, error) {
	if tx == nil {
		return nil, 0, ErrSPCorrupt
	}
	if index == 0 {
		return nil, 0, nil
	}
	count, outputBytes := 0, 0
	var previous uint32
	for _, out := range tx.Outputs {
		if out == nil {
			return nil, 0, fmt.Errorf("%w: nil output", ErrSPCorrupt)
		}
		if !common.SPP2TROutput(out.OutValue.PkScript) {
			continue
		}
		if out.TxOutIndex < 0 || uint64(out.TxOutIndex) > uint64(^uint32(0)) {
			return nil, 0, ErrSPCorrupt
		}
		vout := uint32(out.TxOutIndex)
		if count > 0 && vout <= previous {
			return nil, 0, fmt.Errorf("%w: output order", ErrSPCorrupt)
		}
		outputBytes += wire.VarIntSerializeSize(uint64(vout-previous)) + 32
		previous = vout
		count++
	}
	if count == 0 {
		return nil, 0, nil
	}
	size := 66 + wire.VarIntSerializeSize(uint64(count)) + outputBytes
	if count > spMaxOutputs || size > spMaxRecordBytes {
		return nil, 0, ErrSPCorrupt
	}

	// Accepted limitation: Base's address-based prevout reconstruction is not
	// lossless for P2PK and bare multisig. GetAddress/EncodeAddress can turn
	// those scripts into a P2PKH address before prefetchIndexesFromDB rebuilds
	// the script. SP does not support those source-output categories, including
	// mixed-input transactions involving them. In particular, merely excluding
	// P2PK in SPInputPublicKey does not fix a script already reconstructed as
	// P2PKH. No historical RPC fallback or Base address-format change is made
	// here; a future extension must preserve the ORIGINAL script semantics.
	inputs := make([]common.SPInput, len(tx.Inputs))
	for i, in := range tx.Inputs {
		if in == nil || in.TxOutIndex < 0 || uint64(in.TxOutIndex) > uint64(^uint32(0)) {
			return nil, 0, ErrSPCorrupt
		}
		hash, err := chainhash.NewHashFromStr(in.TxId)
		if err != nil || len(in.TxId) != 64 {
			return nil, 0, fmt.Errorf("%w: input txid", ErrSPCorrupt)
		}
		inputs[i] = common.SPInput{
			Outpoint:        wire.OutPoint{Hash: *hash, Index: uint32(in.TxOutIndex)},
			SignatureScript: in.SignatureScript, Witness: in.Witness, Prevout: &in.OutValue,
		}
	}
	tweak, eligible, err := common.SPComputeTweak(inputs)
	if err != nil || !eligible {
		return nil, 0, err
	}
	hash, err := chainhash.NewHashFromStr(tx.TxId)
	if err != nil || len(tx.TxId) != 64 {
		return nil, 0, fmt.Errorf("%w: transaction txid", ErrSPCorrupt)
	}
	buf := bytes.NewBuffer(make([]byte, 0, size))
	buf.WriteByte(SPSchemaVersion)
	buf.Write(hash[:])
	buf.Write(tweak[:])
	if err := wire.WriteVarInt(buf, 0, uint64(count)); err != nil {
		return nil, 0, err
	}
	previous = 0
	for _, out := range tx.Outputs {
		if !common.SPP2TROutput(out.OutValue.PkScript) {
			continue
		}
		vout := uint32(out.TxOutIndex)
		if err := wire.WriteVarInt(buf, 0, uint64(vout-previous)); err != nil {
			return nil, 0, err
		}
		buf.Write(out.OutValue.PkScript[2:])
		previous = vout
	}
	return buf.Bytes(), count, nil
}

// spRecordHeader validates the bounded v1 header. Output walking below borrows
// key bytes from the record, without allocating an output object for each key.
func spRecordHeader(raw []byte) (*bytes.Reader, int, error) {
	r, count, err := spRecordHeaderStructure(raw)
	if err != nil {
		return nil, 0, err
	}
	if _, err := btcec.ParsePubKey(raw[33:66]); err != nil {
		return nil, 0, ErrSPCorrupt
	}
	return r, count, nil
}

// Daily consistency checks need framing and output positions, but do not
// audit T's curve membership. Queries, spending and full audits continue to
// use spRecordHeader above for strict compressed-public-key validation.
func spRecordHeaderStructure(raw []byte) (*bytes.Reader, int, error) {
	if len(raw) < 67 || len(raw) > spMaxRecordBytes || raw[0] != SPSchemaVersion {
		return nil, 0, ErrSPCorrupt
	}
	if raw[33] != 2 && raw[33] != 3 {
		return nil, 0, ErrSPCorrupt
	}
	r := bytes.NewReader(raw[66:])
	count, err := wire.ReadVarInt(r, 0)
	if err != nil || count == 0 || count > spMaxOutputs || count > uint64(r.Len()/33) {
		return nil, 0, ErrSPCorrupt
	}
	return r, int(count), nil
}

// key is an immutable view into raw. The full-slice bound prevents append from
// overwriting the next output. Callers must not mutate either raw or key.
func spNextRecordOutput(raw []byte, r *bytes.Reader, ordinal int, previous uint32) (uint32, []byte, error) {
	delta, err := wire.ReadVarInt(r, 0)
	if err != nil || (ordinal > 0 && delta == 0) || delta > uint64(^uint32(0)-previous) || r.Len() < 32 {
		return 0, nil, ErrSPCorrupt
	}
	start := len(raw) - r.Len()
	key := raw[start : start+32 : start+32]
	if _, err := r.Seek(32, io.SeekCurrent); err != nil {
		return 0, nil, ErrSPCorrupt
	}
	return previous + uint32(delta), key, nil
}
