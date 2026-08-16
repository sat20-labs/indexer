package indexer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/sat20-labs/indexer/common"
	atomidx "github.com/sat20-labs/indexer/indexer/atom"
)

func (b *IndexerMgr) containAsset(output *common.TxOutput, ticker *common.AssetName) bool {
	if ticker == nil {
		return true
	} else if common.IsPlainAsset(ticker) {
		if len(output.Assets) == 0 {
			return true
		}
		// 如果都是nft，而且是被disable的，也算白聪
		hasOtherAsset := false
		for _, asset := range output.Assets {
			if asset.Name.Type != common.ASSET_TYPE_NFT {
				hasOtherAsset = true
				break
			}
		}
		if hasOtherAsset {
			return false
		}
		// 只有nft
		if b.nft.HasNftInUtxo(output.UtxoId) {
			// 有其他没有被disabled的nft
			return false
		}
		return true
	} else {
		for _, asset := range output.Assets {
			if asset.Name == *ticker {
				return true
			}
		}
	}
	return false
}

func (b *IndexerMgr) hasAssetInUtxoIdNoRPC(utxoId uint64, excludingExotic bool) bool {
	if b.nft.HasNftInUtxo(utxoId) {
		return true
	}
	if b.ns.HasNamesInUtxo(utxoId) {
		return true
	}
	if b.ftIndexer.HasAssetInUtxo(utxoId) {
		return true
	}
	if b.RunesIndexer.IsExistAsset(utxoId) {
		return true
	}
	if b.brc20Indexer.IsExistAsset(utxoId) {
		return true
	}
	if b.atomIndexer.HasAssetInUtxo(utxoId) {
		return true
	}
	if !excludingExotic && b.exotic.HasExoticInUtxo(utxoId) {
		return true
	}
	return false
}

func removeConfirmedPlainDuplicates(unconfirmed map[string]*common.TxOutput, confirmed map[string]struct{}) {
	for outpoint := range unconfirmed {
		if _, ok := confirmed[outpoint]; ok {
			delete(unconfirmed, outpoint)
		}
	}
}

// GetAssetUTXOsInAddressWithTickV3 returns confirmed UTXOs plus, for plain sats
// only, unconfirmed outputs that MiniMemPool has fully classified as asset-free.
func (b *IndexerMgr) GetAssetUTXOsInAddressWithTickV3(address string, ticker *common.AssetName, includeInvalid bool) ([]*common.AssetsInUtxo, error) {
	b.rpcEnter()
	defer b.rpcLeft()

	excludingInvalid := !includeInvalid
	utxos, err := b.GetUTXOsWithAddress(address)
	if err != nil {
		return nil, err
	}

	mid := make([]*common.TxOutput, 0)
	confirmedOutpoints := make(map[string]struct{}, len(utxos))
	for utxoId := range utxos {
		utxo, err := b.rpcService.GetUtxoByID(utxoId)
		if err != nil {
			continue
		}
		info := b.getTxOutputWithUtxoV2(utxo, excludingInvalid)
		if info == nil {
			continue
		}
		confirmedOutpoints[info.OutPointStr] = struct{}{}
		if b.containAsset(info, ticker) {
			mid = append(mid, info)
		}
	}

	// Unconfirmed asset outputs are intentionally not reconstructed. Only
	// outputs proven plain by MiniMemPool are safe to expose for immediate use.
	if common.IsPlainAsset(ticker) {
		for outpoint, info := range b.miniMempool.GetUnconfirmedPlainUtxoByAddress(address) {
			if _, confirmed := confirmedOutpoints[outpoint]; confirmed {
				continue
			}
			mid = append(mid, info)
		}
	}

	sort.Slice(mid, func(i, j int) bool {
		if common.IsPlainAsset(ticker) {
			return mid[i].OutValue.Value > mid[j].OutValue.Value
		}
		a := mid[i].GetAsset(ticker)
		b := mid[j].GetAsset(ticker)
		return a.Cmp(b) > 0
	})

	result := make([]*common.AssetsInUtxo, len(mid))
	for i, v := range mid {
		result[i] = v.ToAssetsInUtxo()
	}
	return result, nil
}

