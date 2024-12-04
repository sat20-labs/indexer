package extension

import (
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
)

// /inscription/utxo
type InscriptionUtxoResp struct {
	rpcwire.BaseResp
	Data *Utxo `json:"data"`
}

// /inscription/utxo-detail
type InscriptionUtxoDetailResp struct {
	rpcwire.BaseResp
	Data *UtxoDetail `json:"data"`
}

// /inscription/utxos
type InscriptionIdListReq struct {
	InscriptionIdList []string `json:"inscriptionIds"`
}

type InscriptionUtxoListResp struct {
	rpcwire.BaseResp
	Data []*Utxo `json:"data"`
}

// /inscription/info
type InscriptionInfoResp struct {
	rpcwire.BaseResp
	Data *Inscription `json:"data"`
}
