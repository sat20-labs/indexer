package exotic

import (
	"strings"

	"github.com/sat20-labs/indexer/common"
)


func (p *ExoticIndexer) HasExoticInUtxo(utxoId uint64) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	info, ok := p.holderInfo[utxoId]
	if !ok {
		var err error
		info, err = p.loadUtxoInfoFromDB(utxoId)
		if err != nil {
			return false
		}
		p.holderInfo[utxoId] = info
	}

	return len(info.Tickers) > 0
}


func (p *ExoticIndexer) GetAssetsWithUtxo(utxoId uint64) map[string]common.AssetOffsets {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	info, ok := p.holderInfo[utxoId]
	if !ok {
		var err error
		info, err = p.loadUtxoInfoFromDB(utxoId)
		if err != nil {
			return nil
		}
		p.holderInfo[utxoId] = info
	}

	result := make(map[string]common.AssetOffsets)
	for name, asset := range info.Tickers {
		result[name] = asset.Offsets.Clone()
	}

	return result
}

func (p *ExoticIndexer) GetExoticsWithType(utxoId uint64, typ string) common.AssetOffsets {
	result := p.GetAssetsWithUtxo(utxoId)
	if result == nil {
		return nil
	}
	return result[typ]
}

func (p *ExoticIndexer) GetTicker(tickerName string) *common.Ticker {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	tickerName = strings.ToLower(tickerName)

	ret, ok := p.tickerMap[tickerName]
	if ok {
		return ret.Ticker
	}
	
	ticker := p.loadTickerFromDB(tickerName)
	if ticker != nil {
		p.tickerMap[tickerName] = &TickInfo{
			Name:   tickerName,
			Ticker: ticker,
		}
	}

	return ticker
}


// 获取该ticker的holder和持有的数量
// return: key, address; value, 资产数量
func (p *ExoticIndexer) GetHolderAndAmountWithTick(tickerName string) map[uint64]int64 {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	tickerName = strings.ToLower(tickerName)
	mp := make(map[uint64]int64, 0)

	utxos, ok := p.utxoMap[tickerName]
	if !ok {
		utxos = p.loadTickerToUtxoMapFromDB(tickerName)
		p.utxoMap[tickerName] = utxos
	}

	for utxo, amount := range utxos {
		info, ok := p.holderInfo[utxo]
		if !ok {
			var err error
			info, err = p.loadUtxoInfoFromDB(utxo)
			if err != nil {
				continue
			}
			p.holderInfo[utxo] = info
		}
		mp[info.AddressId] += amount
		
		// addressId, err := p.baseIndexer.GetUtxoAddress(utxo)
		// if err != nil {
		// 	continue
		// }
		// mp[addressId] += amount
	}

	return mp
}