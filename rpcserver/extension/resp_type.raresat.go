package extension

import (
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
)

type RareSatListData struct {
	rpcwire.ListResp
	List []*rpcwire.ExoticSatRangeUtxo `json:"list"`
}

type RareSatListResp struct {
	rpcwire.BaseResp
	Data *RareSatListData `json:"data"`
}
