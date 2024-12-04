package extension

import (
	"github.com/sat20-labs/indexer/common"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
)

type OrdinalsName struct {
	InscriptionNumber  int64  `json:"inscriptionNumber"`
	Name               string `json:"name"`
	Sat                int64  `json:"sat"`
	Address            string `json:"address"`
	InscriptionId      string `json:"inscriptionId"`
	Utxo               string `json:"utxo"`
	Value              int64  `json:"value"`
	BlockHeight        int64  `json:"height"`
	BlockTimestamp     int64  `json:"timestamp"`
	InscriptionAddress string `json:"inscriptionAddress"`
	Preview            string `json:"preview"`
	KVs                map[string]*common.KeyValueInDB
}

type OrdinalsNameListData struct {
	rpcwire.ListResp
	List []*OrdinalsName `json:"list"`
}

type OrdinalsNameListResp struct {
	rpcwire.BaseResp
	Data *OrdinalsNameListData `json:"data"`
}
