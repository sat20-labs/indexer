package exotic

import (
	"sort"
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)


func (p *ExoticIndexer) UpdateTransfer(block *common.Block, coinbase []*common.Range) {
	p.mutex.Lock()
	

	// 生成所有当前区块的稀有聪
	startTime := time.Now()

	// if block.Height == 738 {
	// 	common.Log.Info("")
	// }

	// 预加载输入
	p.PrepareUpdateTransfer(block, coinbase)

	// 在tx输出中生成稀有聪
	coinbaseInput := common.NewTxOutput(coinbase[0].Size)
	coinbaseInput.UtxoId = block.Transactions[0].Inputs[0].UtxoId
	p.generateRarityAssetWithBlock(block, coinbaseInput)

	// 执行转移
	for i, tx := range block.Transactions[1:] {

		// if tx.TxId == "475ff67b2f2631c6b443635951d81127dcf21898f697d5f7c31e88df836ee756" {
		// 	common.Log.Infof("")
		// }

		change := p.TxInputProcess(i+1, tx, block, coinbase)
		coinbaseInput.Append(change)
	}

	if len(coinbaseInput.Assets) != 0 {
		tx := block.Transactions[0]
		change := p.innerUpdateTransfer(tx, coinbaseInput)
		if !change.Zero() {
			common.Log.Panicf("ExoticIndexer.UpdateTransfer should consume all input assets")
		}
	}

	p.mutex.Unlock()

	common.Log.Infof("ExoticIndexer.UpdateTransfer in %v", time.Since(startTime))

	p.CheckPointWithBlockHeight(block.Height)

	// if !p.CheckSelf() {
	// 	common.Log.Panic("")
	// }
}

func (p *ExoticIndexer) deleteUtxoMap(ticker string, utxo uint64) {
	utxos, ok := p.utxoMap[ticker]
	if ok {
		delete(utxos, utxo)
	}
}

// 增加该utxo下的资产数据，该资产为ticker，持有人，
func (p *ExoticIndexer) addHolder(utxo *common.TxOutputV2, ticker string, assetInfo *common.AssetAbbrInfo) {
	var amt int64
	info, ok := p.holderInfo[utxo.UtxoId]
	if !ok {
		tickers := make(map[string]*common.AssetAbbrInfo, 0)
		assets := assetInfo.Clone()
		tickers[ticker] = assets
		info = &HolderInfo{
			AddressId: utxo.AddressId, 
			Tickers: tickers}
		p.holderInfo[utxo.UtxoId] = info
		amt = assetInfo.AssetAmt()
	} else {
		amt = info.AddTickerAsset(ticker, assetInfo)
	}

	utxovalue, ok := p.utxoMap[ticker]
	if !ok {
		utxovalue = make(map[uint64]int64, 0)
		p.utxoMap[ticker] = utxovalue
	}
	utxovalue[utxo.UtxoId] = amt
}

