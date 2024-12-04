package extension

import (
	"github.com/sat20-labs/indexer/common"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
)

// type TokenAsset struct {
// 	InscriptionID  string          `json:"inscriptionId,omitempty"`
// 	InscriptionNum uint64          `json:"inscriptionNumber,omitempty"`
// 	AssetAmount    int64           `json:"assetAmount,omitempty"`
// 	Ranges         []*common.Range `json:"ranges,omitempty"`
// }

// type UtxoTokenAsset struct {
// 	Ticker      string        `json:"ticker,omitempty"`
// 	Utxo        string        `json:"utxo,omitempty"`
// 	Amount      int64         `json:"amount,omitempty"`
// 	AssetAmount int64         `json:"assetAmount,omitempty"`
// 	AssetList   []*TokenAsset `json:"assets,omitempty"`
// }

type UtxoTokenAsset struct {
	Ticker             string          `json:"ticker,omitempty"`
	Utxo               string          `json:"utxo,omitempty"`
	Amount             int64           `json:"amount,omitempty"`
	InscriptionID      string          `json:"inscriptionId,omitempty"`
	InscriptionNum     uint64          `json:"inscriptionNumber,omitempty"`
	AssetAmount        int64           `json:"assetAmount,omitempty"`
	Ranges             []*common.Range `json:"ranges,omitempty"`
	Address            string          `json:"address,omitempty"`
	OutputValue        uint64          `json:"outputValue,omitempty"`
	Preview            string          `json:"preview,omitempty"`
	Content            string          `json:"content,omitempty"`
	ContentType        string          `json:"contentType,omitempty"`
	ContentLength      uint            `json:"contentLength,omitempty"`
	Timestamp          int64           `json:"timestamp,omitempty"`
	GenesisTransaction string          `json:"genesisTransaction,omitempty"`
	Location           string          `json:"location,omitempty"`
	Output             string          `json:"output,omitempty"`
	Offset             int64           `json:"offset"`
	ContentBody        string          `json:"contentBody"`
	Height             int64           `json:"utxoHeight,omitempty"`
	Confirmation       int             `json:"utxoConfirmation,omitempty"`
}

type TokenListData struct {
	rpcwire.ListResp
	List []*UtxoTokenAsset `json:"list,omitempty"`
}

type TokenListResp struct {
	rpcwire.BaseResp
	Data *TokenListData `json:"data"`
}
