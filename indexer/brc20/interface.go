package brc20

import (
	"sort"
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
	"github.com/sat20-labs/indexer/share/base_indexer"
)

type Brc20TickerOrder int

const (
	// 0: inscribe-mint  1: inscribe-transfer  2: transfer
	BRC20_TICKER_ORDER_DEPLOYTIME_DESC Brc20TickerOrder = iota
	BRC20_TICKER_ORDER_HOLDER_DESC
	BRC20_TICKER_ORDER_TRANSACTION_DESC
)

func (s *BRC20Indexer) TickExisted(ticker string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.tickerMap[strings.ToLower(ticker)] != nil
}

func (s *BRC20Indexer) GetAllTickers() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	ret := make([]string, 0)

	for name := range s.tickerMap {
		ret = append(ret, name)
	}

	return ret
}

func (s *BRC20Indexer) GetTickers(start, limit uint64, order Brc20TickerOrder) (ret []*BRC20TickInfo, total uint64) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, ticker := range s.tickerMap {
		ret = append(ret, ticker)
	}

	switch order {
	case BRC20_TICKER_ORDER_DEPLOYTIME_DESC:
		sort.Slice(ret, func(i, j int) bool {
			return ret[i].Ticker.DeployTime > ret[j].Ticker.DeployTime
		})
	case BRC20_TICKER_ORDER_HOLDER_DESC:
		sort.Slice(ret, func(i, j int) bool {
			return ret[i].Ticker.HolderCount > ret[j].Ticker.HolderCount
		})
	case BRC20_TICKER_ORDER_TRANSACTION_DESC:
		sort.Slice(ret, func(i, j int) bool {
			return ret[i].Ticker.TransactionCount > ret[j].Ticker.TransactionCount
		})
	}

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Name < ret[j].Name
	})
	total = uint64(len(ret))
	end := total
	if start >= end {
		return nil, 0
	}
	if start+limit < end {
		end = start + limit
	}
	return ret[start:end], total
}

func (s *BRC20Indexer) GetTickerMap() (map[string]*common.BRC20Ticker, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	ret := make(map[string]*common.BRC20Ticker)

	for name, tickinfo := range s.tickerMap {
		if tickinfo.Ticker != nil {
			ret[name] = tickinfo.Ticker
			continue
		}

		tickinfo.Ticker = s.getTickerFromDB(tickinfo.Name)
		ret[strings.ToLower(tickinfo.Name)] = tickinfo.Ticker
	}

	return ret, nil
}

func (s *BRC20Indexer) GetTicker(tickerName string) *common.BRC20Ticker {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	ret := s.tickerMap[strings.ToLower(tickerName)]
	if ret == nil {
		return nil
	}
	if ret.Ticker != nil {
		return ret.Ticker
	}

	ret.Ticker = s.getTickerFromDB(ret.Name)
	return ret.Ticker
}

// 获取该ticker的holder和持有的utxo
// return: key, address; value, amt
func (s *BRC20Indexer) GetHoldersWithTick(tickerName string) map[uint64]*common.Decimal {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	tickerName = strings.ToLower(tickerName)
	mp := make(map[uint64]*common.Decimal, 0)

	holders, ok := s.tickerToHolderMap[tickerName]
	if !ok {
		return nil
	}

	for addrId := range holders {
		holderinfo, ok := s.holderMap[addrId]
		if !ok {
			common.Log.Panicf("can't find holder with utxo %d", addrId)
			continue
		}
		info, ok := holderinfo.Tickers[tickerName]
		if ok {
			balance := info.AvailableBalance.Clone()
			balance = balance.Add(info.TransferableBalance)
			mp[addrId] = balance
		}
	}

	return mp
}