func (p *ExoticIndexer) innerUpdateTransfer(tx *common.Transaction,
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

		if len(newOut.Assets) != 0 {
			txOut.Assets = newOut.Assets
			txOut.Offsets = newOut.Offsets

			tickers := make(map[string]bool)
			for _, asset := range newOut.Assets {
				if asset.Name.Protocol == common.PROTOCOL_NAME_ORDX &&
					asset.Name.Type == common.ASSET_TYPE_EXOTIC {
					offsets := newOut.Offsets[asset.Name]
					assetInfo := &common.AssetAbbrInfo{BindingSat: int(asset.BindingSat), Offsets: offsets}
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

// 跟base数据库同步
func (p *ExoticIndexer) UpdateDB() {
	//common.Log.Infof("ExoticIndexer->UpdateDB start...")
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
	// common.Log.Infof("ExoticIndexer->UpdateDB->SetDB(tickerAdded:%d), cost: %.6fs", len(tickerAdded), time.Since(startTime).Seconds())

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
	//common.Log.Infof("ExoticIndexer->UpdateDB->SetDB(ticker.MintAdded(), cost: %v", time.Since(startTime))
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
	//common.Log.Infof("ExoticIndexer->UpdateDB->SetDB(ticker.HolderActionList(%d), cost: %v",len(p.holderActionList), time.Since(startTime))

	err := db.SetDB([]byte(STATUS_KEY), p.status, wb)
	if err != nil {
		common.Log.Panicf("ExoticIndexer->UpdateDB Error setting in db %v", err)
	}

	err = wb.Flush()
	if err != nil {
		common.Log.Panicf("Error ordxwb flushing writes to db %v", err)
	}

	// reset memory buffer
	p.holderInfo = make(map[uint64]*HolderInfo)
	//p.utxoMap = make(map[string]map[uint64]int64)
	p.holderActionList = make([]*HolderAction, 0)
	p.tickerAdded = make(map[string]*common.Ticker)
	for _, info := range p.tickerMap {
		info.MintAdded = make([]*common.Mint, 0)
	}

	common.Log.Infof("ExoticIndexer->UpdateDB takes %v", time.Since(startTime))
}


// 实现新的区块处理接口
func (p *ExoticIndexer) PrepareUpdateTransfer(block *common.Block, coinbase []*common.Range) {

	// if block.Height == 29601 {
	// 	common.Log.Infof("")
	// }

	// pebble 随机读取性能差，调整读的顺序
	// 预加载相关地址的数据: ticker, holder, utxo
	p.db.View(func(txn common.ReadBatch) error {
		// 先加载ticker
		tickerToLoad := make(map[string]bool)

		// 加载mint涉及到地址
		type pair struct {
			key       string
			utxoId    uint64
			addressId uint64
		}

		// 预处理区块本身的交易
		utxoToLoad := make([]*pair, 0)
		for _, tx := range block.Transactions[1:] {
			for _, input := range tx.Inputs {
				// if tx.TxId == "e7115ee426b1a36f7aa9a0463798ec1aa173953a45daa7966bee8096a5254778" {
				// 	common.Log.Infof("utxoId = %d", input.UtxoId)
				// }
				holder, ok := p.holderInfo[input.UtxoId] // 本区块生成的transfer没有在这里面
				if ok {
					for name := range holder.Tickers {
						tickerToLoad[name] = true
					}
					continue
				}
				
				utxoToLoad = append(utxoToLoad, &pair{
					key:       GetHolderInfoKey(input.UtxoId),
					utxoId:    input.UtxoId,
					addressId: input.AddressId,
				})
			}
		}
		// pebble数据库的优化手段: 尽可能将随机读变成按照key的顺序读
		sort.Slice(utxoToLoad, func(i, j int) bool {
			return utxoToLoad[i].key < utxoToLoad[j].key
		})
		for _, v := range utxoToLoad {
			value, err := p.loadUtxoInfoFromDB(v.utxoId)
			if err != nil {
				continue // 没有transfer铭文，忽略
			}
			p.holderInfo[v.utxoId] = value

			for name := range value.Tickers {
				tickerToLoad[name] = true
			}
		}

		tickerKeys := make([]string, len(tickerToLoad))
		for k := range tickerToLoad {
			tickerKeys = append(tickerKeys, GetTickerKey(k))
			if _, ok := p.utxoMap[k]; !ok {
				p.utxoMap[k] = p.loadTickerToUtxoMapFromDB(k)
			}
		}
		sort.Slice(tickerKeys, func(i, j int) bool {
			return tickerKeys[i] < tickerKeys[j]
		})
		for _, key := range tickerKeys {
			ticker := p.loadTickerFromDB(key)
			if ticker == nil {
				continue
			}

			p.tickerMap[strings.ToLower(ticker.Name)] = &TickInfo{
				Name:   strings.ToLower(ticker.Name),
				Ticker: ticker,
			}
		}

		return nil
	})
}
func (p *ExoticIndexer) TxInputProcess(txIndex int, tx *common.Transaction, 
	block *common.Block, coinbase []*common.Range,
) *common.TxOutput {
	var allInput *common.TxOutput
	for _, input := range tx.Inputs {
		utxo := input.UtxoId
		holder, ok := p.holderInfo[utxo]
		if ok {
			tickers := make(map[string]bool)
			for ticker, assetInfo := range holder.Tickers {
				//for _, info := range assetVector {
					asset := common.AssetInfo{
						Name: common.AssetName{
							Protocol: common.PROTOCOL_NAME_ORDX,
							Type:     common.ASSET_TYPE_EXOTIC,
							Ticker:   ticker,
						},
						Amount:     *common.NewDecimal(assetInfo.AssetAmt(), 0),
						BindingSat: 1,
					}
					input.Assets.Add(&asset)
					old, ok := input.Offsets[asset.Name]
					if ok {
						old.Merge(assetInfo.Offsets)
					} else {
						input.Offsets[asset.Name] = assetInfo.Offsets.Clone()
					}
				//}
				tickers[ticker] = true
			}

			action := HolderAction{UtxoId: utxo, AddressId: 0, Tickers: tickers, Action: -1}
			p.holderActionList = append(p.holderActionList, &action)

			delete(p.holderInfo, utxo)
			for name := range holder.Tickers {
				p.deleteUtxoMap(name, utxo)
			}
		}

		// 当前区块生成的各种稀有聪资产

		if allInput == nil {
			allInput = input.Clone()
		} else {
			allInput.Append(&input.TxOutput)
		}
	}

	return p.innerUpdateTransfer(tx, allInput)
}

func (p *ExoticIndexer) UpdateTransferFinished(block *common.Block) {
	
}
