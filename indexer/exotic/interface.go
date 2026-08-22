package exotic

import (
	"strings"

	"github.com/sat20-labs/indexer/common"
)

func (p *ExoticIndexer) HasExoticInUtxo(utxoId uint64) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	info := p.holderInfoForRead(utxoId)
	return info != nil && len(info.Tickers) > 0
}

func (p *ExoticIndexer) GetAssetsWithUtxo(utxoId uint64) map[string]common.AssetOffsets {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	info := p.holderInfoForRead(utxoId)
	if info == nil {
		return nil
	}
	result := make(map[string]common.AssetOffsets, len(info.Tickers))
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

func (p *ExoticIndexer) getAllTickers() []string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	result := make([]string, 0, len(p.tickerMap))
	for k := range p.tickerMap {
		result = append(result, k)
	}
	return result
}

func (p *ExoticIndexer) GetTicker(tickerName string) *common.Ticker {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	tickerName = strings.ToLower(tickerName)

	return p.getTicker(tickerName)
}

func (p *ExoticIndexer) getTicker(tickerName string) *common.Ticker {

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
	return p.getHolderAndAmountWithTick(tickerName)
}

func (p *ExoticIndexer) getHolderAndAmountWithTick(tickerName string) map[uint64]int64 {
	return p.getTickerHolderAmounts(strings.ToLower(tickerName))
}

// 获取某个地址下的资产 return: ticker->amount
func (p *ExoticIndexer) getAssetAmtByAddress(address uint64, tickerName string) int64 {
	utxos := p.baseIndexer.GetUTXOs(address)
	var result int64
	for utxo := range utxos {
		info := p.holderInfoForRead(utxo)
		if info == nil {
			continue
		}

		assetInfo, ok := info.Tickers[tickerName]
		if !ok {
			continue
		}

		result += assetInfo.AssetAmt()
	}
	return result
}
