package extension

import (
	serverCommon "github.com/sat20-labs/ordx/server/define"
)

type RangeReq struct {
	Cursor int `form:"cursor" binding:"omitempty"`
	Size   int `form:"size" binding:"omitempty"`
}

type AddressRangeReq struct {
	serverCommon.AddressReq
	RangeReq
}

type AddressListReq struct {
	AddressList string `form:"addresses" binding:"required"`
}

type AddressTickerRangeReq struct {
	serverCommon.AddressTickerReq
	RangeReq
}

type InscriptionIdReq struct {
	InscriptionId string `form:"inscriptionId" binding:"required"`
}
