package indexer

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/brc20"
	indexer "github.com/sat20-labs/indexer/indexer/common"
	"github.com/sat20-labs/indexer/indexer/ns"
)

func (s *IndexerMgr) processOrdProtocol(block *common.Block) {
	if block.Height < s.ordFirstHeight {
		return
	}

	detectOrdMap := make(map[string]int, 0)
	measureStartTime := time.Now()
	//common.Log.Info("processOrdProtocol ...")
	count := 0
	for _, tx := range block.Transactions {
		id := 0
		for _, input := range tx.Inputs {

			inscriptions, envelopes, err := common.ParseInscription(input.Witness)
			if err != nil {
				continue
			}

			for i, insc := range inscriptions {
				s.handleOrd(input, insc, id, envelopes[i], tx, block)
				id++
				count++
			}
		}
		if id > 0 {
			detectOrdMap[tx.Txid] = id
		}
	}
	//common.Log.Infof("processOrdProtocol loop %d finished. cost: %v", count, time.Since(measureStartTime))

	//time2 := time.Now()
	s.exotic.UpdateTransfer(block)
	s.nft.UpdateTransfer(block)
	s.ns.UpdateTransfer(block)
	s.ftIndexer.UpdateTransfer(block)
	s.brc20Indexer.UpdateTransfer(block)
	s.RunesIndexer.UpdateTransfer(block)

	//common.Log.Infof("processOrdProtocol UpdateTransfer finished. cost: %v", time.Since(time2))

	// 检测是否一致，如果不一致，需要进一步调试。
	// s.detectInconsistent(detectOrdMap, block.Height)

	common.Log.Infof("processOrdProtocol %d,is done: cost: %v", block.Height, time.Since(measureStartTime))
}

func findOutputWithSat(tx *common.Transaction, sat int64) *common.Output {
	for _, out := range tx.Outputs {
		for _, rng := range out.Ordinals {
			if sat >= rng.Start && sat < rng.Start+rng.Size {
				return out
			}
		}
	}
	return nil
}

