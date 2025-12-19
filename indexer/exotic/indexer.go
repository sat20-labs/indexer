package exotic

import (
	"strings"
	"sync"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/base"
	"github.com/sat20-labs/indexer/indexer/db"
)

type TickInfo struct {
	Name           string
	UtxoMap        map[uint64]common.AssetOffsets  // utxoId, 动态的utxo数据
	InscriptionMap map[string]*common.MintAbbrInfo // key: inscriptionId TODO 修改为nftId
	MintAdded      []*common.Mint
	Ticker         *common.Ticker
}

type HolderAction struct {
	UtxoId    uint64
	AddressId uint64
	Tickers   map[string]bool
	Action    int // -1 删除; 1 增加
}

// asset in utxo
type HolderInfo struct {
	AddressId uint64
	IsMinting bool                               // 只要该utxo是minting的reveal tx的input
	Tickers   map[string]map[int64]*common.AssetAbbrInfo // key: ticker->nftId(or 0)。 如果是铸造，每个nftId对应一个AssetAbbrInfo，铸造资产从铸造的reveal输入转移到输出时，就全部合并到nftId=0的项中；否则数组中最多一个元素。
}

func (p *HolderInfo) AddTickerAsset(name string, assetInfo *common.AssetAbbrInfo) int64 {
	tickerAssetVector, ok := p.Tickers[name]
	if !ok {
		tickerAssetVector = make(map[int64]*common.AssetAbbrInfo)
		tickerAssetVector[assetInfo.MintingNftId] = assetInfo.Clone()
		p.Tickers[name] = tickerAssetVector
		return assetInfo.AssetAmt()
	}

	asset, ok := tickerAssetVector[assetInfo.MintingNftId]
	if !ok {
		tickerAssetVector[assetInfo.MintingNftId] = assetInfo.Clone()
	} else {
		asset.Offsets.Merge(assetInfo.Offsets)
	}

	var amt int64
	for _, asset := range tickerAssetVector {
		amt += asset.AssetAmt()
	}

	return amt
}

func (p *HolderInfo) RemoveTickerAsset(name string, assetInfo *common.AssetAbbrInfo) {
	tickerAssetVector, ok := p.Tickers[name]
	if !ok {
		return
	}

	if assetInfo.MintingNftId != 0 {
		delete(tickerAssetVector, assetInfo.MintingNftId)
		if len(tickerAssetVector) == 0 {
			delete(p.Tickers, name)
		}
		return
	}
	
	asset, ok := tickerAssetVector[assetInfo.MintingNftId]
	if !ok {
		return
	}
	
	asset.Offsets.Remove(assetInfo.Offsets)
	if len(asset.Offsets) == 0 {
		delete(tickerAssetVector, assetInfo.MintingNftId)
		if len(tickerAssetVector) == 0 {
			delete(p.Tickers, name)
		}
	}
}

// TODO 加载所有数据，太耗时间和内存，需要优化，参考nft和brc20模块
type ExoticIndexer struct {
	db          common.KVDB
	status      *Status
	baseIndexer *base.BaseIndexer

	mutex sync.RWMutex // 只保护这几个结构

	// exotic sat range
	tickerMap  map[string]*TickInfo        // 用于检索稀有聪. key 稀有聪种类
	holderInfo map[uint64]*HolderInfo      // utxoId -> holder 用于动态更新ticker的holder数据，需要备份到数据库
	utxoMap    map[string]map[uint64]int64 // ticker -> utxoId -> 资产数量. 动态数据，跟随Holder变更，需要保存在数据库中。

	exoticSyncHeight int
	holderActionList []*HolderAction
	tickerAdded      map[string]*common.Ticker // key: ticker
}

var _instance *ExoticIndexer = nil

func getExoticIndexer() *ExoticIndexer {
	return _instance
}

func newExoticTickerInfo(name string) *TickInfo {
	return &TickInfo{
		Name:           name,
		UtxoMap:        make(map[uint64]common.AssetOffsets),
		InscriptionMap: make(map[string]*common.MintAbbrInfo, 0),
		MintAdded:      make([]*common.Mint, 0),
	}
}

func NewExoticIndexer(db common.KVDB) *ExoticIndexer {
	initDefaultExoticAsset()

	_instance = &ExoticIndexer{
		db: db,
	}
	return _instance
}

