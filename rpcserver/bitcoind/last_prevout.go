package bitcoind

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/gin-gonic/gin"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

func readPrevOutTx(rpc bitcoin_rpc.BitcoinRPC, id string) (*wire.MsgTx, error) {
	if len(id) != chainhash.MaxHashStringSize {
		return nil, errors.New("invalid Bitcoin transaction id")
	}
	hash, err := chainhash.NewHashFromStr(id)
	if err != nil || *hash == (chainhash.Hash{}) || hash.String() != id {
		return nil, errors.New("invalid Bitcoin transaction id")
	}
	encoded, err := rpc.GetRawTx(id)
	if err != nil {
		return nil, fmt.Errorf("Bitcoin transaction history unavailable: %w", err)
	}
	if len(encoded) == 0 || len(encoded) > 2*wire.MaxBlockPayload {
		return nil, errors.New("invalid Bitcoin transaction size")
	}
	raw, err := hex.DecodeString(encoded)
	if err != nil {
		return nil, errors.New("invalid Bitcoin transaction hex")
	}
	tx := wire.NewMsgTx(2)
	r := bytes.NewReader(raw)
	if err := tx.Deserialize(r); err != nil || r.Len() != 0 || tx.TxHash() != *hash {
		return nil, errors.New("Bitcoin transaction bytes do not match requested txid")
	}
	var canonical bytes.Buffer
	if err := tx.Serialize(&canonical); err != nil || !bytes.Equal(canonical.Bytes(), raw) {
		return nil, errors.New("noncanonical Bitcoin transaction bytes")
	}
	return tx, nil
}

// GetLastInputPrevOut returns the original output consumed by the last
// input. This is a read-only lookup of TxID-committed data, not a claim of
// confirmation, unspent status, or signature authorization.
func (s *Service) GetLastInputPrevOut(txid string) (wire.OutPoint, *wire.TxOut, error) {
	fail := func(err error) (wire.OutPoint, *wire.TxOut, error) {
		return wire.OutPoint{}, nil, err
	}
	rpc := bitcoin_rpc.ShareBitconRpc
	if rpc == nil {
		return fail(errors.New("Bitcoin evidence backend is unavailable"))
	}
	tx, err := readPrevOutTx(rpc, txid)
	if err != nil {
		return fail(err)
	}
	if len(tx.TxIn) == 0 {
		return fail(errors.New("Bitcoin transaction has no inputs"))
	}
	for _, in := range tx.TxIn {
		if in.PreviousOutPoint.Hash == (chainhash.Hash{}) {
			return fail(errors.New("coinbase or invalid Bitcoin previous output"))
		}
	}
	prev := tx.TxIn[len(tx.TxIn)-1].PreviousOutPoint
	parent, err := readPrevOutTx(rpc, prev.Hash.String())
	if err != nil {
		return fail(err)
	}
	if uint64(prev.Index) >= uint64(len(parent.TxOut)) {
		return fail(errors.New("Bitcoin previous output index is out of bounds"))
	}
	out := parent.TxOut[prev.Index]
	return prev, wire.NewTxOut(out.Value, bytes.Clone(out.PkScript)), nil
}

func (s *Service) getLastInputPrevOut(c *gin.Context) {
	prev, out, err := s.GetLastInputPrevOut(c.Param("txid"))
	if err != nil {
		evidenceError(c, err)
		return
	}
	c.JSON(http.StatusOK, &rpcwire.BitcoinPrevOutResp{BaseResp: evidenceOK(),
		Data: &rpcwire.BitcoinPrevOut{Outpoint: prev.String(), Value: out.Value, PkScript: hex.EncodeToString(out.PkScript)},
	})
}
