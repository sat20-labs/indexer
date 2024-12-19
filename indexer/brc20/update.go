package brc20

import (
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
func (p *BRC20Indexer) UpdateInscribeMint(inUtxo uint64, mint *common.BRC20Mint) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// ticker, ok := p.tickerMap[strings.ToLower(mint.Name)]
	// if !ok {
	// 	// 正常不会走到这里，除非是数据从中间跑
	// 	return
	// }
	// mint.Id = int64(len(ticker.InscriptionMap))
	// for _, rng := range mint.Ordinals {
	// 	ticker.MintInfo.AddMintInfo(rng, mint.Base.InscriptionId)
	// }
	// ticker.MintAdded = append(ticker.MintAdded, mint)
	// ticker.InscriptionMap[mint.Base.InscriptionId] = common.NewMintAbbrInfo(mint)

	// // action := HolderAction{Utxo: mint.Utxo, Action: 1}
	// // p.holderActionList = append(p.holderActionList, &action)
	// // p.addHolder(mint.Ticker, mint.OwnerAddress, mint.Utxo, mint.Utxo, mint.Ordinals)
	// // 这里加holder，容易跟UpdateTransfer形成双重加holder

	// // 应该在这里将input的utxo加入就行，在UpdateTransfer中做真正的处理
	// mintInfo := make(map[string][]*common.Range, 0)
	// mintInfo[mint.Base.InscriptionId] = mint.Ordinals
	// tickers := make(map[string]*common.TickAbbrInfo, 0)
	// tickers[strings.ToLower(mint.Name)] = &common.TickAbbrInfo{MintInfo: mintInfo}
	// action := HolderAction{UtxoId: inUtxo, AddressId: mint.Base.InscriptionAddress, Tickers: tickers, Action: 1}
	// p.holderActionList = append(p.holderActionList, &action)
	// // 仅仅为了让UpdateTransfer能检查到输入的input中有资产，所以该tx的output才会进行资产检查工作
	// p.addHolder(inUtxo, mint.Name, mint.Base.InscriptionAddress, 0, mint.Base.InscriptionId, mint.Ordinals)
}

// transfer
func (p *BRC20Indexer) UpdateInscribeTransfer(inUtxo uint64, mint *common.BRC20Transfer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

}

// 增加该utxo下的资产数据，该资产为ticker，持有人，在mintutxo铸造，资产的聪范围。聪范围可以累加，因为资产都来自不同的utxo。
func (p *BRC20Indexer) addHolder(ticker string, address uint64, index int, amt common.Decimal) {
	ticker = strings.ToLower(ticker)

	info, ok := p.holderMap[address]
	if !ok {
		tickinfo := common.BRC20TickAbbrInfo{AvailableBalance: amt}
		tickers := make(map[string]*common.BRC20TickAbbrInfo, 0)
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

func (p *BRC20Indexer) deleteHolderMap(ticker string, address uint64) {
	holders, ok := p.tickerToHolderMap[ticker]
	if ok {
		delete(holders, address)
		p.tickerToHolderMap[ticker] = holders
	}
}

func (p *BRC20Indexer) UpdateTransfer(block *common.Block) {
	p.mutex.Lock()

	//startTime := time.Now()

	// for _, tx := range block.Transactions[1:] {
	// 	hasAsset := false
	// 	for _, input := range tx.Inputs {
	// 		addressId := p.nftIndexer.GetBaseIndexer().GetAddressId(input.Address.Addresses[0])
	// 		holder, ok := p.holderMap[addressId]
	// 		if ok {
	// 			hasAsset = true
	// 			action := HolderAction{AddressId: 0, Tickers: holder.Tickers, Action: -1}
	// 			p.holderActionList = append(p.holderActionList, &action)
	// 			delete(p.holderMap, utxo)
	// 			for name := range holder.Tickers {
	// 				p.deleteHolderMap(name, utxo)
	// 			}
	// 		}
	// 	}

	// 	if hasAsset {
	// 		for _, output := range tx.Outputs {
	// 			p.innerUpdateTransfer(output)
	// 		}
	// 	}
	// }
	// // 保持顺序
	// tx := block.Transactions[0]
	// for _, output := range tx.Outputs {
	// 	p.innerUpdateTransfer(output)
	// }
	// common.Log.Infof("OrdxIndexer->UpdateTransfer loop %d in %v", len(block.Transactions), time.Since(startTime))

	p.mutex.Unlock()
	p.CheckSelf(block.Height)
}

func (p *BRC20Indexer) innerUpdateTransfer(output *common.Output) {

	

	// 检查是否存在ticker。如果存在，就更新对应的holder数据
	// bUpdated := false
	// tickers := make(map[string]*common.TickAbbrInfo, 0)

	// for _, t := range p.tickerMap {
		
	// }

	// if bUpdated {
	// 	for i, address := range output.Address.Addresses {
	// 		addressId := p.nftIndexer.GetBaseIndexer().GetAddressId(address)
	// 		action := HolderAction{AddressId: addressId, Index: i, Tickers: tickers, Action: 1}
	// 		p.holderActionList = append(p.holderActionList, &action)
	// 	}
	// }
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
			key := GetMintHistoryKey(ticker.Name, v.Base.InscriptionId)
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
