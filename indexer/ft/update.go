package ft

import (
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
	indexer "github.com/sat20-labs/indexer/indexer/common"
)

// 每个deploy都调用
func (p *FTIndexer) UpdateTick(ticker *common.Ticker) {
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
func (p *FTIndexer) UpdateMint(in *common.TxInput, mint *common.Mint) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if strings.Contains(mint.Base.InscriptionId, "ec81dc5b2e62d8bd205da8681995eabcdf2e48a06f17f61abf409b774285831c") {
		common.Log.Infof("") // 1022786333310976
	}

	name := strings.ToLower(mint.Name)

	ticker, ok := p.tickerMap[name]
	if !ok {
		// 正常不会走到这里，除非是数据从中间跑
		return
	}
	mint.Id = int64(len(ticker.InscriptionMap))
	old, ok := ticker.MintInfo[mint.UtxoId]
	if ok {
		old.Append(mint.Offsets)
	} else {
		ticker.MintInfo[mint.UtxoId] = mint.Offsets
	}
	ticker.Ticker.TotalMinted += mint.Offsets.Size() * int64(ticker.Ticker.N)
	p.tickerAdded[name] = ticker.Ticker // 更新
	
	ticker.MintAdded = append(ticker.MintAdded, mint)
	ticker.InscriptionMap[mint.Base.InscriptionId] = common.NewMintAbbrInfo(mint)

	//tickers := make(map[string]*common.AssetAbbrInfo, 0)
	assetInfo := &common.AssetAbbrInfo{MintingNftId: mint.Base.Id, BindingSat: ticker.Ticker.N, Offsets: mint.Offsets.Clone()}
	//tickers[name] = assetInfo
	//action := HolderAction{UtxoId: mint.UtxoId, AddressId: mint.Base.InscriptionAddress, Tickers: tickers, Action: 1}
	//p.holderActionList = append(p.holderActionList, &action)

	// mint 将资产添加到input的资产中
	p.addHolder(&in.TxOutputV2, name, assetInfo)
}

// 将某次无效铸造的结果清除，一般都是在区块处理前先加入，该区块处理过程中发现铸造无效，就清除
func (p *FTIndexer) removeMint(holder *HolderInfo, ticker *TickInfo, utxoId uint64, mintingAsset *common.AssetAbbrInfo) {
	// for i, action := range p.holderActionList {
	// 	if action.UtxoId == utxoId {
	// 		p.holderActionList = common.RemoveIndex(p.holderActionList, i)
	// 		break
	// 	}
	// }
	if utxoId == 1022786333310976 {
		common.Log.Info("")
	}
	
	mintingAsset = mintingAsset.Clone()
	tickerName := ticker.Name
	holder.RemoveTickerAsset(tickerName, mintingAsset)
	if len(holder.Tickers) == 0 {
		delete(p.holderInfo, utxoId)
	}
	

	// 考虑批量铸造时，输入的utxo中有多个铸造，其他铸造是有效的，不能直接删除utxo
	//ticker := p.tickerMap[tickerName]
	old, ok := ticker.MintInfo[utxoId]
	if ok {
		old.Remove(mintingAsset.Offsets)
		if len(old) == 0 {
			delete(ticker.MintInfo, utxoId)
			p.deleteUtxoMap(tickerName, utxoId)
		}
	} else {
		p.deleteUtxoMap(tickerName, utxoId)
	}

	ticker.Ticker.TotalMinted -= mintingAsset.AssetAmt()
	for i, minted := range ticker.MintAdded {
		if minted.Base.Id == mintingAsset.MintingNftId {
			ticker.MintAdded = common.RemoveIndex(ticker.MintAdded, i)
			delete(ticker.InscriptionMap, minted.Base.InscriptionId)
		}
	}
}


