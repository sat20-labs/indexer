package extension

import (
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
)

// /ordinals/inscriptions
type OrdinalsInscriptionList struct {
	rpcwire.ListResp
	List []*Inscription `json:"list"`
}

type OrdinalsInscriptionListResp struct {
	rpcwire.BaseResp
	Data *OrdinalsInscriptionList `json:"data"`
}
