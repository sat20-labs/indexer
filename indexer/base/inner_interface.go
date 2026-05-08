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

func (b *BaseIndexer) GetAddressByUtxo(utxo string) (string, error) {
	if value, ok := b.utxoIndex.Index[utxo]; ok {
		if value.AddressId != common.INVALID_ID {
			return b.GetAddressByID(value.AddressId)
		}
		return common.PkScriptToAddr(value.OutValue.PkScript, b.chaincfgParam)
	}

	utxoId, err := b.GetUtxoInfo(utxo)
	if err != nil {
		return "", err
	}
	addressId, err := b.GetUtxoAddress(utxoId)
	if err != nil {
		return "", err
	}
	return b.GetAddressByID(addressId)
}
