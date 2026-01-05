package brc20

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
	inCommon "github.com/sat20-labs/indexer/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func (s *BRC20Indexer) CheckInscription(nft *common.Nft) bool {
	if nft.Base.CurseType != 0 {
		common.Log.Debugf("%s inscription is cursed, %d", nft.Base.InscriptionId, nft.Base.CurseType)
		return false
	}

	if nft.Base.Reinscription != 0 {
		return false
	}

	return true
}

// deploy
func (s *BRC20Indexer) UpdateInscribeDeploy(input *common.TxInput, ticker *common.BRC20Ticker) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.actionBufferMap[input.UtxoId] = &ActionInfo{
		Action: common.BRC20_Action_InScribe_Deploy,
		Info:   ticker,
	}
}

func (s *BRC20Indexer) updateInscribeDeploy(input *common.TxInput, ticker *common.BRC20Ticker) {
	// 再次检查，因为nft可能会修改reinsription状态
	nft := ticker.Nft
	if !s.CheckInscription(nft) {
		common.Log.Debugf("%s inscription is invalid", nft.Base.InscriptionId)
		return
	}

	name := strings.ToLower(ticker.Name)

	ticker.Id = int64(s.status.TickerCount)
	s.status.TickerCount++
	ticker.TransactionCount++
	tickinfo := newTickerInfo(name)
	tickinfo.Ticker = ticker
	s.tickerMap[name] = tickinfo
	s.tickerAdded = append(s.tickerAdded, ticker)
	s.tickerUpdated[name] = ticker

	action := HolderAction{
		Height:     int(ticker.Nft.Base.BlockHeight),
		TxIndex:    input.InTxIndex,
		TxInIndex:  input.TxInIndex,
		NftId:      ticker.Nft.Base.Id,
		FromAddr:   common.INVALID_ID,
		FromUtxoId: input.UtxoId,
		ToAddr:     ticker.Nft.OwnerAddressId,
		ToUtxoId:   ticker.Nft.UtxoId,
		Ticker:     name,
		Amount:     *common.NewDefaultDecimal(0),
		Action:     common.BRC20_Action_InScribe_Deploy,
	}
	s.holderActionList = append(s.holderActionList, &action)

	common.Log.Debugf("mint-deploy %d: %x deploy ticker = %s",
		ticker.Nft.UtxoId, ticker.Nft.OwnerAddressId, ticker.Name)
}

// mint
func (s *BRC20Indexer) UpdateInscribeMint(input *common.TxInput, mint *common.BRC20Mint) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// if !s.nftIndexer.GetBaseIndexer().IsMainnet() {
	// 	var excludingInscriptions = map[string]bool {
	// 		// "06ccbac09747a62b2b6c000786b0c5c34ded98d50e8e84986f1c4884bc60e461i0": true, // vindicated
	// 		// "23316f7a5e793bed487ece25aeae12aff283403687e024b518eb9c6fe73991a1i0": true, // reinscription
	// 	}
	// 	if _, ok := excludingInscriptions[mint.Nft.Base.InscriptionId]; ok {
	// 		return
	// 	}
	// }

	s.actionBufferMap[input.UtxoId] = &ActionInfo{
		Action: common.BRC20_Action_InScribe_Mint,
		Info:   mint,
	}
}

func (s *BRC20Indexer) updateInscribeMint(input *common.TxInput, mint *common.BRC20Mint) {
	if !s.CheckInscription(mint.Nft) {
		common.Log.Debugf("%s inscription is invalid", mint.Nft.Base.InscriptionId)
		return
	}

	name := strings.ToLower(mint.Name)
	tickerInfo := s.tickerMap[name]
	if tickerInfo == nil {
		return
	}
	ticker := tickerInfo.Ticker
	if ticker.EndInscriptionId != "" {
		// 已经足够了
		return
	}

	ticker.TransactionCount++
	if common.DecimalAdd(&ticker.Minted, &mint.Amt).Cmp(&ticker.Max) >= 0 {
		mint.Amt = *ticker.Max.Sub(&ticker.Minted)
		ticker.EndInscriptionId = mint.Nft.Base.InscriptionId
	}
	ticker.Minted = *ticker.Minted.Add(&mint.Amt)
	s.tickerUpdated[name] = ticker

	mint.Id = int64(ticker.MintCount)
	ticker.MintCount++
	tickerInfo.MintAdded = append(tickerInfo.MintAdded, mint)

	s.loadHolderInfo(mint.Nft.OwnerAddressId, name)

	// 作为invalid的output
	nft := common.TransferNFT{
		NftId:     mint.Nft.Base.Id,
		Id:        mint.Id,
		UtxoId:    mint.Nft.UtxoId,
		Amount:    mint.Amt,
		IsInvalid: true,
	}
	transferInfo := &TransferNftInfo{
		AddressId:   mint.Nft.OwnerAddressId,
		Ticker:      name,
		UtxoId:      mint.Nft.UtxoId,
		TransferNft: &nft,
	}
	s.addTransferNft(transferInfo)

	action := HolderAction{
		Height:     int(mint.Nft.Base.BlockHeight),
		TxIndex:    input.InTxIndex,
		TxInIndex:  input.TxInIndex,
		NftId:      mint.Nft.Base.Id,
		FromAddr:   common.INVALID_ID,
		FromUtxoId: input.UtxoId,
		ToAddr:     mint.Nft.OwnerAddressId,
		ToUtxoId:   mint.Nft.UtxoId,
		Ticker:     name,
		Amount:     *mint.Amt.Clone(),
		Action:     common.BRC20_Action_InScribe_Mint,
	}
	s.holderActionList = append(s.holderActionList, &action)

	s.addHolderBalance(transferInfo, mint.Nft.OwnerAddressId)
	common.Log.Debugf("mint-mint %d: %x mint ticker = %s +%s -> %s",
		mint.NftId, mint.Nft.OwnerAddressId, mint.Name, mint.Amt.String(),
		ticker.Minted.String())
}

