package indexer

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
	indexer "github.com/sat20-labs/indexer/indexer/common"
	"github.com/sat20-labs/indexer/indexer/ns"
	"github.com/sat20-labs/indexer/indexer/ord"
	"github.com/sat20-labs/indexer/indexer/ord/ord0_14_1"
	ordCommon "github.com/sat20-labs/indexer/indexer/ord/common"
)

func (s *IndexerMgr) processOrdProtocol(block *common.Block, coinbase []*common.Range) {
	s.exotic.UpdateTransfer(block, coinbase) // 生成稀有资产，为ordx协议做准备

	if block.Height < s.ordFirstHeight {
		return
	}

	//detectOrdMap := make(map[string]int, 0)
	measureStartTime := time.Now()
	//common.Log.Info("processOrdProtocol ...")
	count := 0
	for _, tx := range block.Transactions {
		id := 0
		for i, input := range tx.Inputs {

			// if tx.TxId == "fbfa79f18e1132adf767a03f64776e412c13229693a5e6af55f8835f101b7615" {
			// 	common.Log.Info("")
			// }

		
			inscriptions2 := ord0_14_1.GetInscriptionsInTxInput(input.Witness, block.Height, i)
			for _, insc := range inscriptions2 {
				s.handleOrd(input, insc, id, tx, block, coinbase) // 尽可能只缓存，不读数据库
				id++
				count++
				if insc.IsCursed {
					if insc.CurseReason != ordCommon.NotAtOffsetZero && 
					insc.CurseReason != ordCommon.Pointer && 
					insc.CurseReason != ordCommon.Pushnum &&  // testnet4: 809cd75a7525b47d49782316cda02dffff83d93702902b037215b4010619dbdei0
					insc.CurseReason != ordCommon.NotInFirstInput && // testnet4: bb5bf322a4cd7117f8b46156705748ba485477a5f9bc306559943ec98147017b
 					insc.CurseReason != ordCommon.UnrecognizedEvenField && // testnet4: b37170d58cac08c65b82d3df9f096bfc2735787fd61b8731a3a57966e136ace8i0 025245e9010c68646a5240115b705381df06bd94730e5d894632771d214a263ci0 6dd8d2b5f1753bc6ea3d193c707b74b6452e1fb38e55fd654544ff5de65203e7i0
					insc.CurseReason != ordCommon.IncompleteField {// testnet4: 55b0a3b554ec73a1a5d9194bef921e9d25b9e729dcd7ad21deb6e68817d620d3i0 b23b28002527ddad7b850ac4544fb7f74b012f109e498f3d4d06096f7d366da4i0
						common.Log.Errorf("%si%d is cursed, reason %d", tx.TxId, id, insc.CurseReason)
					}
				}
			}
		}
	}
	common.Log.Infof("processOrdProtocol loop %d finished. cost: %v", count, time.Since(measureStartTime))
	common.Log.Infof("height: %d, total cursed: %d", block.Height, s.nft.GetStatus().CurseCount)

	//time2 := time.Now()
	s.nft.UpdateTransfer(block, coinbase)
	s.ns.UpdateTransfer(block)
	s.brc20Indexer.UpdateTransfer(block)
	s.RunesIndexer.UpdateTransfer(block)

	s.ftIndexer.UpdateTransfer(block, coinbase) // 依赖前面生成的稀有资产

	//common.Log.Infof("processOrdProtocol UpdateTransfer finished. cost: %v", time.Since(time2))

	common.Log.Infof("processOrdProtocol %d is done, cost: %v", block.Height, time.Since(measureStartTime))
}

func findOutputWithSatPoint(block *common.Block, coinbase []*common.Range,
	index int, tx *common.Transaction, satpoint int64) (*common.TxOutputV2, int64) {
	var outValue int64
	for _, txOut := range tx.Outputs {
		if outValue+txOut.OutValue.Value >= int64(satpoint) {
			return txOut, satpoint
		}
		outValue += txOut.OutValue.Value
	}
	if satpoint > 0 {
		// 如果satpoint大于0，但是不在输出中，就在外面修改satpoint的值，同时直接定位为0
		return tx.Outputs[0], 0 // 遵循ordinals协议的规则
	}

	// 如果satpoint == 0，聪输出在奖励区块中
	// 4bee6242e4ef88e632b7061686ee60f9a0000c85071263ccb44a8aeb83c5072f
	
	// 作为网络费用给到了矿工，位置在手续费的0位置
	var baseOffset int64
	for i := 0; i < index; i++ { // 0 是奖励聪，跳过前面index-1个交易的手续费，
		baseOffset += coinbase[i].Size
	}

	coinbaseTx := block.Transactions[0]
	outValue = 0
	for _, txOut := range coinbaseTx.Outputs {
		if outValue+txOut.OutValue.Value >= baseOffset {
			return txOut, 0
		}
		outValue += txOut.OutValue.Value
	}
	// 没有绑定聪的铭文
	return tx.Outputs[0], 0
}

