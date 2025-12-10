package brc20

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

// deploy
func (s *BRC20Indexer) UpdateInscribeDeploy(ticker *common.BRC20Ticker) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.deployBuffer = append(s.deployBuffer, ticker)
}

func (s *BRC20Indexer) updateInscribeDeploy(ticker *common.BRC20Ticker) {
	// 再次检查，因为nft可能会修改reinsription状态
	nft := ticker.Nft
	if nft.Base.CurseType != 0 { 
		common.Log.Debugf("%s inscription is cursed, %d", nft.Base.InscriptionId, nft.Base.CurseType)
		if nft.Base.BlockHeight < 824544 { // Jubilee
			return
		}
		// vindicated
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

	common.Log.Infof("UpdateInscribeDeploy %s", ticker.Name)
}

// mint
func (s *BRC20Indexer) UpdateInscribeMint(mint *common.BRC20Mint) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.mintOrTransferBuffer = append(s.mintOrTransferBuffer, mint)
}

func (s *BRC20Indexer) updateInscribeMint(mint *common.BRC20Mint) {
	if mint.Nft.Base.CurseType != 0 { 
		common.Log.Debugf("%s inscription is cursed, %d", mint.Nft.Base.InscriptionId, mint.Nft.Base.CurseType)
		if mint.Nft.Base.BlockHeight < 824544 { // Jubilee
			return
		}
		// vindicated
	}

	name := strings.ToLower(mint.Name)
	ticker := s.tickerMap[name]

	// ticker 还没有部署
	if mint.NftId < ticker.Ticker.Nft.Base.Id {
		return
	}
	if ticker.Ticker.EndInscriptionId != "" {
		// 已经足够了
		return
	}

	ticker.Ticker.TransactionCount++
	mintedAmt := ticker.Ticker.Minted.Add(&mint.Amt)
	ticker.Ticker.Minted = *mintedAmt
	if mintedAmt.Cmp(&ticker.Ticker.Max) == 0 {
		ticker.Ticker.EndInscriptionId = mint.Nft.Base.InscriptionId
	}
	s.tickerUpdated[name] = ticker.Ticker

	mint.Id = int64(ticker.Ticker.MintCount)
	ticker.Ticker.MintCount++
	ticker.MintAdded = append(ticker.MintAdded, mint)
	//ticker.InscriptionMap[mint.Nft.Base.InscriptionId] = common.NewBRC20MintAbbrInfo(mint)

	s.loadHolderInfo(mint.Nft.OwnerAddressId, name)

	// 作为inalid的output
	nft := common.TransferNFT{
		NftId:  mint.Nft.Base.Id,
		UtxoId: mint.Nft.UtxoId,
		Amount: mint.Amt,
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
		BRC20ActionHistory: common.BRC20ActionHistory{
			Height:   int(mint.Nft.Base.BlockHeight),
			UtxoId:   mint.Nft.UtxoId,
			NftId:    mint.Nft.Base.Id,
			FromAddr: common.INVALID_ID,
			ToAddr:   mint.Nft.OwnerAddressId,
			Ticker:   name,
			Amount:   mint.Amt,
			Action:   common.BRC20_Action_InScribe_Mint,
		},
	}
	s.holderActionList = append(s.holderActionList, &action)

	s.addHolderBalance(name, mint.Nft.OwnerAddressId, &mint.Amt)
}

// transfer
func (s *BRC20Indexer) UpdateInscribeTransfer(transfer *common.BRC20Transfer) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.mintOrTransferBuffer = append(s.mintOrTransferBuffer, transfer) 
}