// 获取某个地址下的资产 return: ticker->amount
func (s *BRC20Indexer) GetAssetSummaryByAddress(addrId uint64) map[string]*common.Decimal {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make(map[string]*common.Decimal, 0)

	info, ok := s.holderMap[addrId]
	if !ok {
		//common.Log.Errorf("can't find holder with utxo %d", utxo)
		return nil
	}

	for k, v := range info.Tickers {
		org := result[k]
		balance := v.AvailableBalance.Add(v.TransferableBalance)
		result[k] = org.Add(balance)
	}

	return result
}

// return: 按铸造时间排序的铸造历史
func (s *BRC20Indexer) GetMintHistory(tick string, start, limit int) []*common.BRC20MintAbbrInfo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	tickinfo, ok := s.tickerMap[strings.ToLower(tick)]
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
func (s *BRC20Indexer) GetMintHistoryWithAddress(addressId uint64, tick string, start, limit int) ([]*common.MintAbbrInfo, int) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	tickinfo, ok := s.tickerMap[strings.ToLower(tick)]
	if !ok {
		return nil, 0
	}

	result := make([]*common.MintAbbrInfo, 0)
	for _, info := range tickinfo.InscriptionMap {
		if info.Address == addressId {
			result = append(result, info.ToMintAbbrInfo())
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

func (s *BRC20Indexer) GetMintHistoryWithAddressV2(addressId uint64, tick string, start, limit int) ([]*common.MintInfo, int) {
	m, total := s.GetMintHistoryWithAddress(addressId, tick, start, limit)
	result := make([]*common.MintInfo, len(m))
	for i, v := range m {
		result[i] = v.ToMintInfo()
	}
	return result, total
}

// return: mint的总量和次数
func (s *BRC20Indexer) GetMintAmount(tick string) (*common.Decimal, int64) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var amount *common.Decimal
	tickinfo, ok := s.tickerMap[strings.ToLower(tick)]
	if !ok {
		return amount, 0
	}

	for _, info := range tickinfo.InscriptionMap {
		amount = amount.Add(&info.Amount)
	}

	return amount, int64(len(tickinfo.InscriptionMap))
}

func (s *BRC20Indexer) GetMint(tickerName, inscriptionId string) *common.BRC20Mint {

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	ticker := s.tickerMap[strings.ToLower(tickerName)]
	if ticker == nil {
		return nil
	}

	for _, mint := range ticker.MintAdded {
		if mint.Nft.Base.InscriptionId == inscriptionId {
			return mint
		}
	}

	return s.getMintFromDB(tickerName, inscriptionId)
}

// return: 按铸造时间排序的铸造历史
func (s *BRC20Indexer) GetTransferHistory(tick string, start, limit int) []*common.BRC20TransferHistory {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := s.loadTransferHistoryFromDB(tick)

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

func (s *BRC20Indexer) GetUtxoAssets(utxoId uint64) (ret *common.BRC20TransferInfo) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	utxo := base_indexer.ShareBaseIndexer.GetUtxoById(utxoId)
	common.Log.Info("GetUtxoAssets", " utxoId ", utxoId, " testUtxo ", utxo)
	// transferNft, ok := p.transferNftMap[utxoId]
	// if !ok {
	// 	return nil
	// }
	// return &common.BRC20TransferInfo{
	// 	NftId: transferNft.TransferNft.NftId,
	// 	Name: transferNft.Ticker,
	// 	Amt: transferNft.TransferNft.Amount.Clone(),
	// 	Invalid: transferNft.TransferNft.IsInvalid,
	// }

	nfts := s.nftIndexer.GetNftsWithUtxo(utxoId)
	for _, nft := range nfts {
		if nft.Base.Id == 245587 || nft.Base.Id == 245603 {
			common.Log.Infof("test")
		}
		if s.nftIndexer.GetBaseIndexer().IsMainnet() && s.IsExistCursorInscriptionInDB(nft.Base.InscriptionId) {
			continue
		}
		txid, index, err := common.ParseOrdInscriptionID(nft.Base.InscriptionId)
		if err != nil {
			continue
		}
		if index != 0 {
			continue
		}

		content := common.ParseBBRC20AmtContent(string(nft.Base.Content))
		if content == nil {
			continue
		}
		tickerName := strings.ToLower(content.Ticker)
		switch content.Op {
		case "mint":
			for _, ticker := range s.tickerMap {
				if strings.ToLower(ticker.Name) == tickerName {
					for _, v := range ticker.MintAdded {
						if v.Nft.Base.InscriptionId == nft.Base.InscriptionId {
							ret = &common.BRC20TransferInfo{
								NftId:   nft.Base.Id,
								Name:    content.Ticker,
								Amt:     v.Amt.Clone(),
								Invalid: false,
							}
							return
						}
					}
				}
			}
			mint := s.getMintFromDB(tickerName, nft.Base.InscriptionId)
			if mint != nil {
				ret = &common.BRC20TransferInfo{
					NftId:   nft.Base.Id,
					Name:    content.Ticker,
					Amt:     mint.Amt.Clone(),
					Invalid: false,
				}
				return
			}
		case "transfer":
			inscription := s.nftIndexer.GetNftWithInscriptionId(nft.Base.InscriptionId)
			if inscription == nil {
				common.Log.Warnf("inscription not found: %s", nft.Base.InscriptionId)
				continue
			}
			outputUtxo := base_indexer.ShareBaseIndexer.GetUtxoById(inscription.UtxoId)
			geniousUtxo := txid + ":0"
			if geniousUtxo == outputUtxo {
				holder := s.holderMap[nft.OwnerAddressId]
				if holder == nil {
					continue
				}
				ticker := holder.Tickers[tickerName]
				if ticker == nil {
					continue
				}
				transferNft := ticker.TransferableData[nft.UtxoId]
				if transferNft == nil {
					continue
				}
				ret = &common.BRC20TransferInfo{
					NftId:   nft.Base.Id,
					Name:    content.Ticker,
					Amt:     transferNft.Amount.Clone(),
					Invalid: false, // transferNft.IsInvalid,
				}
				return
			} else {
				for _, action := range s.holderActionList {
					if tickerName == strings.ToLower(action.Ticker) && nft.UtxoId == action.UtxoId {
						ret = &common.BRC20TransferInfo{
							NftId:   nft.Base.Id,
							Name:    action.Ticker,
							Amt:     action.Amount.Clone(),
							Invalid: true,
						}
						// switch action.Action {
						// case Action_InScribe_Transfer:
						// 	ret.Invalid = false
						// case Action_Transfer:
						// 	ret.Invalid = true
						// default:
						// 	common.Log.Warnf("action is err")
						// 	continue
						// }
						return
					}
				}

				key := GetTransferHistoryKey(tickerName, nft.UtxoId)
				var result common.BRC20TransferHistory
				err = db.GetValueFromDB([]byte(key), &result, s.db)
				if err != nil {
					continue
				}
				ticker := s.GetTicker(result.Ticker)
				if ticker == nil {
					continue
				}
				amt, err := common.NewDecimalFromString(result.Amount, int(ticker.Decimal))
				if err != nil {
					continue
				}
				ret = &common.BRC20TransferInfo{
					NftId:   nft.Base.Id,
					Name:    result.Ticker,
					Amt:     amt,
					Invalid: true,
				}
			}
			return
		}
	}

	return
}

func (s *BRC20Indexer) IsExistAsset(utxoId uint64) bool {
	ret := s.GetUtxoAssets(utxoId)
	if ret == nil {
		return false
	}
	return !ret.Invalid
}

// transfer
func (s *BRC20Indexer) GetHolderAbbrInfo(addressId uint64, tickerName string) *common.BRC20TickAbbrInfo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	holder := s.holderMap[addressId]
	if holder == nil {
		return nil
	}

	tickerName = strings.ToLower(tickerName)
	tickAbbrInfo := holder.Tickers[tickerName]
	if tickAbbrInfo == nil {
		return nil
	}
	return tickAbbrInfo
}