// transfer
func (s *BRC20Indexer) UpdateInscribeTransfer(input *common.TxInput, transfer *common.BRC20Transfer) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.nftIndexer.GetBaseIndexer().IsMainnet() {
		// 忽略某些不知道什么原因，unisat认为是无效的brc20铭文，仅仅为了让校验数据可以正确匹配
		var excludingInscriptions = map[string]bool{
			// "ea7e189806151db47817a8ab8a114a3b600be927705ad4b243af72701ee0ede4i0": true, // reinscription
		}
		if _, ok := excludingInscriptions[transfer.Nft.Base.InscriptionId]; ok {
			return
		}
	}

	s.actionBufferMap[input.UtxoId] = &ActionInfo{
		Action: common.BRC20_Action_InScribe_Transfer,
		Info:   transfer,
	}
}

func (s *BRC20Indexer) updateInscribeTransfer(input *common.TxInput, transfer *common.BRC20Transfer) {
	if !s.CheckInscription(transfer.Nft) {
		common.Log.Debugf("%s inscription is invalid", transfer.Nft.Base.InscriptionId)
		return
	}
	// if transfer.Nft.Base.InscriptionId == "b73abee8f9cf28bf58bc45476b32ff1f4bcb01f58d0101ea83a094acdeb73cff" {
	// 	common.Log.Infof("")
	// }

	// transfer铭文，实质是将一定数量的token绑定到一个utxo中，所以一个utxo中如果有多个transfer铭文，只有第一个transfer铭文有效
	// 比如：mainnet：e298f7bbea84879bc69a711e27302fc05e34c41fe5d054f22ce0aa1fc8e20add，批量生成多个transfer铭文，只有第一个有效
	_, ok := s.transferNftMap[transfer.Nft.UtxoId]
	if ok {
		common.Log.Debugf("%s inscription is ignored", transfer.Nft.Base.InscriptionId)
		return
	}

	tickerName := strings.ToLower(transfer.Name)
	addressId := transfer.Nft.OwnerAddressId
	holder := s.loadHolderInfo(addressId, tickerName)

	// if transfer.Nft.Base.InscriptionId == "68d47b73d41efc180dec3e2368f373ebe7c52bb48689dcbc8972211a867210f9i0" {
	// 	common.Log.Infof("nftId = %d, utxoId = %d", transfer.Nft.Base.Id, transfer.Nft.UtxoId)
	// }//3804104075771904

	tickAbbrInfo := holder.Tickers[tickerName]
	if tickAbbrInfo == nil {
		return
	}
	if transfer.Amt.Cmp(tickAbbrInfo.AvailableBalance) > 0 {
		// 同一个区块先send出去的资产数量要扣除： e616b84d9917f81de1177e10eaa78617f4b66c0d65c89e6ccebe03b544570e1fi0，前面有4个send
		return
	}
	tickAbbrInfo.AvailableBalance = tickAbbrInfo.AvailableBalance.Sub(&transfer.Amt)
	tickAbbrInfo.TransferableBalance = tickAbbrInfo.TransferableBalance.Add(&transfer.Amt)

	ticker := s.tickerMap[tickerName]
	transfer.Id = int64(ticker.Ticker.TransactionCount)
	ticker.Ticker.TransactionCount++
	s.tickerUpdated[tickerName] = ticker.Ticker

	nft := common.TransferNFT{
		NftId:     transfer.Nft.Base.Id,
		Id:        transfer.Id,
		UtxoId:    transfer.Nft.UtxoId,
		Amount:    transfer.Amt,
		IsInvalid: false,
	}
	transferInfo := &TransferNftInfo{
		AddressId:   transfer.Nft.OwnerAddressId,
		Ticker:      tickerName,
		UtxoId:      transfer.Nft.UtxoId,
		TransferNft: &nft,
	}
	s.addTransferNft(transferInfo)

	action := HolderAction{
		Height:     int(transfer.Nft.Base.BlockHeight),
		TxIndex:    input.InTxIndex,
		TxInIndex:  input.TxInIndex,
		NftId:      transfer.Nft.Base.Id,
		FromAddr:   common.INVALID_ID,
		FromUtxoId: input.UtxoId,
		ToAddr:     transfer.Nft.OwnerAddressId,
		ToUtxoId:   transfer.Nft.UtxoId,
		Ticker:     tickerName,
		Amount:     *transfer.Amt.Clone(),
		Action:     common.BRC20_Action_InScribe_Transfer,
	}
	s.holderActionList = append(s.holderActionList, &action)

	common.Log.Debugf("mint-transfer %d: %x transfer ticker = %s +%s -> %s",
		transfer.NftId, transfer.Nft.OwnerAddressId, transfer.Name,
		transfer.Amt.String(), tickAbbrInfo.TransferableBalance.String())
}

