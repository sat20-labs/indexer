package ordx

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/common"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
	swire "github.com/sat20-labs/satsnet_btcd/wire"
)

func (s *Model) GetSyncHeight() int {
	return s.indexer.GetSyncHeight()
}

func (s *Model) GetBlockInfo(height int) (*common.BlockInfo, error) {
	return s.indexer.GetBlockInfo(height)
}

func (s *Model) IsDeployAllowed(ticker string) (bool, error) {
	
	name := swire.NewAssetNameFromString(ticker)
	if name == nil {
		return false, fmt.Errorf("invalid ticker name")
	}
	
	tickerInfo := s.indexer.GetTickerInfo(name)
	if tickerInfo != nil {
		return false, fmt.Errorf("ticker %s exists", ticker)
	}
	
	return true, nil
}

func (s *Model) GetTickerStatusList() ([]*rpcwire.TickerStatus, error) {
	tickerStatusRespMap, err := s.getTickStatusMap()
	if err != nil {
		return nil, err
	}

	ret := make([]*rpcwire.TickerStatus, 0)
	for _, tickerStatusResp := range tickerStatusRespMap {
		ret = append(ret, tickerStatusResp)
	}

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].ID < ret[j].ID
	})
	return ret, nil
}


func (s *Model) GetMintableTickerList(protocol string) ([]*rpcwire.TickerStatus, error) {
	tickerStatusRespMap, err := s.getMintableTickStatusMap(protocol)
	if err != nil {
		return nil, err
	}

	ret := make([]*rpcwire.TickerStatus, 0)
	for _, tickerStatusResp := range tickerStatusRespMap {
		ret = append(ret, tickerStatusResp)
	}

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].ID < ret[j].ID
	})
	return ret, nil
}

func (s *Model) GetTickerStatus(tickerName string) (*rpcwire.TickerStatus, error) {
	return s.getTicker(tickerName)
}

func (s *Model) GetTickerList(protocol string) ([]string) {
	result := make([]string, 0)
	tickmap := s.indexer.GetTickerMapV2(protocol)
	for k, _ := range tickmap {
		result = append(result, k)
	}
	return result
}

func (s *Model) GetTickerInfo(tickerName string) (*common.TickerInfo, error) {
	return s.getTickerInfo(tickerName)
}

func (s *Model) GetAddressMintHistory(tickerName, address string, start, limit int) (*rpcwire.MintHistory, error) {

	var ticker common.TickerName
	if len(tickerName) < common.MIN_NAME_LEN {
		ticker.Type = tickerName
		ticker.Ticker = ""
	} else {
		ticker.Type = common.ASSET_TYPE_FT
		ticker.Ticker = tickerName
	}
	result := rpcwire.MintHistory{TypeName: ticker.Type, Ticker: tickerName}
	mintmap, total := s.indexer.GetMintHistoryWithAddress(address, &ticker, start, limit)

	result.Total = total
	for _, mintInfo := range mintmap {
		ordxMintInfo := &rpcwire.MintHistoryItem{
			MintAddress:    address,
			HolderAddress:  s.indexer.GetHolderAddress(mintInfo.InscriptionId),
			Balance:        mintInfo.Amount,
			InscriptionID:  mintInfo.InscriptionId,
			InscriptionNum: mintInfo.InscriptionNum,
		}
		result.Items = append(result.Items, ordxMintInfo)
	}

	return &result, nil
}

func (s *Model) GetMintHistory(tickerName string, start, limit int) (*rpcwire.MintHistory, error) {
	result := rpcwire.MintHistory{Ticker: tickerName}
	mintmap := s.indexer.GetMintHistory(tickerName, start, limit)
	for _, mintInfo := range mintmap {
		ordxMintInfo := &rpcwire.MintHistoryItem{
			MintAddress:    s.indexer.GetAddressById(mintInfo.Address),
			HolderAddress:  s.indexer.GetHolderAddress(mintInfo.InscriptionId),
			Balance:        mintInfo.Amount,
			InscriptionID:  mintInfo.InscriptionId,
			InscriptionNum: mintInfo.InscriptionNum,
		}
		result.Items = append(result.Items, ordxMintInfo)
	}
	_, times := s.indexer.GetMintAmount(tickerName)
	result.Total = int(times)

	return &result, nil
}