// 增加该utxo下的资产数据，该资产为ticker，持有人，
func (p *FTIndexer) addHolder(utxo *common.TxOutputV2, ticker string, assetInfo *common.AssetAbbrInfo) {
	info, ok := p.holderInfo[utxo.UtxoId]
	if !ok {
		info = &HolderInfo{
			AddressId: utxo.AddressId, 
			IsMinting: assetInfo.MintingNftId != 0, 
			Tickers: make(map[string]*common.AssetAbbrInfo, 0)}
		p.holderInfo[utxo.UtxoId] = info
	}

	// minting的数据，在区块处理前，提前就加入了holderInfo中
	if assetInfo.MintingNftId != 0 {
		// 铸造中的资产，直接加进来，因为这个时候的utxo中还没有加载资产数据
		newAssetInfo := info.AddTickerAsset(ticker, assetInfo)
		utxovalue, ok := p.utxoMap[ticker]
		if !ok {
			utxovalue = make(map[uint64]int64, 0)
			p.utxoMap[ticker] = utxovalue
		}
		utxovalue[utxo.UtxoId] = newAssetInfo.AssetAmt()
		return
	}

	// 执行transfer的过程中，utxo中已经加载了稀有聪资产数据 （让exotic的结果直接放在tx的输出中）
	// assetInfo是转移过来的资产，有时会对铸造中的资产有影响 

	// 检查当前utxo是否有mining的资产. 
	if info.IsMinting { // 当前utxo涉及minting，也就是当前正在铸造的资产，已经放入info中，而assetInfo是本来就存在的资产，正要放入utxo中
		// utxo是一个inscription的commit tx的输出，而且该输出，在预先处理中，已经添加了铸造资产数据 mintingAsset
		// 这个时候需要对资产信息作检查，判断是否有效
		mintingAsset, ok := info.Tickers[ticker]
		if ok && mintingAsset.MintingNftId != 0 {
			// 检查同名字的铸造，不能在已经铸造有该资产的聪上
			// testnet4: 54bec54ac5c68646753398403bea863c6f015f109b283444b8c8460ee64940ac
			mintingAssetOffsets := mintingAsset.Offsets
			inter := common.IntersectAssetOffsets(mintingAssetOffsets, assetInfo.Offsets)
			if len(inter) != 0 {
				common.Log.Infof("utxo %s mint asset %s on some satoshi with the same asset", utxo.OutPointStr, ticker)
				tickerInfo := p.tickerMap[ticker]
				p.removeMint(info, tickerInfo, utxo.UtxoId, mintingAsset)
				// info 可能被删除，需要重新加进来
				info, ok = p.holderInfo[utxo.UtxoId]
				if !ok {
					info = &HolderInfo{
						AddressId: utxo.AddressId, 
						IsMinting: assetInfo.MintingNftId != 0, 
						Tickers: make(map[string]*common.AssetAbbrInfo, 0)}
					p.holderInfo[utxo.UtxoId] = info
				}
			}
		}
	}
	newAssetInfo := info.AddTickerAsset(ticker, assetInfo)
	utxovalue, ok := p.utxoMap[ticker]
	if !ok {
		utxovalue = make(map[uint64]int64, 0)
		p.utxoMap[ticker] = utxovalue
	}
	utxovalue[utxo.UtxoId] = newAssetInfo.AssetAmt()
}

func (p *FTIndexer) deleteUtxoMap(ticker string, utxo uint64) {
	utxos, ok := p.utxoMap[ticker]
	if ok {
		delete(utxos, utxo)
	}
}

func (p *FTIndexer) UpdateTransfer(block *common.Block, coinbase []*common.Range) {
	p.mutex.Lock()

	startTime := time.Now()

	coinbaseInput := common.NewTxOutput(coinbase[0].Size)
	for _, tx := range block.Transactions[1:] {

		if tx.TxId == "54bec54ac5c68646753398403bea863c6f015f109b283444b8c8460ee64940ac" {
			common.Log.Infof("")
		}

		var allInput *common.TxOutput
		for _, in := range tx.Inputs {
			input := in.Clone()

			utxo := input.UtxoId
		loopback:
			holder, ok := p.holderInfo[utxo]
			if ok {
				for ticker, info := range holder.Tickers {

					assetName := common.AssetName{
						Protocol: common.PROTOCOL_NAME_ORDX,
						Type: common.ASSET_TYPE_FT,
						Ticker: ticker,
					}

					if info.MintingNftId != 0 {
						tickerInfo := p.tickerMap[ticker]
						// 检查是否满足要求的属性
						if indexer.IsRaritySatRequired(&tickerInfo.Ticker.Attr) {
							// 如果是稀有聪铸造，需要有对应的资产
							if tickerInfo.Ticker.Attr.Rarity != "" {
								exoticName := common.AssetName{
									Protocol: common.PROTOCOL_NAME_ORDX,
									Type: common.ASSET_TYPE_EXOTIC,
									Ticker: tickerInfo.Ticker.Attr.Rarity,
								}
								exoticranges := input.Offsets[exoticName]
								if !common.AssetOffsetsContains(exoticranges, info.Offsets) {
									common.Log.Infof("utxo %s mint asset %s, but no enough exotic satoshi", input.OutPointStr, ticker)
									p.removeMint(holder, tickerInfo, input.UtxoId, info) // 修改了 p.holderInfo 需要从头开始
									goto loopback
								}
							}
						}
					}

					asset := common.AssetInfo{
						Name: assetName,
						Amount: *common.NewDecimal(info.AssetAmt(), 0),
						BindingSat: uint32(info.BindingSat),
					}
					input.Assets.Add(&asset)
					input.Offsets[asset.Name] = info.Offsets
				}

				action := HolderAction{UtxoId: utxo, AddressId: 0, Tickers: holder.Tickers, Action: -1}
				p.holderActionList = append(p.holderActionList, &action)
				delete(p.holderInfo, utxo)
				for name := range holder.Tickers {
					p.deleteUtxoMap(name, utxo)
				}
			}

			if allInput == nil {
				allInput = input.Clone()
			} else {
				allInput.Append(input)
			}
		}

		change := p.innerUpdateTransfer(tx, allInput)
		coinbaseInput.Append(change)
	}

	if len(coinbaseInput.Assets) != 0 {
		tx := block.Transactions[0]
		change := p.innerUpdateTransfer(tx, coinbaseInput)
		if !change.Zero() {
			common.Log.Panicf("UpdateTransfer should consume all input assets")
		}
	}

	common.Log.Infof("OrdxIndexer->UpdateTransfer loop %d in %v", len(block.Transactions), time.Since(startTime))

	p.mutex.Unlock()
	//p.CheckSelf(block.Height)
}

