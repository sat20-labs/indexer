package indexer

import (
	"fmt"
	"sort"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
	"lukechampine.com/uint128"

	swire "github.com/sat20-labs/satsnet_btcd/wire"
)

// interface for RPC

func (b *IndexerMgr) IsMainnet() bool {
	return b.chaincfgParam.Name == "mainnet"
}

func (b *IndexerMgr) GetBaseDBVer() string {
	return b.compiling.GetBaseDBVer()
}

func (b *IndexerMgr) GetChainParam() *chaincfg.Params {
	return b.chaincfgParam
}

// return: addressId -> asset amount
func (b *IndexerMgr) GetHoldersWithTick(name string) map[uint64]int64 {

	switch name {
	case common.ASSET_TYPE_NFT:
	case common.ASSET_TYPE_NS:
	case common.ASSET_TYPE_EXOTIC:
	default:
	}

	return b.ftIndexer.GetHolderAndAmountWithTick(name)
}

func (b *IndexerMgr) GetHolderAmountWithTick(name string) int {
	am := b.ftIndexer.GetHoldersWithTick(name)
	return len(am)
}

func (b *IndexerMgr) HasAssetInUtxo(utxo string, excludingExotic bool) bool {
	utxoId, rngs, err := b.rpcService.GetOrdinalsWithUtxo(utxo)
	if err != nil {
		return false
	}

	result := b.ftIndexer.HasAssetInUtxo(utxoId)
	if result {
		return true
	}

	result = b.RunesIndexer.IsExistAsset(utxoId)
	if result {
		return true
	}

	result = b.nft.HasNftInUtxo(utxoId)
	if result {
		return true
	}

	if !excludingExotic && b.exotic.HasExoticInRanges(rngs) {
		return true
	}

	return result
}

// return: utxoId->asset amount
func (b *IndexerMgr) GetAssetUTXOsInAddressWithTick(address string, ticker *common.TickerName) (map[uint64]int64, error) {
	utxos, err := b.rpcService.GetUTXOs(address)
	if err != nil {
		common.Log.Errorf("GetUTXOs %s failed. %v", address, err)
		return nil, err
	}

	bSpecialTicker := false
	result := make(map[uint64]int64)
	switch ticker.Type {
	case common.ASSET_TYPE_NFT:
		var inscmap map[string]int64

		if ticker.Ticker != common.ALL_TICKERS {
			b.mutex.RLock()
			inscmap, bSpecialTicker = b.clmap[*ticker]
			b.mutex.RUnlock()
			if !bSpecialTicker {
				return nil, fmt.Errorf("no assets with ticker %v", ticker)
			}
		}

		for utxoId := range utxos {
			ids := b.GetNftsWithUtxo(utxoId)
			amount := 0
			if bSpecialTicker {
				for _, v := range ids {
					_, ok := inscmap[v]
					if ok {
						amount++
					}
				}
			} else {
				amount = len(ids)
			}

			if amount > 0 {
				result[utxoId] = int64(amount)
			}
		}

	case common.ASSET_TYPE_NS:
		if ticker.Ticker != common.ALL_TICKERS {
			bSpecialTicker = true
		}
		for utxoId := range utxos {
			names := b.GetNamesWithUtxo(utxoId)
			amount := 0
			if bSpecialTicker {
				for _, name := range names {
					_, subName := getSubName(name)
					if subName == ticker.Ticker {
						amount++
					}
				}
			} else {
				amount = len(names)
			}
			if amount > 0 {
				result[utxoId] = int64(amount)
			}
		}

	case common.ASSET_TYPE_EXOTIC:
		if ticker.Ticker != common.ALL_TICKERS {
			bSpecialTicker = true
		}
		for utxoId := range utxos {
			_, rng, err := b.GetOrdinalsWithUtxoId(utxoId)
			if err != nil {
				common.Log.Errorf("GetOrdinalsWithUtxoId failed, %d", utxoId)
				continue
			}

			sr := b.exotic.GetExoticsWithRanges2(rng)
			amount := int64(0)
			for name, rngs := range sr {
				if bSpecialTicker {
					if name == ticker.Ticker {
						amount += (common.GetOrdinalsSize(rngs))
					}
				} else {
					amount += (common.GetOrdinalsSize(rngs))
				}
			}
			if amount > 0 {
				result[utxoId] = amount
			}
		}

	case common.ASSET_TYPE_FT:
		result = b.ftIndexer.GetAssetUtxosWithTicker(b.rpcService.GetAddressId(address), ticker.Ticker)
	}

	return result, nil
}