func (s *Model) GetMintDetailInfo(inscriptionId string) (*rpcwire.MintDetailInfo, error) {
	mintInfo := s.indexer.GetMintInfo(inscriptionId)
	if mintInfo == nil {
		return nil, fmt.Errorf(" GetMintDetails failed %s", inscriptionId)
	}

	ret := &rpcwire.MintDetailInfo{
		InscriptionNum: mintInfo.Base.Id,
		ID:             mintInfo.Id,
		Ticker:         mintInfo.Name,
		InscriptionID:  mintInfo.Base.InscriptionId,
		MintAddress:    s.indexer.GetAddressById(mintInfo.Base.InscriptionAddress),
		Amount:         mintInfo.Amt,
		MintTime:       mintInfo.Base.BlockTime,
		Delegate:       mintInfo.Base.Delegate,
		Content:        mintInfo.Base.Content,
		ContentType:    string(mintInfo.Base.ContentType),
		Ranges:         mintInfo.Ordinals,
	}

	return ret, nil
}

func (s *Model) GetMintPermissionInfo(ticker, address string) (*rpcwire.MintPermissionInfo, error) {
	amount := s.indexer.GetMintPermissionInfo(ticker, address)
	if amount < 0 {
		return nil, fmt.Errorf("GetMintPermission failed. %s %s", ticker, address)
	}

	ret := &rpcwire.MintPermissionInfo{
		Ticker:  ticker,
		Address: address,
		Amount:  amount,
	}

	return ret, nil
}

func (s *Model) GetFeeInfo(address string) (*rpcwire.FeeInfo, error) {
	utxomap, err := s.indexer.GetAssetUTXOsInAddressWithTick(address, 
		&common.TickerName{Protocol:common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_FT, Ticker: "pearl"})
	if err != nil {
		return nil, err
	}

	amount := int64(0)
	for _, v := range utxomap {
		amount += v
	}

	discount := 0
	if amount >= 100000 {
		discount = 100
	} else {
		discount = int(amount / 1000)
	}

	ret := &rpcwire.FeeInfo{
		Address:  address,
		Discount: discount,
	}

	return ret, nil
}

func (s *Model) GetSplittedInscriptionList(tickerName string) []string {
	return s.indexer.GetSplittedInscriptionsWithTick(tickerName)
}

func (s *Model) GetHolderList(tickName string, start, limit int) ([]*rpcwire.Holder, error) {
	// TODO 分页显示
	addressMap := s.indexer.GetHoldersWithTick(tickName)
	result := make([]*rpcwire.Holder, 0)
	for address, amt := range addressMap {
		ordxMintInfo := &rpcwire.Holder{
			Wallet:       s.indexer.GetAddressById(address),
			TotalBalance: amt,
		}
		result = append(result, ordxMintInfo)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalBalance > result[j].TotalBalance
	})
	return result, nil
}

func (s *Model) GetBalanceSummaryList(address string, start int, limit int) ([]*rpcwire.BalanceSummary, error) {
	tickerMap := s.indexer.GetAssetSummaryInAddress(address)

	result := make([]*rpcwire.BalanceSummary, 0)
	for tickName, balance := range tickerMap {
		resp := &rpcwire.BalanceSummary{
			TypeName: tickName.Type,
			Ticker:   tickName.Ticker,
			Balance:  balance,
		}
		resp.Balance = balance
		result = append(result, resp)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Balance > result[j].Balance
	})

	return result, nil
}

func (s *Model) GetAssetsWithUtxos_deprecated(req *rpcwire.UtxosReq) ([]*rpcwire.UtxoAbbrAssets, error) {
	result := make([]*rpcwire.UtxoAbbrAssets, 0)
	for _, utxo := range req.Utxos {
		utxoId := s.indexer.GetUtxoId(utxo)
		assets := s.indexer.GetAssetsWithUtxo(utxoId)

		utxoAssets := rpcwire.UtxoAbbrAssets{Utxo: utxo}
		for ticker, mintinfo := range assets {

			amount := int64(0)
			for _, rng := range mintinfo {
				amount += common.GetOrdinalsSize(rng)
			}

			utxoAssets.Assets = append(utxoAssets.Assets, &rpcwire.AssetAbbrInfo{
				TypeName: ticker.Type,
				Ticker:   ticker.Ticker,
				Amount:   amount,
			})
		}
		sort.Slice(utxoAssets.Assets, func(i, j int) bool {
			return utxoAssets.Assets[i].Amount > utxoAssets.Assets[j].Amount
		})

		result = append(result, &utxoAssets)
	}

	return result, nil
}


func (s *Model) GetExistingUtxos(req *rpcwire.UtxosReq) ([]string, error) {
	result := make([]string, 0)
	for _, utxo := range req.Utxos {
		utxoId := s.indexer.GetUtxoId(utxo)
		if utxoId == common.INVALID_ID {
			continue
		}

		result = append(result, utxo)
	}

	return result, nil
}


