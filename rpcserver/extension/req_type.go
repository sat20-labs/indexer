package extension

import (
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
)

type RangeReq struct {
	Cursor int `form:"cursor" binding:"omitempty"`
	Size   int `form:"size" binding:"omitempty"`
}

type AddressRangeReq struct {
	rpcwire.AddressReq
	RangeReq
}

type AddressListReq struct {
	AddressList string `form:"addresses" binding:"required"`
}

type AddressTickerRangeReq struct {
	rpcwire.AddressTickerReq
	RangeReq
}

type InscriptionIdReq struct {
	InscriptionId string `form:"inscriptionId" binding:"required"`
}
