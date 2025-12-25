package exotic

import (
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
	Tickers   map[string]*common.AssetAbbrInfo // key: ticker
}

func (p *HolderInfo) AddTickerAsset(name string, assetInfo *common.AssetAbbrInfo) int64 {
	tickerAsset, ok := p.Tickers[name]
	if !ok {
		tickerAsset = assetInfo.Clone()
		p.Tickers[name] = tickerAsset
	} else {
		tickerAsset.Offsets.Merge(assetInfo.Offsets)
	}

	return tickerAsset.AssetAmt()
}

func (p *HolderInfo) RemoveTickerAsset(name string, assetInfo *common.AssetAbbrInfo) {
	tickerAsset, ok := p.Tickers[name]
	if !ok {
		return
	}
	
	tickerAsset.Offsets.Remove(assetInfo.Offsets)
	if tickerAsset.AssetAmt() == 0 {
		delete(p.Tickers, name)
	}
}

type ExoticIndexer struct {
	db          common.KVDB
	status      *Status
	baseIndexer *base.BaseIndexer

	mutex sync.RWMutex // 只保护这几个结构

	// 只加载必要的数据
	tickerMap  map[string]*TickInfo        // 用于检索稀有聪. key 稀有聪种类
	holderInfo map[uint64]*HolderInfo      // utxoId -> holder 用于动态更新ticker的holder数据，需要备份到数据库
	utxoMap    map[string]map[uint64]int64 // ticker -> utxoId -> 资产数量. 动态数据，跟随Holder变更，需要保存在数据库中。

	holderActionList []*HolderAction
	tickerAdded      map[string]*common.Ticker // key: ticker
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

	return &ExoticIndexer{
		db: db,
	}
}

func (p *ExoticIndexer) Init(baseIndexer *base.BaseIndexer) {
	p.baseIndexer = baseIndexer
	p.status = initStatusFromDB(p.db)

	ticks := p.loadTickListFromDB()
	if true {
		p.mutex.Lock()

		p.tickerMap = make(map[string]*TickInfo, 0)
		for _, ticker := range ticks {
			p.tickerMap[ticker] = p.initTickInfoFromDB(ticker)
		}

		// 延迟加载
		// p.holderInfo = p.loadHolderInfoFromDB()
		// // 更新ticker数据的utxo数据
		// for utxoId, holder := range p.holderInfo {
		// 	for name, assetInfoMap := range holder.Tickers {
		// 		ticker := p.tickerMap[name]
		// 		ticker.UtxoMap[utxoId] = assetInfoMap.Offsets.Clone()
		// 	}
		// }
		// p.utxoMap = p.loadUtxoMapFromDB()
		p.holderInfo = make(map[uint64]*HolderInfo)
		p.utxoMap = make(map[string]map[uint64]int64)

		p.holderActionList = make([]*HolderAction, 0)
		p.tickerAdded = make(map[string]*common.Ticker, 0)

		p.mutex.Unlock()
	}
}

// 只保存UpdateDB需要用的数据
func (p *ExoticIndexer) Clone() *ExoticIndexer {
	newInst := NewExoticIndexer(p.db)
	newInst.status = p.status.Clone()

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
				newTickerInfo := make(map[string]*common.AssetAbbrInfo)
				for k, assets := range value.Tickers {
					newTickerInfo[k] = assets.Clone()
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

	for name, ticker := range another.tickerAdded {
		n, ok := p.tickerAdded[name]
		if ok && n.TotalMinted == ticker.TotalMinted {
			delete(p.tickerAdded, name)
		}
	}

	p.holderActionList = append([]*HolderAction(nil), p.holderActionList[len(another.holderActionList):]...)

	for key := range another.tickerAdded {
		delete(p.tickerAdded, key)
	}

	for key, value := range another.tickerMap {
		ticker, ok := p.tickerMap[key]
		if ok {
			if ticker.Ticker.TotalMinted == value.Ticker.TotalMinted {
				delete(p.tickerMap, key)
			} else {
				ticker.MintAdded = append([]*common.Mint(nil), ticker.MintAdded[len(value.MintAdded):]...)
			}
		}		
	}

	for name, value := range another.utxoMap {
		n, ok := p.utxoMap[name]
		if ok {
			for utxoId := range value {
				delete(n, utxoId)
				delete(p.holderInfo, utxoId)
			}
		}
	}
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
				tickAssetInfo, ok := holderInfo.Tickers[name]
				if !ok {
					common.Log.Errorf("ExoticIndexer ticker %s's utxo %d not in holders", name, utxo)
					return false
				}
				amountInHolder := tickAssetInfo.AssetAmt()
				if amountInHolder != amoutInUtxo {
					common.Log.Errorf("ExoticIndexer ticker %s's utxo %d assets %d and %d different", name, utxo, amoutInUtxo, amountInHolder)
					return false
				}
			}
		}
	}

	// 最后才设置dbver
	p.setDBVersion()
	common.Log.Infof("ExoticIndexer CheckSelf took %v.", time.Since(startTime))

	return true
}
