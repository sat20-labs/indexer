package common

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/wire"
)

// SPInput is an ephemeral, public-only view of a fully resolved Bitcoin input.
// Prevout must contain the original output script, even for spent outputs.
// Nothing here is a scan key, a spend key or evidence of wallet ownership.
type SPInput struct {
	Outpoint        wire.OutPoint
	SignatureScript []byte
	Witness         wire.TxWitness
	Prevout         *wire.TxOut
}

// SPNUMSInternalKey is the public BIP341 NUMS point used by BIP352.
// Callers must treat this test/reference value as immutable.
var SPNUMSInternalKey = [32]byte{0x50,0x92,0x9b,0x74,0xc1,0xa0,0x49,0x54,0xb7,0x8b,0x4b,0x60,0x35,0xe9,0x7a,0x5e,0x07,0x8a,0x5a,0x0f,0x28,0xec,0x96,0xd5,0x47,0xbf,0xee,0x9a,0xce,0x80,0x3a,0xc0}

const spNUMSHex = "50929b74c1a04954b78b4b6035e97a5e078a5a0f28ec96d547bfee9ace803ac0"

// SPP2TROutput recognizes the exact witness-v1 32-byte program. The bytes need
// not lift to a curve point: retain the original program in the public archive.
func SPP2TROutput(script []byte) bool {
	return len(script) == 34 && script[0] == 0x51 && script[1] == 0x20
}

func spFutureWitness(script []byte) bool {
	return len(script) >= 4 && len(script) <= 42 && script[0] >= 0x52 && script[0] <= 0x60 && int(script[1]) == len(script)-2
}

func spCompressedKey(raw []byte) *btcec.PublicKey {
	if len(raw) != 33 || (raw[0] != 2 && raw[0] != 3) { return nil }
	key, err := btcec.ParsePubKey(raw)
	if err != nil { return nil }
	return key
}

// SPInputPublicKey implements BIP352 v0 input selection, including non-template
// P2PKH scriptSigs, taproot parity and the NUMS script-path exclusion. A nil
// key with no error means an intentionally unsupported input, not missing data.
func SPInputPublicKey(in SPInput) (*btcec.PublicKey, error) {
	if in.Prevout == nil { return nil, fmt.Errorf("silentpay: missing prevout %s", in.Outpoint) }
	script := in.Prevout.PkScript
	switch {
	case SPP2TROutput(script):
		w := in.Witness
		if len(w) == 0 { return nil, fmt.Errorf("silentpay: missing taproot witness %s", in.Outpoint) }
		if len(w) > 1 && len(w[len(w)-1]) > 0 && w[len(w)-1][0] == 0x50 { w = w[:len(w)-1] }
		if len(w) > 1 {
			control := w[len(w)-1]
			if len(control) < 33 || (len(control)-33)%32 != 0 || len(control) > 33+128*32 {
				return nil, fmt.Errorf("silentpay: malformed control block %s", in.Outpoint)
			}
			if hex.EncodeToString(control[1:33]) == spNUMSHex { return nil, nil }
		}
		return spCompressedKey(append([]byte{2}, script[2:]...)), nil
	case len(script) == 22 && script[0] == 0 && script[1] == 20:
		if len(in.Witness) != 2 { return nil, fmt.Errorf("silentpay: missing P2WPKH witness %s", in.Outpoint) }
		key := in.Witness[1]
		if !bytes.Equal(btcutil.Hash160(key), script[2:]) { return nil, fmt.Errorf("silentpay: P2WPKH key mismatch %s", in.Outpoint) }
		return spCompressedKey(key), nil
	case len(script) == 23 && script[0] == 0xa9 && script[1] == 20 && script[22] == 0x87:
		ss := in.SignatureScript
		if len(ss) != 23 || ss[0] != 22 || ss[1] != 0 || ss[2] != 20 { return nil, nil }
		if !bytes.Equal(btcutil.Hash160(ss[1:]), script[2:22]) { return nil, fmt.Errorf("silentpay: redeem script mismatch %s", in.Outpoint) }
		if len(in.Witness) != 2 { return nil, fmt.Errorf("silentpay: missing wrapped P2WPKH witness %s", in.Outpoint) }
		key := in.Witness[1]
		if !bytes.Equal(btcutil.Hash160(key), ss[3:]) { return nil, fmt.Errorf("silentpay: wrapped key mismatch %s", in.Outpoint) }
		return spCompressedKey(key), nil
	case len(script) == 25 && script[0] == 0x76 && script[1] == 0xa9 && script[2] == 20 && script[23] == 0x88 && script[24] == 0xac:
		// P2PKH scriptSig may have consensus-valid malleations. Do not require
		// two canonical pushes or merely take its last 33 bytes.
		for end := len(in.SignatureScript); end >= 33; end-- {
			raw := in.SignatureScript[end-33:end]
			if (raw[0] == 2 || raw[0] == 3) && bytes.Equal(btcutil.Hash160(raw), script[3:23]) {
				return spCompressedKey(raw), nil
			}
		}
		return nil, nil // e.g. an uncompressed legacy pubkey
	default:
		return nil, nil
	}
}