// return: utxoId->asset amount
func (b *IndexerMgr) GetAssetUTXOsInAddressWithTickV2(address string, ticker *swire.AssetName) (map[uint64]*common.TxOutput, error) {
	utxos, err := b.rpcService.GetUTXOs(address)
	if err != nil {
		return nil, err
	}

	result := make(map[uint64]*common.TxOutput)
	for utxoId := range utxos {
		utxo, err := b.rpcService.GetUtxoByID(utxoId)
		if err != nil {
			continue
		}
		info := b.GetTxOutputWithUtxo(utxo)
		if info == nil {
			continue
		}

		if ticker == nil || common.IsPlainAsset(ticker) {
			if len(info.Assets) == 0 {
				result[utxoId] = info
			}
		} else {
			amt := info.GetAsset(ticker)
			if amt == 0 {
				continue
			}
			result[utxoId] = info
		}
	}

	return result, nil
}

// return: ticker -> amount
func (b *IndexerMgr) GetAssetSummaryInAddress(address string) map[common.TickerName]int64 {
	utxos, err := b.rpcService.GetUTXOs(address)
	if err != nil {
		return nil
	}

	result := make(map[common.TickerName]int64)
	nsAsset := b.GetSubNameSummaryWithAddress(address)
	for k, v := range nsAsset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_NS, Ticker: k}
		result[tickName] = v
	}

	nftAsset := b.GetNftAmountWithAddress(address)
	for k, v := range nftAsset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_NFT, Ticker: k}
		result[tickName] = v
	}

	ftAsset := b.ftIndexer.GetAssetSummaryByAddress(utxos)
	for k, v := range ftAsset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_FT, Ticker: k}
		result[tickName] = v
	}

	brc20Asset := b.brc20Indexer.GetAssetSummaryByAddress(b.rpcService.GetAddressId(address))
	for k, v := range brc20Asset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_BRC20, Type: common.ASSET_TYPE_FT, Ticker: k}
		tickInfo := b.brc20Indexer.GetTicker(k)
		if tickInfo != nil {
			result[tickName] = v.ToInt64WithMax(&tickInfo.Max)
		}
	}

	runesAsset := b.RunesIndexer.GetAddressAssets(b.rpcService.GetAddressId(address))
	for _, v := range runesAsset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_RUNES, Type: common.ASSET_TYPE_FT, Ticker: v.Rune}
		tickInfo := b.RunesIndexer.GetRuneInfoWithId(v.Rune)
		if tickInfo != nil {
			// TODO 搞不清铸造的量到底是哪个值，让runes索引器自己提供准确的值
			maxMinted := common.NewDecimalFromUint128(tickInfo.Supply, int(tickInfo.Divisibility))
			amt := common.NewDecimalFromUint128(v.Balance, int(tickInfo.Divisibility))
			result[tickName] = amt.ToInt64WithMax(maxMinted)
		}
	}

	plainUtxoMap := make(map[uint64]int64)
	for utxoId, v := range utxos {
		if b.ftIndexer.HasAssetInUtxo(utxoId) {
			continue
		}
		if b.RunesIndexer.IsExistAsset(utxoId) {
			continue
		}
		if b.nft.HasNftInUtxo(utxoId) {
			continue
		}
		plainUtxoMap[utxoId] = v
	}
	exAssets, plainUtxos := b.getExoticSummaryByAddress(plainUtxoMap)
	for k, v := range exAssets {
		// 如果该range有其他铸造出来的资产，过滤掉（直接使用utxoId过滤）
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_EXOTIC, Ticker: k}
		result[tickName] = v
	}

	var value int64
	for _, u := range plainUtxos {
		value += utxos[u]
	}
	if value != 0 {
		result[common.ASSET_PLAIN_SAT] = value
	}

	return result
}

// return: ticker -> []utxoId
func (b *IndexerMgr) GetAssetUTXOsInAddress(address string) map[common.TickerName][]uint64 {
	utxos, err := b.rpcService.GetUTXOs(address)
	if err != nil {
		return nil
	}

	result := make(map[common.TickerName][]uint64)

	ret := b.getExoticUtxos(utxos)
	for k, v := range ret {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_EXOTIC, Ticker: k}
		result[tickName] = append(result[tickName], v...)
	}

	for utxoId := range utxos {
		ids := b.GetNftsWithUtxo(utxoId)
		if len(ids) > 0 {
			tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_NFT, Ticker: ""}
			result[tickName] = append(result[tickName], utxoId)
		}

		names := b.GetNamesWithUtxo(utxoId)
		if len(names) > 0 {
			for _, name := range names {
				tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_NS, Ticker: name}
				result[tickName] = append(result[tickName], utxoId)
			}
		}
	}

	ret = b.ftIndexer.GetAssetUtxos(utxos)
	for k, v := range ret {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_FT, Ticker: k}
		result[tickName] = v
	}

	return result
}

