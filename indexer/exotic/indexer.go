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
	Id             uint64
	Name           string
	MintInfo       map[uint64]common.AssetOffsets
	InscriptionMap map[string]*common.MintAbbrInfo // key: inscriptionId
	MintAdded      []*common.Mint
	Ticker         *common.Ticker
}

type HolderAction struct {
	UtxoId    uint64
	AddressId uint64
	Tickers   map[string]*common.AssetAbbrInfo
	Action    int // -1 删除; 1 增加
}

// asset in utxo
type HolderInfo struct {
	AddressId uint64
	IsMinting bool  // 只要该utxo是minting的reveal tx的input
	Tickers   map[string]*common.AssetAbbrInfo // key: ticker, 小写
}


type ExoticIndexer struct {
	db          common.KVDB
	baseIndexer *base.BaseIndexer

	mutex sync.RWMutex // 只保护这几个结构

	// exotic sat range
	tickerMap  map[string]*TickInfo // 用于检索稀有聪. key 稀有聪种类
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
		MintInfo:       make(map[uint64]common.AssetOffsets),
		InscriptionMap: make(map[string]*common.MintAbbrInfo, 0),
		MintAdded:      make([]*common.Mint, 0),
	}
}

func NewExoticIndexer(db common.KVDB) *ExoticIndexer {
	_instance = &ExoticIndexer{
		db:              db,
	}
	return _instance
}

func (p *ExoticIndexer) Init(baseIndexer *base.BaseIndexer) {
	p.baseIndexer = baseIndexer
	height := p.baseIndexer.GetSyncHeight()
	// initEpochSat(p.db, height)
	// p.newExoticTickerMap(height)

	ticks := p.getTickListFromDB()
	if true {
		p.mutex.Lock()

		p.tickerMap = make(map[string]*TickInfo, 0)
		for _, ticker := range ticks {
			p.tickerMap[strings.ToLower(ticker)] = p.initTickInfoFromDB(ticker)
		}

		p.holderInfo = p.loadHolderInfoFromDB()
		p.utxoMap = p.loadUtxoMapFromDB()

		p.holderActionList = make([]*HolderAction, 0)
		p.tickerAdded = make(map[string]*common.Ticker, 0)

		p.mutex.Unlock()
	}
	p.exoticSyncHeight = height
}