func (p *ExoticIndexer) Init(baseIndexer *base.BaseIndexer) {
	p.baseIndexer = baseIndexer
	p.status = initStatusFromDB(p.db)
	height := p.baseIndexer.GetSyncHeight()
	// initEpochSat(p.db, height)
	// p.newExoticTickerMap(height)

	ticks := p.getTickListFromDB()
	if true {
		p.mutex.Lock()

		p.tickerMap = make(map[string]*TickInfo, 0)
		for _, ticker := range ticks {
			p.tickerMap[ticker] = p.initTickInfoFromDB(ticker)
		}

		p.holderInfo = p.loadHolderInfoFromDB()
		// 更新ticker数据的utxo数据
		for utxoId, holder := range p.holderInfo {
			for name, assetInfoMap := range holder.Tickers {
				ticker := p.tickerMap[name]
				var offsets common.AssetOffsets
				for _, asset := range assetInfoMap {
					offsets.Merge(asset.Offsets)
				}
				ticker.UtxoMap[utxoId] = offsets
			}
		}
		p.utxoMap = p.loadUtxoMapFromDB()

		p.holderActionList = make([]*HolderAction, 0)
		p.tickerAdded = make(map[string]*common.Ticker, 0)

		p.mutex.Unlock()
	}
	p.exoticSyncHeight = height
}

// 只保存UpdateDB需要用的数据
func (p *ExoticIndexer) Clone() *ExoticIndexer {
	newInst := NewExoticIndexer(p.db)

	newInst.holderActionList = make([]*HolderAction, len(p.holderActionList))
	copy(newInst.holderActionList, p.holderActionList)

	newInst.tickerAdded = make(map[string]*common.Ticker, 0)
	for key, value := range p.tickerAdded {
		newInst.tickerAdded[key] = value
	}

	newInst.tickerMap = make(map[string]*TickInfo, 0)
	for key, value := range p.tickerMap {
		if len(value.MintAdded) > 0 {
			tick := TickInfo{}
			tick.Name = value.Name
			tick.MintAdded = make([]*common.Mint, len(value.MintAdded))
			copy(tick.MintAdded, value.MintAdded)
			newInst.tickerMap[key] = &tick
		}
	}

	// 保存holderActionList对应的数据
	newInst.holderInfo = make(map[uint64]*HolderInfo, 0)
	newInst.utxoMap = make(map[string]map[uint64]int64, 0)
	for _, action := range p.holderActionList {
		if action.Action > 0 {
			value, ok := p.holderInfo[action.UtxoId]
			if ok {
				newTickerInfo := make(map[string]map[int64]*common.AssetAbbrInfo)
				for k, assets := range value.Tickers {
					assetVector := make(map[int64]*common.AssetAbbrInfo)
					for i, v := range assets {
						newAssetInfo := &common.AssetAbbrInfo{
							MintingNftId: v.MintingNftId,
							BindingSat:   v.BindingSat,
							Offsets:      v.Offsets.Clone(),
						}
						assetVector[i] = newAssetInfo
					}
					newTickerInfo[k] = assetVector
				}
				info := HolderInfo{AddressId: value.AddressId, Tickers: newTickerInfo}
				newInst.holderInfo[action.UtxoId] = &info
			} //else {
			// 已经被删除，不存在了
			// common.Log.Panicf("can find utxo %s in holderInfo", action.Utxo)
			//}
		}

		for tickerName := range action.Tickers {
			if action.Action > 0 {
				value, ok := p.utxoMap[tickerName]
				if ok {
					amount, ok := value[action.UtxoId]
					if ok {
						newmap, ok := newInst.utxoMap[tickerName]
						if ok {
							newmap[action.UtxoId] = amount
						} else {
							m := make(map[uint64]int64, 0)
							m[action.UtxoId] = amount
							newInst.utxoMap[tickerName] = m
						}
					} //else {
					// 已经被删除，不存在了
					// common.Log.Panicf("can find utxo %s in utxoMap", action.Utxo)
					//}
				} //else {
				// 已经被删除，不存在了
				// common.Log.Panicf("can find ticker %s in utxoMap", tickerName)
				//}
			}
		}
	}

	return newInst
}

