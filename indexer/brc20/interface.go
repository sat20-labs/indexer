package brc20

import (
	"sort"
	"strings"

	"github.com/sat20-labs/indexer/common"
)

func (p *BRC20Indexer) TickExisted(ticker string) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.tickerMap[strings.ToLower(ticker)] != nil
}


func (p *BRC20Indexer) GetAllTickers() ([]string) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	ret := make([]string, 0)

	for name, _ := range p.tickerMap {
		ret = append(ret, name)
	}

	return ret
}

func (p *BRC20Indexer) GetTickerMap() (map[string]*common.BRC20Ticker, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	ret := make(map[string]*common.BRC20Ticker)

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

func (p *BRC20Indexer) GetTicker(tickerName string) *common.BRC20Ticker {
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

// 获取该ticker的holder和持有的utxo
// return: key, address; value, amt
func (p *BRC20Indexer) GetHoldersWithTick(tickerName string) map[uint64]*common.Decimal {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	tickerName = strings.ToLower(tickerName)
	mp := make(map[uint64]*common.Decimal, 0)

	holders, ok := p.tickerToHolderMap[tickerName]
	if !ok {
		return nil
	}

	for addrId := range holders {
		holderinfo, ok := p.holderMap[addrId]
		if !ok {
			common.Log.Errorf("can't find holder with utxo %d", addrId)
			continue
		}
		info, ok := holderinfo.Tickers[tickerName]
		if ok {
			mp[holderinfo.AddressId] = &info.AvailableBalance
		}
	}

	return mp
}

// 获取某个地址下的资产 return: ticker->amount
func (p *BRC20Indexer) GetAssetSummaryByAddress(addrId uint64) map[string]common.Decimal {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := make(map[string]common.Decimal, 0)

	info, ok := p.holderMap[addrId]
	if !ok {
		//common.Log.Errorf("can't find holder with utxo %d", utxo)
		return nil
	}

	for k, v := range info.Tickers {
		org := result[k]
		result[k] = *org.Add(&v.AvailableBalance)
	}

	return result
}

// return: 按铸造时间排序的铸造历史
func (p *BRC20Indexer) GetMintHistory(tick string, start, limit int) []*common.BRC20MintAbbrInfo {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	tickinfo, ok := p.tickerMap[strings.ToLower(tick)]
	if !ok {
		return nil
	}

	result := make([]*common.BRC20MintAbbrInfo, 0)
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
func (p *BRC20Indexer) GetMintHistoryWithAddress(addressId uint64, tick string, start, limit int) ([]*common.MintInfo, int) {
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

// return: mint的总量和次数
func (p *BRC20Indexer) GetMintAmount(tick string) (*common.Decimal, int64) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var amount common.Decimal
	tickinfo, ok := p.tickerMap[strings.ToLower(tick)]
	if !ok {
		return &amount, 0
	}

	for _, info := range tickinfo.InscriptionMap {
		amount = *amount.Add(&info.Amount)
	}

	return &amount, int64(len(tickinfo.InscriptionMap))
}

func (p *BRC20Indexer) GetMint(tickerName, inscriptionId string) *common.BRC20Mint {

	p.mutex.RLock()
	defer p.mutex.RUnlock()

	ticker := p.tickerMap[strings.ToLower(tickerName)]
	if ticker == nil {
		return nil
	}

	for _, mint := range ticker.MintAdded {
		if mint.Nft.Base.InscriptionId == inscriptionId {
			return mint
		}
	}

	return p.getMintFromDB(tickerName, inscriptionId)
}

// return: 按铸造时间排序的铸造历史
func (p *BRC20Indexer) GetTransferHistory(tick string, start, limit int) []*common.BRC20TransferHistory {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := p.loadTransferHistoryFromDB(tick)

	sort.Slice(result, func(i, j int) bool {
		return result[i].NftId < result[j].NftId
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