func (b *IndexerMgr) GetTxOutputWithUtxoV2(utxo string, excludingInvalid bool) *common.TxOutput {
	b.rpcEnter()
	defer b.rpcLeft()
	return b.getTxOutputWithUtxoV2(utxo, excludingInvalid)
}

func (b *IndexerMgr) getTxOutputWithUtxoV2(utxo string, excludingInvalid bool) *common.TxOutput {
	info, err := b.rpcService.GetUtxoInfo(utxo)
	if err != nil {
		return nil
	}

	output := common.NewTxOutput(0)
	output.UtxoId = info.UtxoId
	output.OutPointStr = utxo
	output.OutValue.Value = info.Value
	output.OutValue.PkScript = info.PkScript

	assetmap := b.GetAssetsWithUtxo(info.UtxoId)
	assetmap2 := b.GetUnbindingAssetsWithUtxoV2(info.UtxoId)
	builder := common.NewTxAssetsBuilder(len(assetmap) + len(assetmap2))
	for k, v := range assetmap {
		offsets := v
		value := v.Size()

		n := 1
		if common.IsOrdxFT(&k) {
			ticker := b.GetTicker(k.Ticker)
			if ticker != nil {
				value = value * int64(ticker.N)
				n = ticker.N
			}
		}

		asset := common.AssetInfo{
			Name:       k,
			Amount:     *common.NewDefaultDecimal(value),
			BindingSat: uint32(n),
		}
		builder.Add(&asset)
		output.Offsets[k] = offsets
	}

	for k, v := range assetmap2 {
		if excludingInvalid && v.Invalid {
			continue
		}
		asset := common.AssetInfo{
			Name:       k,
			Amount:     *v.Amt.Clone(),
			BindingSat: 0,
		}
		if k.Protocol == common.PROTOCOL_NAME_BRC20 {
			output.Offsets[k] = []*common.OffsetRange{{Start: 0, End: 1}}
			output.SatBindingMap[0] = asset.Clone()
		}
		if v.Invalid {
			output.Invalids[k] = v.Invalid
		}
		builder.Add(&asset)
	}
	output.Assets = builder.Build()
	return output
}

func (b *IndexerMgr) GetTxOutputWithUtxoV3(utxo string, excludingInvalid bool) *common.AssetsInUtxo {
	b.rpcEnter()
	defer b.rpcLeft()
	return b.getTxOutputWithUtxoV3(utxo, excludingInvalid)
}

func (b *IndexerMgr) getTxOutputWithUtxoV3(utxo string, excludingInvalid bool) *common.AssetsInUtxo {
	output := b.getTxOutputWithUtxoV2(utxo, excludingInvalid)
	if output == nil {
		return nil
	}
	return output.ToAssetsInUtxo()
}

func genBTCTicker() *common.TickerInfo {
	return &common.TickerInfo{
		AssetName:    common.ASSET_PLAIN_SAT,
		DisplayName:  "BTC",
		MaxSupply:    "21000000000000000",
		Divisibility: 0,
		N:            1,
	}
}

func (b *IndexerMgr) GetTickerInfo(tickerName *common.TickerName) *common.TickerInfo {
	b.rpcEnter()
	defer b.rpcLeft()

	var result *common.TickerInfo
	switch tickerName.Protocol {
	case common.PROTOCOL_NAME_ORDX:
		return b.GetTickerV2(tickerName.Ticker, tickerName.Type)
	case common.PROTOCOL_NAME_BRC20:
		return b.GetBRC20TickerV2(tickerName.Ticker)
	case common.PROTOCOL_NAME_RUNES:
		return b.GetRunesTickerV2(tickerName.Ticker)
	case common.PROTOCOL_NAME_ATOM:
		return b.GetAtomTickerV2(tickerName.Ticker)
	case "":
		if tickerName.Ticker == "" {
			result = genBTCTicker()
			result.AssetName = *tickerName
		}
	}
	return result
}