func (s *IndexerMgr) handleDeployTicker(satpoint int64, in *common.TxInput, out *common.TxOutputV2,
	content *common.OrdxDeployContent, nft *common.Nft) *common.Ticker {
	height := nft.Base.BlockHeight
	// 去掉这个限制
	// if len(content.Ticker) == 4 {
	// 	common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid ticker",
	// 		nft.Base.InscriptionId, content.Ticker)
	// 	return nil
	// }
	if !common.IsValidSat20Name(content.Ticker) {
		if !s.isLptTicker(content.Ticker) {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid ticker",
				nft.Base.InscriptionId, content.Ticker)
			return nil
		}

		// 目前只允许持有足够的pearl的用户可以部署lpt
		if !s.isEligibleUser(out.OutValue.PkScript, content.Des) {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, not eligible user",
				nft.Base.InscriptionId, content.Ticker)
			return nil
		}
	}

	// 名字不跟ticker挂钩
	var reg *ns.NameRegister
	if !common.TickerSeparatedFromName {
		addressId := nft.OwnerAddressId
		reg = s.ns.GetNameRegisterInfo(content.Ticker)
		if reg != nil && s.isSat20Actived(int(height)) {
			if reg.Nft.OwnerAddressId != addressId {
				common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s has owner %d",
					nft.Base.InscriptionId, content.Ticker, reg.Nft.OwnerAddressId)
				return nil
			}
		}
	}

	var err error
	lim := int64(1)
	if content.Lim != "" {
		lim, err = strconv.ParseInt(content.Lim, 10, 64)
		if err != nil {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid lim: %s",
				nft.Base.InscriptionId, content.Ticker, content.Lim)
			return nil
		}
		if lim < 0 {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid lim: %d",
				nft.Base.InscriptionId, content.Ticker, lim)
			return nil
		}
	}

	n := int(1)
	if content.N != "" {
		n, err = strconv.Atoi(content.N)
		if err != nil {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid n: %s",
				nft.Base.InscriptionId, content.Ticker, content.N)
			return nil
		}
		if n <= 0 || n > 100000000 {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid n: %d",
				nft.Base.InscriptionId, content.Ticker, n)
			return nil
		}
		if lim%int64(n) != 0 {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid lim/n: %d %d",
				nft.Base.InscriptionId, content.Ticker, lim, n)
			return nil
		}
	}

	selfmint := 0
	if content.SelfMint != "" {
		selfmint, err = getPercentage(content.SelfMint)
		if err != nil {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid SelfMint: %s",
				nft.Base.InscriptionId, content.Ticker, content.SelfMint)
			return nil
		}
		if selfmint > 100 {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid SelfMint: %s",
				nft.Base.InscriptionId, content.Ticker, content.SelfMint)
			return nil
		}
	}

	max := int64(-1)
	if content.Max != "" {
		max, err = strconv.ParseInt(content.Max, 10, 64)
		if err != nil {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid max: %s",
				nft.Base.InscriptionId, content.Ticker, content.Max)
			return nil
		}
		if max < 0 {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid max: %d",
				nft.Base.InscriptionId, content.Ticker, max)
			return nil
		}
		if max%int64(n) != 0 {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid max/n: %d %d",
				nft.Base.InscriptionId, content.Ticker, max, n)
			return nil
		}
	}
	if selfmint > 0 {
		if content.Max == "" {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, must set max",
				nft.Base.InscriptionId, content.Ticker)
			return nil
		}
	}

	blockStart := -1
	blockEnd := -1
	if content.Block != "" {
		parts := strings.Split(content.Block, "-")
		if len(parts) != 2 {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid block: %s",
				nft.Base.InscriptionId, content.Ticker, content.Block)
			return nil
		}
		var err error
		blockStart, err = strconv.Atoi(parts[0])
		if err != nil {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid block: %s",
				nft.Base.InscriptionId, content.Ticker, content.Block)
			return nil
		}
		if blockStart < 0 {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId:%s, ticker: %s, invalid block: %s",
				nft.Base.InscriptionId, content.Ticker, content.Block)
			return nil
		}
		blockEnd, err = strconv.Atoi(parts[1])
		if err != nil {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid block: %s",
				nft.Base.InscriptionId, content.Ticker, content.Block)
			return nil
		}
		if blockEnd < 0 {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid block: %s",
				nft.Base.InscriptionId, content.Ticker, content.Block)
			return nil
		}
		if blockEnd < blockStart {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid block: %s",
				nft.Base.InscriptionId, content.Ticker, content.Block)
			return nil
		}
	}
	if selfmint < 100 && s.isSat20Actived(int(height)) {
		if s.IsMainnet() {
			if blockStart > 0 && int(height)+common.MIN_BLOCK_INTERVAL > blockStart {
				common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, start of block should be larger than: %d",
					nft.Base.InscriptionId, content.Ticker, height+common.MIN_BLOCK_INTERVAL)
				return nil
			}
		} else {
			if blockStart > 0 && int(height)+5 > blockStart {
				common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, start of block should be larger than: %d",
					nft.Base.InscriptionId, content.Ticker, height+5)
				return nil
			}
		}

	}

	var attr common.SatAttr
	if content.Attr != "" {
		var err error
		attr, err = parseSatAttrString(content.Attr)
		if err != nil {
			common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid attr: %s, ParseSatAttrString err: %v",
				nft.Base.InscriptionId, content.Ticker, content.Attr, err)
			return nil
		}

		if indexer.IsRaritySatRequired(&attr) {
			// 目前只支持稀有聪铸造
			if attr.RegularExp != "" || attr.Template != "" || attr.TrailingZero != 0 {
				common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, invalid attr: %s",
					nft.Base.InscriptionId, content.Ticker, content.Attr)
				return nil
			}
		}
	}

	// 确保输出在output中
	if out.OutValue.Value < satpoint {
		common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, ranges not in output",
			nft.Base.InscriptionId, content.Ticker)
		return nil
	}

	nft.Base.UserData = []byte(content.Ticker)
	ticker := &common.Ticker{
		Base:       common.CloneBaseContent(nft.Base),
		Name:       content.Ticker,
		Desc:       content.Des,
		Type:       common.ASSET_TYPE_FT,
		Limit:      lim,
		N:          n,
		SelfMint:   selfmint,
		Max:        max,
		BlockStart: blockStart,
		BlockEnd:   blockEnd,
		Attr:       attr,
	}

	if !common.TickerSeparatedFromName {
		if reg == nil {
			nft.Base.TypeName = common.ASSET_TYPE_NS
			reg = &ns.NameRegister{
				Nft:  nft,
				Name: strings.ToLower(ticker.Name),
			}

			s.ns.NameRegister(reg)
		}
	}

	return ticker
}

