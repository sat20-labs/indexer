package extension

import (
	"github.com/sat20-labs/ordx/server/define"
)

// /tx/decode2
type TxDecode2Req struct {
	PsbtHex string `json:"psbtHex" binding:"required"`
	Website string `json:"website,omitempty"`
}

type InputInfo struct {
	Txid         string        `json:"txid"`
	Vout         int           `json:"vout"`
	Address      string        `json:"address"`
	Value        int64         `json:"value"`
	Inscriptions []Inscription `json:"inscriptions"`
	// Atomicals    []Atomical    `json:"atomicals"`
	// SighashType  int           `json:"sighashType"`
	Runes []RuneBalance `json:"runes"`
}

type OutputInfo struct {
	Address      string        `json:"address"`
	Value        int64         `json:"value"`
	Inscriptions []Inscription `json:"inscriptions"`
	// Atomicals    []Atomical    `json:"atomicals"`
	Runes []RuneBalance `json:"runes"`
}

type Risk struct {
	Type  RiskType `json:"type"`
	Level string   `json:"level"`
	Title string   `json:"title"`
	Desc  string   `json:"desc"`
}

type TxDecode2Features struct {
	Rbf bool `json:"rbf"`
}

type TxDecode2Data struct {
	InputInfos         []InputInfo             `json:"inputInfos"`
	OutputInfos        []OutputInfo            `json:"outputInfos"`
	FeeRate            string                  `json:"feeRate"`
	Fee                int64                   `json:"fee"`
	Features           *TxDecode2Features      `json:"features"`
	Risks              []Risk                  `json:"risks"`
	IsScammer          bool                    `json:"isScammer"`
	Inscriptions       map[string]*Inscription `json:"inscriptions"`
	RecommendedFeeRate int64                   `json:"recommendedFeeRate"`
	ShouldWarnFeeRate  bool                    `json:"shouldWarnFeeRate"`
}

type TxDecode2Resp struct {
	define.BaseResp
	Data *TxDecode2Data `json:"data"`
}

// /tx/broadcast
type TxBroadcastReq struct {
	Rawtx string `json:"rawtx" binding:"required"`
}

type TxBroadcastResp struct {
	define.BaseResp
	Data string `json:"data"`
}
