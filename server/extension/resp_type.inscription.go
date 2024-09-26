package extension

import (
	"github.com/sat20-labs/indexer/server/define"
)

// /inscription/utxo
type InscriptionUtxoResp struct {
	define.BaseResp
	Data *Utxo `json:"data"`
}

// /inscription/utxo-detail
type InscriptionUtxoDetailResp struct {
	define.BaseResp
	Data *UtxoDetail `json:"data"`
}

// /inscription/utxos
type InscriptionIdListReq struct {
	InscriptionIdList []string `json:"inscriptionIds"`
}

type InscriptionUtxoListResp struct {
	define.BaseResp
	Data []*Utxo `json:"data"`
}

// /inscription/info
type InscriptionInfoResp struct {
	define.BaseResp
	Data *Inscription `json:"data"`
}
