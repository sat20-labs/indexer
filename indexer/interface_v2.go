package indexer

import (
	"fmt"
	"sort"

	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"

	swire "github.com/sat20-labs/satsnet_btcd/wire"
)

// return: utxoId->asset
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

// return: utxoId->asset
func (b *IndexerMgr) GetAssetUTXOsInAddressWithTickV3(address string, ticker *swire.AssetName) (map[uint64]*common.AssetsInUtxo, error) {
	utxos, err := b.rpcService.GetUTXOs(address)
	if err != nil {
		return nil, err
	}

	result := make(map[uint64]*common.AssetsInUtxo)
	for utxoId := range utxos {
		utxo, err := b.rpcService.GetUtxoByID(utxoId)
		if err != nil {
			continue
		}
		info := b.GetTxOutputWithUtxoV3(utxo)
		if info == nil {
			continue
		}

		if ticker == nil || common.IsPlainAsset(ticker) {
			if len(info.Assets) == 0 {
				result[utxoId] = info
			}
		} else {
			for _, asset := range info.Assets {
				if asset.AssetName == *ticker {
					result[utxoId] = info
				}
			}
		}
	}

	return result, nil
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
				offsets = append(offsets, &common.OffsetRange{Start: start, End: start + rng.Size})
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

	assetmap2 := b.GetUnbindingAssetsWithUtxo(info.UtxoId)
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
		Assets:  assets,
		Offsets: offsetmap,
	}
}

func (b *IndexerMgr) GetTxOutputWithUtxoV3(utxo string) *common.AssetsInUtxo {
	info, err := b.rpcService.GetUtxoInfo(utxo)
	if err != nil {
		return nil
	}

	var assetsInUtxo common.AssetsInUtxo
	assetsInUtxo.OutPoint = utxo
	assetsInUtxo.Value = info.Value

	assetmap := b.GetAssetsWithUtxo(info.UtxoId)
	for k, v := range assetmap {
		value := int64(0)
		var offsets []*common.OffsetRange
		for _, rngs := range v {
			for _, rng := range rngs {
				start := common.GetSatOffset(info.Ordinals, rng.Start)
				offsets = append(offsets, &common.OffsetRange{Start: start, End: start + rng.Size})
				value += rng.Size
			}
		}

		sort.Slice(offsets, func(i, j int) bool {
			return offsets[i].Start < offsets[j].Start
		})

		asset := common.DisplayAsset{
			AssetName:  k,
			Amount:     fmt.Sprintf("%d", value),
			BindingSat: true,
			Offsets:    offsets,
		}

		assetsInUtxo.Assets = append(assetsInUtxo.Assets, &asset)
	}

	assetmap2 := b.GetUnbindingAssetsWithUtxoV2(info.UtxoId)
	for k, v := range assetmap2 {
		asset := common.DisplayAsset{
			AssetName:  k,
			Amount:     v.String(),
			BindingSat: false,
		}

		assetsInUtxo.Assets = append(assetsInUtxo.Assets, &asset)
	}

	return &assetsInUtxo
}

func (b *IndexerMgr) GetTickerInfo(tickerName *common.TickerName) *common.TickerInfo {
	var result *common.TickerInfo
	switch tickerName.Protocol {
	case common.PROTOCOL_NAME_ORDX:
		return b.GetTickerV2(tickerName.Ticker)
	case common.PROTOCOL_NAME_BRC20:
		return b.GetBRC20TickerV2(tickerName.Ticker)
	case common.PROTOCOL_NAME_RUNES:
		return b.GetRunesTickerV2(tickerName.Ticker)
	}

	return result
}

