package extension

import (
	serverCommon "github.com/sat20-labs/ordx/server/define"
)

// type Atomical struct {
// 	AtomicalId     string  `json:"atomicalId"`
// 	AtomicalNumber uint64  `json:"atomicalNumber"`
// 	Type           string  `json:"type"`
// 	Ticker         *string `json:"ticker"`
// }

type Rune struct {
	RuneId string `json:"runeId"`
	Rune   string `json:"rune"`
	Amount string `json:"amount"`
}

type Inscription struct {
	Id                 string `json:"inscriptionId"`
	Number             int64  `json:"inscriptionNumber"`
	Address            string `json:"address,omitempty"`
	OutputValue        uint64 `json:"outputValue,omitempty"`
	Preview            string `json:"preview,omitempty"`
	Content            string `json:"content,omitempty"`
	ContentType        string `json:"contentType,omitempty"`
	ContentLength      uint   `json:"contentLength,omitempty"`
	Timestamp          int64  `json:"timestamp,omitempty"`
	GenesisTransaction string `json:"genesisTransaction,omitempty"`
	Location           string `json:"location,omitempty"`
	Output             string `json:"output,omitempty"`
	Offset             int64  `json:"offset"`
	ContentBody        string `json:"contentBody"`
	Height             int64  `json:"utxoHeight,omitempty"`
	Confirmation       int    `json:"utxoConfirmation,omitempty"`
}

type Utxo struct {
	Txid         string         `json:"txid"`
	Vout         int            `json:"vout"`
	Satoshis     uint64         `json:"satoshis"`
	ScriptPk     string         `json:"scriptPk"`
	AddressType  AddressType    `json:"addressType"`
	Inscriptions []*Inscription `json:"inscriptions"`
	// Atomicals    []*Atomical    `json:"atomicals"`
	Runes []*Rune `json:"runes"`
}

type UtxoAssetSummary struct {
	UtxoId uint64 `json:"utxoId"`
	Amount int64  `json:"amount"`
}

type UtxoDetail struct {
	Txid         string         `json:"txid"`
	Vout         int            `json:"outputIndex"`
	Satoshis     uint64         `json:"satoshis"`
	ScriptPk     string         `json:"scriptPk"`
	AddressType  AddressType    `json:"addressType"`
	Inscriptions []*Inscription `json:"inscriptions"`
}

// /version/detail
type VersionDetailData struct {
	Changelogs []interface{} `json:"changelogs"`
	Version    string        `json:"version"`
	Title      string        `json:"title"`
}

type VersionDetailResp struct {
	serverCommon.BaseResp
	Data *VersionDetailData `json:"data"`
}
