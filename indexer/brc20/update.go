package brc20

import (
	"fmt"
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
)

// deploy
func (p *BRC20Indexer) UpdateInscribeDeploy(ticker *common.BRC20Ticker) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	name := strings.ToLower(ticker.Name)
	org, ok := p.tickerMap[name]
	if !ok {
		ticker.Id = int64(len(p.tickerMap))
		tickinfo := newTickerInfo(ticker.Name)
		tickinfo.Ticker = ticker
		p.tickerMap[name] = tickinfo
		p.tickerAdded[name] = ticker
	} else {
		// 仅更新显示内容
		p.tickerAdded[name] = org.Ticker
	}
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
	mint.Id = int64(len(ticker.InscriptionMap))
	ticker.MintAdded = append(ticker.MintAdded, mint)
	ticker.InscriptionMap[mint.Base.Base.InscriptionId] = common.NewBRC20MintAbbrInfo(mint)

	action := HolderAction{AddressId: mint.Base.Base.InscriptionAddress, Ticker: mint.Name, Amount: mint.Amt, Action: 1}
	p.holderActionList = append(p.holderActionList, &action)
	// 仅仅为了让UpdateTransfer能检查到输入的input中有资产，所以该tx的output才会进行资产检查工作
	p.addHolderBalance(mint.Name, mint.Base.Base.InscriptionAddress, 0, mint.Amt)
}

// transfer
func (p *BRC20Indexer) UpdateInscribeTransfer(transfer *common.BRC20Transfer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	nft := common.TransferNFT{
		NftId: transfer.Base.Base.Id,
		UtxoId: transfer.UtxoId,
		Amount: transfer.Amt,
	}

	holder, ok := p.holderMap[transfer.Base.Base.InscriptionAddress]
	if ok {
		tickInfo, ok := holder.Tickers[transfer.Name]
		if ok {
			tickInfo.TransferableData = append(tickInfo.TransferableData, &nft)
			return 
		} 
	}
	
	// 异常路径：没有可用余额，保存起来
	availableBalance := common.NewDecimal(0, transfer.Amt.Precition)
	p.addHolderBalance(transfer.Name, transfer.Base.OwnerAddressId, 0, *availableBalance)

	holder = p.holderMap[transfer.Base.OwnerAddressId]
	tickInfo := holder.Tickers[transfer.Name]
	tickInfo.InvalidTransferableData = []*common.TransferNFT{&nft}

	action := HolderAction{AddressId: transfer.Base.OwnerAddressId, Ticker: transfer.Name, Amount: transfer.Amt, Action: 0}
	p.holderActionList = append(p.holderActionList, &action)
}

// 增加该address下的资产数据
func (p *BRC20Indexer) addHolderBalance(ticker string, address uint64, index int, amt common.Decimal) {
	ticker = strings.ToLower(ticker)

	info, ok := p.holderMap[address]
	if !ok {
		tickinfo := common.BRC20TickAbbrInfo{AvailableBalance: amt}
		tickers := make(map[string]*common.BRC20TickAbbrInfo)
		tickers[ticker] = &tickinfo
		info = &HolderInfo{AddressId: address, Index: index, Tickers: tickers}
		p.holderMap[address] = info
	} else {
		info.AddressId = address
		info.Index = index
		tickinfo, ok := info.Tickers[ticker]
		if !ok {
			tickinfo := common.BRC20TickAbbrInfo{AvailableBalance: amt}
			info.Tickers[ticker] = &tickinfo
		} else {
			tickinfo.AvailableBalance.Add(&amt)
		}
	}

	holders, ok := p.tickerToHolderMap[ticker]
	if !ok {
		holders = make(map[uint64]bool, 0)
		holders[address] = true
		p.tickerToHolderMap[ticker] = holders
	} else {
		holders[address] = true
		p.tickerToHolderMap[ticker] = holders
	}
}


// 减少该address下的资产数据
func (p *BRC20Indexer) subHolderBalance(ticker string, address uint64, index int, amt common.Decimal) error {
	ticker = strings.ToLower(ticker)

	info, ok := p.holderMap[address]
	if ok {
		info.AddressId = address
		info.Index = index
		tickinfo, ok := info.Tickers[ticker]
		if ok {
			cmp := tickinfo.AvailableBalance.Cmp(&amt) 
			if cmp >= 0 {
				tickinfo.AvailableBalance.Sub(&amt)
				if cmp == 0 {
					holders := p.tickerToHolderMap[ticker]
					delete(holders, address)
					p.tickerToHolderMap[ticker] = holders
				}
				return nil
			}
		}
	}
	return fmt.Errorf("no enough balance")
}

func (p *BRC20Indexer) deleteHolderMap(ticker string, address uint64) {
	holders, ok := p.tickerToHolderMap[ticker]
	if ok {
		delete(holders, address)
		p.tickerToHolderMap[ticker] = holders
	}
}

func (p *BRC20Indexer) UpdateTransfer(block *common.Block) {
	p.mutex.Lock()

	startTime := time.Now()

	// 检查transferNft转入到哪个输出
	inputTransferNfts := make(map[int64]*TransferNftInfo)
	for _, tx := range block.Transactions[1:] {
		hasTransfer := false
		for _, input := range tx.Inputs {
			nft, ok := p.transferNftMap[input.UtxoId] 
			if ok {
				inputTransferNfts[nft.TransferNft.NftId]= nft
				hasTransfer = true
				delete(p.transferNftMap, input.UtxoId)
			}
		}

		if hasTransfer {
			for _, output := range tx.Outputs {
				p.innerUpdateTransfer(output, &inputTransferNfts)
			}
		}
	}
	
	if len(inputTransferNfts) != 0 {
		tx := block.Transactions[0]
		for _, output := range tx.Outputs {
			p.innerUpdateTransfer(output, &inputTransferNfts)
		}
	}
	

	common.Log.Infof("BRC20Indexer->UpdateTransfer loop %d in %v", len(block.Transactions), time.Since(startTime))

	p.mutex.Unlock()
	p.CheckSelf(block.Height)
}

