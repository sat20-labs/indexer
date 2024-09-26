package extension

import (
	"github.com/sat20-labs/indexer/server/define"
)

// /buy-btc/channel-list
type BuyBtcChannel struct {
	Channel string `json:"channel"`
}

type BuyBtcChannelListResp struct {
	define.BaseResp
	Data []*BuyBtcChannel `json:"data"`
}

// /buy-btc/create
type BuyBtcCreateReq struct {
	Address string `json:"address"`
	Channel string `json:"channel"`
}

type BuyBtcCreateResp struct {
	define.BaseResp
	Data string `json:"data"`
}