// GetAssetSummaryInAddressV3 includes currently available unconfirmed plain
// outputs in ALL_SAT and PLAIN_SAT. Unconfirmed asset balances remain excluded.
func (b *IndexerMgr) GetAssetSummaryInAddressV3(address string) map[common.TickerName]*common.Decimal {
	b.rpcEnter()
	defer b.rpcLeft()

	utxos, err := b.GetUTXOsWithAddress(address)
	if err != nil {
		return nil
	}
	unconfirmedSpents := b.miniMempool.GetUnconfirmedSpentUtxoByAddress(address)
	unconfirmedPlain := b.miniMempool.GetUnconfirmedPlainUtxoByAddress(address)
	confirmedOutpoints := make(map[string]struct{}, len(utxos))
	for utxoID := range utxos {
		if outpoint, err := b.rpcService.GetUtxoByID(utxoID); err == nil {
			confirmedOutpoints[outpoint] = struct{}{}
		}
	}
	removeConfirmedPlainDuplicates(unconfirmedPlain, confirmedOutpoints)

	result := make(map[common.TickerName]*common.Decimal)
	nsAsset := b.getSubNameSummaryWithAddress(address, unconfirmedSpents)
	for k, v := range nsAsset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_NS, Ticker: k}
		result[tickName] = common.NewDefaultDecimal(v)
	}

	nftAsset := b.getNftAmountWithAddress(address, unconfirmedSpents)
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
	for _, output := range unconfirmedSpents {
		if len(output.Assets) == 0 {
			continue
		}
		for k, v := range brc20Asset {
			tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_BRC20, Type: common.ASSET_TYPE_FT, Ticker: k}
			amt := output.GetAsset(&tickName)
			if amt != nil && amt.Sign() != 0 {
				d := common.DecimalSub(v, amt)
				if d.Sign() < 0 {
					d.SetValue(0)
				}
				v.Value = d.Value
			}
		}
	}
	for k, v := range brc20Asset {
		if v.IsZero() {
			continue
		}
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_BRC20, Type: common.ASSET_TYPE_FT, Ticker: k}
		result[tickName] = v
	}

	runesAsset := b.RunesIndexer.GetAddressAssets(b.rpcService.GetAddressId(address), utxos)
	for _, v := range runesAsset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_RUNES, Type: common.ASSET_TYPE_FT, Ticker: v.Rune}
		result[tickName] = common.NewDecimalFromUint128(v.Balance, int(v.Divisibility))
	}

	atomAsset := b.atomIndexer.GetAssetSummaryByAddress(utxos)
	for k, v := range atomAsset {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ATOM, Type: common.ASSET_TYPE_FT, Ticker: k}
		result[tickName] = common.NewDefaultDecimal(v)
	}

	totalSats := int64(0)
	plainUtxoMap := make(map[uint64]int64)
	for utxoId, v := range utxos {
		totalSats += v
		if b.hasAssetInUtxoIdNoRPC(utxoId, false) {
			continue
		}
		plainUtxoMap[utxoId] = v
	}
	for _, output := range unconfirmedPlain {
		totalSats += output.Value()
	}
	result[common.ASSET_ALL_SAT] = common.NewDefaultDecimal(totalSats)

	exAssets, plainUtxos := b.getExoticSummaryByAddress(plainUtxoMap)
	for k, v := range exAssets {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_EXOTIC, Ticker: k}
		result[tickName] = common.NewDefaultDecimal(v)
	}

	var value int64
	for _, u := range plainUtxos {
		value += utxos[u]
	}
	for _, output := range unconfirmedPlain {
		value += output.Value()
	}
	if value != 0 {
		result[common.ASSET_PLAIN_SAT] = common.NewDefaultDecimal(value)
	}
	return result
}

func (b *IndexerMgr) GetMintHistoryWithAddressV2(address string,
	tick *common.TickerName, start, limit int) ([]*common.MintInfo, int) {
	b.rpcEnter()
	defer b.rpcLeft()

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
		}
	case common.PROTOCOL_NAME_BRC20:
		return b.brc20Indexer.GetMintHistoryWithAddressV2(addressId, tick.Ticker, start, limit)
	case common.PROTOCOL_NAME_RUNES:
		return b.GetRunesMintHistoryWithAddress(addressId, tick.Ticker, start, limit)
	case common.PROTOCOL_NAME_ATOM:
		return b.GetAtomMintHistoryWithAddress(addressId, tick.Ticker, start, limit)
	}
	return nil, 0
}

