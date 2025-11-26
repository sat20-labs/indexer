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
