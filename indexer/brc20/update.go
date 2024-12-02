package brc20

import (
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
)

// 每个deploy都调用
func (p *BRC20Indexer) UpdateTick(ticker *common.Brc20Ticker) {
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

// 每个mint都调用这个函数。
func (p *BRC20Indexer) UpdateMint(inUtxo uint64, mint *common.Brc20Mint) {
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

func (p *BRC20Indexer) UpdateTransferTick(inUtxo uint64, mint *common.Brc20Transfer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
}

// 增加该utxo下的资产数据，该资产为ticker，持有人，在mintutxo铸造，资产的聪范围。聪范围可以累加，因为资产都来自不同的utxo。
func (p *BRC20Indexer) addHolder(utxo uint64, ticker string, address uint64, index int, inscriptionId string, rngs []*common.Range) {
	ticker = strings.ToLower(ticker)

	mintinfo := make(map[string][]*common.Range, 0)
	mintinfo[inscriptionId] = rngs

	info, ok := p.holderInfo[utxo]
	if !ok {
		tickinfo := common.TickAbbrInfo{MintInfo: mintinfo}
		tickers := make(map[string]*common.TickAbbrInfo, 0)
		tickers[ticker] = &tickinfo
		info = &HolderInfo{AddressId: address, Index: index, Tickers: tickers}
		p.holderInfo[utxo] = info
	} else {
		info.AddressId = address
		info.Index = index
		tickinfo, ok := info.Tickers[ticker]
		if !ok {
			tickinfo := common.TickAbbrInfo{MintInfo: mintinfo}
			info.Tickers[ticker] = &tickinfo
		} else {
			tickinfo.MintInfo[inscriptionId] = append(tickinfo.MintInfo[inscriptionId], rngs...)
		}
	}

	utxovalue, ok := p.utxoMap[ticker]
	if !ok {
		newutxovalue := make(map[uint64]int64, 0)
		newutxovalue[utxo] = common.GetOrdinalsSize(rngs)
		p.utxoMap[ticker] = &newutxovalue
	} else {
		(*utxovalue)[utxo] += common.GetOrdinalsSize(rngs)
	}
}

func (p *BRC20Indexer) deleteUtxoMap(ticker string, utxo uint64) {
	utxos, ok := p.utxoMap[ticker]
	if ok {
		delete(*utxos, utxo)
	}
}

func (p *BRC20Indexer) UpdateTransfer(block *common.Block) {
	p.mutex.Lock()

	startTime := time.Now()

	for _, tx := range block.Transactions[1:] {
		hasAsset := false
		for _, input := range tx.Inputs {
			utxo := input.UtxoId
			holder, ok := p.holderInfo[utxo]
			if ok {
				hasAsset = true
				action := HolderAction{UtxoId: utxo, AddressId: 0, Tickers: holder.Tickers, Action: -1}
				p.holderActionList = append(p.holderActionList, &action)
				delete(p.holderInfo, utxo)
				for name := range holder.Tickers {
					p.deleteUtxoMap(name, utxo)
				}
			}
		}

		if hasAsset {
			for _, output := range tx.Outputs {
				p.innerUpdateTransfer(output)
			}
		}
	}
	// 保持顺序
	tx := block.Transactions[0]
	for _, output := range tx.Outputs {
		p.innerUpdateTransfer(output)
	}
	common.Log.Infof("OrdxIndexer->UpdateTransfer loop %d in %v", len(block.Transactions), time.Since(startTime))

	p.mutex.Unlock()
	p.CheckSelf(block.Height)
}

func (p *BRC20Indexer) innerUpdateTransfer(output *common.Output) {

	utxo := common.GetUtxoId(output)

	// 检查是否存在ticker。如果存在，就更新对应的holder数据
	bUpdated := false
	tickers := make(map[string]*common.TickAbbrInfo, 0)

	for _, t := range p.tickerMap {
		mintinfo := make(map[string][]*common.Range, 0)
		for _, r := range output.Ordinals {
			// 求相交区间，只记录相交区间
			interRanges := t.MintInfo.FindIntersections(r)
			for _, rng := range interRanges {
				key := rng.Rng
				value := rng.Value.(*common.RBTreeValue_Mint)
				for _, mintutxo := range value.InscriptionIds {
					mintinfo[mintutxo] = append(mintinfo[mintutxo], key)
				}
			}
		}
		for mintutxo, ranges := range mintinfo {
			for i, address := range output.Address.Addresses {
				addressId := p.nftIndexer.GetBaseIndexer().GetAddressId(address)
				p.addHolder(utxo, t.Name, addressId, i, mintutxo, ranges)
			}
			bUpdated = true
		}
		if len(mintinfo) > 0 {
			tickers[strings.ToLower(t.Name)] = &common.TickAbbrInfo{MintInfo: mintinfo}
		}
	}

	if bUpdated {
		for i, address := range output.Address.Addresses {
			addressId := p.nftIndexer.GetBaseIndexer().GetAddressId(address)
			action := HolderAction{UtxoId: utxo, AddressId: addressId, Index: i, Tickers: tickers, Action: 1}
			p.holderActionList = append(p.holderActionList, &action)
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
		key := GetHolderInfoKey(action.UtxoId)
		if action.Action < 0 {
			err := wb.Delete([]byte(key))
			if err != nil {
				common.Log.Infof("Error deleting db %s: %v\n", key, err)
			}
		} else if action.Action > 0 {
			value, ok := p.holderInfo[action.UtxoId]
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

		for tickerName := range action.Tickers {
			key := GetTickerUtxoKey(tickerName, action.UtxoId)
			if action.Action < 0 {
				err := wb.Delete([]byte(key))
				if err != nil {
					common.Log.Infof("Error deleting db %s: %v\n", key, err)
				}
			} else if action.Action > 0 {
				amount := int64(0)
				value, ok := p.utxoMap[tickerName]
				if ok {
					amount, ok = (*value)[action.UtxoId]
					if ok {
						err := common.SetDB([]byte(key), &amount, wb)
						if err != nil {
							common.Log.Panicf("Error setting %s in db %v", key, err)
						}
					} //else {
					// 已经被删除
					// common.Log.Panicf("can't find %s in map of utxo", action.Utxo)
					//}
				} //else {
				// 已经被删除
				// common.Log.Panicf("can't find %s in map of utxo", tickerName)
				//}
			}
		}
	}
	//common.Log.Infof("OrdxIndexer->UpdateDB->SetDB(ticker.HolderActionList(%d), cost: %v",len(p.holderActionList), time.Since(startTime))

	err := wb.Flush()
	if err != nil {
		common.Log.Panicf("Error ordxwb flushing writes to db %v", err)
	}

	// reset memory buffer
	p.holderActionList = make([]*HolderAction, 0)
	p.tickerAdded = make(map[string]*common.Brc20Ticker)
	for _, info := range p.tickerMap {
		info.MintAdded = make([]*common.Mint, 0)
	}

	common.Log.Infof("OrdxIndexer->UpdateDB takse: %v", time.Since(startTime))
}