func (b *IndexerMgr) GetAssetsWithUtxoV2(utxoId uint64) map[common.TickerName]*common.Decimal {
	b.rpcEnter()
	defer b.rpcLeft()

	result := make(map[common.TickerName]*common.Decimal)
	ftAssets := b.ftIndexer.GetAssetsWithUtxoV2(utxoId)
	for k, v := range ftAssets {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_FT, Ticker: k}
		result[tickName] = common.NewDefaultDecimal(v)
	}
	runesAssets := b.RunesIndexer.GetUtxoAssets(utxoId)
	for _, v := range runesAssets {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_RUNES, Type: common.ASSET_TYPE_FT, Ticker: v.Rune}
		result[tickName] = common.NewDecimalFromUint128(v.Balance, 0)
	}
	brc20Asset := b.brc20Indexer.GetUtxoAssets(utxoId)
	if brc20Asset != nil && !brc20Asset.Invalid {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_BRC20, Type: common.ASSET_TYPE_FT, Ticker: brc20Asset.Name}
		result[tickName] = brc20Asset.Amt
	}
	atomAssets := b.atomIndexer.GetUtxoAssets(utxoId)
	for ticker, amount := range atomAssets {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ATOM, Type: common.ASSET_TYPE_FT, Ticker: ticker}
		result[tickName] = common.NewDefaultDecimal(amount)
	}
	nfts := b.getNftsWithUtxo(utxoId)
	for k, v := range nfts {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_NFT, Ticker: k}
		result[tickName] = common.NewDefaultDecimal(v.Size())
	}
	names := b.getNamesWithUtxo(utxoId)
	for k, v := range names {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_NS, Ticker: k}
		result[tickName] = common.NewDefaultDecimal(v.Size())
	}
	exo := b.getExoticsWithUtxo(utxoId)
	for k, v := range exo {
		tickName := common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: common.ASSET_TYPE_EXOTIC, Ticker: k}
		result[tickName] = common.NewDefaultDecimal(v.Size())
	}
	return result
}

func (b *IndexerMgr) GetTickerMapV2(protocol string, start, limit int) ([]string, int) {
	b.rpcEnter()
	defer b.rpcLeft()
	if start < 0 {
		start = 0
	}
	switch protocol {
	case common.PROTOCOL_NAME_ORDX:
		return b.GetOrdxTickerMapV2(start, limit)
	case common.PROTOCOL_NAME_BRC20:
		return b.GetBRC20TickerMapV2(start, limit)
	case common.PROTOCOL_NAME_RUNES:
		return b.GetRunesTickerMapV2(start, limit)
	case common.PROTOCOL_NAME_ATOM:
		return b.GetAtomTickerMapV2(start, limit)
	}
	return nil, 0
}

func (b *IndexerMgr) GetHoldersWithTickV2(tickerName *common.TickerName) map[uint64]*common.Decimal {
	b.rpcEnter()
	defer b.rpcLeft()
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
	case common.PROTOCOL_NAME_ATOM:
		result = b.atomIndexer.GetHoldersWithTick(tickerName.Ticker)
	}
	return result
}

func (b *IndexerMgr) GetMintAmountV2(tickerName *common.TickerName) (*common.Decimal, int64) {
	b.rpcEnter()
	defer b.rpcLeft()
	switch tickerName.Protocol {
	case common.PROTOCOL_NAME_ORDX:
		amt, times := b.ftIndexer.GetMintAmount(tickerName.Ticker)
		return common.NewDefaultDecimal(amt), times
	case common.PROTOCOL_NAME_BRC20:
		return b.brc20Indexer.GetMintAmount(tickerName.Ticker)
	case common.PROTOCOL_NAME_RUNES:
		return b.GetRunesMintAmount(tickerName.Ticker)
	case common.PROTOCOL_NAME_ATOM:
		return b.atomIndexer.GetMintAmount(tickerName.Ticker)
	}
	return nil, 0
}