func (s *Model) GetUtxoList(address string, tickerName string, start, limit int) ([]*rpcwire.TickerAsset, int, error) {

	var ticker common.TickerName
	if len(tickerName) < common.MIN_NAME_LEN {
		ticker.Type = tickerName
		ticker.Ticker = common.ALL_TICKERS
	} else {
		ticker.Type = common.ASSET_TYPE_FT
		ticker.Ticker = tickerName
	}

	utxos, err := s.indexer.GetAssetUTXOsInAddressWithTick(address, &ticker)
	if err != nil {
		return nil, 0, err
	}

	type UtxoAsset struct {
		Utxo   uint64
		Amount int64
	}
	utxosort := make([]*UtxoAsset, 0)
	for utxo, amout := range utxos {
		utxostr := s.indexer.GetUtxoById(utxo)
		if rpcwire.IsExistUtxoInMemPool(utxostr) {
			common.Log.Infof("IsExistUtxoInMemPool return true %s", utxostr)
			continue
		}
		utxosort = append(utxosort, &UtxoAsset{utxo, amout})
	}
	sort.Slice(utxosort, func(i, j int) bool {
		if utxosort[i].Amount == utxosort[j].Amount {
			return utxosort[i].Utxo < utxosort[j].Utxo
		} else {
			return utxosort[i].Amount > utxosort[j].Amount
		}
	})

	// 分页显示
	totalRecords := len(utxosort)
	if totalRecords < start {
		return nil, 0, nil
	}
	if totalRecords < start+limit {
		limit = totalRecords - start
	}
	end := start + limit
	utxoresult := utxosort[start:end]

	result := make([]*rpcwire.TickerAsset, 0)
	for _, utxoAsset := range utxoresult {
		_, rngs, err := s.indexer.GetOrdinalsWithUtxoId(utxoAsset.Utxo)
		if err != nil {
			common.Log.Errorf("GetOrdinalsForUTXO %d failed, %v", utxoAsset.Utxo, err)
			continue
		}

		assets := s.indexer.GetAssetsWithUtxo(utxoAsset.Utxo)
		for k, mintinfo := range assets {
			if k.Type != ticker.Type || (ticker.Ticker != "" && k.Ticker != ticker.Ticker) {
				continue
			}

			resp := &rpcwire.TickerAsset{
				TypeName: ticker.Type,
				Ticker:   ticker.Ticker,
				Utxo:     s.indexer.GetUtxoById(utxoAsset.Utxo),
			}
			resp.Amount = common.GetOrdinalsSize(rngs)

			for inscriptionId, ranges := range mintinfo {
				asset := rpcwire.InscriptionAsset{}
				asset.AssetAmount = common.GetOrdinalsSize(ranges)
				asset.Ranges = ranges
				asset.InscriptionNum = common.INVALID_INSCRIPTION_NUM
				asset.InscriptionID = inscriptionId

				resp.Assets = append(resp.Assets, &asset)
				resp.AssetAmount += asset.AssetAmount
			}

			sort.Slice(resp.Assets, func(i, j int) bool {
				return resp.Assets[i].InscriptionID < resp.Assets[j].InscriptionID
			})

			result = append(result, resp)
		}

	}

	return result, totalRecords, nil
}

