package base

import (
	"github.com/sat20-labs/ordx/server/define"
)

type HealthStatusResp struct {
	Status    string `json:"status" example:"ok"`
	Version   string `json:"version" example:"0.2.1"`
	BaseDBVer string `json:"basedbver" example:"1.0."`
	OrdxDBVer string `json:"ordxdbver" example:"1.0.0"`
}

type OrdStatusResp struct {
	IndexVersion                  string `json:"indexVersion"`
	DbVersion                     string `json:"dbVersion"`
	SyncInscriptionHeight         uint64 `json:"syncInscriptionHeight"`
	SyncTransferInscriptionHeight uint64 `json:"syncTransferInscriptionHeight"`
	BlessedInscriptions           uint64 `json:"blessedInscriptions"`
	CursedInscriptions            uint64 `json:"cursedInscriptions"`
	AddressCount                  uint64 `json:"addressCount"`
	GenesesAddressCount           uint64 `json:"genesesAddressCount"`
}

type SatRangeResp struct {
	define.BaseResp
	Data *define.ExoticSatRangeUtxo `json:"data"`
}

type SatInfoResp struct {
	define.BaseResp
	Data *define.SatInfo `json:"data"`
}

type SpecificSatReq struct {
	Address string  `json:"address"`
	Sats    []int64 `json:"sats"`
}

type SpecificSatResp struct {
	define.BaseResp
	Data []*define.SpecificSat `json:"data"`
}

type SatributesResp struct {
	define.BaseResp
	Data []string `json:"data"`
}

type SatRangeUtxoResp struct {
	define.BaseResp
	Data []*define.ExoticSatRangeUtxo `json:"data"`
}

type PlainUtxosResp struct {
	define.BaseResp
	Total int                 `json:"total"`
	Data  []*define.PlainUtxo `json:"data"`
}

type AllUtxosResp struct {
	define.BaseResp
	Total      int                 `json:"total"`
	PlainUtxos []*define.PlainUtxo `json:"plainutxos"`
	OtherUtxos []*define.PlainUtxo `json:"otherutxos"`
}

type SpecificExoticUtxoResp struct {
	define.BaseResp
	Data []*define.SpecificExoticUtxo `json:"data"`
}
