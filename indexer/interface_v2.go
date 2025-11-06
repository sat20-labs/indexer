package indexer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/sat20-labs/indexer/common"
)

// return: utxoId->asset
func (b *IndexerMgr) GetAssetUTXOsInAddressWithTickV3(address string, ticker *common.AssetName) (map[uint64]*common.AssetsInUtxo, error) {
	//t1 := time.Now()
	utxos, err := b.rpcService.GetUTXOs(address)
	if err != nil {
		return nil, err
	}
	// common.Log.Infof("GetUTXOs takes %v", time.Since(t1))
	// t1 = time.Now()

	result := make(map[uint64]*common.AssetsInUtxo)
	for utxoId := range utxos {
		utxo, err := b.rpcService.GetUtxoByID(utxoId)
		if err != nil {
			continue
		}
		// 过滤已经被花费的utxo
		if b.IsUtxoSpent(utxo) {
			continue
		}
		info := b.GetTxOutputWithUtxoV3(utxo, true)
		if info == nil {
			continue
		}

		if ticker == nil { // 返回所有
			result[utxoId] = info
		} else if common.IsPlainAsset(ticker) { // 只返回白聪
			if len(info.Assets) == 0 {
				result[utxoId] = info
			} else {
				// 如果都是nft，而且是被disable的，也算白聪
				hasOtherAsset := false
				for _, asset := range info.Assets {
					if asset.AssetName.Type != common.ASSET_TYPE_NFT {
						hasOtherAsset = true
						break
					}
				}
				if hasOtherAsset {
					continue
				}
				// 只有nft
				if b.nft.HasNftInUtxo(utxoId) {
					// 有其他没有被disabled的nft
					continue
				}
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
	//common.Log.Infof("populating takes %v", time.Since(t1))

	return result, nil
}

func (b *IndexerMgr) GetTxOutputWithUtxoV3(utxo string, excludingInvalid bool) *common.AssetsInUtxo {
	//t1 := time.Now()
	info, err := b.rpcService.GetUtxoInfo(utxo)
	//common.Log.Infof("rpcService.GetUtxoInfo takes %v", time.Since(t1))
	if err != nil {
		return nil
	}

	var assetsInUtxo common.AssetsInUtxo
	assetsInUtxo.UtxoId = info.UtxoId
	assetsInUtxo.OutPoint = utxo
	assetsInUtxo.Value = info.Value
	assetsInUtxo.PkScript = info.PkScript

	//t1 = time.Now()
	assetmap := b.GetAssetsWithUtxo(info.UtxoId)
	//common.Log.Infof("GetAssetsWithUtxo takes %v", time.Since(t1))
	//t1 = time.Now()
	for k, v := range assetmap {
		var offsets common.AssetOffsets
		value := int64(0)
		for _, rngs := range v {
			for _, rng := range rngs {
				start := common.GetSatOffset(info.Ordinals, rng.Start)
				offsets.Insert(&common.OffsetRange{Start: start, End: start + rng.Size})
				value += rng.Size
			}
		}

		n := 1
		if common.IsOrdxFT(&k) {
			ticker := b.GetTicker(k.Ticker)
			if ticker != nil {
				value = value * int64(ticker.N)
				n = ticker.N
			}
		}

		asset := common.DisplayAsset{
			AssetName:  k,
			Amount:     fmt.Sprintf("%d", value),
			Precision:  0,
			BindingSat: n,
			Offsets:    offsets,
		}

		assetsInUtxo.Assets = append(assetsInUtxo.Assets, &asset)
	}
	//common.Log.Infof("filling assetsInUtxo takes %v", time.Since(t1))

	assetmap2 := b.GetUnbindingAssetsWithUtxoV2(info.UtxoId)
	for k, v := range assetmap2 {
		if excludingInvalid && v.Invalid {
			continue
		}
		asset := common.DisplayAsset{
			AssetName:  k,
			Amount:     v.Amt.String(),
			Precision:  v.Amt.Precision,
			BindingSat: 0,
			Invalid:    v.Invalid,
		}
		if !v.Invalid && k.Protocol == common.PROTOCOL_NAME_BRC20 {
			asset.Offsets = []*common.OffsetRange{{Start:0, End:1}}
			asset.OffsetToAmts = []*common.OffsetToAmount{{Offset: 0, Amount: v.Amt.String()}}
		}

		assetsInUtxo.Assets = append(assetsInUtxo.Assets, &asset)
	}

	return &assetsInUtxo
}

func genBTCTicker() *common.TickerInfo {
	return &common.TickerInfo{
			AssetName:    common.ASSET_PLAIN_SAT,
			DisplayName:  "BTC",
			MaxSupply:    "21000000000000000", //  sats
			Divisibility: 0,
			N:            1,
		}
}

func (b *IndexerMgr) GetTickerInfo(tickerName *common.TickerName) *common.TickerInfo {
	var result *common.TickerInfo
	switch tickerName.Protocol {
	case common.PROTOCOL_NAME_ORDX:
		return b.GetTickerV2(tickerName.Ticker, tickerName.Type)
	case common.PROTOCOL_NAME_BRC20:
		return b.GetBRC20TickerV2(tickerName.Ticker)
	case common.PROTOCOL_NAME_RUNES:
		return b.GetRunesTickerV2(tickerName.Ticker)
	case "":
		if tickerName.Ticker == "" {
			result = genBTCTicker()
			result.AssetName = *tickerName
		}
	}

	return result
}

func (b *IndexerMgr) GetAssetSummaryInAddressV3(address string) map[common.TickerName]*common.Decimal {
	utxos, err := b.GetUTXOsWithAddress(address)
	if err != nil {
		return nil
	}

	result := make(map[common.TickerName]*common.Decimal)
	nsAsset := b.GetSubNameSummaryWithAddress(address) // TODO 已经广播的utxo没有过滤
	for k, v := range nsAsset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_NS, Ticker: k}
		result[tickName] = common.NewDefaultDecimal(v)
	}

	// 合集
	nftAsset := b.GetNftAmountWithAddress(address) // TODO 已经广播的utxo没有过滤
	for k, v := range nftAsset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_NFT, Ticker: k}
		result[tickName] = common.NewDefaultDecimal(v)
	}

	ftAsset := b.ftIndexer.GetAssetSummaryByAddress(utxos)
	for k, v := range ftAsset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_FT, Ticker: k}
		result[tickName] = common.NewDefaultDecimal(v)
	}

	brc20Asset := b.brc20Indexer.GetAssetSummaryByAddress(b.rpcService.GetAddressId(address))
	for k, v := range brc20Asset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_BRC20, Type: common.ASSET_TYPE_FT, Ticker: k}
		result[tickName] = v
	}

	runesAsset := b.RunesIndexer.GetAddressAssets(b.rpcService.GetAddressId(address), utxos) // TODO 已经广播的utxo没有过滤
	for _, v := range runesAsset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_RUNES, Type: common.ASSET_TYPE_FT, Ticker: v.Rune}
		result[tickName] = common.NewDecimalFromUint128(v.Balance, int(v.Divisibility))
	}

	totalSats := int64(0)
	plainUtxoMap := make(map[uint64]int64)
	for utxoId, v := range utxos {
		totalSats += v
		if b.HasAssetInUtxoId(utxoId, false) {
			continue
		}
		plainUtxoMap[utxoId] = v
	}
	result[common.ASSET_ALL_SAT] = common.NewDefaultDecimal(totalSats)

	exAssets, plainUtxos := b.getExoticSummaryByAddress(plainUtxoMap)
	for k, v := range exAssets {
		// 如果该range有其他铸造出来的资产，过滤掉（直接使用utxoId过滤）
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_EXOTIC, Ticker: k}
		result[tickName] = common.NewDefaultDecimal(v)
	}

	var value int64
	for _, u := range plainUtxos {
		value += utxos[u]
	}
	if value != 0 {
		result[common.ASSET_PLAIN_SAT] = common.NewDefaultDecimal(value)
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
		return b.brc20Indexer.GetMintHistoryWithAddressV2(addressId, tick.Ticker, start, limit)
	case common.PROTOCOL_NAME_RUNES:
		return b.GetRunesMintHistoryWithAddress(addressId, tick.Ticker, start, limit)
	}

	return nil, 0
}