func (b *IndexerMgr) U128ToInt64(runeId string, amt uint128.Uint128) int64 {
	info := b.RunesIndexer.GetRuneInfoWithId(runeId)
	if info == nil {
		return amt.Big().Int64()
	}
	
	return common.Uint128ToInt64(info.Supply, amt)
}


// return: ticker -> assets(amt)
func (b *IndexerMgr) GetUnbindAssetsWithUtxo(utxoId uint64) map[common.TickerName]int64 {
	result := make(map[common.TickerName]int64)
	
	runesAssets := b.RunesIndexer.GetUtxoAssets(utxoId)
	if len(runesAssets) > 0 {
		for _, v := range runesAssets {
			tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_RUNES, Type: common.ASSET_TYPE_FT, Ticker: v.Rune}
			result[tickName] = b.U128ToInt64(v.Rune, v.Balance)
		}
	}

	return result
}

// return: ticker -> assets(inscriptionId->Ranges)
func (b *IndexerMgr) GetAssetsWithUtxo(utxoId uint64) map[common.TickerName]map[string][]*common.Range {
	result := make(map[common.TickerName]map[string][]*common.Range)
	ftAssets := b.ftIndexer.GetAssetsWithUtxo(utxoId)
	if len(ftAssets) > 0 {
		for k, v := range ftAssets {
			tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_FT, Ticker: k}
			result[tickName] = v
		}
	}
	nfts := b.getNftsWithUtxo(utxoId)
	if len(nfts) > 0 {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_NFT, Ticker: ""}
		result[tickName] = nfts
	}
	names := b.getNamesWithUtxo(utxoId)
	if len(names) > 0 {
		for k, v := range names {
			tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_NS, Ticker: k}
			result[tickName] = v
		}
	}
	exo := b.getExoticsWithUtxo(utxoId)
	if len(exo) > 0 {
		for k, v := range exo {
			// 排除哪些已经被铸造成其他资产的稀有聪
			if b.ftIndexer.HasAssetInUtxo(utxoId) {
				continue
			}
			if b.nft.HasNftInUtxo(utxoId) {
				continue
			}
			tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_EXOTIC, Ticker: k}
			result[tickName] = v
		}
	}

	return result
}

func (b *IndexerMgr) GetAssetOffsetWithUtxo(utxo string) []*common.AssetOffsetRange {
	result := make([]*common.AssetOffsetRange, 0)

	utxoId, rngs, err := b.rpcService.GetOrdinalsWithUtxo(utxo)
	if err != nil {
		return nil
	}

	assetmap := b.GetAssetsWithUtxo(utxoId)

	// 白聪打底
	offset := int64(0)
	for _, rng := range rngs {
		result = append(result, &common.AssetOffsetRange{Range: rng, Offset: offset})
		offset += rng.Size
	}

	// 插入资产数据
	for ticker, assets := range assetmap {
		for _, assetRngs := range assets {
			for _, assetRng := range assetRngs {

				for i := 0; i < len(result); {
					rng := result[i]
					intersection := common.InterRange(rng.Range, assetRng)
					if intersection.Start < 0 {
						i++
						continue
					}

					// 分割rng，不处理超出rng的部分
					offset1 := int64(0) // 或者需要加上基数 rng.Offset
					offset2 := intersection.Start - rng.Range.Start
					offset3 := offset2 + intersection.Size
					offset4 := rng.Range.Size

					// 前不相交部分
					rng1 := rng.Clone()
					rng1.Range.Size = offset2 - offset1

					// 相交部分
					rng2 := rng.Clone()
					rng2.Range.Start = rng1.Range.Start + offset2
					rng2.Range.Size = offset3 - offset2
					rng2.Offset = offset2 + rng.Offset
					if rng2.Assets == nil {
						rng2.Assets = make([]*common.TickerName, 0)
					}
					rng2.Assets = append(rng2.Assets, &ticker)

					// 后不相交部分
					rng3 := rng.Clone()
					rng3.Range.Start = rng1.Range.Start + offset3
					rng3.Range.Size = offset4 - offset3
					rng3.Offset = offset3 + rng.Offset

					newResult := make([]*common.AssetOffsetRange, i)
					copy(newResult, result[0:i])
					j := i
					if rng1.Range.Size > 0 {
						newResult = append(newResult, rng1)
						j++
					}
					if rng2.Range.Size > 0 {
						newResult = append(newResult, rng2)
						j++
					}
					if rng3.Range.Size > 0 {
						newResult = append(newResult, rng3)
						j++
					}
					if i+1 < len(result) {
						newResult = append(newResult, result[i+1:]...)
					}
					i = j + 1
					result = newResult
				}
			}
		}
	}

	return result
}