// including all other tickers in the utxo
func (s *Model) GetUtxoList2(address string, tickerName string, start, limit int) ([]*rpcwire.TickerAsset, int, error) {
	var ticker common.TickerName
	if len(tickerName) < common.MIN_NAME_LEN {
		ticker.Type = tickerName
		ticker.Ticker = common.ALL_TICKERS
	} else {
		ticker.Type = common.ASSET_TYPE_FT
		ticker.Ticker = tickerName
	}

	utxos, err := s.indexer.GetAssetUTXOsInAddressWithTick(address, &ticker)
	if err != nil {
		return nil, 0, err
	}
	type UtxoAsset struct {
		Utxo   uint64
		Amount int64
	}
	utxosort := make([]*UtxoAsset, 0)
	for utxo, amout := range utxos {
		utxostr := s.indexer.GetUtxoById(utxo)
		if rpcwire.IsExistUtxoInMemPool(utxostr) {
			common.Log.Infof("IsExistUtxoInMemPool return true %s", utxostr)
			continue
		}
		utxosort = append(utxosort, &UtxoAsset{utxo, amout})
	}
	sort.Slice(utxosort, func(i, j int) bool {
		if utxosort[i].Amount == utxosort[j].Amount {
			return utxosort[i].Utxo < utxosort[j].Utxo
		} else {
			return utxosort[i].Amount > utxosort[j].Amount
		}
	})

	// 分页显示
	totalRecords := len(utxosort)
	if totalRecords < start {
		return nil, 0, nil
	}
	if totalRecords < start+limit {
		limit = totalRecords - start
	}
	end := start + limit
	utxoresult := utxosort[start:end]

	result := make([]*rpcwire.TickerAsset, 0)
	for _, utxoAsset := range utxoresult {
		_, rngs, err := s.indexer.GetOrdinalsWithUtxoId(utxoAsset.Utxo)
		if err != nil {
			common.Log.Errorf("GetOrdinalsForUTXO %d failedm, %v", utxoAsset.Utxo, err)
			continue
		}

		tickAbbrInfoMap := s.indexer.GetAssetsWithUtxo(utxoAsset.Utxo)

		resp := &rpcwire.TickerAsset{
			Ticker: tickerName,
			Utxo:   s.indexer.GetUtxoById(utxoAsset.Utxo),
		}
		resp.Amount = common.GetOrdinalsSize(rngs)
		resp.AssetAmount += 0

		for ticker, tickAbbrInfo := range tickAbbrInfoMap {
			for inscId, ranges := range tickAbbrInfo {
				asset := rpcwire.InscriptionAsset{}
				asset.TypeName = ticker.Type
				asset.Ticker = ticker.Ticker
				asset.AssetAmount = common.GetOrdinalsSize(ranges)
				asset.Ranges = ranges
				asset.InscriptionNum = common.INVALID_INSCRIPTION_NUM
				asset.InscriptionID = inscId

				resp.Assets = append(resp.Assets, &asset)
			}
		}

		result = append(result, resp)
	}

	return result, totalRecords, nil
}

// including assets
func (s *Model) GetUtxoList3(address string, start, limit int) ([]*rpcwire.TickerAsset, int, error) {
	utxos := s.indexer.GetAssetUTXOsInAddress(address)
	type UtxoAsset struct {
		Utxo   uint64
		Ticker *common.TickerName
	}
	utxosort := make([]*UtxoAsset, 0)
	for key, value := range utxos {
		for _, u := range value {
			utxostr := s.indexer.GetUtxoById(u)
			if rpcwire.IsExistUtxoInMemPool(utxostr) {
				common.Log.Infof("IsExistUtxoInMemPool return true %s", utxostr)
				continue
			}
			a := &UtxoAsset{Utxo: u, Ticker: &key}
			utxosort = append(utxosort, a)
		}
	}
	sort.Slice(utxosort, func(i, j int) bool {
		return utxosort[i].Utxo < utxosort[j].Utxo
	})

	// 分页显示
	totalRecords := len(utxosort)
	if totalRecords < start {
		return nil, 0, nil
	}
	if totalRecords < start+limit {
		limit = totalRecords - start
	}
	end := start + limit
	utxoresult := utxosort[start:end]

	result := make([]*rpcwire.TickerAsset, 0)
	for _, utxoAsset := range utxoresult {
		_, rngs, err := s.indexer.GetOrdinalsWithUtxoId(utxoAsset.Utxo)
		if err != nil {
			common.Log.Errorf("GetOrdinalsForUTXO %d failedm, %v", utxoAsset.Utxo, err)
			continue
		}

		tickAbbrInfoMap := s.indexer.GetAssetsWithUtxo(utxoAsset.Utxo)

		resp := &rpcwire.TickerAsset{
			Ticker: "",
			Utxo:   s.indexer.GetUtxoById(utxoAsset.Utxo),
		}
		resp.Amount = common.GetOrdinalsSize(rngs)
		resp.AssetAmount = 0
		for ticker, tickAbbrInfo := range tickAbbrInfoMap {
			for inscId, ranges := range tickAbbrInfo {
				asset := rpcwire.InscriptionAsset{}
				asset.TypeName = ticker.Type
				asset.Ticker = ticker.Ticker
				asset.AssetAmount = common.GetOrdinalsSize(ranges)
				asset.Ranges = ranges
				asset.InscriptionNum = common.INVALID_INSCRIPTION_NUM
				asset.InscriptionID = inscId

				resp.Assets = append(resp.Assets, &asset)
			}
		}
		result = append(result, resp)
	}

	return result, totalRecords, nil
}