func (s *BRC20Indexer) UpdateTransfer(block *common.Block, coinbase []*common.Range) {

	startTime := time.Now()
	s.PrepareUpdateTransfer(block, coinbase)

	// 检查transferNft转入到哪个输出
	for txIndex, tx := range block.Transactions {
		if txIndex == 0 {
			continue
		}
		s.TxInputProcess(txIndex, tx, block, coinbase)
	}

	common.Log.Infof("BRC20Indexer->UpdateTransfer loop %d in %v", len(block.Transactions), time.Since(startTime))
	s.UpdateTransferFinished(block)

	if inCommon.STEP_RUN_MODE && inCommon.CHECK_SELF && !s.CheckSelf(block.Height) {
		common.Log.Panic("")
	}
}

// 增加该address下的资产数据
func (s *BRC20Indexer) addHolderBalance(transfer *TransferNftInfo, address uint64) {

	tickerName := transfer.Ticker
	amt := &transfer.TransferNft.Amount

	info, ok := s.holderMap[address]
	if !ok {
		info = &HolderInfo{Tickers: make(map[string]*common.BRC20TickAbbrInfo)}
		s.holderMap[address] = info
	}
	tickAbbrInfo, ok := info.Tickers[tickerName]
	if !ok {
		tickAbbrInfo = common.NewBRC20TickAbbrInfo(nil, nil)
		info.Tickers[tickerName] = tickAbbrInfo
	}
	tickAbbrInfo.AvailableBalance = tickAbbrInfo.AvailableBalance.Add(amt)

	if tickAbbrInfo.AssetAmt().Cmp(amt) == 0 {
		ticker := s.tickerMap[tickerName].Ticker
		ticker.HolderCount++
		s.tickerUpdated[tickerName] = ticker
	}

	common.Log.Debugf("add %d: %x %s: +%s -> %s (%s, %s)", transfer.TransferNft.NftId, address, tickerName, amt.String(),
		tickAbbrInfo.AssetAmt().String(), tickAbbrInfo.AvailableBalance.String(), tickAbbrInfo.TransferableBalance.String())

	// holders, ok := s.tickerToHolderMap[tickerName]
	// if !ok {
	// 	holders = make(map[uint64]bool, 0)
	// }
	// holders[address] = true
	// s.tickerToHolderMap[tickerName] = holders
}

// 减少该address下的资产数据
func (s *BRC20Indexer) subHolderBalance(transfer *TransferNftInfo, address uint64) {

	// if address == 1378943947 && tickerName == "meme" {
	// 	common.Log.Infof("")
	// }

	tickerName := transfer.Ticker
	amt := &transfer.TransferNft.Amount

	holdInfo, ok := s.holderMap[address]
	if !ok {
		common.Log.Errorf("%d subHolderBalance can't find holder %x", transfer.TransferNft.NftId, address)
		s.printHistoryWithAddress(tickerName, address)
		common.Log.Panic("")
	}
	tickAbbrInfo, ok := holdInfo.Tickers[tickerName]
	if !ok {
		common.Log.Errorf("%d subHolderBalance %x can't find ticker %s", transfer.TransferNft.NftId, address, tickerName)
		s.printHistoryWithAddress(tickerName, address)
		common.Log.Panic("")
	}
	if tickAbbrInfo.TransferableBalance.Cmp(amt) < 0 {
		common.Log.Errorf("%d subHolderBalance %x %s not enough transfer amt, require %s but only %s",
			transfer.TransferNft.NftId, address, tickerName, amt.String(), tickAbbrInfo.TransferableBalance.String())
		s.printHistoryWithAddress(tickerName, address)
		common.Log.Panic("")
	}

	tickAbbrInfo.TransferableBalance = tickAbbrInfo.TransferableBalance.Sub(amt)
	common.Log.Debugf("sub %d: %x %s: -%s -> %s (%s, %s)", transfer.TransferNft.NftId, address, tickerName, amt.String(),
		tickAbbrInfo.AssetAmt().String(), tickAbbrInfo.AvailableBalance.String(), tickAbbrInfo.TransferableBalance.String())

	balance := tickAbbrInfo.AssetAmt()
	if balance.Sign() == 0 {
		// holders := s.tickerToHolderMap[tickerName]
		// delete(holders, address)
		// if len(holders) == 0 {
		// 	delete(s.tickerToHolderMap, tickerName)
		// } else {
		// 	s.tickerToHolderMap[tickerName] = holders
		// }
		ticker := s.tickerMap[tickerName].Ticker
		ticker.HolderCount--
		s.tickerUpdated[tickerName] = ticker

		// 可能有invalid的transfer nft，所以不要删除
		// delete(holdInfo.Tickers, tickerName)
		// if len(holdInfo.Tickers) == 0 {
		// 	delete(s.holderMap, address)
		// }
	}
}