func (s *IndexerMgr) handleMintTicker(satpoint int64, in *common.TxInput, out *common.TxOutputV2,
	content *common.OrdxMintContent, nft *common.Nft) *common.Mint {
	inscriptionId := nft.Base.InscriptionId
	height := nft.Base.BlockHeight
	deployTicker := s.ftIndexer.GetTicker(content.Ticker)
	if deployTicker == nil {
		common.Log.Warnf("IndexerMgr.handleMintTicker: inscriptionId: %s, ticker: %s, no deploy ticker",
			inscriptionId, content.Ticker)
		return nil
	}
	if deployTicker.BlockStart != -1 && int(height) < deployTicker.BlockStart {
		common.Log.Warnf("IndexerMgr.handleMintTicker: inscriptionId: %s, ticker: %s, block height(%d) not in depoly block range(%d-%d)",
			inscriptionId, content.Ticker, height, deployTicker.BlockStart, deployTicker.BlockEnd)
		return nil
	}

	if deployTicker.BlockEnd != -1 && int(height) > deployTicker.BlockEnd {
		common.Log.Warnf("IndexerMgr.handleMintTicker: inscriptionId: %s, ticker: %s, block height(%d) not in depoly block range(%d-%d)",
			inscriptionId, content.Ticker, height, deployTicker.BlockStart, deployTicker.BlockEnd)
		return nil
	}

	amt := deployTicker.Limit
	// check mint limit
	if content.Amt != "" {
		var err error
		amt, err = strconv.ParseInt(content.Amt, 10, 64)
		if err != nil {
			common.Log.Warnf("IndexerMgr.handleMintTicker: inscriptionId: %s, ticker: %s, invalid amt: %s",
				inscriptionId, content.Ticker, content.Amt)
			return nil
		}
		if amt < 0 {
			common.Log.Warnf("IndexerMgr.handleMintTicker: inscriptionId: %s, ticker: %s, invalid amt: %d",
				inscriptionId, content.Ticker, amt)
			return nil
		}

		if amt > deployTicker.Limit {
			common.Log.Warnf("IndexerMgr.handleMintTicker: inscriptionId: %s, ticker: %s, amt(%d) > limit(%d)",
				inscriptionId, content.Ticker, amt, deployTicker.Limit)
			return nil
		}
		if amt%int64(deployTicker.N) != 0 {
			common.Log.Warnf("IndexerMgr.handleMintTicker: inscriptionId: %s, ticker: %s, amt(%d) / n(%d)",
				inscriptionId, content.Ticker, amt, deployTicker.N)
			return nil
		}
	}
	addressId := out.AddressId
	permitAmt := s.getMintAmount(deployTicker.Name, addressId)
	if amt > permitAmt {
		common.Log.Warnf("IndexerMgr.handleMintTicker: inscriptionId: %s, ticker: %s, invalid amt: %s",
			inscriptionId, content.Ticker, content.Amt)
		return nil
	}

	satsNum := int64(amt) / int64(deployTicker.N)
	newRngs := common.AssetOffsets{
		{
			Start: satpoint,
			End:   satpoint + satsNum,
		},
	}

								
	if indexer.IsRaritySatRequired(&deployTicker.Attr) {
		// 如果是稀有聪铸造，需要调整稀有聪范围
		// 因为中间可能存在白聪：383ef74030578308823d524b5ae24820c68b82f6109324da82b6c6e79e3b143ci4
		if deployTicker.Attr.Rarity != "" {
			exoticName := common.AssetName{
				Protocol: common.PROTOCOL_NAME_ORDX,
				Type:     common.ASSET_TYPE_EXOTIC,
				Ticker:   deployTicker.Attr.Rarity,
			}	
			exoticranges := in.Offsets[exoticName]
			newRngs = exoticranges.Pickup(satpoint, satsNum)
			if len(newRngs) == 0 {
				common.Log.Infof("IndexerMgr.handleMintTicker: inscriptionId: %s, ticker: %s, but no enough exotic satoshi", inscriptionId, content.Ticker)
				return nil
			}
		}
	}
	// 禁止在同一个聪上做同样名字的资产的铸造
	// if s.hasSameTickerInRange(content.Ticker, newRngs) {
	// 	common.Log.Warnf("IndexerMgr.handleMintTicker: inscriptionId: %s, ticker: %s, ranges has same ticker",
	// 		inscriptionId, content.Ticker)
	// 	return nil
	// }

	if len(newRngs) == 0 || newRngs.Size() != satsNum {
		common.Log.Warnf("IndexerMgr.handleMintTicker: inscriptionId: %s, ticker: %s, amt(%d), no enough sats %d",
			inscriptionId, content.Ticker, satsNum, newRngs.Size())
		return nil
	}

	// 铸造结果：从指定的nft，往后如果有satsNum个聪，就是铸造成功，这些聪都是输入的一部分就可以，输出在哪里无所谓
	// // 确保newRngs都在output中
	// if !common.RangesContained(out.Ordinals, newRngs) {
	// 	common.Log.Warnf("IndexerMgr.handleMintTicker: inscriptionId: %s, ticker: %s, ranges not in output",
	// 		inscriptionId, content.Ticker)
	// 	return nil
	// }

	nft.Base.TypeName = common.ASSET_TYPE_FT
	mint := &common.Mint{
		Base:    common.CloneBaseContent(nft.Base),
		Name:    content.Ticker,
		UtxoId:  in.UtxoId,  // ordx资产需要从input的聪中分配
		Offsets: newRngs,
		Amt:     int64(amt),
		Desc:    content.Des,
	}

	return mint
}