// return: ticker -> amount
func (b *IndexerMgr) GetAssetSummaryInAddressV2(address string) map[common.TickerName]int64 {
	utxos, err := b.rpcService.GetUTXOs(address)
	if err != nil {
		return nil
	}
	addressId := b.rpcService.GetAddressId(address)

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

	brc20Asset := b.brc20Indexer.GetAssetSummaryByAddress(addressId)
	for k, v := range brc20Asset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_BRC20, Type: common.ASSET_TYPE_FT, Ticker: k}
		ticker := b.brc20Indexer.GetTicker(k)
		if ticker != nil {
			result[tickName] = v.ToInt64WithMax(&ticker.Max)
		}
	}

	runesAsset := b.RunesIndexer.GetAddressAssets(addressId)
	for _, v := range runesAsset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_RUNES, Type: common.ASSET_TYPE_FT, Ticker: v.RuneId}
		ticker := b.RunesIndexer.GetRuneInfoWithName(v.Rune)
		if ticker != nil {
			result[tickName] = common.Uint128ToInt64(ticker.MaxSupply, v.Balance)
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

func (b *IndexerMgr) GetAssetSummaryInAddressV3(address string) map[common.TickerName]*common.Decimal {
	utxos, err := b.rpcService.GetUTXOs(address)
	if err != nil {
		return nil
	}

	result := make(map[common.TickerName]*common.Decimal)
	nsAsset := b.GetSubNameSummaryWithAddress(address)
	for k, v := range nsAsset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_NS, Ticker: k}
		result[tickName] = common.NewDecimal(v, 0)
	}

	nftAsset := b.GetNftAmountWithAddress(address)
	for k, v := range nftAsset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_NFT, Ticker: k}
		result[tickName] = common.NewDecimal(v, 0)
	}

	ftAsset := b.ftIndexer.GetAssetSummaryByAddress(utxos)
	for k, v := range ftAsset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_FT, Ticker: k}
		result[tickName] = common.NewDecimal(v, 0)
	}

	brc20Asset := b.brc20Indexer.GetAssetSummaryByAddress(b.rpcService.GetAddressId(address))
	for k, v := range brc20Asset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_BRC20, Type: common.ASSET_TYPE_FT, Ticker: k}
		result[tickName] = &v
	}

	runesAsset := b.RunesIndexer.GetAddressAssets(b.rpcService.GetAddressId(address))
	for _, v := range runesAsset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_RUNES, Type: common.ASSET_TYPE_FT, Ticker: v.Rune}
		result[tickName] = common.NewDecimalFromUint128(v.Balance, int(v.Divisibility))
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
		result[tickName] = common.NewDecimal(v, 0)
	}

	var value int64
	for _, u := range plainUtxos {
		value += utxos[u]
	}
	if value != 0 {
		result[common.ASSET_PLAIN_SAT] = common.NewDecimal(value, 0)
	}

	return result
}

// return: mint info sorted by inscribed time
func (b *IndexerMgr) GetMintHistoryWithAddressV2(address string,
	tick *common.TickerName, start, limit int) ([]*common.MintInfo, int) {

	addressId := b.GetAddressId(address)

	switch tick.Protocol {
	case common.PROTOCOL_NAME_ORDX:
		switch tick.Type {
		case common.ASSET_TYPE_FT:
			return b.ftIndexer.GetMintHistoryWithAddressV2(addressId, tick.Ticker, start, limit)
		case common.ASSET_TYPE_NFT:

		case common.ASSET_TYPE_NS:

		case common.ASSET_TYPE_EXOTIC:
			return nil, 0
		default:
		}

	case common.PROTOCOL_NAME_BRC20:
		return b.brc20Indexer.GetMintHistoryWithAddress(addressId, tick.Ticker, start, limit)
	case common.PROTOCOL_NAME_RUNES:
		return b.GetRunesMintHistoryWithAddress(addressId, tick.Ticker, start, limit)
	}

	return nil, 0
}

// return: ticker -> asset info (inscriptinId -> asset ranges)
func (b *IndexerMgr) GetAssetsWithUtxoV2(utxoId uint64) map[common.TickerName]*common.Decimal {
	result := make(map[common.TickerName]*common.Decimal)
	ftAssets := b.ftIndexer.GetAssetsWithUtxo(utxoId)
	if len(ftAssets) > 0 {
		for k, v := range ftAssets {
			tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_FT, Ticker: k}
			amt := int64(0)
			for _, rngs := range v {
				amt += common.GetOrdinalsSize(rngs)
			}
			result[tickName] = common.NewDecimal(amt, 0)
		}
	}
	runesAssets := b.RunesIndexer.GetUtxoAssets(utxoId)
	if len(runesAssets) > 0 {
		for _, v := range runesAssets {
			tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_RUNES, Type: common.ASSET_TYPE_FT, Ticker: v.Rune}
			result[tickName] = common.NewDecimalFromUint128(v.Balance, 0)
		}
	}
	nfts := b.getNftsWithUtxo(utxoId)
	if len(nfts) > 0 {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_NFT, Ticker: ""}
		result[tickName] = common.NewDecimal(int64(len(nfts)), 0)
	}
	names := b.getNamesWithUtxo(utxoId)
	if len(names) > 0 {
		for k, v := range names {
			tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_NS, Ticker: k}
			amt := int64(0)
			for _, rngs := range v {
				amt += common.GetOrdinalsSize(rngs)
			}
			result[tickName] = common.NewDecimal(amt, 0)
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
			amt := int64(0)
			for _, rngs := range v {
				amt += common.GetOrdinalsSize(rngs)
			}
			result[tickName] = common.NewDecimal(amt, 0)
		}
	}

	return result
}