// 将一个transfer nft取消，原因可能是作为手续费转给了矿工
func (s *BRC20Indexer) cancelTransferNft(transfer *TransferNftInfo, height, index int,
	tx *common.Transaction) {

	fromAddress := transfer.AddressId

	ticker := s.tickerMap[transfer.Ticker]
	ticker.Ticker.TransactionCount++
	s.tickerUpdated[transfer.Ticker] = ticker.Ticker

	// 当作发送出去，接受者是自己

	s.removeTransferNft(transfer)
	s.subHolderBalance(transfer, fromAddress)

	s.addHolderBalance(transfer, fromAddress)
	//s.addTransferNft(transfer)

	// 检查该transfer nft最后输出到哪个utxoId
	nft := s.nftIndexer.GetNftWithIdWithNoLock(transfer.TransferNft.Id)

	action := HolderAction{
		Height:     height,
		TxIndex:    index,
		TxInIndex:  transfer.TxInIndex,
		NftId:      transfer.TransferNft.NftId,
		FromUtxoId: transfer.UtxoId, // old utxo
		FromAddr:   fromAddress,
		ToAddr:     fromAddress,
		ToUtxoId:   nft.UtxoId,
		Ticker:     transfer.Ticker,
		Amount:     *transfer.TransferNft.Amount.Clone(),
		Action:     common.BRC20_Action_Transfer,
	}
	s.holderActionList = append(s.holderActionList, &action)

	common.Log.Debugf("cancel %d: %x -> %x, ticker = %s, %s",
		action.NftId, action.FromAddr, action.ToAddr, action.Ticker, action.Amount.String())

}

// 需要先加载holderInfo
func (s *BRC20Indexer) removeTransferNft(nft *TransferNftInfo) {

	// if nft.TransferNft.NftId == 1058797 || nft.AddressId == 1378943947 {
	// 	common.Log.Infof("")
	// }

	delete(s.transferNftMap, nft.UtxoId)

	holder, ok := s.holderMap[nft.AddressId]
	if ok {
		tickInfo, ok := holder.Tickers[nft.Ticker]
		if ok {
			delete(tickInfo.TransferableData, nft.UtxoId)
			// if tickInfo.AssetAmt().Sign() == 0 &&
			// 	len(tickInfo.TransferableData) == 0 {
			// 	delete(holder.Tickers, nft.Ticker) // 不能删除，删除会导致 loadHolderInfo 重新加载老的数据
			// }
			// 不能删除，如果删除，就无法删除数据库中对应数据 updateHolderToDB
			// if len(holder.Tickers) == 0 {
			// 	delete(s.holderMap, nft.AddressId)
			// }
		} else {
			// 已经转移过的transfer nft不一定能找到
			// common.Log.Panicf("can't find ticker info %s %d", nft.Ticker, nft.UtxoId)
		}
	} else {
		// 已经转移过的transfer nft不一定能找到
		// common.Log.Panic("can't find ticker info")
	}
}

// 需要先加载holderInfo
func (s *BRC20Indexer) addTransferNft(nft *TransferNftInfo) {
	curr, ok := s.transferNftMap[nft.UtxoId]
	if ok {
		// 多个transfer输出到同一个utxo，这个时候只修改amt，不会影响余额，因为仅用于记录，不用于账户计算
		curr.TransferNft.Amount = *curr.TransferNft.Amount.Add(&nft.TransferNft.Amount)
	} else {
		curr = nft
		s.transferNftMap[nft.UtxoId] = curr
	}

	holder, ok := s.holderMap[nft.AddressId]
	if !ok {
		// 这个nft是一个已经使用过的铭文，但为了继续记录在数据库，还是要保留在holdermap中
		holder = &HolderInfo{
			Tickers: make(map[string]*common.BRC20TickAbbrInfo),
		}
		s.holderMap[nft.AddressId] = holder
	}

	tickAbbrInfo, ok := holder.Tickers[nft.Ticker]
	if !ok {
		tickAbbrInfo = common.NewBRC20TickAbbrInfo(nil, nil)
		holder.Tickers[nft.Ticker] = tickAbbrInfo
	}
	tickAbbrInfo.TransferableData[nft.UtxoId] = curr.TransferNft
}

