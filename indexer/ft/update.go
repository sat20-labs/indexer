package ft

import (
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
	inCommon "github.com/sat20-labs/indexer/indexer/common"
)

// 每个deploy都调用，也可以用于更新
func (p *FTIndexer) UpdateTick(in *common.TxInput, ticker *common.Ticker) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.actionBufferMap == nil {
		p.actionBufferMap = make(map[uint64][]*ActionInfo)
	}

	action := &ActionInfo{
		Action: common.BRC20_Action_InScribe_Deploy,
		Input: in,
		Info: ticker,
	}

	p.actionBufferMap[in.UtxoId] = append(p.actionBufferMap[in.UtxoId], action)
}

func (p *FTIndexer) updateTick(in *common.TxInput, ticker *common.Ticker) {

	name := strings.ToLower(ticker.Name)
	org, ok := p.tickerMap[name]
	if !ok {
		ticker.Id = int64(len(p.tickerMap))
		tickinfo := newTickerInfo(name)
		tickinfo.Ticker = ticker
		p.tickerMap[name] = tickinfo
		p.tickerAdded[name] = ticker
		common.Log.Debugf("FTIndexer.updateTick %s deploy ticker %s", ticker.Base.InscriptionId, ticker.Name)
	} else {
		// 仅更新显示内容
		p.tickerAdded[name] = org.Ticker
		common.Log.Debugf("FTIndexer.updateTick %s update ticker %s", ticker.Base.InscriptionId, ticker.Name)
	}
}

// 每个mint都调用这个函数。
func (p *FTIndexer) UpdateMint(in *common.TxInput, mint *common.Mint) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.actionBufferMap == nil {
		p.actionBufferMap = make(map[uint64][]*ActionInfo)
	}

	action := &ActionInfo{
		Action: common.BRC20_Action_InScribe_Mint,
		Input: in,
		Info: mint,
	}

	p.actionBufferMap[in.UtxoId] = append(p.actionBufferMap[in.UtxoId], action)

}

func (p *FTIndexer) updateMint(in *common.TxInput, mint *common.Mint) {

	// if mint.Base.InscriptionId == "cbaabeb030644cd83462b5befb497d84015140acaf5eacf9f797684e0730beb9i89" {
	// 	common.Log.Infof("")
	// }

	name := mint.Name

	ticker, ok := p.tickerMap[name]
	if !ok {
		// 正常不会走到这里，除非是数据从中间跑
		return
	}
	mint.Id = int64(len(ticker.InscriptionMap))

	assetInfo := &common.AssetAbbrInfo{
		BindingSat: ticker.Ticker.N, 
		Offsets: mint.Offsets.Clone(),
	}
	p.addHolder(&in.TxOutputV2, name, assetInfo) // 加入input的utxoId中，后面在transfer中转移到output

	ticker.Ticker.TotalMinted += mint.Offsets.Size() * int64(ticker.Ticker.N)
	p.tickerAdded[name] = ticker.Ticker // 更新

	ticker.MintAdded = append(ticker.MintAdded, mint)
	ticker.InscriptionMap[mint.Base.InscriptionId] = common.NewMintAbbrInfo(mint)
	common.Log.Debugf("FTIndexer.updateMint %s mint ticker %s %d -> %d", mint.Base.InscriptionId, mint.Name, mint.Amt, ticker.Ticker.TotalMinted)
}


// 增加该utxo下的资产数据，该资产为ticker，持有人，
func (p *FTIndexer) addHolder(utxo *common.TxOutputV2, ticker string, assetInfo *common.AssetAbbrInfo) {
	info, ok := p.holderInfo[utxo.UtxoId]
	if !ok {
		info = &HolderInfo{
			AddressId: utxo.AddressId,
			Tickers:   make(map[string]*common.AssetAbbrInfo, 0),
		}
		p.holderInfo[utxo.UtxoId] = info
	}

	amt := info.AddTickerAsset(ticker, assetInfo)
	utxovalue, ok := p.utxoMap[ticker]
	if !ok {
		utxovalue = make(map[uint64]int64, 0)
		p.utxoMap[ticker] = utxovalue
	}
	utxovalue[utxo.UtxoId] = amt
}

