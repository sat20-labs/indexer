package extension

import (
	baseDefine "github.com/sat20-labs/indexer/server/define"
)

type RareSatListData struct {
	baseDefine.ListResp
	List []*baseDefine.ExoticSatRangeUtxo `json:"list"`
}

type RareSatListResp struct {
	baseDefine.BaseResp
	Data *RareSatListData `json:"data"`
}