// update之后，删除原来instance中的数据
func (p *ExoticIndexer) Subtract(another *ExoticIndexer) {

	p.holderActionList = append([]*HolderAction(nil), p.holderActionList[len(another.holderActionList):]...)

	for key := range another.tickerAdded {
		delete(p.tickerAdded, key)
	}

	for key, value := range another.tickerMap {
		ticker, ok := p.tickerMap[key]
		if ok {
			ticker.MintAdded = append([]*common.Mint(nil), ticker.MintAdded[len(value.MintAdded):]...)
		}
	}

	// 不需要更新 holderInfo 和 utxoMap
}

func newExoticDefaultTicker(name string) *common.Ticker {
	ticker := &common.Ticker{
		Base: &common.InscribeBaseContent{
			Id:       0,
			TypeName: common.ASSET_TYPE_EXOTIC,

			BlockHeight:        0,
			InscriptionAddress: 0,
			BlockTime:          time.Now().Unix(),
			Content:            nil,
			ContentType:        nil,
			InscriptionId:      "",
		},

		Id:         -1,
		Name:       name,
		Type:       common.ASSET_TYPE_EXOTIC,
		Limit:      1,
		N:          1,
		SelfMint:   0,
		Max:        0,
		BlockStart: 0,
		BlockEnd:   0,
		Attr:       common.SatAttr{},
		Desc:       "Ordinals Rare Sats",
	}

	return ticker
}

