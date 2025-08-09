package base

import (
	"github.com/sat20-labs/indexer/common"
)

/*
 提供一些内部接口，在跑数据时供内部模块快速访问。
 只能在跑数据的线程中调用。
*/

func (p *BaseIndexer) GetAddressId(address string) uint64 {
	id, _ := p.getAddressId(address)
	return id
}

func (b *BaseIndexer) IsMainnet() bool {
	return b.chaincfgParam.Name == "mainnet"
}

func (b *BaseIndexer) GetAddressWithUtxo(utxo string) string {
	output, ok := b.utxoIndex.Index[utxo]
	if ok {
		return output.Address.Addresses[0]
	}

	err := b.loadUtxoFromDB(utxo)
	if err != nil {
		common.Log.Errorf("failed to get value of utxo: %s, %v", utxo, err)
		return ""
	}

	output, ok = b.utxoIndex.Index[utxo]
	if ok {
		return output.Address.Addresses[0]
	}
	
	return ""
}