func (b *IndexerMgr) GetMintHistoryV2(tickerName *common.TickerName, start,
	limit int) []*common.MintInfo {
	b.rpcEnter()
	defer b.rpcLeft()

	result := make([]*common.MintInfo, 0)
	switch tickerName.Protocol {
	case common.PROTOCOL_NAME_ORDX:
		var ordxMintInfo []*common.MintAbbrInfo
		switch tickerName.Type {
		case common.ASSET_TYPE_NFT:
			ordxMintInfo, _ = b.GetNftHistory(start, limit)
		case common.ASSET_TYPE_NS:
			ordxMintInfo = b.getNameHistory(start, limit)
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
	case common.PROTOCOL_NAME_ATOM:
		result = b.atomIndexer.GetMintHistory(tickerName.Ticker, start, limit)
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
			}
			return 1
		}
		return 1
	} else if tickerName.Protocol == "" {
		return 1
	} else if tickerName.Protocol == common.PROTOCOL_NAME_ATOM {
		return 1
	}
	return 0
}

func (b *IndexerMgr) IsAllowDeploy(tickerName *common.TickerName) error {
	b.rpcEnter()
	defer b.rpcLeft()
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
		if len(tickerName.Ticker) != 4 && len(tickerName.Ticker) != 5 {
			return fmt.Errorf("only support ticker length = 4 or 5 bytes")
		}
		if !common.IsValidName(tickerName.Ticker) {
			return fmt.Errorf("invalid brc20 ticker name")
		}
		if b.brc20Indexer.TickExisted(tickerName.Ticker) {
			err = fmt.Errorf("existing")
		}
	case common.PROTOCOL_NAME_RUNES:
		err = b.RunesIndexer.IsAllowEtching(tickerName.Ticker)
	case common.PROTOCOL_NAME_ATOM:
		if !atomidx.IsValidTicker(tickerName.Ticker) {
			return fmt.Errorf("invalid atom ticker name")
		}
		if !b.atomIndexer.TickExisted(tickerName.Ticker) {
			err = nil
		} else {
			err = fmt.Errorf("existing")
		}
	}
	return err
}

func (b *IndexerMgr) IsUtxoSpent(utxo string) bool {
	return b.miniMempool.IsSpent(utxo)
}

func (b *IndexerMgr) UnlockOrdinals(utxos []string, pubkey, sig []byte) (map[string]error, error) {
	b.rpcEnter()
	defer b.rpcLeft()
	jsonBytes, err := json.Marshal(utxos)
	if err != nil {
		return nil, err
	}
	if err = common.VerifySignOfMessage(jsonBytes, sig, pubkey); err != nil {
		common.Log.Errorf("verify signature of utxos %v failed, %v", utxos, err)
		return nil, err
	}
	addr, err := common.GetP2TRAddressFromPubkey(pubkey, b.GetChainParam())
	if err != nil {
		return nil, err
	}
	failed := make(map[string]error)
	for _, utxo := range utxos {
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
		if err = b.nft.DisableNftsInUtxo(info.UtxoId, []byte(buf)); err != nil {
			failed[utxo] = err
		}
	}
	return failed, nil
}

func (b *IndexerMgr) GetLockedUTXOsInAddress(address string) ([]*common.AssetsInUtxo, error) {
	b.rpcEnter()
	defer b.rpcLeft()
	utxos, err := b.GetUTXOsWithAddress(address)
	if err != nil {
		return nil, err
	}
	result := make([]*common.AssetsInUtxo, 0)
	for utxoId := range utxos {
		utxo, err := b.rpcService.GetUtxoByID(utxoId)
		if err != nil {
			continue
		}
		if b.ns.HasNamesInUtxo(utxoId) ||
			b.ftIndexer.HasAssetInUtxo(utxoId) ||
			b.RunesIndexer.IsExistAsset(utxoId) ||
			b.brc20Indexer.IsExistAsset(utxoId) ||
			b.atomIndexer.HasAssetInUtxo(utxoId) ||
			b.exotic.HasExoticInUtxo(utxoId) {
			continue
		}
		if !b.nft.HasNftInUtxo(utxoId) {
			continue
		}
		info := b.getTxOutputWithUtxoV3(utxo, true)
		if info != nil {
			result = append(result, info)
		}
	}
	return result, nil
}
