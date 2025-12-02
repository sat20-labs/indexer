package brc20

import (
	"fmt"
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

// deploy
func (s *BRC20Indexer) UpdateInscribeDeploy(ticker *common.BRC20Ticker) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	name := strings.ToLower(ticker.Name)
	org, ok := s.tickerMap[name]
	if !ok {
		ticker.Id = int64(len(s.tickerMap))
		ticker.TransactionCount++
		tickinfo := newTickerInfo(ticker.Name)
		tickinfo.Ticker = ticker
		s.tickerMap[name] = tickinfo
		s.tickerAdded[name] = ticker
	} else {
		// 仅更新显示内容
		s.tickerAdded[name] = org.Ticker
	}

	common.Log.Infof("UpdateInscribeDeploy %s", ticker.Name)
}

// mint
func (s *BRC20Indexer) UpdateInscribeMint(mint *common.BRC20Mint) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	ticker, ok := s.tickerMap[strings.ToLower(mint.Name)]
	if !ok {
		// 正常不会走到这里，除非是数据从中间跑
		return
	}

	ticker.Ticker.TransactionCount++
	mintedAmt := ticker.Ticker.Minted.Add(&mint.Amt)
	ticker.Ticker.Minted = *mintedAmt
	// if strings.ToLower(mint.Name) == "ordi" {
	// 	balanceStr := ticker.Ticker.Minted.String()
	// 	common.Log.Infof("minted:%s, inscriptionId:%s, id:%d", balanceStr, mint.Nft.Base.InscriptionId, mint.Nft.Base.Id)
	// }
	cmpResult := mintedAmt.Cmp(&ticker.Ticker.Max)
	if cmpResult == 0 {
		ticker.Ticker.EndInscriptionId = mint.Nft.Base.InscriptionId
	}
	s.tickerUpdated[strings.ToLower(mint.Name)] = ticker.Ticker

	mint.Id = int64(len(ticker.InscriptionMap))
	ticker.MintAdded = append(ticker.MintAdded, mint)
	ticker.InscriptionMap[mint.Nft.Base.InscriptionId] = common.NewBRC20MintAbbrInfo(mint)

	action := HolderAction{
		Height: int(mint.Nft.Base.BlockHeight),
		// Utxo:     mint.Nft.Base.InscriptionId,
		UtxoId:   mint.Nft.UtxoId,
		NftId:    mint.Nft.Base.Id,
		FromAddr: common.INVALID_ID,
		ToAddr:   mint.Nft.OwnerAddressId,
		Ticker:   mint.Name,
		Amount:   mint.Amt,
		Action:   Action_InScribe_Mint,
	}
	s.holderActionList = append(s.holderActionList, &action)
	s.addHolderBalance(mint.Name, mint.Nft.OwnerAddressId, mint.Amt)
}

// transfer
func (s *BRC20Indexer) UpdateInscribeTransfer(transfer *common.BRC20Transfer) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	addressId := transfer.Nft.OwnerAddressId
	holder := s.holderMap[addressId]
	if holder == nil {
		return
	}

	tickerName := strings.ToLower(transfer.Name)
	tickAbbrInfo := holder.Tickers[tickerName]
	if tickAbbrInfo == nil {
		return
	}
	if transfer.Amt.Cmp(tickAbbrInfo.AvailableBalance) > 0 {
		return
	}
	tickAbbrInfo.AvailableBalance = tickAbbrInfo.AvailableBalance.Sub(&transfer.Amt)
	tickAbbrInfo.TransferableBalance = tickAbbrInfo.TransferableBalance.Add(&transfer.Amt)

	ticker := s.tickerMap[tickerName]
	ticker.Ticker.TransactionCount++
	s.tickerUpdated[tickerName] = ticker.Ticker

	nft := common.TransferNFT{
		NftId:  transfer.Nft.Base.Id,
		UtxoId: transfer.Nft.UtxoId,
		Amount: transfer.Amt,
	}
	transferInfo := &TransferNftInfo{
		AddressId:   transfer.Nft.OwnerAddressId,
		Ticker:      strings.ToLower(transfer.Name),
		UtxoId:      transfer.Nft.UtxoId,
		TransferNft: &nft,
	}
	s.addTransferNft(transferInfo)

	action := HolderAction{
		Height: int(transfer.Nft.Base.BlockHeight),
		// Utxo:     transfer.Nft.Base.InscriptionId,
		UtxoId:   transfer.Nft.UtxoId,
		NftId:    transfer.Nft.Base.Id,
		FromAddr: common.INVALID_ID,
		ToAddr:   transfer.Nft.OwnerAddressId,
		Ticker:   transfer.Name,
		Amount:   transfer.Amt,
		Action:   Action_InScribe_Transfer,
	}
	if action.NftId == 245603 {
		common.Log.Info("action.NftId == 245603")
	}
	s.holderActionList = append(s.holderActionList, &action)
}

