package brc20

import (
	"fmt"
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

// deploy
func (p *BRC20Indexer) UpdateInscribeDeploy(ticker *common.BRC20Ticker) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	name := strings.ToLower(ticker.Name)
	org, ok := p.tickerMap[name]
	if !ok {
		ticker.Id = int64(len(p.tickerMap))
		ticker.TransactionCount++
		tickinfo := newTickerInfo(ticker.Name)
		tickinfo.Ticker = ticker
		p.tickerMap[name] = tickinfo
		p.tickerAdded[name] = ticker
	} else {
		// 仅更新显示内容
		p.tickerAdded[name] = org.Ticker
	}

	common.Log.Infof("UpdateInscribeDeploy %s", ticker.Name)
}

// mint
func (p *BRC20Indexer) UpdateInscribeMint(mint *common.BRC20Mint) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	ticker, ok := p.tickerMap[strings.ToLower(mint.Name)]
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
	p.tickerUpdated[strings.ToLower(mint.Name)] = ticker.Ticker

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
	p.holderActionList = append(p.holderActionList, &action)
	p.addHolderBalance(mint.Name, mint.Nft.OwnerAddressId, mint.Amt)
}

// transfer
func (p *BRC20Indexer) UpdateInscribeTransfer(transfer *common.BRC20Transfer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	addressId := transfer.Nft.OwnerAddressId
	holder := p.holderMap[addressId]
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

	ticker := p.tickerMap[tickerName]
	ticker.Ticker.TransactionCount++
	p.tickerUpdated[tickerName] = ticker.Ticker

	nft := common.TransferNFT{
		NftId:  transfer.Nft.Base.Id,
		UtxoId: transfer.Nft.UtxoId,
		Amount: transfer.Amt,
	}
	transferInfo := &TransferNftInfo{
		AddressId:   transfer.Nft.OwnerAddressId,
		Ticker:      transfer.Name,
		TransferNft: &nft,
	}
	p.addTransferNft(transferInfo)

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
	p.holderActionList = append(p.holderActionList, &action)
}

func (p *BRC20Indexer) UpdateTransfer(block *common.Block) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	startTime := time.Now()

	// 检查transferNft转入到哪个输出
	inputTransferNfts := make(map[int64]*TransferNftInfo)
	for _, tx := range block.Transactions[1:] {
		hasTransfer := false
		for _, input := range tx.Inputs {
			nft, ok := p.transferNftMap[input.UtxoId] // transferNftMap 第一次转移时，先不删除，只设置标志位
			if ok {
				if !nft.TransferNft.IsInvalid {
					inputTransferNfts[nft.TransferNft.NftId] = nft
					tickerName := strings.ToLower(nft.Ticker)

					ticker := p.tickerMap[tickerName]
					ticker.Ticker.TransactionCount++
					p.tickerUpdated[tickerName] = ticker.Ticker

					hasTransfer = true
					nft.TransferNft.IsInvalid = true // 仅设置标志位
				} else {
					// remove
					p.removeTransferNft(nft)
				}
			}
		}

		if hasTransfer {
			for _, output := range tx.Outputs {
				p.innerUpdateTransfer(tx.Txid, output, &inputTransferNfts)
			}
		}
	}

	if len(inputTransferNfts) != 0 {
		tx := block.Transactions[0]
		for _, output := range tx.Outputs {
			p.innerUpdateTransfer(tx.Txid, output, &inputTransferNfts)
		}
	}

	common.Log.Infof("BRC20Indexer->UpdateTransfer loop %d in %v", len(block.Transactions), time.Since(startTime))
	//p.CheckSelf(block.Height)
}

// 增加该address下的资产数据
func (p *BRC20Indexer) addHolderBalance(ticker string, address uint64, amt common.Decimal) {
	tickerName := strings.ToLower(ticker)

	info, ok := p.holderMap[address]
	zeroAmt, _ := common.NewDecimalFromString("0", int(p.tickerMap[tickerName].Ticker.Decimal))
	if !ok {
		tickers := make(map[string]*common.BRC20TickAbbrInfo)
		tickers[tickerName] = common.NewBRC20TickAbbrInfo(&amt, zeroAmt)
		info = &HolderInfo{ /*AddressId: address,*/ Tickers: tickers}
		p.holderMap[address] = info
		p.tickerMap[tickerName].Ticker.HolderCount++
		p.tickerUpdated[tickerName] = p.tickerMap[tickerName].Ticker
	} else {
		// info.AddressId = address
		tickAbbrInfo, ok := info.Tickers[tickerName]
		if !ok {
			p.tickerMap[tickerName].Ticker.HolderCount++
			p.tickerUpdated[tickerName] = p.tickerMap[tickerName].Ticker
			info.Tickers[tickerName] = common.NewBRC20TickAbbrInfo(&amt, zeroAmt)
		} else {
			tickAbbrInfo.AvailableBalance = tickAbbrInfo.AvailableBalance.Add(&amt)
		}
	}

	holders, ok := p.tickerToHolderMap[tickerName]
	if !ok {
		holders = make(map[uint64]bool, 0)
	}
	holders[address] = true
	p.tickerToHolderMap[tickerName] = holders
}

