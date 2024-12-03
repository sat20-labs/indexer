package common

import (
	"time"

	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common/pb"
)

type Range = pb.MyRange

type Input struct {
	Txid     string         `json:"txid"`
	UtxoId   uint64         `json:"utxoid"`
	Address  *ScriptPubKey  `json:"scriptPubKey"`
	Vout     int64          `json:"vout"`
	Ordinals []*Range       `json:"ordinals"`
	Witness  wire.TxWitness `json:"witness"`
}

type ScriptPubKey struct {
	Addresses []string             `json:"addresses"`
	Type     int                   `json:"type"`
	ReqSig   int                   `json:"reqSig"`
	PkScript []byte        		   `json:"pkscript"`
}

type Output struct {
	Height   int           `json:"height"`
	TxId     int           `json:"txid"`
	Value    int64         `json:"value"`
	Address  *ScriptPubKey `json:"scriptPubKey"`
	N        int64         `json:"n"`
	Ordinals []*Range      `json:"ordinals"`
}

type Transaction struct {
	Txid    string    `json:"txid"`
	Inputs  []*Input  `json:"inputs"`
	Outputs []*Output `json:"outputs"`
}

type Block struct {
	Timestamp     time.Time      `json:"timestamp"`
	Height        int            `json:"height"`
	Hash          string         `json:"hash"`
	PrevBlockHash string         `json:"prevBlockHash"`
	Transactions  []*Transaction `json:"transactions"`
}

type UTXOIndex struct {
	Index map[string]*Output
}