func (s *Model) GetDetailAssetWithUtxo(utxo string) (*rpcwire.AssetDetailInfo, error) {
	utxoId, rngs, err := s.indexer.GetOrdinalsWithUtxo(utxo)
	if err != nil {
		common.Log.Errorf("GetUtxoAsset failed, %s", utxo)
		return nil, err
	}

	var result rpcwire.AssetDetailInfo
	result.Utxo = utxo
	result.Value = int64(common.GetOrdinalsSize(rngs))
	result.Ranges = rngs

	// TODO 是否需要做这个过滤？如果需要，所有获取资产的地方都要修改
	// 1.同一个inscriptionId，只出现一次
	// 2.高级别资产，优先显示：比如有ft或ns，就不需要显示nft

	assets := s.indexer.GetAssetsWithUtxo(utxoId)
	for ticker, mintinfo := range assets {

		var tickinfo rpcwire.TickerAsset
		tickinfo.TypeName = ticker.Type
		tickinfo.Ticker = ticker.Ticker
		tickinfo.Utxo = ""
		tickinfo.Amount = 0

		for inscriptionId, mintranges := range mintinfo {
			// _, ok := inscriptionMap[inscriptionId]
			// if ok {
			// 	continue
			// } else {
			// 	inscriptionMap[inscriptionId] = true
			// }

			asset := rpcwire.InscriptionAsset{}
			asset.AssetAmount = common.GetOrdinalsSize(mintranges)
			asset.Ranges = mintranges
			asset.InscriptionNum = common.INVALID_INSCRIPTION_NUM
			asset.InscriptionID = inscriptionId

			tickinfo.Assets = append(tickinfo.Assets, &asset)
			tickinfo.AssetAmount += asset.AssetAmount
		}

		sort.Slice(tickinfo.Assets, func(i, j int) bool {
			return tickinfo.Assets[i].InscriptionID < tickinfo.Assets[j].InscriptionID
		})

		if tickinfo.AssetAmount > 0 {
			result.Assets = append(result.Assets, &tickinfo)
		}
	}

	sort.Slice(result.Assets, func(i, j int) bool {
		return result.Assets[i].AssetAmount > result.Assets[j].AssetAmount
	})

	return &result, nil
}

func (s *Model) GetAssetOffsetWithUtxo(utxo string) ([]*common.AssetOffsetRange, error) {
	ret := s.indexer.GetAssetOffsetWithUtxo(utxo)
	if len(ret) == 0 {
		return nil, fmt.Errorf("can't find this utxo %s", utxo)
	}
	return ret, nil
}

func (s *Model) GetDetailAssetWithRanges(req *rpcwire.RangesReq) (*rpcwire.AssetDetailInfo, error) {

	var result rpcwire.AssetDetailInfo
	result.Ranges = req.Ranges
	result.Utxo = ""
	result.Value = common.GetOrdinalsSize(req.Ranges)

	assets := s.indexer.GetAssetsWithRanges(req.Ranges)
	for tickerName, info := range assets {

		var tickinfo rpcwire.TickerAsset
		tickinfo.Ticker = tickerName
		tickinfo.Utxo = ""
		tickinfo.Amount = 0

		for mintutxo, mintranges := range info {
			asset := rpcwire.InscriptionAsset{}
			asset.AssetAmount = common.GetOrdinalsSize(mintranges)
			asset.Ranges = mintranges
			asset.InscriptionNum = common.INVALID_INSCRIPTION_NUM
			asset.InscriptionID = mintutxo

			tickinfo.Assets = append(tickinfo.Assets, &asset)
			tickinfo.AssetAmount += asset.AssetAmount
		}

		result.Assets = append(result.Assets, &tickinfo)
	}

	sort.Slice(result.Assets, func(i, j int) bool {
		return result.Assets[i].AssetAmount > result.Assets[j].AssetAmount
	})

	return &result, nil
}