func (s *IndexerMgr) handleBrc20DeployTicker(satpoint int64, out *common.TxOutputV2,
	content *common.BRC20DeployContent, nft *common.Nft) *common.BRC20Ticker {

	ticker := &common.BRC20Ticker{
		Nft:  nft,
		Name: content.Ticker, // 保留原型
		// Limit:      lim,
		// SelfMint:   selfmint,
		// Max:        max,
		Decimal:            uint8(18),
		DeployTime:         nft.Base.BlockTime,
		StartInscriptionId: nft.Base.InscriptionId,
		EndInscriptionId:   "",
		HolderCount:        0,
		TransactionCount:   0,
	}

	if content.SelfMint == "true" {
		ticker.SelfMint = true
	}

	// dec
	if content.Decimal != "" {
		dec, err := strconv.ParseUint(content.Decimal, 10, 64)
		if err != nil || dec > 18 {
			// dec invalid
			common.Log.Warnf("deploy, but dec invalid. ticker: %s, dec: %s", content.Ticker, content.Decimal)
			return nil
		}
		ticker.Decimal = uint8(dec)
	}

	// max
	max, err := ParseBrc20Amount(content.Max, int(ticker.Decimal))
	if err != nil {
		// max invalid
		common.Log.Warnf("deploy, but max invalid. ticker: %s, max: '%s'", content.Ticker, content.Max)
		return nil
	}
	if max.Sign() < 0 || max.IsOverflowUint64() {
		common.Log.Warnf("deploy, but max invalid (range)")
		return nil
		// return
	}

	if max.Sign() == 0 {
		if ticker.SelfMint {
			ticker.Max = *max.GetMaxUint64()
		} else {
			common.Log.Warnf("deploy, but max invalid (0)")
			return nil
		}
	} else {
		ticker.Max = *max
	}

	// minted
	minted, err := common.NewDecimalFromString("0", int(ticker.Decimal))
	if err != nil {
		// minted invalid
		common.Log.Warnf("deploy, but minted invalid. ticker: %s, minted: '%s'", content.Ticker, minted.String())
		return nil
	}
	ticker.Minted = *minted

	// lim
	lim, err := ParseBrc20Amount(content.Lim, int(ticker.Decimal))
	if err != nil {
		// limit invalid
		common.Log.Warnf("deploy, but limit invalid. ticker: %s, limit: '%s'", content.Ticker, content.Lim)
		return nil
	}
	if lim.Sign() < 0 || lim.IsOverflowUint64() {
		common.Log.Warnf("deploy, but lim invalid (range)")
		return nil
	}
	if lim.Sign() == 0 {
		if ticker.SelfMint {
			ticker.Limit = *max
		} else {
			common.Log.Warnf("deploy, but lim invalid (0)")
			return nil
		}
	} else {
		ticker.Limit = *lim
	}

	return ticker
}

func (s *IndexerMgr) handleBrc20MintTicker(satpoint int64, out *common.TxOutputV2,
	content *common.BRC20MintContent, nft *common.Nft) *common.BRC20Mint {
	ticker := s.brc20Indexer.GetTicker(content.Ticker)
	if ticker == nil {
		common.Log.Warnf("IndexerMgr.handleBrc20MintTicker: inscriptionId: %s, ticker: %s, no deploy ticker",
			nft.Base.InscriptionId, content.Ticker)
		return nil
	}

	if ticker.SelfMint {
		if nft.Base.Parent != ticker.Nft.Base.InscriptionId {
			return nil
		}
	}

	mint := &common.BRC20Mint{
		BRC20MintInDB: common.BRC20MintInDB{
			NftId: nft.Base.Id,
			Name: strings.ToLower(content.Ticker),
		},
		Nft: nft, 
	}

	// recover for decimal panic
	defer func() {
		if r := recover(); r != nil {
			common.Log.Warnf("IndexerMgr.handleBrc20MintTicker: inscriptionId: %s, ticker: %s, panic: %v",
				nft.Base.InscriptionId, content.Ticker, r)
		}
	}()

	// check mint amount
	amt, err := ParseBrc20Amount(content.Amt, int(ticker.Decimal))
	if err != nil {
		common.Log.Warnf("mint %s, but invalid amount(%s)", content.Ticker, content.Amt)
		return nil
	}

	if amt.Sign() <= 0 || amt.Cmp(&ticker.Limit) > 0 {
		common.Log.Warnf("mint %s, invalid amount(%s), limit(%s)", content.Ticker, content.Amt, ticker.Limit.String())
		return nil
	}

	// check max
	mintedAmt := ticker.Minted.Add(amt)
	cmpResult := mintedAmt.Cmp(&ticker.Max)
	if cmpResult > 0 {
		amt = ticker.Max.Sub(&ticker.Minted)
		common.Log.Debugf("mint %s, invalid amount(%s), max(%s), change to %s", content.Ticker, content.Amt, ticker.Max.String(), amt.String())
	}
	if amt.Sign() <= 0 {
		common.Log.Debugf("mint %s, invalid amount(%s)", content.Ticker, amt.String())
		return nil
	}
	mint.Amt = *amt
	return mint
}

func (s *IndexerMgr) handleBrc20TransferTicker(satpoint int64, out *common.TxOutputV2,
	content *common.BRC20TransferContent, nft *common.Nft) *common.BRC20Transfer {
	inscriptionId := nft.Base.InscriptionId
	ticker := s.brc20Indexer.GetTicker(content.Ticker)
	if ticker == nil {
		common.Log.Warnf("IndexerMgr.handleBrc20TransferTicker: inscriptionId: %s, ticker: %s, no deploy ticker",
			inscriptionId, content.Ticker)
		return nil
	}

	transfer := &common.BRC20Transfer{
		BRC20TransferInDB: common.BRC20TransferInDB{
			Name: strings.ToLower(content.Ticker),
		},
		Nft:  nft,
	}

	// check amount
	amt, err := ParseBrc20Amount(content.Amt, int(ticker.Decimal))
	if err != nil {
		common.Log.Warnf("transfer, but invalid amount")
		return nil
	}
	if amt.Sign() <= 0 || amt.Cmp(&ticker.Max) > 0 {
		common.Log.Warnf("transfer, invalid amount(range)")
		return nil
	}

	transfer.Amt = *amt

	return transfer
}