func (p *FTIndexer) deleteUtxoMap(ticker string, utxo uint64) {
	utxos, ok := p.utxoMap[ticker]
	if ok {
		delete(utxos, utxo)
	}
}

func (p *FTIndexer) UpdateTransfer(block *common.Block, coinbase []*common.Range) {

	if block.Height < p.enableHeight {
		return
	}

	// if block.Height == 31905 {
	// 	common.Log.Infof("")
	// }

	p.mutex.Lock()

	startTime := time.Now()

	coinbaseInput := common.NewTxOutput(coinbase[0].Size)
	for _, tx := range block.Transactions[1:] {

		// if tx.TxId == "cbaabeb030644cd83462b5befb497d84015140acaf5eacf9f797684e0730beb9" {
		// 	common.Log.Infof("")
		// }

		var allInput *common.TxOutput
		for _, in := range tx.Inputs {
			input := in.Clone()

			utxo := input.UtxoId

			holder := p.holderInfo[utxo]
			actions, ok := p.actionBufferMap[utxo]
			if ok {
				// 需要对将铸造结果先合并到input中，同时处理可能的资产铸造有效性问题
				for _, action := range actions {
					switch action.Action {
					case common.BRC20_Action_InScribe_Deploy:
						p.updateTick(action.Input, action.Info.(*common.Ticker))
					case common.BRC20_Action_InScribe_Mint:
						mint := action.Info.(*common.Mint)
						if holder != nil {
							existingAsset, ok := holder.Tickers[mint.Name]
							if ok {
								// 检查同名字的铸造，不能在已经铸造有该资产的聪上
								// testnet4: 54bec54ac5c68646753398403bea863c6f015f109b283444b8c8460ee64940ac
								mintingAssetOffsets := mint.Offsets
								inter := common.IntersectAssetOffsets(mintingAssetOffsets, existingAsset.Offsets)
								if len(inter) != 0 {
									common.Log.Infof("%s mint asset %s on some satoshi with the same asset", 
										mint.Base.InscriptionId, mint.Name)
									// 这次铸造无效
									continue
								}
							}
						}
						// 没有冲突，加入holder中
						p.updateMint(action.Input, mint)
						holder = p.holderInfo[utxo]
					}
				}
			}

			if holder != nil {

				tickers := make(map[string]bool)
				for ticker, assetInfo := range holder.Tickers {

					tickerInfo := p.tickerMap[ticker]

					assetName := common.AssetName{
						Protocol: common.PROTOCOL_NAME_ORDX,
						Type:     common.ASSET_TYPE_FT,
						Ticker:   ticker,
					}
					asset := common.AssetInfo{
						Name:       assetName,
						Amount:     *common.NewDecimal(assetInfo.AssetAmt(), 0),
						BindingSat: uint32(tickerInfo.Ticker.N),
					}
					input.Assets.Add(&asset)
					input.Offsets[asset.Name] = assetInfo.Offsets.Clone()
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

	p.actionBufferMap = make(map[uint64][]*ActionInfo)

	common.Log.Infof("FTIndexer->UpdateTransfer loop %d in %v", len(block.Transactions), time.Since(startTime))

	p.mutex.Unlock()
	p.CheckPointWithBlockHeight(block.Height)

	if inCommon.STEP_RUN_MODE && !p.CheckSelf(block.Height) {
		common.Log.Panic("")
	}
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

	}
	return change
}

// 跟basic数据库同步
func (p *FTIndexer) UpdateDB() {
	//common.Log.Infof("OrdxIndexer->UpdateDB start...")

	if p.nftIndexer.GetBaseIndexer().GetHeight() < p.enableHeight {
		return
	}

	startTime := time.Now()

	wb := p.db.NewWriteBatch()
	defer wb.Close()

	for _, v := range p.tickerAdded {
		key := GetTickerKey(strings.ToLower(v.Name))
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