func (s *Model) GetAbbrAssetsWithUtxo(utxo string) ([]*rpcwire.AssetAbbrInfo, error) {
	result := make([]*rpcwire.AssetAbbrInfo, 0)
	utxoId := s.indexer.GetUtxoId(utxo)
	assets := s.indexer.GetAssetsWithUtxo(utxoId)
	for ticker, mintinfo := range assets {

		amount := int64(0)
		for _, rng := range mintinfo {
			amount += common.GetOrdinalsSize(rng)
		}

		result = append(result, &rpcwire.AssetAbbrInfo{
			TypeName: ticker.Type,
			Ticker:   ticker.Ticker,
			Amount:   amount,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Amount > result[j].Amount
	})

	return result, nil
}

func (s *Model) GetSeedsWithUtxo(utxo string) ([]*rpcwire.Seed, error) {
	result := make([]*rpcwire.Seed, 0)
	assets := s.indexer.GetAssetsWithUtxo(s.indexer.GetUtxoId(utxo))
	for ticker, info := range assets {
		assetRanges := make([]*common.Range, 0)
		for _, rngs := range info {
			assetRanges = append(assetRanges, rngs...)
		}
		seed := rpcwire.Seed{TypeName: ticker.Type, Ticker: ticker.Ticker, Seed: common.GenerateSeed2(assetRanges)}
		result = append(result, &seed)
	}

	return result, nil
}

func (s *Model) GetSatRangeWithUtxo(utxo string) (*rpcwire.UtxoInfo, error) {
	utxoId := uint64(common.INVALID_ID)
	var err error
	if len(utxo) < 64 {
		utxoId, err = strconv.ParseUint(utxo, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	result := rpcwire.UtxoInfo{}
	if utxoId == common.INVALID_ID {
		result.Id, result.Ranges, err = s.indexer.GetOrdinalsWithUtxo(utxo)
		result.Utxo = utxo
	} else {
		result.Utxo, result.Ranges, err = s.indexer.GetOrdinalsWithUtxoId(utxoId)
		result.Id = utxoId
	}
	if err != nil {
		common.Log.Warnf("GetSatRangeWithUtxo %s failed, %v", utxo, err)
		return nil, err
	}

	return &result, nil
}

func (s *Model) GetNSStatusList(start, limit int) (*rpcwire.NSStatusData, error) {
	status := s.indexer.GetNSStatus()

	ret := rpcwire.NSStatusData{Version: status.Version, Total: (status.NameCount), Start: uint64(start)}
	names := s.indexer.GetNames(start, limit)
	for _, name := range names {
		info := s.indexer.GetNameInfo(name)
		if info != nil {
			item := s.nameToItem(info)
			ret.Names = append(ret.Names, item)
		}
	}

	return &ret, nil
}

func (s *Model) GetNameInfo(name string) (*rpcwire.OrdinalsName, error) {
	info := s.indexer.GetNameInfo(name)
	if info == nil {
		return nil, fmt.Errorf("can't find name %s", name)
	}

	ret := rpcwire.OrdinalsName{NftItem: *s.nameToItem(info)}
	for k, v := range info.KVs {
		item := rpcwire.KVItem{Key: k, Value: v.Value, InscriptionId: v.InscriptionId}
		ret.KVItemList = append(ret.KVItemList, &item)
	}

	return &ret, nil
}

func (s *Model) GetNameValues(name, prefix string, start, limit int) (*rpcwire.OrdinalsName, error) {
	info := s.indexer.GetNameInfo(name)
	if info == nil {
		return nil, fmt.Errorf("can't find name %s", name)
	}

	type FilterResult struct {
		Key   string
		Value *common.KeyValueInDB
	}

	filter := make([]*FilterResult, 0)
	for k, v := range info.KVs {
		if strings.HasPrefix(k, prefix) {
			filter = append(filter, &FilterResult{Key: k, Value: v})
		}
	}

	sort.Slice(filter, func(i, j int) bool {
		return filter[i].Key > filter[j].Key
	})

	totalRecords := len(filter)
	if totalRecords < start {
		return nil, fmt.Errorf("start exceeds boundary")
	}
	if totalRecords < start+limit {
		limit = totalRecords - start
	}
	end := start + limit
	newFilter := filter[start:end]

	ret := rpcwire.OrdinalsName{NftItem: *s.nameToItem(info)}
	for _, kv := range newFilter {
		item := rpcwire.KVItem{Key: kv.Key, Value: kv.Value.Value, InscriptionId: kv.Value.InscriptionId}
		ret.KVItemList = append(ret.KVItemList, &item)
	}
	ret.Total = totalRecords
	ret.Start = start

	return &ret, nil
}

func (s *Model) GetNameRouting(name string) (*rpcwire.NameRouting, error) {
	info := s.indexer.GetNameInfo(name)
	if info == nil {
		return nil, fmt.Errorf("can't find name %s", name)
	}

	ret := rpcwire.NameRouting{Holder: info.OwnerAddress, InscriptionId: info.Base.InscriptionId, P: "btcname", Op: "routing", Name: info.Name}
	for k, v := range info.KVs {
		switch k {
		case "ord_handle":
			ret.Handle = v.Value
		case "ord_index":
			ret.Index = v.Value
		}
	}

	return &ret, nil
}

func (s *Model) GetNamesWithAddress(address, sub string, start, limit int) (*rpcwire.NamesWithAddressData, error) {
	ret := rpcwire.NamesWithAddressData{Address: address}
	var names []*common.NameInfo
	var total int
	if sub == "" {
		names, total = s.indexer.GetNamesWithAddress(address, start, limit)
	} else {
		if sub == "PureName" {
			sub = ""
		}
		names, total = s.indexer.GetSubNamesWithAddress(address, sub, start, limit)
	}

	for _, info := range names {
		data := rpcwire.OrdinalsName{NftItem: *s.nameToItem(info)}
		// 暂时不要传回kv
		// for k, v := range info.KVs {
		// 	item := rpcwire.KVItem{Key: k, Value: v.Value, InscriptionId: v.InscriptionId}
		// 	data.KVItemList = append(data.KVItemList, &item)
		// }
		ret.Names = append(ret.Names, &data)
	}
	ret.Total = total

	return &ret, nil
}

func (s *Model) GetNamesWithFilters(address, sub, filters string, start, limit int) (*rpcwire.NamesWithAddressData, error) {
	ret := rpcwire.NamesWithAddressData{Address: address}
	var names []*common.NameInfo
	var total int

	if sub == "PureName" {
		sub = ""
	}
	names, total = s.indexer.GetSubNamesWithFilters(address, sub, filters, start, limit)

	for _, info := range names {
		data := rpcwire.OrdinalsName{NftItem: *s.nameToItem(info)}
		// 暂时不要传回kv
		// for k, v := range info.KVs {
		// 	item := rpcwire.KVItem{Key: k, Value: v.Value, InscriptionId: v.InscriptionId}
		// 	data.KVItemList = append(data.KVItemList, &item)
		// }
		ret.Names = append(ret.Names, &data)
	}
	ret.Total = total

	return &ret, nil
}

func (s *Model) GetNamesWithSat(sat int64) (*rpcwire.NamesWithAddressData, error) {
	ret := rpcwire.NamesWithAddressData{}
	names := s.indexer.GetNamesWithSat(sat)
	for _, info := range names {
		data := rpcwire.OrdinalsName{NftItem: *s.nameToItem(info)}
		for k, v := range info.KVs {
			item := rpcwire.KVItem{Key: k, Value: v.Value, InscriptionId: v.InscriptionId}
			data.KVItemList = append(data.KVItemList, &item)
		}
		ret.Names = append(ret.Names, &data)
	}
	ret.Total = len(names)

	sort.Slice(ret.Names, func(i, j int) bool {
		return ret.Names[i].Name < ret.Names[j].Name
	})

	return &ret, nil
}

func (s *Model) GetNameWithInscriptionId(id string) (*rpcwire.OrdinalsName, error) {
	info := s.indexer.GetNameWithInscriptionId(id)
	if info == nil {
		return nil, fmt.Errorf("can't find name with %s", id)
	}

	ret := rpcwire.OrdinalsName{NftItem: *s.nameToItem(info)}
	for k, v := range info.KVs {
		item := rpcwire.KVItem{Key: k, Value: v.Value, InscriptionId: v.InscriptionId}
		ret.KVItemList = append(ret.KVItemList, &item)
	}

	return &ret, nil
}

func (s *Model) GetNameCheckResult(req *rpcwire.NameCheckReq) ([]*rpcwire.NameCheckResult, error) {
	result := make([]*rpcwire.NameCheckResult, 0)
	for _, name := range req.Names {
		name = common.PreprocessName(name)
		nc := rpcwire.NameCheckResult{Name: name}
		if common.IsValidSNSName(name) {
			if s.indexer.IsNameExist(name) {
				nc.Result = 1
			} else {
				nc.Result = 0
			}
		} else {
			nc.Result = -1
		}
		result = append(result, &nc)
	}
	return result, nil
}

func (s *Model) AddCollection(req *rpcwire.AddCollectionReq) error {
	if strings.Contains(req.Ticker, "-") {
		return fmt.Errorf("ticker name contains symbol -")
	}
	ids := make([]string, 0)
	for _, id := range req.Data {
		ids = append(ids, id.Id)
	}

	return s.indexer.AddCollection(req.Type, req.Ticker, ids)
}

func (s *Model) GetNftStatusList(start, limit int) (*rpcwire.NftStatusData, error) {
	status := s.indexer.GetNftStatus()

	ret := rpcwire.NftStatusData{Version: status.Version, Total: (status.Count), Start: uint64(start)}
	ids, _ := s.indexer.GetNfts(start, limit)
	for _, id := range ids {
		info := s.indexer.GetNftInfo(id)
		if info != nil {
			item := s.nftToItem(info)
			ret.Nfts = append(ret.Nfts, item)
		}
	}

	return &ret, nil
}

func (s *Model) GetNftInfo(id int64) (*rpcwire.NftInfo, error) {
	info := s.indexer.GetNftInfo(id)
	if info == nil {
		return nil, fmt.Errorf("can't find nft %d", id)
	}

	ret := rpcwire.NftInfo{
		NftItem:      *s.nftToItem(info),
		ContentType:  info.Base.ContentType,
		Content:      info.Base.Content,
		MetaProtocol: info.Base.MetaProtocol,
		MetaData:     info.Base.MetaData,
		Parent:       info.Base.Parent,
		Delegate:     info.Base.Delegate,
	}

	return &ret, nil
}

func (s *Model) GetNftsWithAddress(address string, start, limit int) (*rpcwire.NftsWithAddressData, int, error) {
	ret := rpcwire.NftsWithAddressData{Address: address}
	nfts, total := s.indexer.GetNftsWithAddress(address, start, limit)
	for _, nft := range nfts {
		utxo := s.indexer.GetUtxoById(nft.UtxoId)
		item := s.nftToItem(nft)
		item.Address = address
		item.Utxo = utxo
		item.Value = s.getUtxoValue2(utxo)
		ret.Nfts = append(ret.Nfts, item)

	}
	ret.Amount = len(ret.Nfts)

	return &ret, total, nil
}

func (s *Model) GetNftsWithSat(sat int64) (*rpcwire.NftsWithAddressData, error) {
	ret := rpcwire.NftsWithAddressData{}
	nfts := s.indexer.GetNftsWithSat(sat)
	address := s.indexer.GetAddressById(nfts.OwnerAddressId)
	utxo := s.indexer.GetUtxoById(nfts.UtxoId)
	value := s.getUtxoValue2(utxo)
	for _, info := range nfts.Nfts {
		item := s.baseContentToNftItem(info)
		item.Address = address
		item.Utxo = utxo
		item.Value = value
		ret.Nfts = append(ret.Nfts, item)

	}
	ret.Amount = len(ret.Nfts)

	sort.Slice(ret.Nfts, func(i, j int) bool {
		return ret.Nfts[i].Id < ret.Nfts[j].Id
	})

	return &ret, nil
}

func (s *Model) GetNftInfoWithInscriptionId(inscriptionId string) (*rpcwire.NftInfo, error) {
	info := s.indexer.GetNftInfoWithInscriptionId(inscriptionId)
	if info == nil {
		return nil, fmt.Errorf("can't find nft %s", inscriptionId)
	}

	ret := rpcwire.NftInfo{
		NftItem:      *s.nftToItem(info),
		ContentType:  info.Base.ContentType,
		Content:      info.Base.Content,
		MetaProtocol: info.Base.MetaProtocol,
		MetaData:     info.Base.MetaData,
		Parent:       info.Base.Parent,
		Delegate:     info.Base.Delegate,
	}

	return &ret, nil
}

func (s *Model) baseContentToNftItem(info *common.InscribeBaseContent) *rpcwire.NftItem {
	return &rpcwire.NftItem{
		Id:                 info.Id,
		Name:               info.TypeName,
		Sat:                info.Sat,
		InscriptionId:      info.InscriptionId,
		BlockHeight:        int(info.BlockHeight),
		BlockTime:          info.BlockTime,
		InscriptionAddress: s.indexer.GetAddressById(info.InscriptionAddress)}
}

func (s *Model) nftToItem(info *common.Nft) *rpcwire.NftItem {
	item := s.baseContentToNftItem(info.Base)
	item.Address = s.indexer.GetAddressById(info.OwnerAddressId)
	item.Utxo = s.indexer.GetUtxoById(info.UtxoId)
	item.Value = s.getUtxoValue(info.UtxoId)
	return item
}

func (s *Model) nameToItem(info *common.NameInfo) *rpcwire.NftItem {
	item := s.baseContentToNftItem(info.Base)
	item.Address = info.OwnerAddress
	item.Utxo = info.Utxo
	if info.Utxo == "" {
		common.Log.Errorf("database has been corrupted, should rebuild it!!!")
	} else {
		item.Value = s.getUtxoValue2(info.Utxo)
	}
	item.Id = info.Id
	item.Name = info.Name
	return item
}

func (s *Model) getUtxoValue(utxoId uint64) int64 {
	_, rngs, err := s.indexer.GetOrdinalsWithUtxoId(utxoId)
	if err != nil {
		return 0
	}
	return common.GetOrdinalsSize(rngs)
}

func (s *Model) getUtxoValue2(utxo string) int64 {
	_, rngs, err := s.indexer.GetOrdinalsWithUtxo(utxo)
	if err != nil {
		return 0
	}
	return common.GetOrdinalsSize(rngs)
}
