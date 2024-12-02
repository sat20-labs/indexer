package brc20

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sat20-labs/indexer/common"
)

func (p *BRC20Indexer) TickExisted(ticker string) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.tickerMap[strings.ToLower(ticker)] != nil
}

func (p *BRC20Indexer) GetTickerMap() (map[string]*common.Brc20Ticker, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	ret := make(map[string]*common.Brc20Ticker)

	for name, tickinfo := range p.tickerMap {
		if tickinfo.Ticker != nil {
			ret[name] = tickinfo.Ticker
			continue
		}

		tickinfo.Ticker = p.getTickerFromDB(tickinfo.Name)
		ret[strings.ToLower(tickinfo.Name)] = tickinfo.Ticker
	}

	return ret, nil
}

func (p *BRC20Indexer) GetTicker(tickerName string) *common.Brc20Ticker {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	ret := p.tickerMap[strings.ToLower(tickerName)]
	if ret == nil {
		return nil
	}
	if ret.Ticker != nil {
		return ret.Ticker
	}

	ret.Ticker = p.getTickerFromDB(ret.Name)
	return ret.Ticker
}

func (p *BRC20Indexer) GetMint(inscriptionId string) *common.Mint {

	tickerName, err := p.GetTickerWithInscriptionId(inscriptionId)
	if err != nil {
		common.Log.Errorf(err.Error())
		return nil
	}

	p.mutex.RLock()
	defer p.mutex.RUnlock()

	ticker := p.tickerMap[strings.ToLower(tickerName)]
	if ticker == nil {
		return nil
	}

	for _, mint := range ticker.MintAdded {
		if mint.Base.InscriptionId == inscriptionId {
			return mint
		}
	}

	return p.getMintFromDB(tickerName, inscriptionId)
}

// 获取该ticker的holder和持有的utxo
// return: key, address; value, utxos
func (p *BRC20Indexer) GetHoldersWithTick(tickerName string) map[uint64][]uint64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	tickerName = strings.ToLower(tickerName)
	mp := make(map[uint64][]uint64, 0)

	utxos, ok := p.utxoMap[tickerName]
	if !ok {
		return nil
	}

	for utxo := range *utxos {
		info, ok := p.holderInfo[utxo]
		if !ok {
			common.Log.Errorf("can't find holder with utxo %d", utxo)
			continue
		}
		mp[info.AddressId] = append(mp[info.AddressId], utxo)
	}

	return mp
}

// 获取该ticker的holder和持有的数量
// return: key, address; value, 资产数量
func (p *BRC20Indexer) GetHolderAndAmountWithTick(tickerName string) map[uint64]int64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	tickerName = strings.ToLower(tickerName)
	mp := make(map[uint64]int64, 0)

	utxos, ok := p.utxoMap[tickerName]
	if !ok {
		return nil
	}

	for utxo, amount := range *utxos {
		info, ok := p.holderInfo[utxo]
		if !ok {
			common.Log.Errorf("can't find holder with utxo %d", utxo)
			continue
		}
		mp[info.AddressId] += amount
	}

	return mp
}

// 获取某个地址下有某个资产的utxos
func (p *BRC20Indexer) GetAssetUtxosWithTicker(address uint64, ticker string) map[uint64]int64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	ticker = strings.ToLower(ticker)
	result := make(map[uint64]int64, 0)

	utxos, ok := p.utxoMap[ticker]
	if !ok {
		return nil
	}

	for utxo, amout := range *utxos {
		info, ok := p.holderInfo[utxo]
		if !ok {
			common.Log.Errorf("can't find holder with utxo %d", utxo)
			continue
		}
		if info.AddressId == address {
			result[utxo] = amout
		}
	}

	return result
}