var err_no_find_holder = fmt.Errorf("no find holder")
var err_no_enough_balance = fmt.Errorf("not enough balance")

// 减少该address下的资产数据
func (p *BRC20Indexer) subHolderBalance(ticker string, address uint64, amt common.Decimal) error {
	tickerName := strings.ToLower(ticker)
	holdInfo, ok := p.holderMap[address]
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
					holders := p.tickerToHolderMap[tickerName]
					delete(holders, address)
					if len(holders) == 0 {
						delete(p.tickerToHolderMap, tickerName)
					} else {
						p.tickerToHolderMap[tickerName] = holders
					}
					p.tickerMap[tickerName].Ticker.HolderCount--
					p.tickerUpdated[tickerName] = p.tickerMap[tickerName].Ticker

					delete(holdInfo.Tickers, tickerName)
					if len(holdInfo.Tickers) == 0 {
						delete(p.holderMap, address)
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

func (p *BRC20Indexer) removeTransferNft(nft *TransferNftInfo) {
	delete(p.transferNftMap, nft.TransferNft.UtxoId)
	
	// nft是同一个指针，不需要再查找
	// holder, ok := p.holderMap[nft.AddressId]
	// if ok {
	// 	tickerName := strings.ToLower(nft.Ticker)
	// 	tickInfo, ok := holder.Tickers[tickerName]
	// 	if ok {
	// 		tickInfo.TransferableData[nft.TransferNft.UtxoId].IsInvalid = true
	// 		// delete(tickInfo.TransferableData, nft.TransferNft.UtxoId)
	// 	} else {
	// 		common.Log.Panic("can't find ticker info")
	// 	}
	// } else {
	// 	common.Log.Panic("can't find ticker info")
	// }
}

func (p *BRC20Indexer) addTransferNft(nft *TransferNftInfo) {
	p.transferNftMap[nft.TransferNft.UtxoId] = nft

	// bValid := false
	holder, ok := p.holderMap[nft.AddressId]
	if ok {
		tickerName := strings.ToLower(nft.Ticker)
		tickAbbrInfo, ok := holder.Tickers[tickerName]
		if ok {
			tickAbbrInfo.TransferableData[nft.TransferNft.UtxoId] = nft.TransferNft
			// bValid = true
		}
	}

	// if !bValid {
	// TODO:
	// 异常路径：没有可用余额，先将transfer铭文保存起来，以后可能可以用（TODO 要看BRC20协议是否这样处理）
	// availableBalance := common.NewDecimal(0, nft.TransferNft.Amount.Precision)
	// 仅为了创建tickInfo
	// p.addHolderBalance(nft.Ticker, nft.AddressId, *availableBalance)

	// holder = p.holderMap[nft.AddressId]
	// tickInfo := holder.Tickers[strings.ToLower(nft.Ticker)]
	// tickInfo.InvalidTransferableData[nft.TransferNft.UtxoId] = nft.TransferNft
	// }
}

func (p *BRC20Indexer) innerUpdateTransfer(txId string, output *common.Output, inputTransferNfts *map[int64]*TransferNftInfo) {
	// 检查是否存在nft。如果存在，就更新对应的holder数据
	utxoId := common.GetUtxoId(output)
	ids := p.nftIndexer.GetNftsWithUtxo(utxoId)
	for _, nft := range ids {
		transferNft, ok := (*inputTransferNfts)[nft.Base.Id]
		if ok {
			// transfer occur

			err := p.subHolderBalance(transferNft.Ticker, transferNft.AddressId,
				transferNft.TransferNft.Amount)
			if err == err_no_find_holder {
				common.Log.Error("innerUpdateTransfer call subHolderBalance err: ", err)
				continue
				// return
			} else if err == err_no_enough_balance {
				common.Log.Error("innerUpdateTransfer call subHolderBalance err: ", err)
				continue
			}
			// p.addHolderBalance(transferNft.Ticker, transferNft.AddressId,
			// 	transferNft.TransferNft.Amount)
			toAddressId := p.nftIndexer.GetBaseIndexer().GetAddressId(output.Address.Addresses[0])
			p.addHolderBalance(transferNft.Ticker, toAddressId,
				transferNft.TransferNft.Amount)

			action := HolderAction{
				Height: output.Height,
				// Utxo:     common.ToUtxo(txId, int(output.N)),
				UtxoId:   utxoId,
				NftId:    transferNft.TransferNft.NftId,
				FromAddr: transferNft.AddressId,
				ToAddr:   toAddressId,
				Ticker:   transferNft.Ticker,
				Amount:   transferNft.TransferNft.Amount,
				Action:   Action_Transfer,
			}
			p.holderActionList = append(p.holderActionList, &action)

			delete((*inputTransferNfts), nft.Base.Id)
		}
	}
}

// 跟basic数据库同步
func (p *BRC20Indexer) UpdateDB() {
	//common.Log.Infof("BRC20Indexer->UpdateDB start...")
	startTime := time.Now()

	wb := p.db.NewWriteBatch()
	defer wb.Close()

	// new ticker for deploy
	for _, v := range p.tickerAdded {
		key := GetTickerKey(v.Name)
		err := db.SetDB([]byte(key), v, wb)
		if err != nil {
			common.Log.Panicf("Error setting %s in db %v", key, err)
		}
	}

	// static info for ticker for mint/transfer
	for _, ticker := range p.tickerUpdated {
		key := GetTickerKey(ticker.Name)
		err := db.SetDB([]byte(key), ticker, wb)
		if err != nil {
			common.Log.Panicf("Error setting %s in db %v", key, err)
		}
	}

	// mint history
	for _, ticker := range p.tickerMap {
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
	for _, action := range p.holderActionList {
		switch action.Action {
		case Action_InScribe_Mint:
			// toKey := GetHolderInfoKey(action.ToAddr, action.Ticker)
			toKey := GetHolderInfoKey(action.ToAddr)
			value, ok := p.holderMap[action.ToAddr]
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
			value, ok := p.holderMap[action.ToAddr]
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
			// fromKey := GetHolderInfoKey(action.FromAddr, action.Ticker)
			fromKey := GetHolderInfoKey(action.FromAddr)
			value, ok := p.holderMap[action.FromAddr]
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

			// toKey := GetHolderInfoKey(action.ToAddr, action.Ticker)
			toKey := GetHolderInfoKey(action.ToAddr)
			value, ok = p.holderMap[action.ToAddr]
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

			// key := GetTransferHistoryKey(action.Ticker, action.Utxo)
			key := GetTransferHistoryKey(action.Ticker, action.UtxoId)
			err := db.SetDB([]byte(key), &history, wb)
			if err != nil {
				common.Log.Panicf("Error setting %s in db %v", key, err)
			}
		}
		// 更新holder数据
		// if action.FromAddr != common.INVALID_ID {
		// 	fromKey := GetHolderInfoKey(action.FromAddr, action.Ticker)
		// 	value, ok := p.holderMap[action.FromAddr]
		// 	if ok {
		// 		err := db.SetDB([]byte(fromKey), value, wb)
		// 		if err != nil {
		// 			common.Log.Panicf("Error setting %s in db %v", fromKey, err)
		// 		}
		// 	} else {
		// 		//已经被删除
		// 		err := wb.Delete([]byte(fromKey))
		// 		if err != nil {
		// 			common.Log.Infof("Error deleting db %s: %v\n", fromKey, err)
		// 		}
		// 	}
		// }
		// toKey := GetHolderInfoKey(action.ToAddr, action.Ticker)
		// value, ok := p.holderMap[action.ToAddr]
		// if ok {
		// 	err := db.SetDB([]byte(toKey), value, wb)
		// 	if err != nil {
		// 		common.Log.Panicf("Error setting %s in db %v", toKey, err)
		// 	}
		// } else {
		// 	//已经被删除
		// 	err := wb.Delete([]byte(toKey))
		// 	if err != nil {
		// 		common.Log.Infof("Error deleting db %s: %v\n", toKey, err)
		// 	}
		// }

		// if action.Action == Action_Transfer {
		// 	// 保存历史记录
		// 	history := common.BRC20TransferHistory{
		// 		Height:   action.Height,
		// 		Utxo:     action.Utxo,
		// 		NftId:    action.NftId,
		// 		FromAddr: action.FromAddr,
		// 		ToAddr:   action.ToAddr,
		// 		Ticker:   action.Ticker,
		// 		Amount:   action.Amount.String(),
		// 	}

		// 	key := GetTransferHistoryKey(action.Ticker, action.Utxo)
		// 	err := db.SetDB([]byte(key), &history, wb)
		// 	if err != nil {
		// 		common.Log.Panicf("Error setting %s in db %v", key, err)
		// 	}
		// }
	}

	err := wb.Flush()
	if err != nil {
		common.Log.Panicf("Error ordxwb flushing writes to db %v", err)
	}

	// reset memory buffer
	p.holderActionList = make([]*HolderAction, 0)
	p.tickerAdded = make(map[string]*common.BRC20Ticker)
	p.tickerUpdated = make(map[string]*common.BRC20Ticker)
	for _, info := range p.tickerMap {
		info.MintAdded = make([]*common.BRC20Mint, 0)
	}

	common.Log.Infof("BRC20Indexer->UpdateDB takse: %v", time.Since(startTime))
}
