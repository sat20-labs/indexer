package brc20

import (
	"sort"
	"strings"

	"github.com/sat20-labs/indexer/common"
)

type Brc20TickerOrder int

const (
	// 0: inscribe-mint  1: inscribe-transfer  2: transfer
	BRC20_TICKER_ORDER_DEPLOYTIME_DESC Brc20TickerOrder = iota
	BRC20_TICKER_ORDER_HOLDER_DESC
	BRC20_TICKER_ORDER_TRANSACTION_DESC
)

func (s *BRC20Indexer) TickExisted(tickerName string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.loadTickInfo(strings.ToLower(tickerName)) != nil
}

func (s *BRC20Indexer) GetAllTickers() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.getAllTickers()
}

func (s *BRC20Indexer) getAllTickers() []string {
	ret := s.loadTickListFromDB()

	for _, ticker := range s.tickerAdded {
		ret = append(ret, ticker.Name)
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

func (s *BRC20Indexer) GetTicker(tickerName string) *common.BRC20Ticker {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	info := s.loadTickInfo(strings.ToLower(tickerName))
	if info == nil {
		return nil
	}

	return info.Ticker
}

// 获取该ticker的holder和持有的资产数量
// return: key, address; value, amt
func (s *BRC20Indexer) GetHoldersWithTick(tickerName string) map[uint64]*common.Decimal {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.getHoldersWithTick(tickerName)
}

func (s *BRC20Indexer) getHoldersWithTick(tickerName string) map[uint64]*common.Decimal {

	name := strings.ToLower(tickerName)
	result := s.loadHoldersInTickerFromDB(name)

	// 根据缓存更新
	for addressId, holder := range s.holderMap {
		info, ok := holder.Tickers[name]
		if ok {
			result[addressId] = info.AssetAmt()
		}
	}
	return result
}

// 获取某个地址下的资产 return: ticker->amount
func (s *BRC20Indexer) GetAssetSummaryByAddress(addrId uint64) map[string]*common.Decimal {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := s.loadHolderInfoFromDBV2(addrId)


	// 根据缓存更新
	holder, ok := s.holderMap[addrId]
	if ok {
		for name, info := range holder.Tickers {
			result[name] = info.AssetAmt()
		}
	}

	return result
}

// 获取某个地址下的资产 return: ticker->amount
func (s *BRC20Indexer) hasAssetInAddress(addrId uint64) bool {

	info, ok := s.holderMap[addrId]
	if ok {
		for _, v := range info.Tickers {
			if v.AvailableBalance.Sign() != 0 {
				return true
			}
			if v.TransferableBalance.Sign() != 0 {
				return true
			}
		}
	}

	return s.checkHolderAssetFromDB(addrId)
}

// return: 按铸造时间排序的铸造历史
func (s *BRC20Indexer) GetMintHistory(tick string, start, limit int) []*common.BRC20MintAbbrInfo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// tickinfo, ok := s.tickerMap[strings.ToLower(tick)]
	// if !ok {
	// 	return nil
	// }

	result := make([]*common.BRC20MintAbbrInfo, 0)
	// for _, info := range tickinfo.InscriptionMap {
	// 	result = append(result, info)
	// }

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

	// tickinfo, ok := s.tickerMap[strings.ToLower(tick)]
	// if !ok {
	// 	return nil, 0
	// }

	result := make([]*common.MintAbbrInfo, 0)
	// for _, info := range tickinfo.InscriptionMap {
	// 	if info.Address == addressId {
	// 		result = append(result, info.ToMintAbbrInfo())
	// 	}
	// }

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

	ticker := s.GetTicker(strings.ToLower(tick))
	if ticker == nil {
		return nil, 0
	}

	return ticker.Minted.Clone(), int64(ticker.MintCount)
}

func (s *BRC20Indexer) GetMint(tickerName string, id int64) *common.BRC20Mint {

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	ticker := s.tickerMap[strings.ToLower(tickerName)]
	if ticker == nil {
		return nil
	}

	for _, mint := range ticker.MintAdded {
		if mint.Nft.Base.Id == id {
			return mint
		}
	}

	return s.loadMintFromDB(tickerName, id)
}

// return: 按铸造时间排序的铸造历史
func (s *BRC20Indexer) GetTransferHistory(tick string, start, limit int) []*common.BRC20ActionHistory {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := s.loadTickerHistory(tick)

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

func (s *BRC20Indexer) GetUtxoAssets(utxoId uint64) (*common.BRC20TransferInfo) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	transferNft := s.loadTransferNft(utxoId)
	if transferNft != nil {
		return &common.BRC20TransferInfo{
			NftId:   transferNft.TransferNft.NftId,
			Name:    transferNft.Ticker,
			Amt:     transferNft.TransferNft.Amount.Clone(),
			Invalid: transferNft.TransferNft.IsInvalid,
		}
	}

	return nil

	// 检查是否可能是mint的结果
	// nfts := s.nftIndexer.GetNftsWithUtxo(utxoId)
	// for _, nft := range nfts {
	// 	txid, index, err := common.ParseOrdInscriptionID(nft.Base.InscriptionId)
	// 	if err != nil {
	// 		continue
	// 	}
	// 	if index != 0 {
	// 		continue
	// 	}

	// 	switch string(nft.Base.ContentType) {
	// 	case "application/json":
	// 		fallthrough
	// 	case "text/plain;charset=utf-8":
	// 		fallthrough
	// 	case "text/plain":
	// 	default:
	// 		continue
	// 	}
	// 	if s.nftIndexer.GetBaseIndexer().IsMainnet() && s.IsExistCursorInscriptionInDB(nft.Base.InscriptionId) {
	// 		continue
	// 	}
	// 	content := common.ParseBBRC20AmtContent(string(nft.Base.Content))
	// 	if content == nil {
	// 		continue
	// 	}
	// 	tickerName := strings.ToLower(content.Ticker)
	// 	switch content.Op {
	// 	case "mint":
	// 		// 对于mint的结果，只有在mint的输出还没有被使用时，才返回资产数据，否则就当作一个完全的白聪
	// 		utxo := base_indexer.ShareBaseIndexer.GetUtxoById(utxoId)
	// 		// common.Log.Info("GetUtxoAssets", " utxoId ", utxoId, " testUtxo ", utxo)
	// 		if !strings.Contains(utxo, txid) {
	// 			continue
	// 		}
	// 		ticker := s.tickerMap[tickerName]
	// 		if ticker != nil {
	// 			for _, v := range ticker.MintAdded {
	// 				if v.Nft.Base.InscriptionId == nft.Base.InscriptionId {
	// 					ret = &common.BRC20TransferInfo{
	// 						NftId:   nft.Base.Id,
	// 						Name:    content.Ticker,
	// 						Amt:     v.Amt.Clone(),
	// 						Invalid: true,
	// 					}
	// 					return
	// 				}
	// 			}
	// 		}
	// 		mint := s.loadMintFromDB(tickerName, nft.Base.Id)
	// 		if mint != nil {
	// 			ret = &common.BRC20TransferInfo{
	// 				NftId:   nft.Base.Id,
	// 				Name:    content.Ticker,
	// 				Amt:     mint.Amt.Clone(),
	// 				Invalid: true,
	// 			}
	// 			return
	// 		}

	// 	}
	// }

	//return
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

	return s.getHolderAbbrInfo(addressId, strings.ToLower(tickerName))
}


// transfer
func (s *BRC20Indexer) getHolderAbbrInfo(addressId uint64, tickerName string) *common.BRC20TickAbbrInfo {

	holder := s.loadHolderInfo(addressId, tickerName)
	if holder == nil {
		return nil
	}

	tickAbbrInfo := holder.Tickers[tickerName]
	if tickAbbrInfo == nil {
		return nil
	}
	return tickAbbrInfo
}

func (s *BRC20Indexer) CheckEmptyAddress(wantToDelete map[string]uint64) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	hasAssetAddress := make(map[string]uint64)
	needCheckInDB := make([]uint64, 0)
	idToStr := make(map[uint64]string)
	for k, v := range wantToDelete {
		hasAsset := false
		info, ok := s.holderMap[v]
		if ok {
			for _, v := range info.Tickers {
				if v.AvailableBalance.Sign() != 0 {
					hasAsset = true
					break
				}
				if v.TransferableBalance.Sign() != 0 {
					hasAsset = true
					break
				}
			}
		}
		if hasAsset {
			hasAssetAddress[k] = v
		} else {
			needCheckInDB = append(needCheckInDB, v)
			idToStr[v] = k
		}
	}

	if len(needCheckInDB) > 0 {
		hasAssetList := s.checkHolderAssetFromDBV2(needCheckInDB)
		for _, addressId := range hasAssetList {
			hasAssetAddress[idToStr[addressId]] = addressId
		}
	}

	for k := range hasAssetAddress {
		delete(wantToDelete, k)
	}
}