// return: ticker -> assets(inscriptionId->Ranges)
func (b *IndexerMgr) GetAssetsWithRanges(ranges []*common.Range) map[string]map[string][]*common.Range {
	result := b.ftIndexer.GetAssetsWithRanges(ranges)
	if result == nil {
		result = make(map[string]map[string][]*common.Range)
	}
	ret := b.getNftsWithRanges(ranges)
	if len(ret) > 0 {
		result[common.ASSET_TYPE_NFT] = ret
	}
	ret = b.getNamesWithRanges(ranges)
	if len(ret) > 0 {
		result[common.ASSET_TYPE_NS] = ret
	}
	ret = b.exotic.GetExoticsWithRanges2(ranges)
	if len(ret) > 0 {
		result[common.ASSET_TYPE_EXOTIC] = ret
	}

	return result
}

func (b *IndexerMgr) GetMintHistory(tick string, start, limit int) []*common.MintAbbrInfo {
	switch tick {
	case common.ASSET_TYPE_NFT:
		r, _ := b.GetNftHistory(start, limit)
		return r
	case common.ASSET_TYPE_NS:
		return b.GetNameHistory(start, limit)
	case common.ASSET_TYPE_EXOTIC:
	default:

	}
	return b.ftIndexer.GetMintHistory(tick, start, limit)
}

func (b *IndexerMgr) GetMintHistoryWithAddress(address string, tick *common.TickerName,
	start, limit int) ([]*common.MintAbbrInfo, int) {
	addressId := b.GetAddressId(address)

	switch tick.Protocol {
	case common.PROTOCOL_NAME_ORDX:
		switch tick.Type {
		case common.ASSET_TYPE_FT:
			return b.ftIndexer.GetMintHistoryWithAddress(addressId, tick.Ticker, start, limit)
		case common.ASSET_TYPE_NFT:
			return b.GetNftHistoryWithAddress(addressId, start, limit)
		case common.ASSET_TYPE_NS:
			return b.GetNameHistoryWithAddress(addressId, start, limit)
		case common.ASSET_TYPE_EXOTIC:
			return nil, 0
		default:
		}

	case common.PROTOCOL_NAME_BRC20:
		//return b.brc20Indexer.GetMintHistoryWithAddress(addressId, tick.Ticker, start, limit)
	case common.PROTOCOL_NAME_RUNES:

	}

	return nil, 0

}

func (b *IndexerMgr) GetMintInfo(inscriptionId string) *common.Mint {
	nft := b.nft.GetNftWithInscriptionId(inscriptionId)
	if nft == nil {
		common.Log.Errorf("can't find ticker by %s", inscriptionId)
		return nil
	}

	switch nft.Base.TypeName {
	case common.ASSET_TYPE_NFT:
		return &common.Mint{
			Base:     nft.Base,
			Amt:      1,
			Ordinals: []*common.Range{{Start: nft.Base.Sat, Size: 1}},
		}
	case common.ASSET_TYPE_NS:
		return &common.Mint{
			Base:     nft.Base,
			Amt:      1,
			Ordinals: []*common.Range{{Start: nft.Base.Sat, Size: 1}},
		}
	}

	return b.ftIndexer.GetMint(inscriptionId)
}

func (b *IndexerMgr) GetNftWithInscriptionId(inscriptionId string) *common.Nft {
	return b.nft.GetNftWithInscriptionId(inscriptionId)
}

func (b *IndexerMgr) AddCollection(ntype, ticker string, ids []string) error {

	key := getCollectionKey(ntype, ticker)
	switch ntype {
	case common.ASSET_TYPE_NFT:
		err := common.GobSetDB1(key, ids, b.localDB)
		if err != nil {
			common.Log.Errorf("AddCollection %s %s failed: %v", ntype, ticker, err)
		} else {
			b.mutex.Lock()
			b.clmap[common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: ntype, Ticker: ticker}] = inscriptionIdsToCollectionMap(ids)
			b.mutex.Unlock()
		}
		return err
	case common.ASSET_TYPE_NS:
	}

	return fmt.Errorf("not support asset type %s", ntype)
}