func (s *BRC20Indexer) updateInscribeTransfer(transfer *common.BRC20Transfer) {
	if transfer.Nft.Base.CurseType != 0 { 
		common.Log.Debugf("%s inscription is cursed, %d", transfer.Nft.Base.InscriptionId, transfer.Nft.Base.CurseType)
		if transfer.Nft.Base.BlockHeight < 824544 { // Jubilee
			return
		}
		// vindicated
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
		return
	}
	tickAbbrInfo.AvailableBalance = tickAbbrInfo.AvailableBalance.Sub(&transfer.Amt)
	tickAbbrInfo.TransferableBalance = tickAbbrInfo.TransferableBalance.Add(&transfer.Amt)

	ticker := s.tickerMap[tickerName]
	transfer.Id = int64(ticker.Ticker.TransactionCount)
	ticker.Ticker.TransactionCount++
	s.tickerUpdated[tickerName] = ticker.Ticker

	nft := common.TransferNFT{
		NftId:  transfer.Nft.Base.Id,
		Id:     transfer.Id,
		UtxoId: transfer.Nft.UtxoId,
		Amount: transfer.Amt,
	}
	transferInfo := &TransferNftInfo{
		AddressId:   transfer.Nft.OwnerAddressId,
		Ticker:      tickerName,
		UtxoId:      transfer.Nft.UtxoId,
		TransferNft: &nft,
	}
	s.addTransferNft(transferInfo)

	action := HolderAction{
		BRC20ActionHistory: common.BRC20ActionHistory{
			Height:   int(transfer.Nft.Base.BlockHeight),
			UtxoId:   transfer.Nft.UtxoId,
			NftId:    transfer.Nft.Base.Id,
			FromAddr: common.INVALID_ID,
			ToAddr:   transfer.Nft.OwnerAddressId,
			Ticker:   tickerName,
			Amount:   transfer.Amt,
			Action:   common.BRC20_Action_InScribe_Transfer,
		},
	}
	s.holderActionList = append(s.holderActionList, &action)
}

