package ordx

import "github.com/sat20-labs/indexer/rpcserver/wire"

// holder
type HolderV3 struct {
	Wallet       string `json:"wallet,omitempty"`
	TotalBalance string `json:"total_balance,omitempty"`
}

type HolderListDataV3 struct {
	wire.ListResp
	Detail []*HolderV3 `json:"detail"`
}

type HolderListRespV3 struct {
	wire.BaseResp
	Data *HolderListDataV3 `json:"data"`
}

// mint history
type MintHistoryRespV3 struct {
	wire.BaseResp
	Data *MintHistoryDataV3 `json:"data"`
}

type MintHistoryDataV3 struct {
	wire.ListResp
	Detail *MintHistoryV3 `json:"detail"`
}

type MintHistoryV3 struct {
	TypeName string               `json:"type"`
	Ticker   string               `json:"ticker,omitempty"`
	Total    int                  `json:"total,omitempty"`
	Start    int                  `json:"start,omitempty"`
	Limit    int                  `json:"limit,omitempty"`
	Items    []*MintHistoryItemV3 `json:"items,omitempty"`
}

type MintHistoryItemV3 struct {
	MintAddress    string `json:"mintaddress,omitempty" example:"bc1p9jh2caef2ejxnnh342s4eaddwzntqvxsc2cdrsa25pxykvkmgm2sy5ycc5"`
	HolderAddress  string `json:"holderaddress,omitempty"`
	Balance        string `json:"balance,omitempty" example:"546" description:"Balance of the holder"`
	InscriptionID  string `json:"inscriptionId,omitempty" example:"bac89275b4c0a0ba6aaa603d749a1c88ae3033da9f6d6e661a28fb40e8dca362i0"`
	InscriptionNum int64  `json:"inscriptionNumber,omitempty" example:"67269474" description:"Inscription number of the holder"`
}
