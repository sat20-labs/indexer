package btclucky

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"time"

	btcblockchain "github.com/btcsuite/btcd/blockchain"
	btcbtcjson "github.com/btcsuite/btcd/btcjson"
	btcbtcutil "github.com/btcsuite/btcd/btcutil"
	btcchaincfg "github.com/btcsuite/btcd/chaincfg"
	btcchainhash "github.com/btcsuite/btcd/chaincfg/chainhash"
	btctxscript "github.com/btcsuite/btcd/txscript"
	btcwire "github.com/btcsuite/btcd/wire"
)

const coinbaseTag = "satoshinet"

// The first leaf is supplied later by the miner's coinbase. Only siblings
// along its path are needed to compute its Merkle root.
func newBlockTemplate(t *btcbtcjson.GetBlockTemplateResult, network string) (*BlockTemplate, error) {
	if t == nil || t.CoinbaseValue == nil {
		return nil, fmt.Errorf("missing coinbase value")
	}
	result := &BlockTemplate{GetBlockTemplateResult: *t, Network: network, MerkleBranch: []string{}}
	target, err := targetFromTemplate(t)
	if err != nil {
		return nil, err
	}
	result.Target = fmt.Sprintf("%064x", target)
	hashes := make([]btcchainhash.Hash, 1+len(t.Transactions))
	for i, tx := range t.Transactions {
		h, err := btcchainhash.NewHashFromStr(tx.TxID)
		if err != nil || len(tx.TxID) != 64 || tx.Data == "" {
			return nil, fmt.Errorf("invalid template transaction %d", i)
		}
		hashes[i+1] = *h
	}
	for len(hashes) > 1 {
		result.MerkleBranch = append(result.MerkleBranch, hashes[1].String())
		// Skip index zero, which depends on the as-yet unknown coinbase.
		for i := 2; i < len(hashes); i += 2 {
			right := min(i+1, len(hashes)-1)
			hashes[i/2] = hashPair(hashes[i], hashes[right])
		}
		hashes = hashes[:(len(hashes)+1)/2]
	}
	return result, nil
}

type miningHeader [btcwire.MaxBlockHeaderPayload]byte

func (t *BlockTemplate) work(params *btcchaincfg.Params, address string, extraNonce uint64) (*btcwire.MsgTx, btcwire.BlockHeader, error) {
	var header btcwire.BlockHeader
	cb, err := buildCoinbaseTx(&t.GetBlockTemplateResult, params, address, extraNonce)
	if err != nil {
		return nil, header, err
	}
	root := cb.TxHash()
	for _, s := range t.MerkleBranch {
		sibling, err := btcchainhash.NewHashFromStr(s)
		if err != nil {
			return nil, header, err
		}
		root = hashPair(root, *sibling)
	}
	prev, err := btcchainhash.NewHashFromStr(t.PreviousHash)
	if err != nil {
		return nil, header, err
	}
	bits, err := strconv.ParseUint(t.Bits, 16, 32)
	if err != nil {
		return nil, header, err
	}
	header = btcwire.BlockHeader{
		Version:    t.Version,
		PrevBlock:  *prev,
		MerkleRoot: root,
		Timestamp:  time.Unix(t.CurTime, 0),
		Bits:       uint32(bits),
	}
	return cb, header, nil
}

// Keep the exact transaction order and witness bytes delivered by Core.
func (t *BlockTemplate) blockBytes(cb *btcwire.MsgTx, header btcwire.BlockHeader) ([]byte, error) {
	var buf bytes.Buffer
	if err := header.Serialize(&buf); err != nil {
		return nil, err
	}
	if err := btcwire.WriteVarInt(&buf, 0, uint64(len(t.Transactions))+1); err != nil {
		return nil, err
	}
	if err := cb.Serialize(&buf); err != nil {
		return nil, err
	}
	for _, tx := range t.Transactions {
		raw, err := hex.DecodeString(tx.Data)
		if err != nil {
			return nil, err
		}
		buf.Write(raw)
	}
	return buf.Bytes(), nil
}

func (h *miningHeader) hashNonce(nonce uint32) btcchainhash.Hash {
	// The nonce occupies the last four bytes of the serialized Bitcoin header.
	binary.LittleEndian.PutUint32(h[len(h)-4:], nonce)
	return btcchainhash.DoubleHashH(h[:])
}

func btcPayToAddrScript(addr string, params *btcchaincfg.Params) ([]byte, error) {
	decoded, err := btcbtcutil.DecodeAddress(addr, params)
	if err != nil {
		return nil, err
	}
	if !decoded.IsForNet(params) {
		return nil, fmt.Errorf("address %s is not for btc network %s", addr, params.Name)
	}
	return btctxscript.PayToAddrScript(decoded)
}

func buildCoinbaseTx(template *btcbtcjson.GetBlockTemplateResult, params *btcchaincfg.Params, rewardAddress string, extraNonce uint64) (*btcwire.MsgTx, error) {
	pkScript, err := btcPayToAddrScript(rewardAddress, params)
	if err != nil {
		return nil, fmt.Errorf("invalid btc lucky reward address: %w", err)
	}
	if template.CoinbaseValue == nil {
		return nil, fmt.Errorf("missing coinbase value")
	}

	extraNonceBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(extraNonceBytes, extraNonce)
	sigScript, err := btctxscript.NewScriptBuilder().
		AddInt64(template.Height).
		AddData([]byte(coinbaseTag)).
		AddData(extraNonceBytes).
		Script()
	if err != nil {
		return nil, err
	}

	tx := btcwire.NewMsgTx(1)
	tx.AddTxIn(&btcwire.TxIn{
		PreviousOutPoint: btcwire.OutPoint{
			Index: ^uint32(0),
		},
		SignatureScript: sigScript,
		Sequence:        ^uint32(0),
	})
	tx.AddTxOut(btcwire.NewTxOut(*template.CoinbaseValue, pkScript))

	if template.DefaultWitnessCommitment != "" {
		commitment, err := hex.DecodeString(template.DefaultWitnessCommitment)
		if err != nil {
			return nil, fmt.Errorf("invalid default witness commitment: %w", err)
		}
		tx.TxIn[0].Witness = btcwire.TxWitness{make([]byte, 32)}
		tx.AddTxOut(btcwire.NewTxOut(0, commitment))
	}

	return tx, nil
}

func hashPair(left, right btcchainhash.Hash) btcchainhash.Hash {
	var data [64]byte
	copy(data[:32], left[:])
	copy(data[32:], right[:])
	first := sha256.Sum256(data[:])
	second := sha256.Sum256(first[:])
	var result btcchainhash.Hash
	copy(result[:], second[:])
	return result
}

func targetFromTemplate(template *btcbtcjson.GetBlockTemplateResult) (*big.Int, error) {
	if template.Target != "" {
		target, ok := new(big.Int).SetString(template.Target, 16)
		if !ok {
			return nil, fmt.Errorf("invalid template target %q", template.Target)
		}
		return target, nil
	}
	bits64, err := strconv.ParseUint(template.Bits, 16, 32)
	if err != nil {
		return nil, err
	}
	return btcblockchain.CompactToBig(uint32(bits64)), nil
}
