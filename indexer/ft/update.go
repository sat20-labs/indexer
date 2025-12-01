package ft

import (
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
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

	if strings.Contains(mint.Base.InscriptionId, "383ef74030578308823d524b5ae24820c68b82f6109324da82b6c6e79e3b143c") {
		common.Log.Infof("")
	}

	name := strings.ToLower(mint.Name)

	ticker, ok := p.tickerMap[name]
	if !ok {
		// 正常不会走到这里，除非是数据从中间跑
		return
	}
	mint.Id = int64(len(ticker.InscriptionMap))

	assetInfo := &common.AssetAbbrInfo{
		MintingNftId: mint.Base.Id, 
		BindingSat: ticker.Ticker.N, 
		Offsets: mint.Offsets.Clone(),
	}
	if !p.addHolder(&in.TxOutputV2, name, assetInfo) {
		// 同一个聪上重复铸造导致的失败
		common.Log.Warnf("FTIndexer UpdateMint %s invalid mint %s", name, in.TxId)
		return
	}

	old, ok := ticker.UtxoMap[mint.UtxoId]
	if ok {
		// 批量铸造。如果是稀有聪，其offset需要调整
		// 最合理方案是根据聪的属性调整，但这里还不知道聪的属性
		old.Merge(mint.Offsets)
	} else {
		ticker.UtxoMap[mint.UtxoId] = mint.Offsets
	}
	ticker.Ticker.TotalMinted += mint.Offsets.Size() * int64(ticker.Ticker.N)
	p.tickerAdded[name] = ticker.Ticker // 更新

	ticker.MintAdded = append(ticker.MintAdded, mint)
	ticker.InscriptionMap[mint.Base.InscriptionId] = common.NewMintAbbrInfo(mint)

}