func spTaggedHash(tag string, message []byte) [32]byte {
	t := sha256.Sum256([]byte(tag))
	h := sha256.New()
	h.Write(t[:]); h.Write(t[:]); h.Write(message)
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result
}

func spSerializeOutpoint(op wire.OutPoint) [36]byte {
	var out [36]byte
	copy(out[:32], op.Hash[:])
	binary.LittleEndian.PutUint32(out[32:], op.Index)
	return out
}

func spTweakFromAggregate(outpoints []wire.OutPoint, aggregate *btcec.PublicKey) ([33]byte, error) {
	var result [33]byte
	if len(outpoints) == 0 || aggregate == nil { return result, fmt.Errorf("silentpay: empty input aggregate") }
	lowest := spSerializeOutpoint(outpoints[0])
	for _, op := range outpoints[1:] {
		candidate := spSerializeOutpoint(op)
		if bytes.Compare(candidate[:], lowest[:]) < 0 { lowest = candidate }
	}
	message := append(lowest[:], aggregate.SerializeCompressed()...)
	hash := spTaggedHash("BIP0352/Inputs", message)
	var scalar btcec.ModNScalar
	if scalar.SetByteSlice(hash[:]) || scalar.IsZero() { return result, fmt.Errorf("silentpay: invalid input hash scalar") }
	var a, product btcec.JacobianPoint
	aggregate.AsJacobian(&a)
	btcec.ScalarMultNonConst(&scalar, &a, &product)
	if product.Z.IsZero() { return result, fmt.Errorf("silentpay: invalid tweak point") }
	product.ToAffine()
	copy(result[:], btcec.NewPublicKey(&product.X, &product.Y).SerializeCompressed())
	return result, nil
}

// SPComputeTweak returns T=input_hash*A. Eligibility also requires at least one
// P2TR output in the transaction; the caller checks outputs before calling this.
// Every input contributes to the minimum outpoint, even unsupported input types.
// Duplicate pubkeys are summed repeatedly; intermediate infinity is permitted.
func SPComputeTweak(inputs []SPInput) ([33]byte, bool, error) {
	var zero [33]byte
	if len(inputs) == 0 { return zero, false, nil }
	var sum btcec.JacobianPoint
	outpoints := make([]wire.OutPoint, len(inputs))
	for i, in := range inputs {
		if in.Outpoint.Hash == ([32]byte{}) && in.Outpoint.Index == 0xffffffff { return zero, false, nil }
		if in.Prevout == nil { return zero, false, fmt.Errorf("silentpay: missing prevout %s", in.Outpoint) }
		outpoints[i] = in.Outpoint
	}
	for _, in := range inputs { if spFutureWitness(in.Prevout.PkScript) { return zero, false, nil } }
	for _, in := range inputs {
		key, err := SPInputPublicKey(in)
		if err != nil { return zero, false, err }
		if key == nil { continue }
		var point, next btcec.JacobianPoint
		key.AsJacobian(&point)
		btcec.AddNonConst(&sum, &point, &next)
		sum = next
	}
	if sum.Z.IsZero() { return zero, false, nil }
	sum.ToAffine()
	tweak, err := spTweakFromAggregate(outpoints, btcec.NewPublicKey(&sum.X, &sum.Y))
	if err != nil { return zero, false, err }
	return tweak, true, nil
}
