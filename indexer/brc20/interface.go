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

func (p *BRC20Indexer) GetMint(inscriptionId string) *common.BRC20Mint {

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
		if mint.Base.Base.InscriptionId == inscriptionId {
			return mint
		}
	}

	return p.getMintFromDB(tickerName, inscriptionId)
}

// 获取该ticker的holder和持有的utxo
// return: key, address; value, amt
func (p *BRC20Indexer) GetHoldersWithTick(tickerName string) map[uint64]common.Decimal {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	tickerName = strings.ToLower(tickerName)
	mp := make(map[uint64]common.Decimal, 0)

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
			mp[holderinfo.AddressId] = info.AvailableBalance
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
		org.Add(&v.AvailableBalance)
		result[k] = org
	}
	
	return result
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
func (p *BRC20Indexer) GetMintHistoryWithAddress(addressId uint64, tick string, start, limit int) ([]*common.BRC20MintAbbrInfo, int) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	tickinfo, ok := p.tickerMap[strings.ToLower(tick)]
	if !ok {
		return nil, 0
	}

	result := make([]*common.BRC20MintAbbrInfo, 0)
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
func (p *BRC20Indexer) GetMintAmount(tick string) (common.Decimal, int64) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var amount common.Decimal
	tickinfo, ok := p.tickerMap[strings.ToLower(tick)]
	if !ok {
		return amount, 0
	}

	for _, info := range tickinfo.InscriptionMap {
		amount.Add(&info.Amount)
	}

	return amount, int64(len(tickinfo.InscriptionMap))
}
