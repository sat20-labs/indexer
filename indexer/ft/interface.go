package ft

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sat20-labs/indexer/common"
)

func (p *FTIndexer) TickExisted(ticker string) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.tickerMap[strings.ToLower(ticker)] != nil
}

func (p *FTIndexer) GetAllTickers() []string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	type pair struct {
		id int64
		name string
	}
	mid := make([]*pair, 0)
	for name, v := range p.tickerMap {
		mid = append(mid, &pair{
			id: v.Ticker.Id,
			name: name,
		})
	}
	sort.Slice(mid, func(i, j int) bool {
		return mid[i].id < mid[j].id
	})
	
	ret := make([]string, 0)
	for _, item := range mid {
		ret = append(ret, item.name)
	}

	return ret
}


func (p *FTIndexer) GetTickerMap() (map[string]*common.Ticker, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	ret := make(map[string]*common.Ticker)

	for name, tickinfo := range p.tickerMap {
		if tickinfo.Ticker != nil {
			ret[name] = tickinfo.Ticker
			continue
		}

		tickinfo.Ticker = p.getTickerFromDB(tickinfo.Name)
		ret[tickinfo.Name] = tickinfo.Ticker
	}

	return ret, nil
}

func (p *FTIndexer) GetTicker(tickerName string) *common.Ticker {
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

func (p *FTIndexer) GetMint(inscriptionId string) *common.Mint {

	tickerName, err := p.GetTickerWithInscriptionId(inscriptionId)
	if err != nil {
		common.Log.Errorf(err.Error())
		return nil
	}

	p.mutex.RLock()
	defer p.mutex.RUnlock()

	ticker := p.tickerMap[tickerName]
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
func (p *FTIndexer) GetHoldersWithTick(tickerName string) map[uint64][]uint64 {
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
func (p *FTIndexer) GetHolderAndAmountWithTick(tickerName string) map[uint64]int64 {
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
func (p *FTIndexer) GetAssetUtxosWithTicker(address uint64, ticker string) map[uint64]int64 {
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
func (p *FTIndexer) GetAssetSummaryByAddress(utxos map[uint64]int64) map[string]int64 {
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
				amount += common.GetOrdinalsSize(rng) * int64(v.N)
			}
			result[k] += amount
		}
	}

	return result
}

// 获取某个地址下有资产的utxos。key是ticker，value是utxos
func (p *FTIndexer) GetAssetUtxos(utxos map[uint64]int64) map[string][]uint64 {
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
func (p *FTIndexer) GetTickersWithUtxo(utxo uint64) []string {
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
func (p *FTIndexer) GetAssetsWithUtxo(utxo uint64) map[string]map[string][]*common.Range {
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


// 获取utxo的资产详细信息
// return: ticker -> assets(inscriptionId->Ranges)
func (p *FTIndexer) GetAssetsWithUtxoV2(utxo uint64) map[string]int64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	result := make(map[string]int64, 0)

	holders := p.holderInfo[utxo]
	if holders != nil {
		for ticker, info := range holders.Tickers {
			// deep copy
			value := int64(0)
			for _, mintRngs := range info.MintInfo {
				value += common.GetOrdinalsSize(mintRngs) * int64(info.N)	
			}
			result[ticker] = value
		}
		return result
	}

	return nil
}

// return: ticker -> assets(inscriptionId->Ranges)
func (p *FTIndexer) GetAssetsWithRanges(rngs []*common.Range) map[string]map[string][]*common.Range {
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
func (p *FTIndexer) HasAssetInUtxo(utxo uint64) bool {
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
func (p *FTIndexer) GetAssetRangesWithUtxo(utxo uint64, tickerName string) map[string][]*common.Range {
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


// 获取该utxo中哪些range有指定的tick资产
// return: inscriptionId -> assets range
func (p *FTIndexer) GetAssetsWithUtxoV3(utxo uint64, tickerName string) int64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	tickerName = strings.ToLower(tickerName)

	holder := p.holderInfo[utxo]
	if holder == nil {
		return 0
	}

	tickinfo := holder.Tickers[tickerName]
	if tickinfo == nil {
		return 0
	}

	result := int64(0)
	for _, ranges := range tickinfo.MintInfo {
		result += common.GetOrdinalsSize(ranges) * int64(tickinfo.N)
	}

	return result
}

// 检查该Range是否有ticker存在
func (p *FTIndexer) CheckTickersWithSatRange(ticker string, rng *common.Range) bool {
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
func (p *FTIndexer) GetTickerWithInscriptionId(inscriptionId string) (string, error) {
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
func (p *FTIndexer) GetMintHistory(tick string, start, limit int) []*common.MintAbbrInfo {
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
func (p *FTIndexer) GetMintHistoryWithAddress(addressId uint64, tick string, start, limit int) ([]*common.MintAbbrInfo, int) {
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


// return: 按铸造时间排序的铸造历史
func (p *FTIndexer) GetMintHistoryWithAddressV2(addressId uint64, tick string, start, limit int) ([]*common.MintInfo, int) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	tickinfo, ok := p.tickerMap[strings.ToLower(tick)]
	if !ok {
		return nil, 0
	}

	result := make([]*common.MintInfo, 0)
	for _, info := range tickinfo.InscriptionMap {
		if info.Address == addressId {
			result = append(result, info.ToMintInfo())
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


// return: mint的总量
func (p *FTIndexer) GetMintAmountWithAddressId(addressId uint64, tick string) (int64) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	tickinfo, ok := p.tickerMap[strings.ToLower(tick)]
	if !ok {
		return 0
	}

	amount := int64(0)
	for _, info := range tickinfo.InscriptionMap {
		if info.Address == addressId {
			amount += info.Amount.Int64()
		}
	}

	return amount
}

// return: mint的总量和次数
func (p *FTIndexer) GetMintAmount(tick string) (int64, int64) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	tickinfo, ok := p.tickerMap[strings.ToLower(tick)]
	if !ok {
		return 0, 0
	}

	amount := int64(0)
	for _, info := range tickinfo.InscriptionMap {
		amount += info.Amount.Int64()
	}

	return amount, int64(len(tickinfo.InscriptionMap))
}

func (p *FTIndexer) GetSplittedInscriptionsWithTick(tickerName string) []string {
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
