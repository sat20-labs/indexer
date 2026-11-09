package btclucky

import (
	"time"

	"github.com/btcsuite/btcd/btcjson"
)

// BlockTemplate is shared by every miner. Its ETag identifies the exact bytes
// received over HTTP. All transaction data includes witness serialization.
type BlockTemplate struct {
	btcjson.GetBlockTemplateResult
	Network      string   `json:"network"`
	MerkleBranch []string `json:"merkle_branch"`
	ID           string   `json:"-"`
}

type FoundBlockRecord struct {
	lucky         bool
	BlockHash     string    `json:"blockHash"`
	BlockHeight   int64     `json:"blockHeight"`
	CoinbaseTxID  string    `json:"coinbaseTxid"`
	Vout          uint32    `json:"vout"`
	Amount        int64     `json:"amount"`
	RewardAddress string    `json:"rewardAddress"`
	Submitted     bool      `json:"submitted"`
	SubmitResult  string    `json:"submitResult"`
	CreatedAt     time.Time `json:"createdAt"`
}

type MiningBackend interface {
	CurrentTemplate() (*BlockTemplate, error)
	SubmitBlock([]byte) (*FoundBlockRecord, error)
}