func (s *BRC20Indexer) UpdateTransfer(block *common.Block) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	startTime := time.Now()

	// 检查transferNft转入到哪个输出
	inputTransferNfts := make(map[int64]*TransferNftInfo)
	for _, tx := range block.Transactions[1:] {
		hasTransfer := false
		for _, input := range tx.Inputs {
			nft, ok := s.transferNftMap[input.UtxoId] // transferNftMap 第一次转移时，先不删除，只设置标志位
			if ok {
				if !nft.TransferNft.IsInvalid {
					inputTransferNfts[nft.TransferNft.NftId] = nft
					
					ticker := s.tickerMap[nft.Ticker]
					ticker.Ticker.TransactionCount++
					s.tickerUpdated[nft.Ticker] = ticker.Ticker

					hasTransfer = true
					nft.TransferNft.IsInvalid = true // 仅设置标志位
				} else {
				 	// 已经转移过的transfer铭文，不需要再处理，直接删除就行
					s.removeTransferNft(nft) // 从当前地址中删除数据
				}
			}
		}

		if hasTransfer {
			for _, output := range tx.Outputs {
				s.innerUpdateTransfer(tx.Txid, output, &inputTransferNfts)
			}
		}
	}

	// if len(inputTransferNfts) != 0 {
	// 	tx := block.Transactions[0]
	// 	for _, output := range tx.Outputs {
	// 		p.innerUpdateTransfer(tx.Txid, output, &inputTransferNfts)
	// 	}
	// }

	common.Log.Infof("BRC20Indexer->UpdateTransfer loop %d in %v", len(block.Transactions), time.Since(startTime))
	//p.CheckSelf(block.Height)
}

// 增加该address下的资产数据
func (s *BRC20Indexer) addHolderBalance(ticker string, address uint64, amt common.Decimal) {
	tickerName := strings.ToLower(ticker)

	info, ok := s.holderMap[address]
	zeroAmt, _ := common.NewDecimalFromString("0", int(s.tickerMap[tickerName].Ticker.Decimal))
	if !ok {
		tickers := make(map[string]*common.BRC20TickAbbrInfo)
		tickers[tickerName] = common.NewBRC20TickAbbrInfo(&amt, zeroAmt)
		info = &HolderInfo{ /*AddressId: address,*/ Tickers: tickers}
		s.holderMap[address] = info
		s.tickerMap[tickerName].Ticker.HolderCount++
		s.tickerUpdated[tickerName] = s.tickerMap[tickerName].Ticker
	} else {
		// info.AddressId = address
		tickAbbrInfo, ok := info.Tickers[tickerName]
		if !ok {
			s.tickerMap[tickerName].Ticker.HolderCount++
			s.tickerUpdated[tickerName] = s.tickerMap[tickerName].Ticker
			info.Tickers[tickerName] = common.NewBRC20TickAbbrInfo(&amt, zeroAmt)
		} else {
			tickAbbrInfo.AvailableBalance = tickAbbrInfo.AvailableBalance.Add(&amt)
		}
	}

	holders, ok := s.tickerToHolderMap[tickerName]
	if !ok {
		holders = make(map[uint64]bool, 0)
	}
	holders[address] = true
	s.tickerToHolderMap[tickerName] = holders
}

var err_no_find_holder = fmt.Errorf("no find holder")
var err_no_enough_balance = fmt.Errorf("not enough balance")

// 减少该address下的资产数据
func (s *BRC20Indexer) subHolderBalance(ticker string, address uint64, amt common.Decimal) error {
	tickerName := strings.ToLower(ticker)
	holdInfo, ok := s.holderMap[address]
	if ok {
		// info.AddressId = address
		tickAbbrInfo, ok := holdInfo.Tickers[tickerName]
		if ok {
			if tickAbbrInfo.TransferableBalance.Cmp(&amt) >= 0 {
				tickAbbrInfo.TransferableBalance = tickAbbrInfo.TransferableBalance.Sub(&amt)
				// balanceStr := tickinfo.TransferableBalance.String()
				// common.Log.Infof("%s", balanceStr)

				balance := tickAbbrInfo.AvailableBalance.Add(tickAbbrInfo.TransferableBalance)
				balanceIsZero := balance.IsZero()
				if balanceIsZero {
					holders := s.tickerToHolderMap[tickerName]
					delete(holders, address)
					if len(holders) == 0 {
						delete(s.tickerToHolderMap, tickerName)
					} else {
						s.tickerToHolderMap[tickerName] = holders
					}
					s.tickerMap[tickerName].Ticker.HolderCount--
					s.tickerUpdated[tickerName] = s.tickerMap[tickerName].Ticker

					delete(holdInfo.Tickers, tickerName)
					if len(holdInfo.Tickers) == 0 {
						delete(s.holderMap, address)
					}
				}
				return nil
			} else {
				return err_no_enough_balance
			}
		}
	}
	return err_no_find_holder
}