func (s *BRC20Indexer) innerUpdateTransfer(index int, txId string, output *common.TxOutputV2,
	inputTransferNfts map[int64]*TransferNftInfo) {
	// 检查是否存在nft。如果存在，就更新对应的holder数据
	utxoId := output.UtxoId
	// if txId == "672f335b384bc15e9a05cc6583e6c586e77a82e9ac1eda7960efe2af372514f5" {
	// 	common.Log.Infof("")
	// }
	// ids := s.nftIndexer.GetNftsWithUtxo(utxoId) // 有可能多个transfer nft，合并输出到一个output中
	//for _, nft := range ids {

	for _, asset := range output.Assets {
		if asset.Name.Protocol != common.PROTOCOL_NAME_ORD ||
			asset.Name.Type != common.ASSET_TYPE_NFT {
			continue
		}
		sat, err := strconv.ParseInt(asset.Name.Ticker, 10, 64)
		if err != nil {
			common.Log.Panicf("innerUpdateTransfer3 ParseInt %s failed, %v", asset.Name.Ticker, err)
		}
		ids := s.nftIndexer.GetNftsWithSatNoDisable(sat)
		for _, nftId := range ids.Nfts {

			transfer, ok := inputTransferNfts[nftId]
			if ok {
				fromAddressId := transfer.AddressId
				toAddressId := output.AddressId

				s.removeTransferNft(transfer)
				flag := common.BRC20_Action_Transfer_Spent
				method := "spend"
				oldUtxoId := transfer.UtxoId
				if !transfer.TransferNft.IsInvalid {
					method = "transfer"
					flag = common.BRC20_Action_Transfer
					transfer.TransferNft.IsInvalid = true

					ticker := s.tickerMap[transfer.Ticker]
					ticker.Ticker.TransactionCount++
					s.tickerUpdated[transfer.Ticker] = ticker.Ticker

					// transfer occur
					s.subHolderBalance(transfer, fromAddressId)
					s.addHolderBalance(transfer, toAddressId)

					// 再继续加入s.transferNftMap，方便跟踪。在下一次转移时，可以删除，不需要再记录
					transfer.AddressId = toAddressId
					transfer.UtxoId = utxoId
					s.addTransferNft(transfer)
				}
				action := HolderAction{
					Height:     output.OutHeight,
					TxIndex:    index,
					TxInIndex:  transfer.TxInIndex,
					ToUtxoId:   utxoId,
					NftId:      transfer.TransferNft.NftId,
					FromAddr:   fromAddressId,
					ToAddr:     toAddressId,
					Ticker:     transfer.Ticker,
					Amount:     *transfer.TransferNft.Amount.Clone(),
					Action:     flag,
					FromUtxoId: oldUtxoId, // old utxo
				}
				s.holderActionList = append(s.holderActionList, &action)
				delete(inputTransferNfts, nftId)

				common.Log.Debugf("%s %d: %x -> %x, ticker = %s, %s",
					method, action.NftId, action.FromAddr, action.ToAddr, action.Ticker,
					action.Amount.String())
			}
		}
	}
}

func (s *BRC20Indexer) updateHolderToDB(address uint64, ticker string, writeToDB bool,
	updateTickerAddr bool, wb common.WriteBatch) {
	addressTickerKey := GetHolderInfoKey(address, ticker)
	tickerAddressKey := GetTickerToHolderKey(ticker, address)
	holder, ok := s.holderMap[address]
	if ok {
		value, ok := holder.Tickers[ticker]
		if ok {
			amt := value.AssetAmt()
			if writeToDB {
				if amt.Sign() == 0 && len(value.TransferableData) == 0 {
					// 我们保留一个空记录，以便维持该addressId不被删除
					// err := wb.Delete([]byte(addressTickerKey))
					err := db.SetDB([]byte(addressTickerKey), value, wb)
					if err != nil {
						common.Log.Panicf("Error deleting db %s: %v\n", addressTickerKey, err)
					}
				} else {
					err := db.SetDB([]byte(addressTickerKey), value, wb)
					if err != nil {
						common.Log.Panicf("Error setting %s in db %v", addressTickerKey, err)
					}
				}
			}
			if updateTickerAddr {
				if amt.Sign() > 0 {
					if writeToDB {
						err := db.SetDB([]byte(tickerAddressKey), amt, wb)
						if err != nil {
							common.Log.Panicf("Error setting %s in db %v", tickerAddressKey, err)
						}
					}
				} else {
					err := wb.Delete([]byte(tickerAddressKey))
					if err != nil {
						common.Log.Panicf("Error deleting db %s: %v\n", tickerAddressKey, err)
					}
				}
			}
		} else {
			// 我们保留一个空记录，以便维持该addressId不被删除
			value = &common.BRC20TickAbbrInfo{
				AvailableBalance:    nil,
				TransferableBalance: nil,
				TransferableData:    nil,
			}
			err := db.SetDB([]byte(addressTickerKey), value, wb)
			//err := wb.Delete([]byte(addressTickerKey))
			if err != nil {
				common.Log.Panicf("Error deleting db %s: %v\n", addressTickerKey, err)
			}
			//delete(holder.Tickers, ticker) // 防止多次删除

			if updateTickerAddr {
				err = wb.Delete([]byte(tickerAddressKey))
				if err != nil {
					common.Log.Panicf("Error deleting db %s: %v\n", tickerAddressKey, err)
				}
			}
		}
	} else {
		// 可能重入
	}
}

func (s *BRC20Indexer) updateUtxoToDB(utxoId uint64, writeToDB bool, wb common.WriteBatch) {
	transferKey := GetUtxoToTransferKey(utxoId)
	transferNft, ok := s.transferNftMap[utxoId]
	if ok {
		if writeToDB {
			err := db.SetDB([]byte(transferKey), transferNft, wb)
			if err != nil {
				common.Log.Panicf("Error setting %s in db %v", transferKey, err)
			}
		}
	} else {
		err := wb.Delete([]byte(transferKey))
		if err != nil {
			common.Log.Panicf("Error deleting db %s: %v\n", transferKey, err)
		}
	}
}