func (s *BRC20Indexer) UpdateTransfer(block *common.Block) {
	if block.Height < s.enableHeight {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	startTime := time.Now()

	if block.Height == 90419 {
		common.Log.Infof("")
	}

	baseIndexer := s.nftIndexer.GetBaseIndexer()

	// 预加载相关地址的数据
	s.db.View(func(txn common.ReadBatch) error {
		// 处理区块涉及到的铸造
		// 先加载所有ticker
		addressToLoad := make(map[uint64]map[string]bool)
		newTickers := make(map[string]bool)
		for _, deploy := range s.deployBuffer {
			name := strings.ToLower(deploy.Name)
			_, ok := s.tickerMap[name]
			if ok {
				continue
			}
			newTickers[name] = true
		}
		for _, item := range s.mintOrTransferBuffer {
			var name string
			var addressId uint64
			mint, ok := item.(*common.BRC20Mint)
			if ok {
				name = strings.ToLower(mint.Name)
				addressId = mint.Nft.OwnerAddressId
			} else {
				transfer, ok := item.(*common.BRC20Transfer)
				if ok {
					name = strings.ToLower(transfer.Name)
					addressId = transfer.Nft.OwnerAddressId
				}
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
			newTickers[name] = true
		}
		tickerKeys := make([]string, len(newTickers))
		for k := range newTickers {
			tickerKeys = append(tickerKeys, encodeTickerName(k))
		}
		sort.Slice(tickerKeys, func(i, j int) bool {
			return tickerKeys[i] < tickerKeys[j]
		})
		for _, key := range tickerKeys {
			var ticker common.BRC20Ticker
			key := DB_PREFIX_TICKER + key
			err := db.GetValueFromDB([]byte(key), &ticker, s.db)
			if err != nil {
				continue
			} 

			s.tickerMap[strings.ToLower(ticker.Name)] = &BRC20TickInfo{
				Name: strings.ToLower(ticker.Name),
				Ticker: &ticker,
			}
		}
		// 处理所有的deploy
		for _, deploy := range s.deployBuffer {
			name := strings.ToLower(deploy.Name)
			_, ok := s.tickerMap[name]
			if ok {
				continue
			}
			s.updateInscribeDeploy(deploy)
		}
		// 加载mint涉及到地址
		type pair struct {
			utxoId    uint64
			addressId uint64
			tx        *common.Transaction
			ticker    string
		}
		addressToLoadVector := make([]*pair, 0)
		for addressId, tickers := range addressToLoad {
			for name := range tickers {
				addressToLoadVector = append(addressToLoadVector, &pair{
					addressId: addressId,
					ticker: name,
				})
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
			holder.Tickers[v.ticker] = &value
		}
		// 处理所有的mint 和 transfer，按顺序
		for _, item := range s.mintOrTransferBuffer {
			mint, ok := item.(*common.BRC20Mint)
			if ok {
				// mint 必须在deploy后面
				s.updateInscribeMint(mint)
			} else {
				transfer, ok := item.(*common.BRC20Transfer)
				if ok {
					s.updateInscribeTransfer(transfer)
				}
			}
		}


		// 处理区块本身的交易
		utxoToLoad := make([]*pair, 0)
		transferTxMap := make(map[*common.Transaction]map[string]bool)
		for _, tx := range block.Transactions[1:] {
			for _, input := range tx.Inputs {
				var ticker string
				nft, ok := s.transferNftMap[input.UtxoId]
				if ok {
					ticker = nft.Ticker
					if !nft.TransferNft.IsInvalid {
						tickers, ok := transferTxMap[tx]
						if !ok {
							tickers = make(map[string]bool)
							transferTxMap[tx] = tickers
						}
						tickers[nft.Ticker] = true // 影响输出的结果
					}
				}
				utxoToLoad = append(utxoToLoad, &pair{
					utxoId: 	input.UtxoId,
					addressId: 	baseIndexer.GetAddressId(input.Address.Addresses[0]),
					tx:     	tx,
					ticker:     ticker,
				})
			}
		}
		// pebble数据库的优化手段: 尽可能将随机读变成按照key的顺序读
		sort.Slice(utxoToLoad, func(i, j int) bool {
			return utxoToLoad[i].utxoId < utxoToLoad[j].utxoId
		})
		

		
		tickerToLoad := make(map[string]bool)
		for _, v := range utxoToLoad {
			if v.ticker == "" {
				var value TransferNftInfo
				key := GetUtxoToTransferKey(v.utxoId)
				err := db.GetValueFromTxn([]byte(key), &value, txn)
				if err != nil {
					continue
				}
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
			tickerToLoad[v.ticker] = true
			
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
				addressId := baseIndexer.GetAddressId(output.Address.Addresses[0])
				tickers, ok := addressToLoad[addressId]
				if !ok {
					tickers = make(map[string]bool)
					addressToLoad[addressId] = tickers
				}
				for name := range names {
					tickers[name] = true
				}
			}
		}
		for addressId, tickers := range addressToLoad {
			for name := range tickers {
				addressToLoadVector = append(addressToLoadVector, &pair{
					addressId: addressId,
					ticker: name,
				})
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
			holder.Tickers[v.ticker] = &value
		}

		for name := range tickerToLoad {
			s.loadTickInfo(name)
		}

		return nil
	})

	// 检查transferNft转入到哪个输出
	inputTransferNfts := make(map[int64]*TransferNftInfo)
	for _, tx := range block.Transactions[1:] {
		if tx.Txid == "442c7a43d638a27e7043e6d7a450bd6d14f5836477b7fc5ede302f5e60c905f5" {
			common.Log.Infof("utxoId = %d", common.GetUtxoId(tx.Outputs[0]))
		}

		hasTransfer := false
		for _, input := range tx.Inputs {
			// if input.UtxoId == 3804104075509760 || input.UtxoId == 3804104075771904 {
			// 	common.Log.Infof("utxoId = %d", tx.Outputs[0].UtxoId)
			// }
			transfer, ok := s.transferNftMap[input.UtxoId] // transferNftMap 第一次转移时，先不删除，只设置标志位
			if ok {
				inputTransferNfts[transfer.TransferNft.NftId] = transfer
				hasTransfer = true
				if !transfer.TransferNft.IsInvalid {
					ticker := s.tickerMap[transfer.Ticker]
					ticker.Ticker.TransactionCount++
					s.tickerUpdated[transfer.Ticker] = ticker.Ticker
				}
				s.removeTransferNft(transfer) // 从当前地址中删除数据
			}
		}

		if hasTransfer {
			for _, output := range tx.Outputs {
				s.innerUpdateTransfer(tx.Txid, output, inputTransferNfts)
			}
		}
	}

	// if len(inputTransferNfts) != 0 {
	// 	tx := block.Transactions[0]
	// 	for _, output := range tx.Outputs {
	// 		p.innerUpdateTransfer(tx.Txid, output, &inputTransferNfts)
	// 	}
	// }

	s.deployBuffer = nil
	s.mintOrTransferBuffer = nil

	common.Log.Infof("BRC20Indexer->UpdateTransfer loop %d in %v", len(block.Transactions), time.Since(startTime))
	//p.CheckSelf(block.Height)
}

// 增加该address下的资产数据
func (s *BRC20Indexer) addHolderBalance(tickerName string, address uint64, amt *common.Decimal) {

	
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

	ticker := s.tickerMap[tickerName].Ticker
	ticker.HolderCount++
	s.tickerUpdated[tickerName] = ticker

	// holders, ok := s.tickerToHolderMap[tickerName]
	// if !ok {
	// 	holders = make(map[uint64]bool, 0)
	// }
	// holders[address] = true
	// s.tickerToHolderMap[tickerName] = holders
}

var err_no_find_holder = fmt.Errorf("no find holder")
var err_no_enough_balance = fmt.Errorf("not enough balance")

// 减少该address下的资产数据
func (s *BRC20Indexer) subHolderBalance(tickerName string, address uint64, amt *common.Decimal) error {

	// if address == 1378943947 && tickerName == "meme" {
	// 	common.Log.Infof("")
	// }

	holdInfo, ok := s.holderMap[address]
	if !ok {
		return err_no_find_holder
	}
	tickAbbrInfo, ok := holdInfo.Tickers[tickerName]
	if !ok {
		return err_no_find_holder
	}
	if tickAbbrInfo.TransferableBalance.Cmp(amt) < 0 {
		return err_no_enough_balance
	}

	tickAbbrInfo.TransferableBalance = tickAbbrInfo.TransferableBalance.Sub(amt)
	// balanceStr := tickinfo.TransferableBalance.String()
	// common.Log.Infof("%s", balanceStr)

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
	return nil
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
			if tickInfo.AssetAmt().Sign() == 0 &&
				len(tickInfo.TransferableData) == 0 {
				delete(holder.Tickers, nft.Ticker)
			}
			// 不能删除，如果删除，就无法删除数据库中对应数据 updateHolderToDB
			// if len(holder.Tickers) == 0 {
			// 	delete(s.holderMap, nft.AddressId)
			// }
		} else {
			common.Log.Panicf("can't find ticker info %s %d", nft.Ticker, nft.UtxoId)
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
		// 多个transfer输出到同一个utxo，这个时候只修改amt
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

func (s *BRC20Indexer) innerUpdateTransfer(txId string, output *common.Output, 
	inputTransferNfts map[int64]*TransferNftInfo) {
	// 检查是否存在nft。如果存在，就更新对应的holder数据
	utxoId := common.GetUtxoId(output)
	// if utxoId == 3804104076034048 {
	// 	common.Log.Infof("")
	// }
	baseIndexer := s.nftIndexer.GetBaseIndexer()
	ids := s.nftIndexer.GetNftsWithUtxo(utxoId) // 有可能多个transfer nft，合并输出到一个output中
	for _, nft := range ids {
		transfer, ok := inputTransferNfts[nft.Base.Id]
		if ok {
			fromAddressId := transfer.AddressId
			toAddressId := baseIndexer.GetAddressId(output.Address.Addresses[0])
			
			flag := common.BRC20_Action_Transfer_Spent
			if !transfer.TransferNft.IsInvalid {
				flag = common.BRC20_Action_Transfer
				transfer.TransferNft.IsInvalid = true

				// transfer occur
				err := s.subHolderBalance(transfer.Ticker, fromAddressId,
					&transfer.TransferNft.Amount)
				if err == err_no_find_holder {
					common.Log.Panic("innerUpdateTransfer call subHolderBalance err: ", err)
					continue
					// return
				} else if err == err_no_enough_balance {
					common.Log.Panic("innerUpdateTransfer call subHolderBalance err: ", err)
					continue
				}

				s.addHolderBalance(transfer.Ticker, toAddressId,
					&transfer.TransferNft.Amount)

				// 再继续加入s.transferNftMap，方便跟踪。在下一次转移时，可以删除，不需要再记录
				transfer.AddressId = toAddressId
				transfer.UtxoId = utxoId
				s.addTransferNft(transfer)
			}
			action := HolderAction{
				BRC20ActionHistory: common.BRC20ActionHistory{
					Height:   output.Height,
					UtxoId:   utxoId,
					NftId:    transfer.TransferNft.NftId,
					FromAddr: fromAddressId,
					ToAddr:   toAddressId,
					Ticker:   transfer.Ticker,
					Amount:   transfer.TransferNft.Amount,
					Action:   flag,
				},
				FromUtxoId: transfer.UtxoId, // old utxo
			}
			s.holderActionList = append(s.holderActionList, &action)
			delete(inputTransferNfts, nft.Base.Id)
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
			if writeToDB {
				err := db.SetDB([]byte(addressTickerKey), value, wb)
				if err != nil {
					common.Log.Panicf("Error setting %s in db %v", addressTickerKey, err)
				}
			}
			if updateTickerAddr {
				amt := value.AssetAmt()
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
			err := wb.Delete([]byte(addressTickerKey))
			if err != nil {
				common.Log.Panicf("Error deleting db %s: %v\n", addressTickerKey, err)
			}
			delete(holder.Tickers, ticker) // 防止多次删除

			if updateTickerAddr {
				err = wb.Delete([]byte(tickerAddressKey))
				if err != nil {
					common.Log.Panicf("Error deleting db %s: %v\n", tickerAddressKey, err)
				}
			}
		}
	} else {
		// 可能重入
		//common.Log.Panicf("no find holder info in holderMap :%d", address)
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
			s.updateUtxoToDB(action.UtxoId, false, wb)

		case common.BRC20_Action_InScribe_Transfer:
			s.updateHolderToDB(action.ToAddr, action.Ticker, false, false, wb)
			s.updateUtxoToDB(action.UtxoId, false, wb)

		case common.BRC20_Action_Transfer:
			if action.FromAddr == common.INVALID_ID {
				common.Log.Panic("action.FromAddr == common.INVALID_ID when action.Action == Action_Transfer")
			}
			s.updateHolderToDB(action.FromAddr, action.Ticker, false, true, wb)
			
			transferKey := GetUtxoToTransferKey(action.FromUtxoId)
			err := wb.Delete([]byte(transferKey))
			if err != nil {
				common.Log.Panicf("Error deleting db %s: %v\n", transferKey, err)
			}
			
			s.updateHolderToDB(action.ToAddr, action.Ticker, false, true, wb)
			s.updateUtxoToDB(action.UtxoId, false, wb)
		
		case common.BRC20_Action_Transfer_Spent:
			if action.FromAddr == common.INVALID_ID {
				common.Log.Panic("action.FromAddr == common.INVALID_ID when action.Action == Action_Transfer")
			}
			s.updateHolderToDB(action.FromAddr, action.Ticker, false, true, wb)
		}

		// 保存历史记录
		key := GetTransferHistoryKey(action.Ticker, action.UtxoId)
		err := db.SetDB([]byte(key), &action.BRC20ActionHistory, wb)
		if err != nil {
			common.Log.Panicf("Error setting %s in db %v", key, err)
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