func (s *BRC20Indexer) removeTransferNft(nft *TransferNftInfo) {
	delete(s.transferNftMap, nft.UtxoId)

	holder, ok := s.holderMap[nft.AddressId]
	if ok {
		tickInfo, ok := holder.Tickers[nft.Ticker]
		if ok {
			delete(tickInfo.TransferableData, nft.UtxoId)
		} else {
			common.Log.Panic("can't find ticker info")
		}
	} else {
		// 已经转移过的transfer nft不一定能找到
		// common.Log.Panic("can't find ticker info")
	}
}

func (s *BRC20Indexer) addTransferNft(nft *TransferNftInfo) {
	curr, ok := s.transferNftMap[nft.UtxoId]
	if ok {
		// 多个transfer输出到同一个utxo，这个时候只修改amt
		curr.TransferNft.Amount = *curr.TransferNft.Amount.Add(&nft.TransferNft.Amount)
	} else {
		curr = nft
		s.transferNftMap[nft.UtxoId] = nft
	}

	holder, ok := s.holderMap[nft.AddressId]
	if !ok {
		// 一个transfer铭文转移时的接受地址，其holder可能为空。nft是一个已经使用过的铭文。
		holder = &HolderInfo{
			Tickers: make(map[string]*common.BRC20TickAbbrInfo),
		}
		holder.Tickers[nft.Ticker] = common.NewBRC20TickAbbrInfo(nil, nil)
		s.holderMap[nft.AddressId] = holder
	}
	
	tickAbbrInfo, ok := holder.Tickers[nft.Ticker]
	if ok {
		tickAbbrInfo.TransferableData[nft.UtxoId] = curr.TransferNft
	}
}

func (s *BRC20Indexer) innerUpdateTransfer(txId string, output *common.Output, inputTransferNfts *map[int64]*TransferNftInfo) {
	// 检查是否存在nft。如果存在，就更新对应的holder数据
	utxoId := common.GetUtxoId(output)
	ids := s.nftIndexer.GetNftsWithUtxo(utxoId) // 有可能多个transfer nft，合并输出到一个output中
	for _, nft := range ids {
		transferNft, ok := (*inputTransferNfts)[nft.Base.Id]
		if ok {
			// transfer occur

			fromAddressId := transferNft.AddressId
			toAddressId := s.nftIndexer.GetBaseIndexer().GetAddressId(output.Address.Addresses[0])

			// 在下一次转移时，可以删除，不需要再记录
			transferNft.AddressId = toAddressId
			transferNft.UtxoId = utxoId
			s.addTransferNft(transferNft)

			err := s.subHolderBalance(transferNft.Ticker, fromAddressId,
				transferNft.TransferNft.Amount)
			if err == err_no_find_holder {
				common.Log.Panic("innerUpdateTransfer call subHolderBalance err: ", err)
				continue
				// return
			} else if err == err_no_enough_balance {
				common.Log.Panic("innerUpdateTransfer call subHolderBalance err: ", err)
				continue
			}
			// p.addHolderBalance(transferNft.Ticker, fromAddrId,
			// 	transferNft.TransferNft.Amount)
			
			s.addHolderBalance(transferNft.Ticker, toAddressId,
				transferNft.TransferNft.Amount)

			action := HolderAction{
				Height: output.Height,
				// Utxo:     common.ToUtxo(txId, int(output.N)),
				UtxoId:   utxoId,
				NftId:    transferNft.TransferNft.NftId,
				FromAddr: fromAddressId,
				ToAddr:   toAddressId,
				Ticker:   transferNft.Ticker,
				Amount:   transferNft.TransferNft.Amount,
				Action:   Action_Transfer,
			}
			s.holderActionList = append(s.holderActionList, &action)
			delete((*inputTransferNfts), nft.Base.Id)
		}
	}
}

