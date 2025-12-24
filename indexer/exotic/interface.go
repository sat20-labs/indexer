package exotic

import (
	"strings"

	"github.com/sat20-labs/indexer/common"
)


func (p *ExoticIndexer) HasExoticInUtxo(utxoId uint64) bool {
	return false
}


func (p *ExoticIndexer) GetAssetsWithUtxo(utxo uint64) map[string]common.AssetOffsets {
	return nil
}


func (p *ExoticIndexer) GetExoticsWithType(utxoId uint64, typ string) common.AssetOffsets {
	return nil
}


func (p *ExoticIndexer) GetTicker(tickerName string) *common.Ticker {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	ret := p.tickerMap[strings.ToLower(tickerName)]
	if ret == nil {
		return nil
	}
	if ret.Ticker != nil {
		return ret.Ticker
	}

	return nil
}


// 获取该ticker的holder和持有的数量
// return: key, address; value, 资产数量
func (p *ExoticIndexer) GetHolderAndAmountWithTick(tickerName string) map[uint64]int64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	tickerName = strings.ToLower(tickerName)
	mp := make(map[uint64]int64, 0)

	utxos, ok := p.utxoMap[tickerName]
	if !ok {
		return nil
	}

	for utxo, amount := range utxos {
		info, ok := p.holderInfo[utxo]
		if !ok {
			common.Log.Errorf("can't find holder with utxo %d", utxo)
			continue
		}
		mp[info.AddressId] += amount
	}

	return mp
}