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
	
	return p.getAllTickers()
}

func (p *FTIndexer) GetTickersWithRange(start, limit int) ([]string, int) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	

	tickers := p.getAllTickers()
	total := len(tickers)
	if start < 0 {
		start = 0
	}
	if start >= total {
		return nil, 0
	}
	if limit < 0 {
		limit = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	return tickers[start:end], total
}

func (p *FTIndexer) getAllTickers() []string {
	
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

	return p.getTicker(tickerName)
}

func (p *FTIndexer) getTicker(tickerName string) *common.Ticker {

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
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	tickerName, err := p.getTickerWithInscriptionId(inscriptionId)
	if err != nil {
		common.Log.Errorf(err.Error())
		return nil
	}

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
	return p.getHoldersWithTick(tickerName)
}

func (p *FTIndexer) getHoldersWithTick(tickerName string) map[uint64][]uint64 {
	mp := make(map[uint64][]uint64, 0)

	utxos, ok := p.utxoMap[tickerName]
	if !ok {
		return nil
	}

	for utxo := range utxos {
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
	return p.getHolderAndAmountWithTick(tickerName)
}

func (p *FTIndexer) getHolderAndAmountWithTick(tickerName string) map[uint64]int64 {
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

	for utxo, amout := range utxos {
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
			result[k] += v.AssetAmt()
		}
	}

	return result
}


// 获取某个地址下的资产 return: ticker->amount
func (p *FTIndexer) getAssetAmtByAddress(address uint64, tickerName string) int64 {
	utxos := p.nftIndexer.GetBaseIndexer().GetUTXOs(address)
	var result int64
	for utxo := range utxos {
		info, ok := p.holderInfo[utxo]
		if !ok {
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
func (p *FTIndexer) GetAssetsWithUtxo(utxo uint64) map[string]common.AssetOffsets {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	result := make(map[string]common.AssetOffsets, 0)

	holders := p.holderInfo[utxo]
	if holders != nil {
		for ticker, assetInfo := range holders.Tickers {
			// deep copy
			result[ticker] = assetInfo.Offsets.Clone()
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
		for ticker, assetInfo := range holders.Tickers {
			// deep copy
			result[ticker] = assetInfo.AssetAmt()
		}
		return result
	}

	return nil
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

// 获取该utxo中有指定的tick资产的数量
// return: inscriptionId -> assets amount
func (p *FTIndexer) GetAssetsWithUtxoV3(utxo uint64, tickerName string) int64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	tickerName = strings.ToLower(tickerName)

	holder := p.holderInfo[utxo]
	if holder == nil {
		return 0
	}

	tickAssetInfo := holder.Tickers[tickerName]
	if tickAssetInfo == nil {
		return 0
	}

	return tickAssetInfo.AssetAmt()
}

// return: mint的ticker名字
func (p *FTIndexer) getTickerWithInscriptionId(inscriptionId string) (string, error) {

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
func (p *FTIndexer) GetMintAmountWithAddressId(addressId uint64, tick string) int64 {
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

	return p.getMintAmount(tick)
}

// return: mint的总量和次数
func (p *FTIndexer) getMintAmount(tick string) (int64, int64) {

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
	return nil
	// tickerName = strings.ToLower(tickerName)

	// mintMap := p.getMintListFromDB(tickerName)
	// result := make([]string, 0)

	// p.mutex.RLock()
	// defer p.mutex.RUnlock()

	// inscMap := make(map[string]string, 0)

	// utxos, ok := p.utxoMap[tickerName]
	// if !ok {
	// 	return nil
	// }

	// for utxo := range utxos {
	// 	holder, ok := p.holderInfo[utxo]
	// 	if !ok {
	// 		common.Log.Errorf("can't find holder with utxo %d", utxo)
	// 		continue
	// 	}

	// 	for name, tickinfo := range holder.Tickers {
	// 		if strings.EqualFold(name, tickerName) {
	// 			for mintutxo, newRngs := range tickinfo.Offsets {
	// 				mintinfo := mintMap[mintutxo]
	// 				oldRngs := mintinfo.Offsets

	// 				if len(oldRngs) != len(newRngs) {
	// 					inscMap[mintutxo] = mintinfo.Base.InscriptionId
	// 					//break 不能跳出，有更多的在后面
	// 				} else {
	// 					// newRng的顺序可能是错乱的
	// 					if !common.RangesContained(oldRngs, newRngs) ||
	// 						!common.RangesContained(newRngs, oldRngs) {
	// 						inscMap[mintutxo] = mintinfo.Base.InscriptionId
	// 						//break
	// 					}
	// 				}
	// 			}
	// 		}
	// 	}
	// }

	// for _, id := range inscMap {
	// 	common.Log.Warnf("Splited inscription ID %s", id)
	// 	result = append(result, id)
	// }

	// sort.Slice(result, func(i, j int) bool {
	// 	return result[i] < result[j]
	// })

	// return result
}