// FT
// return: ticker's name -> ticker info
func (b *IndexerMgr) GetTickerMapV2(protocol string) map[string]*common.TickerInfo {
	switch protocol {
	case common.PROTOCOL_NAME_ORDX:
		return b.GetOrdxTickerMapV2()
	case common.PROTOCOL_NAME_BRC20:
		return b.GetBRC20TickerMapV2()
	case common.PROTOCOL_NAME_RUNES:
		return b.GetRunesTickerMapV2()
	}
	return nil
}

// return: addressId -> asset amount
func (b *IndexerMgr) GetHoldersWithTickV2(tickerName *common.TickerName) map[uint64]*common.Decimal {
	result := make(map[uint64]*common.Decimal)
	switch tickerName.Protocol {
	case common.PROTOCOL_NAME_ORDX:
		holders := b.ftIndexer.GetHolderAndAmountWithTick(tickerName.Ticker)
		for k, v := range holders {
			result[k] = common.NewDecimal(v, 0)
		}
	case common.PROTOCOL_NAME_BRC20:
		result = b.brc20Indexer.GetHoldersWithTick(tickerName.Ticker)
	case common.PROTOCOL_NAME_RUNES:
		result = b.RunesIndexer.GetHoldersWithTick(tickerName.Ticker)
	}

	return result
}

// return: asset amount, mint times
func (b *IndexerMgr) GetMintAmountV2(tickerName *common.TickerName) (*common.Decimal, int64) {
	switch tickerName.Protocol {
	case common.PROTOCOL_NAME_ORDX:
		amt, times := b.ftIndexer.GetMintAmount(tickerName.Ticker)
		return common.NewDecimal(amt, 0), times
	case common.PROTOCOL_NAME_BRC20:
		return b.brc20Indexer.GetMintAmount(tickerName.Ticker)
	case common.PROTOCOL_NAME_RUNES:
		return b.GetRunesMintAmount(tickerName.Ticker)
	}
	return nil, 0
}

// return:  mint info sorted by inscribed time
func (b *IndexerMgr) GetMintHistoryV2(tickerName *common.TickerName, start,
	limit int) []*common.MintInfo {
	result := make([]*common.MintInfo, 0)
	switch tickerName.Protocol {
	case common.PROTOCOL_NAME_ORDX:
		var ordxMintInfo []*common.MintAbbrInfo
		switch tickerName.Type {
		case common.ASSET_TYPE_NFT:
			ordxMintInfo, _ = b.GetNftHistory(start, limit)
		case common.ASSET_TYPE_NS:
			ordxMintInfo = b.GetNameHistory(start, limit)
		case common.ASSET_TYPE_EXOTIC:
		default:
			ordxMintInfo = b.ftIndexer.GetMintHistory(tickerName.Ticker, start, limit)
		}

		for _, info := range ordxMintInfo {
			m := info.ToMintInfo()
			m.Address = b.GetAddressById(info.Address)
			result = append(result, m)
		}
	case common.PROTOCOL_NAME_BRC20:
		brc20MintInfo := b.brc20Indexer.GetMintHistory(tickerName.Ticker, start, limit)
		for _, info := range brc20MintInfo {
			m := info.ToMintInfo()
			m.Address = b.GetAddressById(info.Address)
			result = append(result, m)
		}
	case common.PROTOCOL_NAME_RUNES:
		result, _ = b.GetRunesMintHistory(tickerName.Ticker, start, limit)
	}
	return result
}
