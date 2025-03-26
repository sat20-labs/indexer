package utils

import (
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/share/base_indexer"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

func IsExistingInMemPool(utxo string) bool {
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

	if IsExistingInMemPool(utxo) {
		return false
	}

	return true
}