// 只保存UpdateDB需要用的数据
func (s *ExoticIndexer) Clone() *ExoticIndexer {
	newInst := NewExoticIndexer(s.db)
	
	newInst.holderActionList = make([]*HolderAction, len(s.holderActionList))
	copy(newInst.holderActionList, s.holderActionList)

	newInst.tickerAdded = make(map[string]*common.Ticker, 0)
	for key, value := range s.tickerAdded {
		newInst.tickerAdded[key] = value
	}

	newInst.tickerMap = make(map[string]*TickInfo, 0)
	for key, value := range s.tickerMap {
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
	for _, action := range s.holderActionList {
		if action.Action > 0 {
			value, ok := s.holderInfo[action.UtxoId]
			if ok {
				info := HolderInfo{AddressId: value.AddressId, Tickers: value.Tickers}
				newInst.holderInfo[action.UtxoId] = &info
			} //else {
			// 已经被删除，不存在了
			// common.Log.Panicf("can find utxo %s in holderInfo", action.Utxo)
			//}
		}

		for tickerName := range action.Tickers {
			if action.Action > 0 {
				value, ok := s.utxoMap[tickerName]
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
func (s *ExoticIndexer) Subtract(another *ExoticIndexer) {

	s.holderActionList = append([]*HolderAction(nil), s.holderActionList[len(another.holderActionList):]...)

	for key := range another.tickerAdded {
		delete(s.tickerAdded, key)
	}

	for key, value := range another.tickerMap {
		ticker, ok := s.tickerMap[key]
		if ok {
			ticker.MintAdded = append([]*common.Mint(nil), ticker.MintAdded[len(value.MintAdded):]...)
		}
	}

	// 不需要更新 holderInfo 和 utxoMap
}

func (p *ExoticIndexer) getExoticDefaultTicker(name string) *common.Ticker {
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
		Limit:      100000000,
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

func (p *ExoticIndexer) generateAssetWithBlock(block *common.Block, coinbase []*common.Range) {

}

func (p *ExoticIndexer) UpdateTransfer(block *common.Block, coinbase []*common.Range) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 生成所有当前区块的稀有聪
	startTime := time.Now()

	p.generateAssetWithBlock(block, coinbase)

	coinbaseInput := common.NewTxOutput(coinbase[0].Size)
	for _, tx := range block.Transactions[1:] {
		var allInput *common.TxOutput
		for _, input := range tx.Inputs {
			utxo := input.UtxoId
			holder, ok := p.holderInfo[utxo]
			if ok {
				for ticker, info := range holder.Tickers {
					asset := common.AssetInfo{
						Name: common.AssetName{
							Protocol: common.PROTOCOL_NAME_ORD,
							Type: common.ASSET_TYPE_EXOTIC,
							Ticker: ticker,
						},
						Amount: *common.NewDecimal(info.AssetAmt(), 0),
						BindingSat: 1,
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
	info, ok := p.holderInfo[utxo.UtxoId]
	if !ok {
		tickers := make(map[string]*common.AssetAbbrInfo, 0)
		tickers[ticker] = assetInfo
		info = &HolderInfo{AddressId: utxo.AddressId, IsMinting: assetInfo.IsMinting, Tickers: tickers}
		p.holderInfo[utxo.UtxoId] = info
	}

	_, ok = info.Tickers[ticker]
	if ok {
		common.Log.Panicf("ExoticIndexer.addHolder utxo %s already has asset %s", utxo.OutPointStr, ticker)
	} else {
		info.Tickers[ticker] = assetInfo
	}
	
	utxovalue, ok := p.utxoMap[ticker]
	if !ok {
		utxovalue = make(map[uint64]int64, 0)
		p.utxoMap[ticker] = utxovalue
	}
	utxovalue[utxo.UtxoId] = assetInfo.AssetAmt()
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
		
		tickers := make(map[string]*common.AssetAbbrInfo)
		if len(newOut.Assets) != 0 {
			txOut.Assets = newOut.Assets
			txOut.Offsets = newOut.Offsets
			
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

func (p *ExoticIndexer) getMoreExoticRangesToHeight(startHeight, endHeight int) map[string][]*common.Range {
	if p.baseIndexer.GetHeight() < 0 {
		return nil
	}

	var result map[string][]*common.Range
	p.db.View(func(txn common.ReadBatch) error {
		result = p.getMoreRodarmorRarityRangesToHeight(startHeight, endHeight, txn)
		// TODO
		//result[Alpha] = p.GetRangesForAlpha(startHeight, endHeight)
		//result[Omega] = p.GetRangesForOmega(startHeight, endHeight)
		if endHeight >= 9 {
			result[Block9] = p.getRangeForBlock(9, txn)
		}
		if endHeight >= 78 {
			result[Block78] = p.getRangeForBlock(78, txn)
		}
		validBlock := make([]int, 0)
		for h := range NakamotoBlocks {
			if h <= endHeight {
				validBlock = append(validBlock, h)
			}
		}
		result[Nakamoto] = p.getRangesForBlocks(validBlock, txn)

		result[FirstTransaction] = FirstTransactionRanges
		if endHeight >= 1000 {
			result[Vintage] = p.getRangeToBlock(1000, txn)
		}
		return nil
	})

	return result
}

func initEpochSat(ldb common.KVDB, height int) {

	ldb.View(func(txn common.ReadBatch) error {
		currentEpoch := height / HalvingInterval
		underpays := int64(0)

		for epoch := (height / HalvingInterval); epoch > 0; epoch-- {

			value := &common.BlockValueInDB{}
			key := db.GetBlockDBKey(210000 * epoch)
			err := db.GetValueFromTxn(key, value, txn)
			if err != nil {
				common.Log.Panicf("GetValueFromDB %s failed. %v", key, err)
			}

			if epoch == currentEpoch {
				underpays = int64(Epoch(int64(epoch)).GetStartingSat()) - value.Ordinals.Start
			}
			SetEpochStartingSat(int64(epoch), value.Ordinals.Start)
		}

		for epoch := currentEpoch + 1; epoch < MAX_EPOCH; epoch++ {
			SetEpochStartingSat(int64(epoch), int64(Epoch(int64(epoch)).GetStartingSat())-underpays)
		}
		return nil
	})

}

// 跟base数据库同步
func (p *ExoticIndexer) UpdateDB() {
	//common.Log.Infof("NftIndexer->UpdateDB start...")
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

	common.Log.Infof("ExoticIndexer->UpdateDB takes %v", time.Since(startTime))
}

func (p *ExoticIndexer) CheckSelf() bool {
	return true
}