func (p *BRC20Indexer) innerUpdateTransfer(output *common.Output, inputTransferNfts *map[int64]*TransferNftInfo) {

	// 检查是否存在nft。如果存在，就更新对应的holder数据
	utxoId := common.GetUtxoId(output)
	ids := p.nftIndexer.GetNftsWithUtxo(utxoId)
	for _, nft := range ids {
		transferNft, ok := (*inputTransferNfts)[nft.Base.Id]
		if ok {
			// transfer occur

			err := p.subHolderBalance(transferNft.Ticker, transferNft.AddressId, transferNft.Index, transferNft.TransferNft.Amount)
			if err != nil {
				return
			}
			action := HolderAction{AddressId: transferNft.AddressId, Index: transferNft.Index, 
				Ticker: transferNft.Ticker, Amount: transferNft.TransferNft.Amount, Action: -1}
			p.holderActionList = append(p.holderActionList, &action)

			p.addHolderBalance(transferNft.Ticker, nft.OwnerAddressId, 0, transferNft.TransferNft.Amount)
			action2 := HolderAction{AddressId: nft.OwnerAddressId, Index: 0, 
				Ticker: transferNft.Ticker, Amount: transferNft.TransferNft.Amount, Action: 1}
			p.holderActionList = append(p.holderActionList, &action2)

			delete((*inputTransferNfts), nft.Base.Id)
		}
	}
}

// 跟basic数据库同步
func (p *BRC20Indexer) UpdateDB() {
	//common.Log.Infof("OrdxIndexer->UpdateDB start...")
	startTime := time.Now()

	wb := p.db.NewWriteBatch()
	defer wb.Cancel()

	for _, v := range p.tickerAdded {
		key := GetTickerKey(v.Name)
		err := common.SetDB([]byte(key), v, wb)
		if err != nil {
			common.Log.Panicf("Error setting %s in db %v", key, err)
		}
	}
	// common.Log.Infof("OrdxIndexer->UpdateDB->SetDB(tickerAdded:%d), cost: %.6fs", len(tickerAdded), time.Since(startTime).Seconds())

	//startTime = time.Now()
	for _, ticker := range p.tickerMap {
		for _, v := range ticker.MintAdded {
			key := GetMintHistoryKey(ticker.Name, v.Base.Base.InscriptionId)
			err := common.SetDB([]byte(key), v, wb)
			if err != nil {
				common.Log.Panicf("Error setting %s in db %v", key, err)
			}
		}
	}
	//common.Log.Infof("OrdxIndexer->UpdateDB->SetDB(ticker.MintAdded(), cost: %v", time.Since(startTime))
	//startTime = time.Now()

	for _, action := range p.holderActionList {
		key := GetHolderInfoKey(action.AddressId)
		if action.Action < 0 {
			err := wb.Delete([]byte(key))
			if err != nil {
				common.Log.Infof("Error deleting db %s: %v\n", key, err)
			}
		} else if action.Action > 0 {
			value, ok := p.holderMap[action.AddressId]
			if ok {
				err := common.SetDB([]byte(key), value, wb)
				if err != nil {
					common.Log.Panicf("Error setting %s in db %v", key, err)
				}
			} //else {
			//已经被删除
			//common.Log.Panicf("can't find %s in map of holderInfo", key)
			//}
		}

		// for tickerName := range action.Tickers {
		// 	key := GetHolderInfoKey(tickerName, action.AddressId)
		// 	if action.Action < 0 {
		// 		err := wb.Delete([]byte(key))
		// 		if err != nil {
		// 			common.Log.Infof("Error deleting db %s: %v\n", key, err)
		// 		}
		// 	} else if action.Action > 0 {
		// 		amount := int64(0)
		// 		value, ok := p.tickerToHolderMap[tickerName]
		// 		if ok {
		// 			amount, ok = (value)[action.AddressId]
		// 			if ok {
		// 				err := common.SetDB([]byte(key), &amount, wb)
		// 				if err != nil {
		// 					common.Log.Panicf("Error setting %s in db %v", key, err)
		// 				}
		// 			} //else {
		// 			// 已经被删除
		// 			// common.Log.Panicf("can't find %s in map of utxo", action.Utxo)
		// 			//}
		// 		} //else {
		// 		// 已经被删除
		// 		// common.Log.Panicf("can't find %s in map of utxo", tickerName)
		// 		//}
		// 	}
		// }
	}
	//common.Log.Infof("OrdxIndexer->UpdateDB->SetDB(ticker.HolderActionList(%d), cost: %v",len(p.holderActionList), time.Since(startTime))

	err := wb.Flush()
	if err != nil {
		common.Log.Panicf("Error ordxwb flushing writes to db %v", err)
	}

	// reset memory buffer
	p.holderActionList = make([]*HolderAction, 0)
	p.tickerAdded = make(map[string]*common.BRC20Ticker)
	for _, info := range p.tickerMap {
		info.MintAdded = make([]*common.BRC20Mint, 0)
	}

	common.Log.Infof("OrdxIndexer->UpdateDB takse: %v", time.Since(startTime))
}