func (p *FTIndexer) innerUpdateTransfer(tx *common.Transaction, 
	input *common.TxOutput) *common.TxOutput {

	change := input
	for _, txOut := range tx.Outputs {

		if txOut.OutPointStr == "1f7601d5272dc6dd28dd8ce36064515287c9551022c0f3da5a74fa431ce3ea4d:0" {
			common.Log.Infof("")
		}

		if txOut.OutValue.Value == 0 {
			continue
		}
		newOut, newChange, err := change.Cut(txOut.OutValue.Value)
		if err != nil {
			common.Log.Panicf("innerUpdateTransfer Cut failed, %v", err)
		}
		change = newChange
		
		if len(newOut.Assets) != 0 {
			tickers := make(map[string]*common.AssetAbbrInfo)
			// 只处理ordx资产
			for _, asset := range newOut.Assets {
				if asset.Name.Protocol == common.PROTOCOL_NAME_ORDX && 
				asset.Name.Type == common.ASSET_TYPE_FT {
					offsets := newOut.Offsets[asset.Name]
					assetInfo := &common.AssetAbbrInfo{BindingSat: int(asset.BindingSat), Offsets: offsets.Clone()}
					tickers[asset.Name.Ticker] = assetInfo
					
					p.addHolder(txOut, asset.Name.Ticker, assetInfo)
				}
			}

			if len(tickers) > 0 {
				addressId := txOut.AddressId
				action := HolderAction{UtxoId: txOut.UtxoId, AddressId: addressId, Tickers: tickers, Action: 1}
				p.holderActionList = append(p.holderActionList, &action)
			}
		}

		// 处理完成，稀有聪资产数据清空，避免下一轮影响稀有聪资产数据的处理
		txOut.Assets = nil
		txOut.Offsets = make(map[common.AssetName]common.AssetOffsets)
	}
	return change
}

// 跟basic数据库同步
func (p *FTIndexer) UpdateDB() {
	//common.Log.Infof("OrdxIndexer->UpdateDB start...")
	startTime := time.Now()

	wb := p.db.NewWriteBatch()
	defer wb.Close()

	for _, v := range p.tickerAdded {
		key := GetTickerKey(v.Name)
		err := db.SetDB([]byte(key), v, wb)
		if err != nil {
			common.Log.Panicf("Error setting %s in db %v", key, err)
		}
	}
	// common.Log.Infof("OrdxIndexer->UpdateDB->SetDB(tickerAdded:%d), cost: %.6fs", len(tickerAdded), time.Since(startTime).Seconds())

	//startTime = time.Now()
	for _, ticker := range p.tickerMap {
		for _, v := range ticker.MintAdded {
			key := GetMintHistoryKey(ticker.Name, v.Base.InscriptionId)
			err := db.SetDB([]byte(key), v, wb)
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
				err := db.SetDB([]byte(key), value, wb)
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
					amount, ok = value[action.UtxoId]
					if ok {
						err := db.SetDB([]byte(key), &amount, wb)
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
	p.tickerAdded = make(map[string]*common.Ticker)
	for _, info := range p.tickerMap {
		info.MintAdded = make([]*common.Mint, 0)
	}

	common.Log.Infof("OrdxIndexer->UpdateDB takse: %v", time.Since(startTime))
}
