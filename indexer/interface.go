package indexer

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/ordx/common"
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
	switch ticker.TypeName {
	case common.ASSET_TYPE_NFT:
		var inscmap map[string]int64

		if ticker.Name != common.ALL_TICKERS {
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
		if ticker.Name != common.ALL_TICKERS {
			bSpecialTicker = true
		}
		for utxoId := range utxos {
			names := b.GetNamesWithUtxo(utxoId)
			amount := 0
			if bSpecialTicker {
				for _, name := range names {
					_, subName := getSubName(name)
					if subName == ticker.Name {
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
		if ticker.Name != common.ALL_TICKERS {
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
					if name == ticker.Name {
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
		result = b.ftIndexer.GetAssetUtxosWithTicker(b.rpcService.GetAddressId(address), ticker.Name)
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
		tickName := common.TickerName{TypeName: common.ASSET_TYPE_NS, Name: k}
		result[tickName] = v
	}

	nftAsset := b.GetNftAmountWithAddress(address)
	for k, v := range nftAsset {
		tickName := common.TickerName{TypeName: common.ASSET_TYPE_NFT, Name: k}
		result[tickName] = v
	}

	ftAsset := b.ftIndexer.GetAssetSummaryByAddress(utxos)
	for k, v := range ftAsset {
		tickName := common.TickerName{TypeName: common.ASSET_TYPE_FT, Name: k}
		result[tickName] = v
	}

	plainUtxoMap := make(map[uint64]int64)
	for utxoId, v := range utxos {
		if b.ftIndexer.HasAssetInUtxo(utxoId) {
			continue
		}
		if b.nft.HasNftInUtxo(utxoId) {
			continue
		}
		plainUtxoMap[utxoId] = v
	}
	exAssets := b.getExoticSummaryByAddress(plainUtxoMap)
	for k, v := range exAssets {
		// 如果该range有其他铸造出来的资产，过滤掉（直接使用utxoId过滤）
		tickName := common.TickerName{TypeName: common.ASSET_TYPE_EXOTIC, Name: k}
		result[tickName] = v
	}

	return result
}

// return: ticker -> []utxoId
func (b *IndexerMgr) GetAssetUTXOsInAddress(address string) map[*common.TickerName][]uint64 {
	utxos, err := b.rpcService.GetUTXOs(address)
	if err != nil {
		return nil
	}

	result := make(map[*common.TickerName][]uint64)

	ret := b.getExoticUtxos(utxos)
	for k, v := range ret {
		tickName := &common.TickerName{TypeName: common.ASSET_TYPE_EXOTIC, Name: k}
		result[tickName] = append(result[tickName], v...)
	}

	for utxoId := range utxos {
		ids := b.GetNftsWithUtxo(utxoId)
		if len(ids) > 0 {
			tickName := &common.TickerName{TypeName: common.ASSET_TYPE_NFT, Name: ""}
			result[tickName] = append(result[tickName], utxoId)
		}

		names := b.GetNamesWithUtxo(utxoId)
		if len(names) > 0 {
			for _, name := range names {
				tickName := &common.TickerName{TypeName: common.ASSET_TYPE_NS, Name: name}
				result[tickName] = append(result[tickName], utxoId)
			}
		}
	}

	ret = b.ftIndexer.GetAssetUtxos(utxos)
	for k, v := range ret {
		tickName := &common.TickerName{TypeName: common.ASSET_TYPE_FT, Name: k}
		result[tickName] = v
	}

	return result
}

// return: ticker -> assets(inscriptionId->Ranges)
func (b *IndexerMgr) GetAssetsWithUtxo(utxoId uint64) map[*common.TickerName]map[string][]*common.Range {
	result := make(map[*common.TickerName]map[string][]*common.Range)
	ftAssets := b.ftIndexer.GetAssetsWithUtxo(utxoId)
	if len(ftAssets) > 0 {
		for k, v := range ftAssets {
			tickName := &common.TickerName{TypeName: common.ASSET_TYPE_FT, Name: k}
			result[tickName] = v
		}
	}
	nfts := b.getNftsWithUtxo(utxoId)
	if len(nfts) > 0 {
		tickName := &common.TickerName{TypeName: common.ASSET_TYPE_NFT, Name: ""}
		result[tickName] = nfts
	}
	names := b.getNamesWithUtxo(utxoId)
	if len(names) > 0 {
		for k, v := range names {
			tickName := &common.TickerName{TypeName: common.ASSET_TYPE_NS, Name: k}
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
			tickName := &common.TickerName{TypeName: common.ASSET_TYPE_EXOTIC, Name: k}
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

				var newResult []*common.AssetOffsetRange
				for i := 0; i < len(result); i++ {
					rng := result[i]
					intersection := common.InterRange(rng.Range, assetRng)
					if intersection.Start < 0 {
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
					rng2.Assets = append(rng2.Assets, ticker)

					// 后不相交部分
					rng3 := rng.Clone()
					rng3.Range.Start = rng1.Range.Start + offset3
					rng3.Range.Size = offset4 - offset3
					rng3.Offset = offset3 + rng.Offset

					newResult = result[:i]
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
					newResult = append(newResult, result[i:]...)
					i = j
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

func (b *IndexerMgr) GetMintHistoryWithAddress(address string, tick *common.TickerName, start, limit int) ([]*common.MintAbbrInfo, int) {
	addressId := b.GetAddressId(address)
	switch tick.TypeName {
	case common.ASSET_TYPE_NFT:
		return b.GetNftHistoryWithAddress(addressId, start, limit)
	case common.ASSET_TYPE_NS:
		return b.GetNameHistoryWithAddress(addressId, start, limit)
	case common.ASSET_TYPE_EXOTIC:
		return nil, 0
	default:

	}
	return b.ftIndexer.GetMintHistoryWithAddress(addressId, tick.Name, start, limit)
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
			b.clmap[common.TickerName{TypeName: ntype, Name: ticker}] = inscriptionIdsToCollectionMap(ids)
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
