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
func (p *FTIndexer) UpdateMint(out *common.TxOutputV2, mint *common.Mint) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	name := strings.ToLower(mint.Name)

	ticker, ok := p.tickerMap[name]
	if !ok {
		// 正常不会走到这里，除非是数据从中间跑
		return
	}
	mint.Id = int64(len(ticker.InscriptionMap))
	ticker.MintInfo[mint.UtxoId] = mint.Offsets
	ticker.Ticker.TotalMinted += mint.Offsets.Size() * int64(ticker.Ticker.N)
	
	ticker.MintAdded = append(ticker.MintAdded, mint)
	ticker.InscriptionMap[mint.Base.InscriptionId] = common.NewMintAbbrInfo(mint)

	tickers := make(map[string]*common.AssetAbbrInfo, 0)
	assetInfo := &common.AssetAbbrInfo{IsMinting: true, BindingSat: ticker.Ticker.N, Offsets: mint.Offsets}
	tickers[name] = assetInfo
	action := HolderAction{UtxoId: mint.UtxoId, AddressId: mint.Base.InscriptionAddress, Tickers: tickers, Action: 1}
	p.holderActionList = append(p.holderActionList, &action)

	// mint 将资产添加到input的资产中
	p.addHolder(out, name, assetInfo)
}

// 增加该utxo下的资产数据，该资产为ticker，持有人，
func (p *FTIndexer) addHolder(utxo *common.TxOutputV2, ticker string, assetInfo *common.AssetAbbrInfo) {
	info, ok := p.holderInfo[utxo.UtxoId]
	if !ok {
		tickers := make(map[string]*common.AssetAbbrInfo, 0)
		tickers[ticker] = assetInfo
		info = &HolderInfo{AddressId: utxo.AddressId, IsMinting: assetInfo.IsMinting, Tickers: tickers}
		p.holderInfo[utxo.UtxoId] = info
	}

	if assetInfo.IsMinting {
		// 铸造中的资产，直接加进来
		info.Tickers[ticker] = assetInfo
		utxovalue, ok := p.utxoMap[ticker]
		if !ok {
			utxovalue = make(map[uint64]int64, 0)
			p.utxoMap[ticker] = utxovalue
		}
		utxovalue[utxo.UtxoId] = assetInfo.AssetAmt()
		return
	}

	// 执行transfer的过程中
	// 转移过来的资产，有时会对铸造中的资产有影响 （让exotic的结果直接放在tx的输出中）

	// 检查当前utxo是否有mining的资产. 
	if info.IsMinting { // 当前utxo涉及minting
		oldAsset, ok := info.Tickers[ticker]
		if ok {
			// 检查同名字的铸造，不能在已经铸造有该资产的聪上
			assetName := common.AssetName{
				Protocol: common.PROTOCOL_NAME_ORDX,
				Type: common.ASSET_TYPE_FT,
				Ticker: ticker,
			}
			assetOffsets := utxo.Offsets[assetName]
			inter := common.IntersectAssetOffsets(assetOffsets, oldAsset.Offsets)
			if len(inter) != 0 {
				common.Log.Infof("utxo %s mint asset %s on some satoshi with the same asset", utxo.OutPointStr, ticker)
				p.clearOneMint(utxo.UtxoId, ticker, oldAsset)
			}
		}
		
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
				exoticranges := utxo.Offsets[exoticName]
				if !common.AssetOffsetsContains(exoticranges, assetInfo.Offsets) {
					common.Log.Infof("utxo %s mint asset %s, but no enough exotic satoshi", utxo.OutPointStr, ticker)
					p.clearOneMint(utxo.UtxoId, ticker, oldAsset)
				}
			}
		}
	}
	info.Tickers[ticker] = assetInfo
	utxovalue, ok := p.utxoMap[ticker]
	if !ok {
		utxovalue = make(map[uint64]int64, 0)
		p.utxoMap[ticker] = utxovalue
	}
	utxovalue[utxo.UtxoId] = assetInfo.AssetAmt()
}

func (p *FTIndexer) clearOneMint(utxoId uint64, tickerName string, mintingAsset *common.AssetAbbrInfo) {
	for i, action := range p.holderActionList {
		if action.UtxoId == utxoId {
			p.holderActionList = common.RemoveIndex(p.holderActionList, i)
			break
		}
	}

	ticker := p.tickerMap[tickerName]
	delete(ticker.MintInfo, utxoId)

	ticker.Ticker.TotalMinted -= mintingAsset.AssetAmt()
	for i, minted := range ticker.MintAdded {
		if minted.UtxoId == utxoId {
			ticker.MintAdded = common.RemoveIndex(ticker.MintAdded, i)
			delete(ticker.InscriptionMap, minted.Base.InscriptionId)
		}
	}
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
		var allInput *common.TxOutput
		for _, in := range tx.Inputs {
			input := in.Clone()

			utxo := input.UtxoId
			holder, ok := p.holderInfo[utxo]
			if ok {
				for ticker, info := range holder.Tickers {
					asset := common.AssetInfo{
						Name: common.AssetName{
							Protocol: common.PROTOCOL_NAME_ORDX,
							Type: common.ASSET_TYPE_NFT,
							Ticker: ticker,
						},
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
	p.CheckSelf(block.Height)
}

func (p *FTIndexer) innerUpdateTransfer(tx *common.Transaction, 
	input *common.TxOutput) *common.TxOutput {

	change := input
	for _, txOut := range tx.Outputs {
		if txOut.OutValue.Value == 0 {
			continue
		}
		newOut, newChange, err := change.Cut(txOut.OutValue.Value)
		if err != nil {
			common.Log.Panicf("innerUpdateTransfer Cut failed, %v", err)
		}
		change = newChange
		
		tickers := make(map[string]*common.AssetAbbrInfo)
		if len(newOut.Assets) != 0 {
			for _, asset := range newOut.Assets {
				offsets := newOut.Offsets[asset.Name]
				assetInfo := &common.AssetAbbrInfo{BindingSat: int(asset.BindingSat), Offsets: offsets}
				tickers[asset.Name.Ticker] = assetInfo
				p.addHolder(txOut, asset.Name.Ticker, assetInfo)
			}

			addressId := txOut.AddressId
			action := HolderAction{UtxoId: txOut.UtxoId, AddressId: addressId, Tickers: tickers, Action: 1}
			p.holderActionList = append(p.holderActionList, &action)
		}
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