// return: ticker -> asset info (inscriptinId -> asset ranges)
func (b *IndexerMgr) GetAssetsWithUtxoV2(utxoId uint64) map[common.TickerName]*common.Decimal {
	result := make(map[common.TickerName]*common.Decimal)
	ftAssets := b.ftIndexer.GetAssetsWithUtxoV2(utxoId)
	if len(ftAssets) > 0 {
		for k, v := range ftAssets {
			tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_FT, Ticker: k}
			result[tickName] = common.NewDefaultDecimal(v)
		}
	}
	runesAssets := b.RunesIndexer.GetUtxoAssets(utxoId)
	if len(runesAssets) > 0 {
		for _, v := range runesAssets {
			tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_RUNES, Type: common.ASSET_TYPE_FT, Ticker: v.Rune}
			result[tickName] = common.NewDecimalFromUint128(v.Balance, 0)
		}
	}
	brc20Asset := b.brc20Indexer.GetUtxoAssets(utxoId)
	if brc20Asset != nil && !brc20Asset.Invalid {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_BRC20, Type: common.ASSET_TYPE_FT, Ticker: brc20Asset.Name}
		result[tickName] = brc20Asset.Amt
	}
	nfts := b.getNftsWithUtxo(utxoId)
	if len(nfts) > 0 {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_NFT, Ticker: ""}
		result[tickName] = common.NewDefaultDecimal(int64(len(nfts)))
	}
	names := b.getNamesWithUtxo(utxoId)
	if len(names) > 0 {
		for k, v := range names {
			tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_NS, Ticker: k}
			amt := int64(0)
			for _, rngs := range v {
				amt += common.GetOrdinalsSize(rngs)
			}
			result[tickName] = common.NewDefaultDecimal(amt)
		}
	}
	exo := b.getExoticsWithUtxo(utxoId)
	if len(exo) > 0 {
		for k, v := range exo {
			if b.HasAssetInUtxoId(utxoId, true) {
				continue
			}
			tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_EXOTIC, Ticker: k}
			amt := int64(0)
			for _, rngs := range v {
				amt += common.GetOrdinalsSize(rngs)
			}
			result[tickName] = common.NewDefaultDecimal(amt)
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
			result[k] = common.NewDefaultDecimal(v)
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
		return common.NewDefaultDecimal(amt), times
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

func (b *IndexerMgr) GetBindingSat(tickerName *common.TickerName) int {
	if tickerName == nil {
		return 1
	}
	if tickerName.Protocol == common.PROTOCOL_NAME_ORDX {
		if tickerName.Type == common.ASSET_TYPE_FT {
			ticker := b.GetTicker(tickerName.Ticker)
			if ticker != nil {
				return ticker.N
			} else {
				return 1
			}
		} else {
			return 1
		}
	} else if tickerName.Protocol == "" {
		return 1
	}
	
	return 0
}



func (b *IndexerMgr) IsAllowDeploy(tickerName *common.TickerName) error {

	if tickerName.Type != common.ASSET_TYPE_FT {
		return fmt.Errorf("invalid asset type")
	}

	var err error
	switch tickerName.Protocol {
	case common.PROTOCOL_NAME_ORDX:
		if !common.IsValidSat20Name(tickerName.Ticker) {
			return fmt.Errorf("invalid ordx ticker name")
		}
		if b.ftIndexer.TickExisted(tickerName.Ticker) {
			err = fmt.Errorf("existing")
		}
	case common.PROTOCOL_NAME_BRC20:
		if len(tickerName.Ticker) != 4 || !common.IsValidName(tickerName.Ticker) {
			return fmt.Errorf("invalid brc20 ticker name")
		}
		if b.brc20Indexer.TickExisted(tickerName.Ticker) {
			err = fmt.Errorf("existing")
		}
	case common.PROTOCOL_NAME_RUNES:
		err = b.RunesIndexer.IsAllowEtching(tickerName.Ticker)
	}
	return err
}


func (b *IndexerMgr) IsUtxoSpent(utxo string) bool {
	return b.miniMempool.IsSpent(utxo)
}

// 某个用户将某个utxo中的所有铭文都解锁，不再生效，这个操作在该索引器中永久生效，但数据没上链
func (b *IndexerMgr) UnlockOrdinals(utxos []string, pubkey, sig []byte) (map[string]error, error) {

	jsonBytes, err := json.Marshal(utxos)
	if err != nil {
		return nil, err
	}

	err = common.VerifySignOfMessage(jsonBytes, sig, pubkey)
	if err != nil {
		common.Log.Errorf("verify signature of utxos %v failed, %v", utxos, err)
		return nil, err
	}

	addr, err := common.GetP2TRAddressFromPubkey(pubkey, b.GetChainParam())
	if err != nil {
		return nil, err
	}

	failed := make(map[string]error)
	for _, utxo := range utxos {
		// 确保目前该utxo没有被花费，并且在该地址下
		if b.IsUtxoSpent(utxo) {
			failed[utxo] = fmt.Errorf("spent")
			continue
		}
		info, err := b.rpcService.GetUtxoInfo(utxo)
		if err != nil {
			failed[utxo] = err
			continue
		}
		addr2, err := common.PkScriptToAddr(info.PkScript, b.GetChainParam())
		if err != nil {
			failed[utxo] = err
			continue
		}
		if addr != addr2 {
			failed[utxo] = fmt.Errorf("not owner")
			continue
		}

		buf := fmt.Sprintf("%s-%s-%s", utxo, hex.EncodeToString(pubkey), hex.EncodeToString(sig))
		err = b.nft.DisableNftsInUtxo(info.UtxoId, []byte(buf))
		if err != nil {
			failed[utxo] = err
		}
	}
	return failed, nil
}

// 获取哪些因为存在铭文而被锁定的utxo
func (b *IndexerMgr) GetLockedUTXOsInAddress(address string) ([]*common.AssetsInUtxo, error) {
	//t1 := time.Now()
	utxos, err := b.rpcService.GetUTXOs(address)
	if err != nil {
		return nil, err
	}
	// common.Log.Infof("GetUTXOs takes %v", time.Since(t1))
	// t1 = time.Now()

	result := make([]*common.AssetsInUtxo, 0)
	for utxoId := range utxos {
		utxo, err := b.rpcService.GetUtxoByID(utxoId)
		if err != nil {
			continue
		}
		if b.IsUtxoSpent(utxo) {
			continue
		}

		// 如果有其他资产存在，会优先识别为其他资产，而不是铭文
		if b.HasNameInUtxo(utxoId) {
			continue
		}
		if b.ftIndexer.HasAssetInUtxo(utxoId) {
			continue
		}
		if b.RunesIndexer.IsExistAsset(utxoId) {
			continue
		}
		if b.brc20Indexer.IsExistAsset(utxoId) {
			continue
		}
		_, rngs, err := b.GetOrdinalsWithUtxoId(utxoId)
		if err == nil {
			if b.exotic.HasExoticInRanges(rngs) {
				continue
			}
		}
		
		// 只剩下铭文的可能性
		if !b.nft.HasNftInUtxo(utxoId) {
			continue
		}
		info := b.GetTxOutputWithUtxoV3(utxo, true)
		if info == nil {
			continue
		}
		// 没有其他资产了，只有nft
		result = append(result, info)
	}
	//common.Log.Infof("populating takes %v", time.Since(t1))

	return result, nil
}