func (s *IndexerMgr) handleNameRegister(content *common.OrdxRegContent, nft *common.Nft) {

	name := strings.ToLower(content.Name)

	reg := &ns.NameRegister{
		Nft:  nft,
		Name: name,
	}
	nft.Base.TypeName = common.ASSET_TYPE_NS
	nft.Base.UserData = []byte(name)

	s.ns.NameRegister(reg)

	if len(content.KVs) > 0 {
		update := &ns.NameUpdate{
			InscriptionId: nft.Base.InscriptionId,
			BlockHeight:   int(nft.Base.BlockHeight),
			Sat:           nft.Base.Sat,
			Name:          name,
			KVs:           ns.ParseKVs(content.KVs),
		}
		s.ns.NameUpdate(update)
	}
}

func (s *IndexerMgr) handleNameUpdate(content *common.OrdxUpdateContentV2, nft *common.Nft) {

	content.Name = strings.ToLower(content.Name)

	reg := s.ns.GetNameRegisterInfo(content.Name)
	if reg == nil {
		common.Log.Warnf("IndexerMgr.handleNameUpdate: %s, Name %s not exist", nft.Base.InscriptionId, content.Name)
		return
	}

	// 只需要当前owner持有该nft就可以修改，而不必在sat上继续铸造
	if nft.OwnerAddressId != reg.Nft.OwnerAddressId {
		common.Log.Warnf("IndexerMgr.handleNameUpdate: %s, Name %s has different owner", nft.Base.InscriptionId, content.Name)
		return
	}

	// if nft.Base.Sat != reg.Nft.Base.Sat {
	// 	common.Log.Warnf("IndexerMgr.handleNameUpdate: %s, name: %s, invalid sat: %d : %d",
	// 		nft.Base.InscriptionId, content.Name, reg.Nft.Base.Sat, nft.Base.Sat)
	// 	return
	// }

	// 如果是一个ticker，看看是否要修改显示封面（不允许修改跟铸币相关的属性）
	ticker := s.ftIndexer.GetTicker(content.Name)
	if ticker != nil {
		delegate := ""
		for k, v := range content.KVs {
			switch k {
			case "Delegate":
				delegate = v
			}
		}
		if delegate != "" {
			ticker.Base.Delegate = delegate
			s.ftIndexer.UpdateTick(ticker)
		}
	}

	kvs := make([]*ns.KeyValue, 0)
	for k, v := range content.KVs {
		// 对于需要做持有者检查的属性，简单忽略就行，不影响其他有效属性
		if k == "avatar" {
			avatar := s.nft.GetNftWithInscriptionId(v)
			if avatar == nil || avatar.OwnerAddressId != nft.OwnerAddressId {
				common.Log.Warnf("IndexerMgr.handleNameUpdate: %s, name: %s, invalid avatar: %v, ignore it",
					nft.Base.InscriptionId, content.Name, v)
				continue
			}
		}
		kvs = append(kvs, &ns.KeyValue{Key: k, Value: v})
	}

	update := &ns.NameUpdate{
		InscriptionId: nft.Base.InscriptionId,
		BlockHeight:   int(nft.Base.BlockHeight),
		Name:          content.Name,
		KVs:           kvs,
	}
	nft.Base.TypeName = common.ASSET_TYPE_NFT

	s.ns.NameUpdate(update)
}

func (s *IndexerMgr) handleNameRouting(content *common.OrdxUpdateContentV2, nft *common.Nft) {

	content.Name = strings.ToLower(content.Name)

	reg := s.ns.GetNameRegisterInfo(content.Name)
	if reg == nil {
		common.Log.Warnf("IndexerMgr.handleNameRouting: %s, Name %s not exist", nft.Base.InscriptionId, content.Name)
		return
	}

	// 只需要当前owner持有该nft就可以修改，而不必在sat上继续铸造
	if nft.OwnerAddressId != reg.Nft.OwnerAddressId {
		common.Log.Warnf("IndexerMgr.handleNameRouting: %s, Name %s has different owner", nft.Base.InscriptionId, content.Name)
		return
	}

	kvs := make([]*ns.KeyValue, 0)
	for k, v := range content.KVs {
		kvs = append(kvs, &ns.KeyValue{Key: k, Value: v})
	}

	update := &ns.NameUpdate{
		InscriptionId: nft.Base.InscriptionId,
		BlockHeight:   int(nft.Base.BlockHeight),
		Name:          content.Name,
		KVs:           kvs,
	}
	nft.Base.TypeName = common.ASSET_TYPE_NFT

	s.ns.NameUpdate(update)
}

