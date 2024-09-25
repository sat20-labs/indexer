package define

import (
	"github.com/sat20-labs/ordx/common"
	"github.com/sat20-labs/ordx/share/base_indexer"
	"github.com/sat20-labs/ordx/share/bitcoin_rpc"
	"gopkg.in/yaml.v2"
)

func ParseRpcService(data any) (*RPCService, error) {
	rpcServiceRaw, err := yaml.Marshal(data)
	if err != nil {
		return nil, err
	}
	ret := &RPCService{}
	err = yaml.Unmarshal(rpcServiceRaw, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func IsExistUtxoInMemPool(utxo string) bool {
	isExist, err := bitcoin_rpc.IsExistUtxoInMemPool(utxo)
	if err != nil {
		common.Log.Errorf("GetUnspendTxOutput %s failed. %v", utxo, err)
		return false
	}
	return isExist
}

func IsAvailableUtxoId(utxoId uint64) bool {
	return IsAvailableUtxo(base_indexer.ShareBaseIndexer.GetUtxoById(utxoId))
}

func IsAvailableUtxo(utxo string) bool {
	//Find common utxo (that is, utxo with non-ordinal attributes)
	if base_indexer.ShareBaseIndexer.HasAssetInUtxo(utxo, false) {
		return false
	}

	if IsExistUtxoInMemPool(utxo) {
		return false
	}

	return true
}