// 跟basic数据库同步
func (s *BRC20Indexer) UpdateDB() {
	if s.nftIndexer.GetBaseIndexer().GetHeight() < s.enableHeight {
		return
	}

	//common.Log.Infof("BRC20Indexer->UpdateDB start...")
	startTime := time.Now()

	wb := s.db.NewWriteBatch()
	defer wb.Close()

	for _, ticker := range s.tickerUpdated {
		key := GetTickerKey(strings.ToLower(ticker.Name))
		err := db.SetDB([]byte(key), ticker, wb)
		if err != nil {
			common.Log.Panicf("Error setting %s in db %v", key, err)
		}
	}

	// mint history
	for _, ticker := range s.tickerMap {
		for _, v := range ticker.MintAdded {
			key := GetMintHistoryKey(ticker.Name, v.Nft.Base.Id)
			value := &common.BRC20MintInDB{
				Id:    v.Id,
				Name:  v.Name,
				Amt:   v.Amt,
				NftId: v.Nft.Base.Id,
			}
			err := db.SetDB([]byte(key), &value, wb)
			if err != nil {
				common.Log.Panicf("Error setting %s in db %v", key, err)
			}
		}
	}

	// transfer history
	// 因为重复写入一些很大数据，会导致底层数据库缓存panic，这里只删除，不写入，后面统一写入
	for _, action := range s.holderActionList {
		switch action.Action {
		case common.BRC20_Action_InScribe_Mint:
			s.updateHolderToDB(action.ToAddr, action.Ticker, false, true, wb)
			s.updateUtxoToDB(action.ToUtxoId, false, wb)

		case common.BRC20_Action_InScribe_Transfer:
			s.updateHolderToDB(action.ToAddr, action.Ticker, false, false, wb)
			s.updateUtxoToDB(action.ToUtxoId, false, wb)

		case common.BRC20_Action_Transfer:
			s.updateHolderToDB(action.FromAddr, action.Ticker, false, true, wb)
			s.updateUtxoToDB(action.FromUtxoId, false, wb)

			s.updateHolderToDB(action.ToAddr, action.Ticker, false, true, wb)
			s.updateUtxoToDB(action.ToUtxoId, false, wb)

		case common.BRC20_Action_Transfer_Spent:
			s.updateHolderToDB(action.FromAddr, action.Ticker, false, true, wb)
			s.updateUtxoToDB(action.FromUtxoId, false, wb)
		}

		// 保存历史记录
		key := GetTransferHistoryKey(action.Ticker, action.ToUtxoId, action.NftId)
		err := db.SetDB([]byte(key), action, wb)
		if err != nil {
			common.Log.Panicf("Error setting %s in db %v", key, err)
		}

		if action.FromAddr != common.INVALID_ID {
			if action.Action == common.BRC20_Action_Transfer {
				// transfer
				key = GetHolderTransferHistoryKey(action.Ticker, action.FromAddr, action.NftId)
				value := &TransferNftHistory{UtxoId: []uint64{action.FromUtxoId, action.ToUtxoId}}
				err := db.SetDB([]byte(key), value, wb)
				if err != nil {
					common.Log.Panicf("Error setting %s in db %v", key, err)
				}

				key = GetHolderTransferHistoryKey(action.Ticker, action.ToAddr, action.NftId)
				value = &TransferNftHistory{UtxoId: []uint64{action.ToUtxoId}}
				err = db.SetDB([]byte(key), value, wb)
				if err != nil {
					common.Log.Panicf("Error setting %s in db %v", key, err)
				}
			} else {
				// spent
				key = GetHolderTransferHistoryKey(action.Ticker, action.FromAddr, action.NftId)
				value := &TransferNftHistory{UtxoId: []uint64{action.FromUtxoId, action.ToUtxoId}}
				err := db.SetDB([]byte(key), value, wb)
				if err != nil {
					common.Log.Panicf("Error setting %s in db %v", key, err)
				}
			}
		} else {
			// inscribe
			key = GetHolderTransferHistoryKey(action.Ticker, action.ToAddr, action.NftId)
			value := &TransferNftHistory{UtxoId: []uint64{action.ToUtxoId}}
			err := db.SetDB([]byte(key), value, wb)
			if err != nil {
				common.Log.Panicf("Error setting %s in db %v", key, err)
			}
		}

	}
	// 写入最终结果
	for addressId, holder := range s.holderMap {
		for name := range holder.Tickers {
			s.updateHolderToDB(addressId, name, true, true, wb)
		}
	}
	for utxoId := range s.transferNftMap {
		s.updateUtxoToDB(utxoId, true, wb)
	}

	err := db.SetDB([]byte(BRC20_DB_STATUS_KEY), s.status, wb)
	if err != nil {
		common.Log.Panicf("BRC20Indexer->UpdateDB Error setting in db %v", err)
	}

	err = wb.Flush()
	if err != nil {
		common.Log.Panicf("Error ordxwb flushing writes to db %v", err)
	}

	// reset memory buffer
	s.tickerMap = make(map[string]*BRC20TickInfo)
	s.holderMap = make(map[uint64]*HolderInfo)
	s.transferNftMap = make(map[uint64]*TransferNftInfo)
	s.holderActionList = make([]*HolderAction, 0)
	s.tickerAdded = nil
	s.tickerUpdated = make(map[string]*common.BRC20Ticker)

	common.Log.Infof("BRC20Indexer->UpdateDB takse: %v", time.Since(startTime))
}

