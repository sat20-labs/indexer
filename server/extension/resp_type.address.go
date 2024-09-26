package extension

import (
	serverCommon "github.com/sat20-labs/indexer/server/define"
)

// /address/summary
type TokenSummary struct {
	Name    string `json:"name"`
	Balance uint64 `json:"balance"`
}

type OrdinalsSummary struct {
	Count uint64 `json:"count"`
}

type NameSummary struct {
	Name    string `json:"name"`
	Count   uint64 `json:"count"`
	Balance uint64 `json:"balance"`
}

type ExoticSummary struct {
	Name    string `json:"name"`
	Balance uint64 `json:"balance"`
}

type AssetSummary struct {
	TotalSatoshis     uint64          `json:"totalSatoshis"`
	BtcSatoshis       uint64          `json:"btcSatoshis"`
	AssetSatoshis     uint64          `json:"assetSatoshis"`
	InscriptionCount  uint64          `json:"inscriptionCount"`
	RunesCount        uint64          `json:"runesCount"`
	TokenSummaryList  []TokenSummary  `json:"token"`
	OrdinalsSummary   OrdinalsSummary `json:"ordinals"`
	NameSummaryList   []NameSummary   `json:"name"`
	ExoticSummaryList []ExoticSummary `json:"exotic"`
}

type AssetsSummaryResp struct {
	serverCommon.BaseResp
	Data *AssetSummary `json:"data"`
}

// /address/balance
type Balance struct {
	ConfirmAmount            string `json:"confirm_amount"`
	PendingAmount            string `json:"pending_amount"`
	Amount                   string `json:"amount"`
	ConfirmBtcAmount         string `json:"confirm_btc_amount"`
	PendingBtcAmount         string `json:"pending_btc_amount"`
	BtcAmount                string `json:"btc_amount"`
	ConfirmInscriptionAmount string `json:"confirm_inscription_amount"`
	PendingInscriptionAmount string `json:"pending_inscription_amount"`
	InscriptionAmount        string `json:"inscription_amount"`
	UsdValue                 string `json:"usd_value"`
}

type BalanceResp struct {
	serverCommon.BaseResp
	Data *Balance `json:"data"`
}

// /address/multi-assets
// type AddressMultiAssetsData struct {
// 	TotalSatoshis    uint64 `json:"totalSatoshis"`
// 	BtcSatoshis      uint64 `json:"btcSatoshis"`
// 	AssetSatoshis    uint64 `json:"assetSatoshis"`
// 	InscriptionCount uint64 `json:"inscriptionCount"`
// 	// AtomicalsCount   uint64 `json:"atomicalsCount"`
// 	// Brc20Count       uint64 `json:"brc20Count"`
// 	// Brc20Count5Byte  uint64 `json:"brc20Count5Byte"`
// 	// Arc20Count       uint64 `json:"arc20Count"`
// 	RunesCount uint64 `json:"runesCount"`
// }

type MultiAddressAssetsResp struct {
	serverCommon.BaseResp
	Data []*AssetSummary `json:"data"`
}

// /address/find-group-assets
type AddressFindGroupAssetsGroup struct {
	Type       int      `json:"type"`
	AddressArr []string `json:"address_arr"`
	PubkeyArr  []string `json:"pubkey_arr"`
}

type AddressFindGroupAssetsReq struct {
	Groups []AddressFindGroupAssetsGroup `json:"groups"`
}

type AddressFindGroupAssetsData struct {
	AddressFindGroupAssetsGroup
	SatoshisArr []int `json:"satoshis_arr"`
}

type AddressFindGroupAssetsResp struct {
	serverCommon.BaseResp
	Data []*AddressFindGroupAssetsData `json:"data"`
}

// /address/unavailable-utxo
type AddressUtxoResp struct {
	serverCommon.BaseResp
	Data []*Utxo `json:"data"`
}

// /address/inscriptions
type AddressInscriptionData struct {
	serverCommon.ListResp
	InscriptionList []*Inscription `json:"list"`
}

type AddressInscriptionResp struct {
	serverCommon.BaseResp
	Data *AddressInscriptionData `json:"data"`
}

// /address/search
type AddressSearchResp struct {
	serverCommon.BaseResp
	Data []*Inscription `json:"data"`
}