func (s *IndexerMgr) handleOrdX(satpoint int64, in *common.TxInput, out *common.TxOutputV2,
	insc *ord.InscriptionResult, nft *common.Nft) {
	ordxInfo, bOrdx := ord.IsOrdXProtocol(insc)
	if !bOrdx {
		return
	}

	ordxType := common.GetBasicContent(ordxInfo)
	switch ordxType.Op {
	case "deploy":
		deployInfo := common.ParseDeployContent(ordxInfo)
		if deployInfo == nil {
			return
		}
		// common.Log.Infof("indexer.handleOrdX: prepare deploy ticker, content: %s", deployInfo)

		if s.ftIndexer.TickExisted(deployInfo.Ticker) {
			common.Log.Warnf("ticker %s exists", deployInfo.Ticker)
			return
		}

		ticker := s.handleDeployTicker(satpoint, in, out, deployInfo, nft)
		if ticker == nil {
			return
		}

		s.ftIndexer.UpdateTick(ticker)

	case "mint":
		mintInfo := common.ParseMintContent(ordxInfo)
		if mintInfo == nil {
			return
		}
		// common.Log.Infof("IndexerMgr.handleOrdX: prepare mint ticker is succ: %v", mintInfo)

		if !s.ftIndexer.TickExisted(mintInfo.Ticker) {
			common.Log.Warnf("ticker %s does not exist", mintInfo.Ticker)
			return
		}

		mint := s.handleMintTicker(satpoint, in, out, mintInfo, nft)
		if mint == nil {
			return
		}

		s.ftIndexer.UpdateMint(in, mint)

	default:
		//common.Log.Warnf("handleOrdX unknown ordx type: %s, content: %s, txid: %s", ordxType, content, tx.Txid)
	}
}

func (s *IndexerMgr) handleBrc20(inUtxoId uint64, satpoint int64, out *common.TxOutputV2,
	insc *ord.InscriptionResult, nft *common.Nft) {

	content := string(insc.Inscription.Body)
	ordxBaseContent := common.ParseBrc20BaseContent(content)
	if ordxBaseContent == nil {
		common.Log.Debugf("invalid content %s", content)
		return
	}

	if out.OutValue.Value == 0 {
		common.Log.Debugf("invalid brc20 inscription %s", nft.Base.InscriptionId)
		return
	}

	switch strings.ToLower(ordxBaseContent.Op) {
	case "deploy":
		deployInfo := common.ParseBrc20DeployContent(content)
		if deployInfo == nil {
			return
		}
		if len(deployInfo.Ticker) == 5 {
			if deployInfo.SelfMint != "true" {
				common.Log.Errorf("deploy, tick length 5, but not self_mint")
				return
			}

			if s.IsMainnet() && nft.Base.BlockHeight < 837090 {
				common.Log.Errorf("deploy, tick length 5, but not enabled")
				return
			}
		}

		if s.brc20Indexer.TickExisted(deployInfo.Ticker) {
			common.Log.Warnf("ticker %s exists", deployInfo.Ticker)
			return
		}

		ticker := s.handleBrc20DeployTicker(satpoint, out, deployInfo, nft)
		if ticker == nil {
			return
		}

		s.brc20Indexer.UpdateInscribeDeploy(ticker)

	case "mint":
		mintInfo := common.ParseBrc20MintContent(content)
		if mintInfo == nil {
			return
		}
		// if mintInfo.BRC20BaseContent.Ticker != "box1" {
		// 	return
		// } else {
		// 	common.Log.Info("mint brc20 ticker is box1")
		// }

		mint := s.handleBrc20MintTicker(satpoint, out, mintInfo, nft)
		if mint == nil {
			return
		}
		//common.Log.Infof("nft.Base.InscriptionId: %s, nft.Base.Id: %d", nft.Base.InscriptionId, nft.Base.Id)
		s.brc20Indexer.UpdateInscribeMint(mint)

	case "transfer":
		transferInfo := common.ParseBrc20TransferContent(content)
		if transferInfo == nil {
			return
		}
		// if transferInfo.BRC20BaseContent.Ticker != "box1" {
		// 	return
		// } else {
		// 	common.Log.Info("transfer brc20 ticker is box1")
		// }

		transfer := s.handleBrc20TransferTicker(satpoint, out, transferInfo, nft)
		if transfer == nil {
			return
		}

		s.brc20Indexer.UpdateInscribeTransfer(transfer)

	default:
		//common.Log.Warnf("handleOrdX unknown ordx type: %s, content: %s, txid: %s", ordxType, content, tx.Txid)
	}
}