// 实现新的区块处理接口
func (s *BRC20Indexer) PrepareUpdateTransfer(block *common.Block, coinbase []*common.Range) {
	if block.Height < s.enableHeight {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// if block.Height == 29601 {
	// 	common.Log.Infof("")
	// }

	// pebble 随机读取性能差，调整读的顺序
	// 预加载相关地址的数据: ticker, holder, utxo
	s.db.View(func(txn common.ReadBatch) error {
		// 处理区块涉及到的铸造，将铸造的结果直接放在交易的输出上，跟其他索引器放在输入上有所不同，因为涉及有效性判断
		// 先加载所有ticker
		addressToLoad := make(map[uint64]map[string]bool) // 该地址哪些ticker被影响
		tickerToLoad := make(map[string]bool)
		addingTransfer := make(map[uint64]string)
		for _, item := range s.actionBufferMap {
			var name string
			var addressId uint64
			switch item.Action {
			case common.BRC20_Action_InScribe_Deploy:
				deploy := item.Info.(*common.BRC20Ticker)
				name = strings.ToLower(deploy.Name)
				addressId = deploy.Nft.OwnerAddressId
			case common.BRC20_Action_InScribe_Mint:
				mint := item.Info.(*common.BRC20Mint)
				name = strings.ToLower(mint.Name)
				addressId = mint.Nft.OwnerAddressId
			case common.BRC20_Action_InScribe_Transfer:
				transfer := item.Info.(*common.BRC20Transfer)
				name = strings.ToLower(transfer.Name)
				addressId = transfer.Nft.OwnerAddressId

				addingTransfer[transfer.Nft.UtxoId] = transfer.Name
			}

			tickers, ok := addressToLoad[addressId]
			if !ok {
				tickers = make(map[string]bool)
				addressToLoad[addressId] = tickers
			}
			tickers[name] = true

			_, ok = s.tickerMap[name]
			if ok {
				continue
			}
			tickerToLoad[name] = true
		}

		// 加载mint涉及到地址
		type pair struct {
			utxoId    uint64
			addressId uint64
			tx        *common.Transaction
			ticker    string
		}

		// 预处理区块本身的交易
		utxoToLoad := make([]*pair, 0)
		transferTxMap := make(map[*common.Transaction]map[string]bool) // 该交易影响哪些ticker
		for _, tx := range block.Transactions[1:] {
			for _, input := range tx.Inputs {
				// if tx.TxId == "228e74cf8cfef4be045380f60457b2472fc5a671715beea91e4b954b7b27f022" {
				// 	common.Log.Infof("utxoId = %d", input.UtxoId)
				// }
				nft, ok := s.transferNftMap[input.UtxoId] // 本区块生成的transfer没有在这里面
				if ok {
					if !nft.TransferNft.IsInvalid {
						tickers, ok := transferTxMap[tx]
						if !ok {
							tickers = make(map[string]bool)
							transferTxMap[tx] = tickers
						}
						tickers[nft.Ticker] = true // 影响输出的结果

						tickers, ok = addressToLoad[nft.AddressId]
						if !ok {
							tickers = make(map[string]bool)
							addressToLoad[nft.AddressId] = tickers
						}
						tickers[nft.Ticker] = true
					}
					continue
				}
				ticker, ok := addingTransfer[input.UtxoId]
				if ok {
					tickers, ok := transferTxMap[tx]
					if !ok {
						tickers = make(map[string]bool)
						transferTxMap[tx] = tickers
					}
					tickers[ticker] = true // 影响输出的结果
					continue
				}
				utxoToLoad = append(utxoToLoad, &pair{
					utxoId:    input.UtxoId,
					addressId: input.AddressId,
					tx:        tx,
					ticker:    "",
				})
			}
		}
		// pebble数据库的优化手段: 尽可能将随机读变成按照key的顺序读
		sort.Slice(utxoToLoad, func(i, j int) bool {
			return utxoToLoad[i].utxoId < utxoToLoad[j].utxoId
		})

		for _, v := range utxoToLoad {
			if v.ticker == "" {
				var value TransferNftInfo
				// if v.utxoId == 31980498781208576 {
				// 	common.Log.Infof("")
				// }
				key := GetUtxoToTransferKey(v.utxoId)
				err := db.GetValueFromTxn([]byte(key), &value, txn)
				if err != nil {
					continue // 没有transfer铭文，忽略
				}
				common.Log.Debugf("brc20 preload load transfer nft %d %s from %x %s",
					value.TransferNft.NftId, value.Ticker, v.utxoId, v.tx.TxId)
				v.ticker = value.Ticker
				s.transferNftMap[v.utxoId] = &value

				if !value.TransferNft.IsInvalid {
					tickers, ok := transferTxMap[v.tx]
					if !ok {
						tickers = make(map[string]bool)
						transferTxMap[v.tx] = tickers
					}
					tickers[value.Ticker] = true // 影响输出的结果
				}
			}
			_, ok := s.tickerMap[v.ticker]
			if !ok {
				tickerToLoad[v.ticker] = true
			}

			tickers, ok := addressToLoad[v.addressId]
			if !ok {
				tickers = make(map[string]bool)
				addressToLoad[v.addressId] = tickers
			}
			tickers[v.ticker] = true
		}

		// 对存在资产转移的tx，加载其输出的地址
		for tx, names := range transferTxMap {
			for _, output := range tx.Outputs {
				if output.OutValue.Value == 0 {
					continue
				}
				tickers, ok := addressToLoad[output.AddressId]
				if !ok {
					tickers = make(map[string]bool)
					addressToLoad[output.AddressId] = tickers
				}
				for name := range names {
					tickers[name] = true
					tickerToLoad[name] = true
				}
			}
		}
		addressToLoadVector := make([]*pair, 0)
		for addressId, tickers := range addressToLoad {
			for name := range tickers {
				addressToLoadVector = append(addressToLoadVector, &pair{
					addressId: addressId,
					ticker:    name,
				})
				tickerToLoad[name] = true
			}
		}
		sort.Slice(addressToLoadVector, func(i, j int) bool {
			return addressToLoadVector[i].addressId < addressToLoadVector[j].addressId
		})
		for _, v := range addressToLoadVector {
			holder, ok := s.holderMap[v.addressId]
			if !ok {
				holder = &HolderInfo{
					Tickers: make(map[string]*common.BRC20TickAbbrInfo),
				}
				s.holderMap[v.addressId] = holder
			}
			_, ok = holder.Tickers[v.ticker]
			if ok {
				continue
			}
			var value common.BRC20TickAbbrInfo
			key := GetHolderInfoKey(v.addressId, v.ticker)
			err := db.GetValueFromTxn([]byte(key), &value, txn)
			if err != nil {
				continue
			}
			// 有很多没有资产的数据，这些也加入ticker中
			if value.AssetAmt().Sign() == 0 && len(value.TransferableData) == 0 {
				continue
			}
			holder.Tickers[v.ticker] = &value
		}

		tickerKeys := make([]string, 0, len(tickerToLoad))
		for k := range tickerToLoad {
			if _, ok := s.tickerMap[k]; ok {
				continue
			}
			tickerKeys = append(tickerKeys, GetTickerKey(k))
		}
		sort.Slice(tickerKeys, func(i, j int) bool {
			return tickerKeys[i] < tickerKeys[j]
		})
		for _, key := range tickerKeys {
			var ticker common.BRC20Ticker
			err := db.GetValueFromDB([]byte(key), &ticker, s.db)
			if err != nil {
				// n, _ := parseTickerKey(key)
				// common.Log.Panicf("brc20 preload load ticker %s %s failed, %v", key, n, err)
				// 可能是本区块刚部署的ticker
				continue
			}

			s.tickerMap[strings.ToLower(ticker.Name)] = &BRC20TickInfo{
				Name:   strings.ToLower(ticker.Name),
				Ticker: &ticker,
			}
			//common.Log.Infof("load ticker %s", ticker.Name)
		}

		return nil
	})
}
func (s *BRC20Indexer) TxInputProcess(txIndex int, tx *common.Transaction,
	block *common.Block, coinbase []*common.Range,
) *common.TxOutput {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// if tx.TxId == "672f335b384bc15e9a05cc6583e6c586e77a82e9ac1eda7960efe2af372514f5" {
	// 	common.Log.Infof("utxoId = %d", tx.Outputs[0].UtxoId)
	// }

	inputTransferNfts := make(map[int64]*TransferNftInfo)
	hasTransfer := false
	for index, input := range tx.Inputs {
		// if input.UtxoId == 3804104075509760 || input.UtxoId == 3804104075771904 {
		// 	common.Log.Infof("utxoId = %d", tx.Outputs[0].UtxoId)
		// }

		// 将结果直接放在交易的输出上
		// 按顺序执行每一个动作。 每个input最多只有一个动作。
		item, ok := s.actionBufferMap[input.UtxoId]
		if ok {
			switch item.Action {
			case common.BRC20_Action_InScribe_Deploy:
				deploy := item.Info.(*common.BRC20Ticker)
				name := strings.ToLower(deploy.Name)
				_, ok := s.tickerMap[name]
				if ok {
					continue
				}
				s.updateInscribeDeploy(input, deploy)
			case common.BRC20_Action_InScribe_Mint:
				mint := item.Info.(*common.BRC20Mint)
				s.updateInscribeMint(input, mint)
			case common.BRC20_Action_InScribe_Transfer:
				transfer := item.Info.(*common.BRC20Transfer)
				s.updateInscribeTransfer(input, transfer)
			}
		}

		transfer, ok := s.transferNftMap[input.UtxoId] // transferNftMap 第一次转移时，先不删除，只设置标志位
		if ok {
			transfer.TxInIndex = index
			inputTransferNfts[transfer.TransferNft.NftId] = transfer
			hasTransfer = true
		}
	}

	if hasTransfer {
		for _, output := range tx.Outputs {
			s.innerUpdateTransfer(txIndex, tx.TxId, output, inputTransferNfts)
		}

		// testnet4: 19206e5c580194fce3a513682998e918e40b9c2a2afaa64f63e55a217b7ec023
		// 该交易有很多个transfer nft作为手续费给到了矿工，需要将这些transfer nft作废，
		// 比如其中的一个：3f04ce47dc1ed5fc04243d3282dae6d472111fe584b2318d0715b6a1c9bb9664i0
		if len(inputTransferNfts) != 0 {
			for _, transfer := range inputTransferNfts {
				if !transfer.TransferNft.IsInvalid {
					transfer.TransferNft.IsInvalid = true
					s.cancelTransferNft(transfer, block.Height, txIndex, tx)
				}
			}
		}
	}
	return nil
}

func (s *BRC20Indexer) UpdateTransferFinished(block *common.Block) {
	s.actionBufferMap = make(map[uint64]*ActionInfo)
	s.CheckPointWithBlockHeight(block.Height)
}
