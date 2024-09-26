package extension

import (
	"github.com/sat20-labs/indexer/server/define"
)

// /ordinals/inscriptions
type OrdinalsInscriptionList struct {
	define.ListResp
	List []*Inscription `json:"list"`
}

type OrdinalsInscriptionListResp struct {
	define.BaseResp
	Data *OrdinalsInscriptionList `json:"data"`
}