// 将某次无效铸造的结果清除，一般都是在区块处理前先加入，该区块处理过程中发现铸造无效，就清除
func (p *FTIndexer) removeMint(holder *HolderInfo, ticker *TickInfo, utxoId uint64, mintingAsset *common.AssetAbbrInfo) {
	// for i, action := range p.holderActionList {
	// 	if action.UtxoId == utxoId {
	// 		p.holderActionList = common.RemoveIndex(p.holderActionList, i)
	// 		break
	// 	}
	// }
	// if utxoId == 1022786333310976 {
	// 	common.Log.Info("")
	// }

	mintingAsset = mintingAsset.Clone()
	tickerName := ticker.Name
	holder.RemoveTickerAsset(tickerName, mintingAsset)
	if len(holder.Tickers) == 0 {
		delete(p.holderInfo, utxoId)
	}

	// 考虑批量铸造时，输入的utxo中有多个铸造，其他铸造是有效的，不能直接删除utxo
	// ticker := p.tickerMap[tickerName]
	old, ok := ticker.UtxoMap[utxoId]
	if ok {
		old.Remove(mintingAsset.Offsets)
		if len(old) == 0 {
			delete(ticker.UtxoMap, utxoId)
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
func (p *FTIndexer) addHolder(utxo *common.TxOutputV2, ticker string, assetInfo *common.AssetAbbrInfo) bool {
	info, ok := p.holderInfo[utxo.UtxoId]
	if !ok {
		info = &HolderInfo{
			AddressId: utxo.AddressId,
			IsMinting: assetInfo.MintingNftId != 0,
			Tickers:   make(map[string]map[int64]*common.AssetAbbrInfo, 0),
		}
		p.holderInfo[utxo.UtxoId] = info
	}

	// 执行transfer的过程中，utxo中已经加载了稀有聪资产数据 （让exotic的结果直接放在tx的输出中）
	// assetInfo是转移过来的资产，有时会对铸造中的资产有影响

	// 检查当前utxo是否有mining的资产.
	if info.IsMinting { // 当前utxo涉及minting，也就是当前正在铸造的资产，已经放入info中，而assetInfo是本来就存在的资产，正要放入utxo中
		// utxo是一个inscription的commit tx的输出，而且该输出，在预先处理中，已经添加了铸造资产数据 mintingAsset
		// 这个时候需要对资产信息作检查，判断是否有效
		mintingAssetVector, ok := info.Tickers[ticker]
		if ok {
			for _, mintingAsset := range mintingAssetVector {
				if mintingAsset.MintingNftId != 0 {
					// 检查同名字的铸造，不能在已经铸造有该资产的聪上
					// testnet4: 54bec54ac5c68646753398403bea863c6f015f109b283444b8c8460ee64940ac
					mintingAssetOffsets := mintingAsset.Offsets
					inter := common.IntersectAssetOffsets(mintingAssetOffsets, assetInfo.Offsets)
					if len(inter) != 0 {
						common.Log.Infof("utxo %s mint asset %s on some satoshi with the same asset", utxo.OutPointStr, ticker)
						tickerInfo := p.tickerMap[ticker]
						if assetInfo.MintingNftId != 0 {
							// 两个铸造资产冲突，后面的无效
							//p.removeMint(info, tickerInfo, utxo.UtxoId, assetInfo)
							if len(info.Tickers) == 0 {
								delete(p.holderInfo, utxo.UtxoId)
							}
							return false
						} else {
							p.removeMint(info, tickerInfo, utxo.UtxoId, mintingAsset)
						}
						// info 可能被删除，需要重新加进来
						info, ok = p.holderInfo[utxo.UtxoId]
						if !ok {
							info = &HolderInfo{
								AddressId: utxo.AddressId,
								IsMinting: assetInfo.MintingNftId != 0,
								Tickers:   make(map[string]map[int64]*common.AssetAbbrInfo, 0)}
							p.holderInfo[utxo.UtxoId] = info
						}
					}
				}
			}
		}
	}
	amt := info.AddTickerAsset(ticker, assetInfo)
	utxovalue, ok := p.utxoMap[ticker]
	if !ok {
		utxovalue = make(map[uint64]int64, 0)
		p.utxoMap[ticker] = utxovalue
	}
	utxovalue[utxo.UtxoId] = amt

	return true
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

		if tx.TxId == "383ef74030578308823d524b5ae24820c68b82f6109324da82b6c6e79e3b143c" {
			common.Log.Infof("")
		}

		var allInput *common.TxOutput
		for _, in := range tx.Inputs {
			input := in.Clone()

			utxo := input.UtxoId
		//loopback:
			holder, ok := p.holderInfo[utxo]
			if ok {
				// 如果是批量铸造，需要能区分各个铸造
				tickers := make(map[string]bool)
				for ticker, assetVector := range holder.Tickers {

					tickerInfo := p.tickerMap[ticker]

					var amt int64
					var offsets common.AssetOffsets
					for _, info := range assetVector {
						// if info.MintingNftId != 0 {
						// 	// 检查是否满足要求的属性
						// 	if indexer.IsRaritySatRequired(&tickerInfo.Ticker.Attr) {
						// 		// 如果是稀有聪铸造，需要有对应的资产
						// 		if tickerInfo.Ticker.Attr.Rarity != "" {
						// 			exoticName := common.AssetName{
						// 				Protocol: common.PROTOCOL_NAME_ORDX,
						// 				Type:     common.ASSET_TYPE_EXOTIC,
						// 				Ticker:   tickerInfo.Ticker.Attr.Rarity,
						// 			}
						// 			// 如果是稀有聪铸造，其mint数据中的offset可能不够，
						// 			// 因为中间可能存在白聪：383ef74030578308823d524b5ae24820c68b82f6109324da82b6c6e79e3b143ci4
									
						// 			exoticranges := input.Offsets[exoticName]
						// 			if !common.AssetOffsetsContains(exoticranges, info.Offsets) {
						// 				common.Log.Infof("utxo %s mint asset %s, but no enough exotic satoshi", input.OutPointStr, ticker)
						// 				p.removeMint(holder, tickerInfo, input.UtxoId, info) // 修改了 p.holderInfo 需要从头开始
						// 				goto loopback
						// 			}
						// 		}
						// 	}
						// }
						amt += info.AssetAmt()
						offsets.Merge(info.Offsets)
					}

					assetName := common.AssetName{
						Protocol: common.PROTOCOL_NAME_ORDX,
						Type:     common.ASSET_TYPE_FT,
						Ticker:   ticker,
					}
					asset := common.AssetInfo{
						Name:       assetName,
						Amount:     *common.NewDecimal(amt, 0),
						BindingSat: uint32(tickerInfo.Ticker.N),
					}
					input.Assets.Add(&asset)
					input.Offsets[asset.Name] = offsets
					tickers[ticker] = true
				}

				action := HolderAction{UtxoId: utxo, AddressId: 0, Tickers: tickers, Action: -1}
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

	common.Log.Infof("FTIndexer->UpdateTransfer loop %d in %v", len(block.Transactions), time.Since(startTime))

	p.mutex.Unlock()
	// if !p.CheckSelf(block.Height) {
	// 	common.Log.Panicf("FTIndexer UpdateTransfer CheckSelf %d failed", block.Height)
	// }
}

func (p *FTIndexer) innerUpdateTransfer(tx *common.Transaction,
	input *common.TxOutput) *common.TxOutput {

	change := input
	for _, txOut := range tx.Outputs {

		if txOut.OutPointStr == "9aeb2adcaa104ee63a02247717a8a6a3d14f82cac4fa77863a9cea848ebdd653:0" {
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
			tickers := make(map[string]bool)
			// 只处理ordx资产
			for _, asset := range newOut.Assets {
				if asset.Name.Protocol == common.PROTOCOL_NAME_ORDX &&
					asset.Name.Type == common.ASSET_TYPE_FT {
					offsets := newOut.Offsets[asset.Name]
					assetInfo := &common.AssetAbbrInfo{
						BindingSat: int(asset.BindingSat), 
						Offsets: offsets.Clone()}
					p.addHolder(txOut, asset.Name.Ticker, assetInfo)

					tickers[asset.Name.Ticker] = true
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
