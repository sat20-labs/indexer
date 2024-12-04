package extension

import (
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
)

// /buy-btc/channel-list
type BuyBtcChannel struct {
	Channel string `json:"channel"`
}

type BuyBtcChannelListResp struct {
	rpcwire.BaseResp
	Data []*BuyBtcChannel `json:"data"`
}

// /buy-btc/create
type BuyBtcCreateReq struct {
	Address string `json:"address"`
	Channel string `json:"channel"`
}

type BuyBtcCreateResp struct {
	rpcwire.BaseResp
	Data string `json:"data"`
}