// 跟basic数据库同步
func (s *BRC20Indexer) UpdateDB() {
	//common.Log.Infof("BRC20Indexer->UpdateDB start...")
	startTime := time.Now()

	wb := s.db.NewWriteBatch()
	defer wb.Close()

	// new ticker for deploy
	for _, v := range s.tickerAdded {
		key := GetTickerKey(v.Name)
		err := db.SetDB([]byte(key), v, wb)
		if err != nil {
			common.Log.Panicf("Error setting %s in db %v", key, err)
		}
	}

	// static info for ticker for mint/transfer
	for _, ticker := range s.tickerUpdated {
		key := GetTickerKey(ticker.Name)
		err := db.SetDB([]byte(key), ticker, wb)
		if err != nil {
			common.Log.Panicf("Error setting %s in db %v", key, err)
		}
	}

	// mint history
	for _, ticker := range s.tickerMap {
		for _, v := range ticker.MintAdded {
			key := GetMintHistoryKey(ticker.Name, v.Nft.Base.InscriptionId)
			err := db.SetDB([]byte(key), v, wb)
			if err != nil {
				common.Log.Panicf("Error setting %s in db %v", key, err)
			}
		}
	}

	// holder status
	// transfer history
	for _, action := range s.holderActionList {
		switch action.Action {
		case Action_InScribe_Mint:
			toKey := GetHolderInfoKey(action.ToAddr)
			value, ok := s.holderMap[action.ToAddr]
			// tickerMinted := p.tickerMap[strings.ToLower(action.Ticker)].Ticker.Minted.String()
			// common.Log.Infof("action.Ticker:%s, str:%s", action.Ticker, tickerMinted)
			if ok {
				err := db.SetDB([]byte(toKey), value, wb)
				if err != nil {
					common.Log.Panicf("Error setting %s in db %v", toKey, err)
				}
			} else {
				err := wb.Delete([]byte(toKey))
				if err != nil {
					common.Log.Panicf("Error deleting db %s: %v\n", toKey, err)
				}
				// common.Log.Panicf("no find holder info in holderMap :%d", action.ToAddr)
			}
		case Action_InScribe_Transfer:
			// toKey := GetHolderInfoKey(action.ToAddr, action.Ticker)
			toKey := GetHolderInfoKey(action.ToAddr)
			value, ok := s.holderMap[action.ToAddr]
			if ok {
				err := db.SetDB([]byte(toKey), value, wb)
				if err != nil {
					common.Log.Panicf("Error setting %s in db %v", toKey, err)
				}
			} else {
				err := wb.Delete([]byte(toKey))
				if err != nil {
					common.Log.Panicf("Error deleting db %s: %v\n", toKey, err)
				}
				// common.Log.Infof("no find holder info in holderMap :%d", action.ToAddr)
			}
		case Action_Transfer:
			if action.FromAddr == common.INVALID_ID {
				common.Log.Panic("action.FromAddr == common.INVALID_ID when action.Action == Action_Transfer")
			}
			fromKey := GetHolderInfoKey(action.FromAddr)
			value, ok := s.holderMap[action.FromAddr]
			if ok {
				err := db.SetDB([]byte(fromKey), value, wb)
				if err != nil {
					common.Log.Panicf("Error setting %s in db %v", fromKey, err)
				}
			} else {
				//已经被删除
				err := wb.Delete([]byte(fromKey))
				if err != nil {
					common.Log.Infof("Error deleting db %s: %v\n", fromKey, err)
				}
			}

			toKey := GetHolderInfoKey(action.ToAddr)
			value, ok = s.holderMap[action.ToAddr]
			if ok {
				err := db.SetDB([]byte(toKey), value, wb)
				if err != nil {
					common.Log.Panicf("Error setting %s in db %v", toKey, err)
				}
			} else {
				err := wb.Delete([]byte(toKey))
				if err != nil {
					common.Log.Panicf("Error deleting db %s: %v\n", toKey, err)
				}
				// common.Log.Infof("no find holder info in holderMap :%d", action.ToAddr)
			}

			// 保存历史记录
			history := common.BRC20TransferHistory{
				Height: action.Height,
				// Utxo:     action.Utxo,
				UtxoId:   action.UtxoId,
				NftId:    action.NftId,
				FromAddr: action.FromAddr,
				ToAddr:   action.ToAddr,
				Ticker:   action.Ticker,
				Amount:   action.Amount.String(),
			}
			key := GetTransferHistoryKey(strings.ToLower(action.Ticker), action.UtxoId)
			err := db.SetDB([]byte(key), &history, wb)
			if err != nil {
				common.Log.Panicf("Error setting %s in db %v", key, err)
			}
		}
	}

	err := wb.Flush()
	if err != nil {
		common.Log.Panicf("Error ordxwb flushing writes to db %v", err)
	}

	// reset memory buffer
	s.holderActionList = make([]*HolderAction, 0)
	s.tickerAdded = make(map[string]*common.BRC20Ticker)
	s.tickerUpdated = make(map[string]*common.BRC20Ticker)
	for _, info := range s.tickerMap {
		info.MintAdded = make([]*common.BRC20Mint, 0)
	}

	common.Log.Infof("BRC20Indexer->UpdateDB takse: %v", time.Since(startTime))
}