func (s *IndexerMgr) handleOrd(input *common.TxInput,
	insc *ord.InscriptionResult, inscriptionId int, tx *common.Transaction,
	block *common.Block, coinbase []*common.Range) {

	satpoint := int64(0)
	if insc.Inscription.Pointer != nil {
		satpoint = int64(common.GetSatpoint(insc.Inscription.Pointer))
		if int64(satpoint) >= input.OutValue.Value {
			satpoint = 0
		}
	}
	index := int(insc.TxInIndex)

	var output *common.TxOutputV2
	
	// 遵循ordinals的规则
	output, satpoint = findOutputWithSatPoint(block, coinbase, index, tx, satpoint)

	// 1. 先保存nft数据
	nft := s.handleNft(input, output, satpoint, insc, inscriptionId, tx, block)
	if nft == nil {
		return
	}

	if input.OutValue.Value == 0 {
		// 虽然ordinals.com解析出了这个交易，但是我们认为该交易没有输入的sat，也就是无法将数据绑定到某一个sat上，违背了协议原则
		// 特殊交易，ordx不支持，不处理
		// c1e0db6368a43f5589352ed44aa1ff9af33410e4a9fd9be0f6ac42d9e4117151
		// TODO 0605版本中，没有把这个nft编译进来
		return
	}

	// 2. 再看看是否ordx协议
	protocol, content := ord.GetProtocol(insc)
	switch protocol {
	case "ordx":
		s.handleOrdX(satpoint, input, output, insc, nft)
	case "sns":
		domain := common.ParseDomainContent(string(insc.Inscription.Body))
		if domain == nil {
			domain = common.ParseDomainContent(string(content))
		}
		if domain != nil {
			switch domain.Op {
			case "reg": // https://docs.btcname.id/docs/overview/chapter-4-thinking-about-.btc-domain-name/calibration-rules
			// 不支持该方式注册名字
			//s.handleSnsName(domain.Name, nft)
			case "update":
				var updateInfo *common.OrdxUpdateContentV2
				// 如果有metadata，那么不处理FIELD_CONTENT的内容
				if string(insc.Inscription.Metaprotocol) == "sns" && len(insc.Inscription.Metadata) != 0 {
					updateInfo = common.ParseUpdateContent(string(content))
					updateInfo.P = "sns"
					value, ok := updateInfo.KVs["key"]
					if ok {
						delete(updateInfo.KVs, "key")
						updateInfo.KVs[value] = nft.Base.InscriptionId
					}
				} else {
					updateInfo = common.ParseUpdateContent(string(content))
				}

				if updateInfo == nil {
					return
				}
				s.handleNameUpdate(updateInfo, nft)
			}
		}
	case "brc-20":
		// if s.IsMainnet() && s.brc20Indexer.IsExistCursorInscriptionInDB(nft.Base.InscriptionId) {
		// 	return
		// }
		if nft.Base.CurseType != 0 { 
			common.Log.Infof("%s inscription is cursed, %d", nft.Base.InscriptionId, nft.Base.CurseType)
			if block.Height < 824544 { // Jubilee
				return
			}
			// vindicated
		}

		s.handleBrc20(input.UtxoId, satpoint, output, insc, nft)

	case "primary-name":
		primaryNameContent := common.ParseCommonContent(string(insc.Inscription.Body))
		if primaryNameContent != nil {
			switch primaryNameContent.Op {
			case "update":
				s.handleNameUpdate(primaryNameContent, nft)
			}
		}
		// {
		// 	"p": "primary-name",
		// 	"op": "update",
		// 	"name": "btcname.btc",
		// 	"avatar": "41479dbcb749ec04872b77c5cb4a67dc7b13f746ba2e86ba70854d0cdaed0646i0"
		//   }
		// type: application/json
		// content: { "p": "sns", "op": "reg", "name": "1866.sats"}
		// or ： text/plain;charset=utf-8 {"p":"sns","op":"reg","name":"good.sats"}
	case "btcname":
		commonContent := common.ParseCommonContent(string(insc.Inscription.Body))
		if commonContent != nil {
			switch commonContent.Op {
			case "routing":
				s.handleNameRouting(commonContent, nft)
			}
		}
		/*
			{
				"p":"btcname",
				"op":"routing",
				"name":"xxx.btc",
				"ord_handle":"xxx",
				"ord_index":"xxxi0",
				"btc_p2phk":"1xxx",
				"btc_p2sh":"3xxx",
				"btc_segwit":"bc1qxxx",
				"btc_lightning":"xxx",
				"eth_address":"0xxxx",
				"matic_address":"0xxxx",
				"sol_address":"xxx",
				"avatar":"xxxi0"
			}
		*/

	default:
		// 3. 如果content中的内容格式，符合 *.* 或者 * , 并且字段在32字节以内，符合名字规范，就把它当做一个名字来处理
		// text/plain;charset=utf-8 abc
		// 或者简单文本 xxx.xx 或者 xx
		if protocol == "" {
			s.handleSnsName(string(insc.Inscription.Body), nft)
		}
	}

}

func (s *IndexerMgr) handleSnsName(name string, nft *common.Nft) {
	if common.IsValidSNSName(name) {
		info := s.ns.GetNameRegisterInfo(name)
		if info != nil {
			common.Log.Debugf("%s Name %s exist, registered at %s",
				nft.Base.InscriptionId, name, info.Nft.Base.InscriptionId)
			return
		}

		regInfo := &common.OrdxRegContent{
			OrdxBaseContent: common.OrdxBaseContent{P: "sns", Op: "reg"},
			Name:            name}

		s.handleNameRegister(regInfo, nft)
	}
}

func (s *IndexerMgr) handleNft(input *common.TxInput, output *common.TxOutputV2, satpoint int64, 
	insc *ord.InscriptionResult, inscriptionId int, tx *common.Transaction, block *common.Block) *common.Nft {

	//if s.nft.Base.IsEnabled() {
	sat := int64(-1)
	if input.OutValue.Value > 0 {
		sat = int64(common.ToUtxoId(output.Height, output.TxIndex, inscriptionId))
	}

	//addressId1 := s.compiling.GetAddressId(input.Address.Addresses[0])
	addressId2 := output.AddressId
	nft := common.Nft{
		Base: &common.InscribeBaseContent{
			InscriptionId:      tx.TxId + "i" + strconv.Itoa(inscriptionId),
			InscriptionAddress: addressId2, // TODO 这个地址不是铭刻者，模型的问题，比较难改，直接使用输出地址
			BlockHeight:        int32(block.Height),
			BlockTime:          block.Timestamp.Unix(),
			ContentType:        insc.Inscription.ContentType,
			Content:            insc.Inscription.Body,
			ContentEncoding:    insc.Inscription.ContentEncoding,
			MetaProtocol:       insc.Inscription.Metaprotocol,
			MetaData:           insc.Inscription.Metadata,
			Parent:             common.ParseInscriptionId(insc.Inscription.Parent),
			Delegate:           common.ParseInscriptionId(insc.Inscription.Delegate),
			Sat:                sat,
			CurseType:          int32(insc.CurseReason),
			TypeName:           common.ASSET_TYPE_NFT,
		},
		OwnerAddressId: addressId2,
		UtxoId:         output.UtxoId,
		Offset:         satpoint, // 在input中的偏移
	}
	s.nft.NftMint(input, &nft)
	if !insc.IsCursed && nft.Base.CurseType != 0 {
		insc.IsCursed = true
		insc.CurseReason = ordCommon.Curse(nft.Base.CurseType)
	}
	return &nft
	// }
	// return nil
}

func getSatInRange(common []*common.Range, satpoint int64) int64 {
	for _, rng := range common {
		if satpoint >= (rng.Size) {
			satpoint -= (rng.Size)
		} else {
			return rng.Start + int64(satpoint)
		}
	}

	return -1
}