// 获取某个地址下的资产 return: ticker->amount
func (p *BRC20Indexer) GetAssetSummaryByAddress(utxos map[uint64]int64) map[string]int64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := make(map[string]int64, 0)

	for utxo := range utxos {
		info, ok := p.holderInfo[utxo]
		if !ok {
			//common.Log.Errorf("can't find holder with utxo %d", utxo)
			continue
		}

		for k, v := range info.Tickers {
			amount := int64(0)
			for _, rng := range v.MintInfo {
				amount += common.GetOrdinalsSize(rng)
			}
			result[k] += amount
		}
	}

	return result
}

// 获取某个地址下有资产的utxos。key是ticker，value是utxos
func (p *BRC20Indexer) GetAssetUtxos(utxos map[uint64]int64) map[string][]uint64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	result := make(map[string][]uint64, 0)

	for utxo := range utxos {
		info, ok := p.holderInfo[utxo]
		if !ok {
			continue
		}
		for name := range info.Tickers {
			result[name] = append(result[name], utxo)
		}
	}

	return result
}

// 检查utxo里面包含哪些资产
// return: ticker list
func (p *BRC20Indexer) GetTickersWithUtxo(utxo uint64) []string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	result := make([]string, 0)

	holders := p.holderInfo[utxo]
	if holders != nil {
		for name := range holders.Tickers {
			result = append(result, name)
		}
	}

	return result
}

// 获取utxo的资产详细信息
// return: ticker -> assets(inscriptionId->Ranges)
func (p *BRC20Indexer) GetAssetsWithUtxo(utxo uint64) map[string]map[string][]*common.Range {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	result := make(map[string]map[string][]*common.Range, 0)

	holders := p.holderInfo[utxo]
	if holders != nil {
		for ticker, info := range holders.Tickers {
			// deep copy
			mintInfo := make(map[string][]*common.Range, 0)
			for mintUtxo, mintRngs := range info.MintInfo {
				rngs := make([]*common.Range, 0)
				for _, rng := range mintRngs {
					rngs = append(rngs, &common.Range{Start: rng.Start, Size: rng.Size})
				}
				mintInfo[mintUtxo] = rngs
			}
			result[ticker] = mintInfo
		}
		return result
	}

	return nil
}

// return: ticker -> assets(inscriptionId->Ranges)
func (p *BRC20Indexer) GetAssetsWithRanges(rngs []*common.Range) map[string]map[string][]*common.Range {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := make(map[string]map[string][]*common.Range)
	for tickerName, v := range p.tickerMap {
		for _, rng := range rngs {
			intersections := v.MintInfo.FindIntersections(rng)
			for _, it := range intersections {
				mintinfo := it.Value.(*common.RBTreeValue_Mint)
				tickinfo, ok := result[tickerName]
				if ok {
					tickinfo[mintinfo.InscriptionIds[0]] = append(tickinfo[mintinfo.InscriptionIds[0]], it.Rng)
				} else {
					tickinfo = make(map[string][]*common.Range)
					tickinfo[mintinfo.InscriptionIds[0]] = []*common.Range{it.Rng}
					result[tickerName] = tickinfo
				}
			}
		}
	}

	return result
}

// 检查utxo是否有资产
func (p *BRC20Indexer) HasAssetInUtxo(utxo uint64) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	holder, ok := p.holderInfo[utxo]
	if !ok {
		return false
	}

	return len(holder.Tickers) > 0
}

// 获取该utxo中哪些range有指定的tick资产
// return: inscriptionId -> assets range
func (p *BRC20Indexer) GetAssetRangesWithUtxo(utxo uint64, tickerName string) map[string][]*common.Range {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	tickerName = strings.ToLower(tickerName)

	holder := p.holderInfo[utxo]
	if holder == nil {
		return nil
	}

	tickinfo := holder.Tickers[tickerName]
	if tickinfo == nil {
		return nil
	}

	result := make(map[string][]*common.Range, 0)
	for id, ranges := range tickinfo.MintInfo {
		result[id] = ranges
	}

	return result
}

