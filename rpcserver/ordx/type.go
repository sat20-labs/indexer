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