func (b *IndexerMgr) GetCollection(ntype, ticker string, ids []string) ([]string, error) {

	key := getCollectionKey(ntype, ticker)
	value := make([]string, 0)
	switch ntype {
	case common.ASSET_TYPE_NFT:
		err := b.localDB.View(func(txn *badger.Txn) error {
			return common.GetValueFromDB(key, txn, value)
		})
		if err != nil {
			common.Log.Errorf("GetCollection %s %s failed: %v", ntype, ticker, err)
		}
		return value, err
	case common.ASSET_TYPE_NS:
	}

	return nil, fmt.Errorf("not support asset type %s", ntype)
}

func (b *IndexerMgr) GetTxOutputWithUtxo(utxo string) *common.TxOutput {
	info, err := b.rpcService.GetUtxoInfo(utxo)
	if err != nil {
		return nil
	}

	var assets common.TxAssets
	offsetmap := make(map[swire.AssetName]common.AssetOffsets)

	assetmap := b.GetAssetsWithUtxo(info.UtxoId)
	for k, v := range assetmap {
		value := int64(0)
		var offsets []*common.OffsetRange
		for _, rngs := range v {
			for _, rng := range rngs {
				start := common.GetSatOffset(info.Ordinals, rng.Start)
				offsets = append(offsets, &common.OffsetRange{Start: start, End: start+rng.Size})
				value += rng.Size
			}
		}

		sort.Slice(offsets, func(i, j int) bool {
			return offsets[i].Start < offsets[j].Start
		})

		asset := swire.AssetInfo{
			Name:       k,
			Amount:     value,
			BindingSat: 1,
		}

		if assets == nil {
			assets = swire.TxAssets{asset}
		} else {
			assets.Add(&asset)
		}

		offsetmap[k] = offsets
	}

	assetmap2 := b.GetUnbindAssetsWithUtxo(info.UtxoId)
	for k, v := range assetmap2 {
		asset := swire.AssetInfo{
			Name:       k,
			Amount:     v,
			BindingSat: 0,
		}
		
		if assets == nil {
			assets = swire.TxAssets{asset}
		} else {
			assets.Add(&asset)
		}
	}

	return &common.TxOutput{
		OutPointStr: utxo,
		OutValue: wire.TxOut{
			Value:    common.GetOrdinalsSize(info.Ordinals),
			PkScript: info.PkScript,
		},
		Assets: assets,
		Offsets: offsetmap,
	}
}

func (b *IndexerMgr) GetTickerInfo(tickerName *common.TickerName) *common.TickerInfo {
	var result *common.TickerInfo
	switch tickerName.Protocol {
	case common.PROTOCOL_NAME_ORDX:
		ticker := b.ftIndexer.GetTicker(tickerName.Ticker)
		if ticker != nil {
			result = &common.TickerInfo{}
			result.Protocol = common.PROTOCOL_NAME_ORDX
			result.Type = common.ASSET_TYPE_FT
			result.Name = ticker.Name
			result.Divisibility = 0
			result.NumLenght = 64
			result.MaxSupply = fmt.Sprintf("%d", ticker.Max)
			minted, _ := b.ftIndexer.GetMintAmount(tickerName.Ticker)
			result.MintedAmt = fmt.Sprintf("%d", minted)
		}
	case common.PROTOCOL_NAME_BRC20:
		brc20 := b.brc20Indexer.GetTicker(tickerName.Ticker)
		if brc20 != nil {
			result = &common.TickerInfo{}
			result.Protocol = common.PROTOCOL_NAME_BRC20
			result.Type = common.ASSET_TYPE_FT
			result.Name = brc20.Name
			result.Divisibility = int(brc20.Decimal)
			result.NumLenght = 128
			result.MaxSupply = brc20.Max.String()
			minted, _ := b.brc20Indexer.GetMintAmount(tickerName.Ticker)
			result.MintedAmt = minted.String()
		}
	case common.PROTOCOL_NAME_RUNES:
		rune := b.RunesIndexer.GetRuneInfoWithName(tickerName.Ticker)
		if rune != nil {
			result = &common.TickerInfo{}
			result.Protocol = common.PROTOCOL_NAME_RUNES
			result.Type = common.ASSET_TYPE_FT
			result.Name = rune.Name
			result.Divisibility = int(rune.Divisibility)
			result.NumLenght = 128
			result.MintedAmt = rune.Supply.String()
			if rune.MintInfo != nil {
				maxSupply := rune.MintInfo.Amount.Mul(rune.MintInfo.Cap)
				result.MintedAmt = maxSupply.String()
			}
		}
	}

	return result
}