// 检查该Range是否有ticker存在
func (p *BRC20Indexer) CheckTickersWithSatRange(ticker string, rng *common.Range) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	tickinfo := p.tickerMap[strings.ToLower(ticker)]
	if tickinfo == nil {
		return false
	}

	value := tickinfo.MintInfo.FindIntersections(rng)
	return len(value) > 0
}

// return: mint的ticker名字
func (p *BRC20Indexer) GetTickerWithInscriptionId(inscriptionId string) (string, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	for _, tickinfo := range p.tickerMap {
		for k := range tickinfo.InscriptionMap {
			if k == inscriptionId {
				return tickinfo.Name, nil
			}
		}
	}

	return "", fmt.Errorf("can't find inscription id %s", inscriptionId)
}

// return: 按铸造时间排序的铸造历史
func (p *BRC20Indexer) GetMintHistory(tick string, start, limit int) []*common.MintAbbrInfo {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	tickinfo, ok := p.tickerMap[strings.ToLower(tick)]
	if !ok {
		return nil
	}

	result := make([]*common.MintAbbrInfo, 0)
	for _, info := range tickinfo.InscriptionMap {
		result = append(result, info)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].InscriptionNum < result[j].InscriptionNum
	})

	end := len(result)
	if start >= end {
		return nil
	}
	if start+limit < end {
		end = start + limit
	}

	return result[start:end]
}

// return: 按铸造时间排序的铸造历史
func (p *BRC20Indexer) GetMintHistoryWithAddress(addressId uint64, tick string, start, limit int) ([]*common.MintAbbrInfo, int) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	tickinfo, ok := p.tickerMap[strings.ToLower(tick)]
	if !ok {
		return nil, 0
	}

	result := make([]*common.MintAbbrInfo, 0)
	for _, info := range tickinfo.InscriptionMap {
		if info.Address == addressId {
			result = append(result, info)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].InscriptionNum < result[j].InscriptionNum
	})

	total := len(result)
	end := total
	if start >= end {
		return nil, 0
	}
	if start+limit < end {
		end = start + limit
	}

	return result[start:end], total
}

// return: mint的总量和次数
func (p *BRC20Indexer) GetMintAmount(tick string) (int64, int64) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	tickinfo, ok := p.tickerMap[strings.ToLower(tick)]
	if !ok {
		return 0, 0
	}

	amount := int64(0)
	for _, info := range tickinfo.InscriptionMap {
		amount += info.Amount
	}

	return amount, int64(len(tickinfo.InscriptionMap))
}

func (p *BRC20Indexer) GetSplittedInscriptionsWithTick(tickerName string) []string {
	tickerName = strings.ToLower(tickerName)

	mintMap := p.getMintListFromDB(tickerName)
	result := make([]string, 0)

	p.mutex.RLock()
	defer p.mutex.RUnlock()

	inscMap := make(map[string]string, 0)

	utxos, ok := p.utxoMap[tickerName]
	if !ok {
		return nil
	}

	for utxo := range *utxos {
		holder, ok := p.holderInfo[utxo]
		if !ok {
			common.Log.Errorf("can't find holder with utxo %d", utxo)
			continue
		}

		for name, tickinfo := range holder.Tickers {
			if strings.EqualFold(name, tickerName) {
				for mintutxo, newRngs := range tickinfo.MintInfo {
					mintinfo := mintMap[mintutxo]
					oldRngs := mintinfo.Ordinals

					if len(oldRngs) != len(newRngs) {
						inscMap[mintutxo] = mintinfo.Base.InscriptionId
						//break 不能跳出，有更多的在后面
					} else {
						// newRng的顺序可能是错乱的
						if !common.RangesContained(oldRngs, newRngs) ||
							!common.RangesContained(newRngs, oldRngs) {
							inscMap[mintutxo] = mintinfo.Base.InscriptionId
							//break
						}
					}
				}
			}
		}
	}

	for _, id := range inscMap {
		common.Log.Warnf("Splited inscription ID %s", id)
		result = append(result, id)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})

	return result
}