// func (s *IndexerMgr) hasSameTickerInRange(ticker string, rngs []*common.Range) bool {
// 	for _, rng := range rngs {
// 		if s.ftIndexer.CheckTickersWithSatRange(ticker, rng) {
// 			return true
// 		}
// 	}
// 	return false
// }

func (s *IndexerMgr) getMintAmountByAddressId(ticker string, address uint64) int64 {
	return s.ftIndexer.GetMintAmountWithAddressId(address, ticker)
}

// 有资格的地址：跟引导节点建立了通道，而且该通道持有足够的资产
func (s *IndexerMgr) isEligibleUser(pkScript []byte, pubkey string) bool {
	assetName := common.NewAssetNameFromString(common.CORENODE_STAKING_ASSET_NAME)
	amt := common.CORENODE_STAKING_ASSET_AMOUNT
	if !s.IsMainnet() {
		assetName = common.NewAssetNameFromString(common.TESTNET_CORENODE_STAKING_ASSET_NAME)
		amt = common.TESTNET_CORENODE_STAKING_ASSET_AMOUNT
	}

	pubkeyBytes, err := hex.DecodeString(pubkey)
	if err != nil {
		common.Log.Errorf("DecodeString %s failed", pubkey)
		return false
	}

	address, err := common.GetBTCAddressFromPkScript(pkScript, s.chaincfgParam)
	if err != nil {
		common.Log.Errorf("GetBTCAddressFromPkScript %v failed, %v", pkScript, err)
		return false
	}

	address2, err := common.GetP2TRAddressFromPubkey(pubkeyBytes, s.chaincfgParam)
	if err != nil {
		common.Log.Errorf("GetP2TRAddressFromPubkey %s failed, %v", pubkey, err)
		return false
	}
	if address != address2 {
		common.Log.Errorf("address %s != address2 %s", address, address2)
		return false
	}

	address3, err := common.GetCoreNodeChannelAddress(pubkeyBytes, s.chaincfgParam)
	if err != nil {
		common.Log.Errorf("GetCoreNodeChannelAddress %s failed, %v", pubkey, err)
		return false
	}

	addrmap := s.GetHoldersWithTick(assetName.Ticker)
	//addressId := s.compiling.GetAddressId(address3) address3 不是跑数据过程中交易相关地址，不能通过这个函数获取
	addressId := s.rpcService.GetAddressId(address3)
	value := addrmap[addressId]
	result := value >= amt
	if !result {
		common.Log.Errorf("not enough assets, value %d, amt %d", value, amt)
	}
	return result
}

func (s *IndexerMgr) isSat20Actived(height int) bool {
	if s.IsMainnet() {
		return height >= 845000
	} else if s.chaincfgParam.Name == "testnet3" {
		return height >= 2810000
	} else {
		return height >= 0
	}
}

func (b *IndexerMgr) getMintAmount(ticker string, addressId uint64) int64 {
	deployTicker := b.ftIndexer.GetTicker(ticker)

	if deployTicker == nil {
		common.Log.Warnf("IndexerMgr.getMintAmount: ticker: %s, no deploy ticker", ticker)
		return -1
	}

	nftOwnAddressId := b.nft.GetNftHolderWithInscriptionId(deployTicker.Base.InscriptionId)
	isOwner := addressId == nftOwnAddressId

	amt := int64(0)

	mintAmount, _ := b.GetMintAmount(deployTicker.Name)
	if deployTicker.SelfMint > 0 {
		ownerMinted := b.getMintAmountByAddressId(deployTicker.Name, nftOwnAddressId)
		if isOwner {
			limit := (deployTicker.Max * int64(deployTicker.SelfMint)) / 100
			amt = limit - ownerMinted
		} else {
			if deployTicker.SelfMint == 100 {
				amt = 0
			} else {
				limit := (deployTicker.Max * int64(100-deployTicker.SelfMint)) / 100
				amt = limit - (mintAmount - ownerMinted)
			}
		}
	} else {
		// == 0
		if deployTicker.Max < 0 {
			// no limit
			amt = common.MaxSupply
		} else {
			amt = deployTicker.Max - mintAmount
		}
	}
	return amt
}

func (b *IndexerMgr) isLptTicker(name string) bool {

	// 支持lpt： xxx.lptnnn or xxx.runes.lptnnn
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return false
	}

	org := parts[0]

	var protocol, lpt string
	if len(parts) == 3 {
		protocol = parts[1]
		lpt = parts[2]
	} else if len(parts) == 2 {
		protocol = common.PROTOCOL_NAME_ORDX
		lpt = parts[1]
	} else {
		return false
	}

	switch protocol {
	case common.PROTOCOL_NAME_ORDX:
		if !b.ftIndexer.TickExisted(org) {
			return false
		}
	case common.PROTOCOL_NAME_RUNES:
		if !b.RunesIndexer.IsExistRuneWithId(org) {
			return false
		}
	case common.PROTOCOL_NAME_BRC20:
		if !b.brc20Indexer.TickExisted(org) {
			return false
		}
	default:
		return false
	}

	return lpt == "lpt"
	// TODO 暂时不支持各个核心通道部署自己的流动性质押代币，只统一使用lpt,

	// num, has := strings.CutPrefix(lpt, "lpt")
	// if !has {
	// 	return false
	// }
	// _, err :=  strconv.Atoi(num)
	// return err == nil

}

func ParseBrc20Amount(amt string, dec int) (*common.Decimal, error) {
	if len(amt) > 0 {
		if amt[0] == '+' || amt[0] == '-' {
			return nil, fmt.Errorf("invalid amount: %s", amt)
		}
	}
	ret, err := common.NewDecimalFromString(amt, dec)
	return ret, err
}