func (s *IndexerMgr) handleDeployTicker(rngs []*common.Range, satpoint int, out *common.Output,
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
		if !s.isEligibleUser(out.Address.Addresses[0], content.Des) {
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
	}

	newRngs := reAlignRange(rngs, satpoint, 1)
	if len(newRngs) == 0 {
		common.Log.Warnf("IndexerMgr.handleDeployTicker: inscriptionId: %s, ticker: %s, satpoint %d ",
			nft.Base.InscriptionId, content.Ticker, satpoint)
		return nil
	}

	// 确保newRngs都在output中
	if !common.RangesContained(out.Ordinals, newRngs) {
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

func (s *IndexerMgr) handleMintTicker(rngs []*common.Range, satpoint int, out *common.Output,
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
	addressId := s.compiling.GetAddressId(out.Address.Addresses[0])
	permitAmt := s.getMintAmount(deployTicker.Name, addressId)
	if amt > permitAmt {
		common.Log.Warnf("IndexerMgr.handleMintTicker: inscriptionId: %s, ticker: %s, invalid amt: %s",
			inscriptionId, content.Ticker, content.Amt)
		return nil
	}

	var newRngs []*common.Range
	satsNum := int64(amt) / int64(deployTicker.N)
	var sat int64 = nft.Base.Sat
	if indexer.IsRaritySatRequired(&deployTicker.Attr) {
		// check trz=N
		if deployTicker.Attr.TrailingZero > 0 {
			if satsNum != 1 || !indexer.EndsWithNZeroes(deployTicker.Attr.TrailingZero, nft.Base.Sat) {
				common.Log.Warnf("IndexerMgr.handleMintTicker: inscriptionId: %s, ticker: %s, invalid sat: %d, trailingZero: %d",
					inscriptionId, content.Ticker, nft.Base.Sat, deployTicker.Attr.TrailingZero)
				return nil
			}
		}

		newRngs = skipOffsetRange(rngs, satpoint)
		// check rarity
		if deployTicker.Attr.Rarity != "" {
			exoticranges := s.exotic.GetExoticsWithType(newRngs, deployTicker.Attr.Rarity)
			size := int64(0)
			newRngs2 := make([]*common.Range, 0)
			for _, exrng := range exoticranges {
				size += exrng.Range.Size
				newRngs2 = append(newRngs2, exrng.Range)

			}
			if size < (satsNum) {
				common.Log.Warnf("IndexerMgr.handleMintTicker: inscriptionId: %s, ticker: %s, invalid sat: %d, size %d, rarity: %s",
					inscriptionId, content.Ticker, sat, size, deployTicker.Attr.Rarity)
				return nil
			}
			newRngs = newRngs2
		}
		newRngs = reSizeRange(newRngs, satsNum)
	} else {
		newRngs = reAlignRange(rngs, satpoint, satsNum)
	}

	if len(newRngs) == 0 || common.GetOrdinalsSize(newRngs) != satsNum {
		common.Log.Warnf("IndexerMgr.handleMintTicker: inscriptionId: %s, ticker: %s, amt(%d), no enough sats %d",
			inscriptionId, content.Ticker, satsNum, common.GetOrdinalsSize(newRngs))
		return nil
	}

	// 铸造结果：从指定的nft，往后如果有satsNum个聪，就是铸造成功，这些聪都是输入的一部分就可以，输出在哪里无所谓
	// // 确保newRngs都在output中
	// if !common.RangesContained(out.Ordinals, newRngs) {
	// 	common.Log.Warnf("IndexerMgr.handleMintTicker: inscriptionId: %s, ticker: %s, ranges not in output",
	// 		inscriptionId, content.Ticker)
	// 	return nil
	// }

	// 禁止在同一个聪上做同样名字的铸造
	if s.hasSameTickerInRange(content.Ticker, newRngs) {
		common.Log.Warnf("IndexerMgr.handleMintTicker: inscriptionId: %s, ticker: %s, ranges has same ticker",
			inscriptionId, content.Ticker)
		return nil
	}

	nft.Base.TypeName = common.ASSET_TYPE_FT
	mint := &common.Mint{
		Base:     common.CloneBaseContent(nft.Base),
		Name:     content.Ticker,
		Ordinals: newRngs,
		Amt:      int64(amt),
		Desc:     content.Des,
	}

	return mint
}

func (s *IndexerMgr) handleBrc20DeployTicker(rngs []*common.Range, satpoint int, out *common.Output,
	content *common.BRC20DeployContent, nft *common.Nft) *common.BRC20Ticker {

	ticker := &common.BRC20Ticker{
		Nft:  nft,
		Name: content.Ticker,
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
		if err != nil || dec > common.MAX_PRECISION {
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

func (s *IndexerMgr) handleBrc20MintTicker(rngs []*common.Range, satpoint int, out *common.Output,
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

	mint := &common.BRC20Mint{Nft: nft, Name: content.Ticker}

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
		common.Log.Warnf("mint %s, invalid amount(%s), max(%s), change to %s", content.Ticker, content.Amt, ticker.Max.String(), amt.String())
	}
	mint.Amt = *amt
	return mint
}

func (s *IndexerMgr) handleBrc20TransferTicker(rngs []*common.Range, satpoint int, out *common.Output,
	content *common.BRC20TransferContent, nft *common.Nft) *common.BRC20Transfer {
	inscriptionId := nft.Base.InscriptionId
	ticker := s.brc20Indexer.GetTicker(content.Ticker)
	if ticker == nil {
		common.Log.Warnf("IndexerMgr.handleBrc20TransferTicker: inscriptionId: %s, ticker: %s, no deploy ticker",
			inscriptionId, content.Ticker)
		return nil
	}

	transfer := &common.BRC20Transfer{
		Nft:  nft,
		Name: content.Ticker,
		// UtxoId: nft.UtxoId,
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

func (s *IndexerMgr) handleOrdX(inUtxoId uint64, input []*common.Range, satpoint int, out *common.Output,
	fields map[int][]byte, nft *common.Nft) {
	ordxInfo, bOrdx := common.IsOrdXProtocol(fields)
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

		ticker := s.handleDeployTicker(input, satpoint, out, deployInfo, nft)
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

		mint := s.handleMintTicker(input, satpoint, out, mintInfo, nft)
		if mint == nil {
			return
		}

		s.ftIndexer.UpdateMint(inUtxoId, mint)

	default:
		//common.Log.Warnf("handleOrdX unknown ordx type: %s, content: %s, txid: %s", ordxType, content, tx.Txid)
	}
}

func (s *IndexerMgr) handleBrc20(inUtxoId uint64, input []*common.Range, satpoint int, out *common.Output,
	fields map[int][]byte, nft *common.Nft) {

	content := string(fields[common.FIELD_CONTENT])
	ordxBaseContent := common.ParseBrc20BaseContent(content)
	if ordxBaseContent == nil {
		common.Log.Errorf("invalid content %s", content)
		return
	}

	if common.GetOrdinalsSize(out.Ordinals) == 0 {
		common.Log.Errorf("invalid brc20 inscription %s", nft.Base.InscriptionId)
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

		ticker := s.handleBrc20DeployTicker(input, satpoint, out, deployInfo, nft)
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

		mint := s.handleBrc20MintTicker(input, satpoint, out, mintInfo, nft)
		if mint == nil {
			return
		}
		common.Log.Infof("nft.Base.InscriptionId: %s, nft.Base.Id: %d", nft.Base.InscriptionId, nft.Base.Id)
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

		transfer := s.handleBrc20TransferTicker(input, satpoint, out, transferInfo, nft)
		if transfer == nil {
			return
		}

		s.brc20Indexer.UpdateInscribeTransfer(transfer)

	default:
		//common.Log.Warnf("handleOrdX unknown ordx type: %s, content: %s, txid: %s", ordxType, content, tx.Txid)
	}
}

func (s *IndexerMgr) handleOrd(input *common.Input,
	fields map[int][]byte, inscriptionId int, envelope []byte, tx *common.Transaction, block *common.Block) {

	satpoint := 0
	if fields[common.FIELD_POINT] != nil {
		satpoint = common.GetSatpoint(fields[common.FIELD_POINT])
		if int64(satpoint) >= common.GetOrdinalsSize(input.Ordinals) {
			satpoint = 0
		}
	}

	var output *common.Output
	sat := getSatInRange(input.Ordinals, satpoint)
	if sat > 0 {
		output = findOutputWithSat(tx, sat)
		if output == nil {
			output = findOutputWithSat(block.Transactions[0], sat)
			if output == nil {
				common.Log.Errorf("processOrdProtocol: tx: %s, findOutputWithSat %d failed", tx.Txid, sat)
				return
			}
		}
	} else {
		// 99e70421ab229d1ccf356e594512da6486e2dd1abdf6c2cb5014875451ee8073:0  788312
		// c1e0db6368a43f5589352ed44aa1ff9af33410e4a9fd9be0f6ac42d9e4117151:0  788200
		// 输入为0，输出也只有一个，也为0

		output = tx.Outputs[0]
	}

	// 1. 先保存nft数据
	nft := s.handleNft(input, output, satpoint, fields, inscriptionId, tx, block)
	if nft == nil {
		return
	}

	if len(input.Ordinals) == 0 {
		// 虽然ordinals.com解析出了这个交易，但是我们认为该交易没有输入的sat，也就是无法将数据绑定到某一个sat上，违背了协议原则
		// 特殊交易，ordx不支持，不处理
		// c1e0db6368a43f5589352ed44aa1ff9af33410e4a9fd9be0f6ac42d9e4117151
		// TODO 0605版本中，没有把这个nft编译进来
		return
	}

	// 2. 再看看是否ordx协议
	protocol, content := common.GetProtocol(fields)
	switch protocol {
	case "ordx":
		s.handleOrdX(input.UtxoId, input.Ordinals, satpoint, output, fields, nft)
	case "sns":
		domain := common.ParseDomainContent(string(fields[common.FIELD_CONTENT]))
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
				if string(fields[common.FIELD_META_PROTOCOL]) == "sns" && fields[common.FIELD_META_DATA] != nil {
					updateInfo = common.ParseUpdateContent(string(content))
					updateInfo.P = "sns"
					value, ok := updateInfo.KVs["key"]
					if ok {
						delete(updateInfo.KVs, "key")
						updateInfo.KVs[value] = nft.Base.InscriptionId
					}
				} else {
					updateInfo = common.ParseUpdateContent(string(fields[common.FIELD_CONTENT]))
				}

				if updateInfo == nil {
					return
				}
				s.handleNameUpdate(updateInfo, nft)
			}
		}
	case "brc-20":
		if s.IsMainnet() && s.brc20Indexer.IsExistCursorInscriptionInDB(nft.Base.InscriptionId) {
			return
		}
		if brc20.IsCursed(envelope, inscriptionId, block.Height) {
			common.Log.Infof("%s inscription is cursed", nft.Base.InscriptionId)
			return
		}

		s.handleBrc20(input.UtxoId, input.Ordinals, satpoint, output, fields, nft)
		// if inscriptionId == 0 {
		// TODO brc20 只处理tx中的第一个铭文？
		// s.handleBrc20(input.UtxoId, input.Ordinals, satpoint, output, fields, nft)
		// }

	case "primary-name":
		primaryNameContent := common.ParseCommonContent(string(fields[common.FIELD_CONTENT]))
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
		commonContent := common.ParseCommonContent(string(fields[common.FIELD_CONTENT]))
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
			s.handleSnsName(string(fields[common.FIELD_CONTENT]), nft)
		}
	}

}

func (s *IndexerMgr) handleSnsName(name string, nft *common.Nft) {
	if common.IsValidSNSName(name) {
		info := s.ns.GetNameRegisterInfo(name)
		if info != nil {
			common.Log.Warnf("%s Name %s exist, registered at %s",
				nft.Base.InscriptionId, name, info.Nft.Base.InscriptionId)
			return
		}

		regInfo := &common.OrdxRegContent{
			OrdxBaseContent: common.OrdxBaseContent{P: "sns", Op: "reg"},
			Name:            name}

		s.handleNameRegister(regInfo, nft)
	}
}

func (s *IndexerMgr) handleNft(input *common.Input, output *common.Output, satpoint int,
	fields map[int][]byte, inscriptionId int, tx *common.Transaction, block *common.Block) *common.Nft {

	//if s.nft.Base.IsEnabled() {
	sat := int64(-1)
	if len(input.Ordinals) > 0 {
		newRngs := reAlignRange(input.Ordinals, satpoint, 1)
		sat = newRngs[0].Start
	}

	//addressId1 := s.compiling.GetAddressId(input.Address.Addresses[0])
	addressId2 := s.compiling.GetAddressId(output.Address.Addresses[0])
	utxoId := common.GetUtxoId(output)
	nft := common.Nft{
		Base: &common.InscribeBaseContent{
			InscriptionId:      tx.Txid + "i" + strconv.Itoa(inscriptionId),
			InscriptionAddress: addressId2, // TODO 这个地址不是铭刻者，模型的问题，比较难改，直接使用输出地址
			BlockHeight:        int32(block.Height),
			BlockTime:          block.Timestamp.Unix(),
			ContentType:        (fields[common.FIELD_CONTENT_TYPE]),
			Content:            fields[common.FIELD_CONTENT],
			ContentEncoding:    fields[common.FIELD_CONTENT_ENCODING],
			MetaProtocol:       (fields[common.FIELD_META_PROTOCOL]),
			MetaData:           fields[common.FIELD_META_DATA],
			Parent:             common.ParseInscriptionId(fields[common.FIELD_PARENT]),
			Delegate:           common.ParseInscriptionId(fields[common.FIELD_DELEGATE]),
			Sat:                sat,
			TypeName:           common.ASSET_TYPE_NFT,
		},
		OwnerAddressId: addressId2,
		UtxoId:         utxoId,
	}
	s.nft.NftMint(&nft)
	return &nft
	// }
	// return nil
}

func getSatInRange(common []*common.Range, satpoint int) int64 {
	for _, rng := range common {
		if satpoint >= int(rng.Size) {
			satpoint -= int(rng.Size)
		} else {
			return rng.Start + int64(satpoint)
		}
	}

	return -1
}

func (s *IndexerMgr) hasSameTickerInRange(ticker string, rngs []*common.Range) bool {
	for _, rng := range rngs {
		if s.ftIndexer.CheckTickersWithSatRange(ticker, rng) {
			return true
		}
	}
	return false
}

func (s *IndexerMgr) getMintAmountByAddressId(ticker string, address uint64) int64 {
	return s.ftIndexer.GetMintAmountWithAddressId(address, ticker)
}

// 有资格的地址：跟引导节点建立了通道，而且该通道持有足够的资产
func (s *IndexerMgr) isEligibleUser(address, pubkey string) bool {
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