func (p *ExoticIndexer) UpdateTransfer(block *common.Block, coinbase []*common.Range) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 生成所有当前区块的稀有聪
	startTime := time.Now()

	// if block.Height == 738 {
	// 	common.Log.Info("")
	// }

	coinbaseInput := common.NewTxOutput(coinbase[0].Size)
	coinbaseInput.UtxoId = block.Transactions[0].Inputs[0].UtxoId
	p.generateRarityAssetWithBlock(block, coinbaseInput)

	// 执行转移
	for _, tx := range block.Transactions[1:] {

		// if tx.TxId == "475ff67b2f2631c6b443635951d81127dcf21898f697d5f7c31e88df836ee756" {
		// 	common.Log.Infof("")
		// }

		var allInput *common.TxOutput
		for _, input := range tx.Inputs {
			utxo := input.UtxoId
			holder, ok := p.holderInfo[utxo]
			if ok {
				tickers := make(map[string]bool)
				for ticker, assetVector := range holder.Tickers {
					for _, info := range assetVector {
						asset := common.AssetInfo{
							Name: common.AssetName{
								Protocol: common.PROTOCOL_NAME_ORDX,
								Type:     common.ASSET_TYPE_EXOTIC,
								Ticker:   ticker,
							},
							Amount:     *common.NewDecimal(info.AssetAmt(), 0),
							BindingSat: 1,
						}
						input.Assets.Add(&asset)
						old, ok := input.Offsets[asset.Name]
						if ok {
							old.Merge(info.Offsets)
						} else {
							input.Offsets[asset.Name] = info.Offsets.Clone()
						}
					}
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

		change := p.innerUpdateTransfer(tx, allInput)
		coinbaseInput.Append(change)
	}

	if block.Height == 501726 {
		common.Log.Infof("")
	}

	if len(coinbaseInput.Assets) != 0 {
		tx := block.Transactions[0]
		change := p.innerUpdateTransfer(tx, coinbaseInput)
		if !change.Zero() {
			common.Log.Panicf("ExoticIndexer.UpdateTransfer should consume all input assets")
		}
	}

	common.Log.Infof("ExoticIndexer.UpdateTransfer in %v", time.Since(startTime))
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
		tickers := make(map[string]map[int64]*common.AssetAbbrInfo, 0)
		assets := make(map[int64]*common.AssetAbbrInfo)
		assets[assetInfo.MintingNftId] = assetInfo.Clone()
		tickers[ticker] = assets
		info = &HolderInfo{
			AddressId: utxo.AddressId, 
			IsMinting: assetInfo.MintingNftId != 0, 
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
	p.holderActionList = make([]*HolderAction, 0)
	p.tickerAdded = make(map[string]*common.Ticker)
	for _, info := range p.tickerMap {
		info.MintAdded = make([]*common.Mint, 0)
	}

	common.Log.Infof("ExoticIndexer->UpdateDB takes %v", time.Since(startTime))
}

func (s *ExoticIndexer) setDBVersion() {
	err := db.SetRawValueToDB([]byte(ORDX_DB_VER_KEY), []byte(ORDX_DB_VERSION), s.db)
	if err != nil {
		common.Log.Panicf("SetRawValueToDB failed %v", err)
	}
}

func (s *ExoticIndexer) GetDBVersion() string {
	value, err := db.GetRawValueFromDB([]byte(ORDX_DB_VER_KEY), s.db)
	if err != nil {
		common.Log.Errorf("GetRawValueFromDB failed %v", err)
		return ""
	}

	return string(value)
}

func (p *ExoticIndexer) CheckSelf() bool {
	//common.Log.Infof("ExoticIndexer->CheckSelf ...")
	startTime := time.Now()
	for name := range p.tickerMap {
		//common.Log.Infof("checking ticker %s", name)
		holdermap := p.GetHolderAndAmountWithTick(name)
		holderAmount := int64(0)
		for _, amt := range holdermap {
			holderAmount += amt
		}

		ticker := p.GetTicker(name)
		mintAmount := ticker.TotalMinted
		if holderAmount != mintAmount {
			common.Log.Errorf("ExoticIndexer ticker %s amount incorrect. %d %d", name, mintAmount, holderAmount)
			return false
		}

		common.Log.Infof("ExoticIndexer %s amount: %d, holders: %d", name, mintAmount, len(holdermap))

		utxos, ok := p.utxoMap[name]
		if !ok {
			if holderAmount != 0 {
				common.Log.Errorf("ExoticIndexer ticker %s has no asset utxos", name)
				return false
			}
		} else {
			amontInUtxos := int64(0)
			for utxo, amoutInUtxo := range utxos {
				amontInUtxos += amoutInUtxo

				holderInfo, ok := p.holderInfo[utxo]
				if !ok {
					common.Log.Errorf("ExoticIndexer ticker %s's utxo %d not in holdermap", name, utxo)
					return false
				}
				tickInfoVector, ok := holderInfo.Tickers[name]
				if !ok {
					common.Log.Errorf("ExoticIndexer ticker %s's utxo %d not in holders", name, utxo)
					return false
				}

				var amountInHolder int64
				for _, info := range tickInfoVector {
					amountInHolder += info.AssetAmt()
				}

				if amountInHolder != amoutInUtxo {
					common.Log.Errorf("ExoticIndexer ticker %s's utxo %d assets %d and %d different", name, utxo, amoutInUtxo, amountInHolder)
					return false
				}
			}
		}
	}

	// // 需要高度到达一定高度才需要检查
	// if p.baseIndexer.IsMainnet() && p.exoticSyncHeight == 920000 {
	// 	// 需要区分主网和测试网
	// 	name := "pearl"
	// 	ticker := p.GetTicker(name)
	// 	if ticker == nil {
	// 		common.Log.Errorf("ExoticIndexer can't find %s in db", name)
	// 		return false
	// 	}

	// 	holdermap := p.GetHolderAndAmountWithTick(name)
	// 	holderAmount := int64(0)
	// 	for _, amt := range holdermap {
	// 		holderAmount += amt
	// 	}

	// 	mintAmount, _ := p.GetMintAmount(name)
	// 	if holderAmount != mintAmount {
	// 		common.Log.Errorf("ExoticIndexer ticker amount incorrect. %d %d", mintAmount, holderAmount)
	// 		return false
	// 	}

	// 	// 1.2.0 版本升级后，pearl的数量增加了105张。原因是之前铸造时，部分输出少于amt的铸造，被错误的识别为无效的铸造。
	// 	// 但实际上，这些铸造是有效的，铸造时已经提供了大于10000的聪，只是大部分铸造出来的pearl，都给了矿工，只有546或者330留在铸造者手里
	// 	// 比如： 5647d570edcbe45d4953915f7b9063e9b39b83432ae2ae13fdbd5283abb83367i0 等
	// 	if ticker.BlockStart == 828200 {
	// 		if holderAmount != 156271012 {
	// 			common.Log.Errorf("ExoticIndexer %s amount incorrect. %d", name, holderAmount)
	// 			return false
	// 		}
	// 	} else {
	// 		common.Log.Errorf("ExoticIndexer Incorrect %s", name)
	// 		return false
	// 	}
	// }

	// 检查holderinfo？
	// for utxo, holderInfo := range s.holderInfo {

	// }

	// 最后才设置dbver
	p.setDBVersion()
	common.Log.Infof("ExoticIndexer CheckSelf took %v.", time.Since(startTime))

	return true
}
