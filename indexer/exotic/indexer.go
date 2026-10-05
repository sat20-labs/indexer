package exotic

import (
	"sort"
	"sync"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/base"
	"github.com/sat20-labs/indexer/indexer/db"
)

type TickInfo struct {
	Name           string
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

func (p *HolderInfo) Clone() *HolderInfo {
	newTickerInfo := make(map[string]*common.AssetAbbrInfo)
	for k, assets := range p.Tickers {
		newTickerInfo[k] = assets.Clone()
	}
	return &HolderInfo{
		AddressId: p.AddressId,
		Tickers:   newTickerInfo}
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
	tickerMap  map[string]*TickInfo   // 没几个，全部加载
	holderInfo map[uint64]*HolderInfo // utxoId -> holder 用于动态更新ticker的holder数据，需要备份到数据库

	// Unflushed state only. Durable balances are authoritative in Badger.
	utxoMap              map[string]map[uint64]int64 // pending ticker/UTXO values
	utxoDeleted          map[string]map[uint64]bool  // pending ticker/UTXO deletes
	holderBalanceTouched map[string]int64            // encoded ticker/address key -> aggregate amount

	holderActionList []*HolderAction
	tickerAdded      map[string]*common.Ticker // key: ticker
}

func newExoticTickerInfo(name string) *TickInfo {
	return &TickInfo{
		Name:           name,
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
		p.holderInfo = make(map[uint64]*HolderInfo)

		p.utxoMap = make(map[string]map[uint64]int64)
		p.utxoDeleted = make(map[string]map[uint64]bool)
		p.holderBalanceTouched = make(map[string]int64)

		p.holderActionList = make([]*HolderAction, 0)
		p.tickerAdded = make(map[string]*common.Ticker, 0)

		p.mutex.Unlock()
	}
}

// 只保存UpdateDB需要用的数据
func (p *ExoticIndexer) Clone(baseIndexer *base.BaseIndexer) *ExoticIndexer {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	newInst := NewExoticIndexer(p.db)
	newInst.status = p.status.Clone()
	newInst.baseIndexer = baseIndexer

	newInst.holderActionList = make([]*HolderAction, len(p.holderActionList))
	copy(newInst.holderActionList, p.holderActionList)

	newInst.tickerAdded = make(map[string]*common.Ticker, 0)
	for key, value := range p.tickerAdded {
		newInst.tickerAdded[key] = value.Clone()
	}

	newInst.tickerMap = make(map[string]*TickInfo, 0)
	for key, value := range p.tickerMap {
		//if len(value.MintAdded) > 0 {
		tick := TickInfo{}
		tick.Name = value.Name
		tick.MintAdded = make([]*common.Mint, len(value.MintAdded))
		copy(tick.MintAdded, value.MintAdded)
		tick.Ticker = value.Ticker.Clone()
		newInst.tickerMap[key] = &tick
		//}
	}

	// 保存holderActionList对应的数据
	newInst.holderInfo = make(map[uint64]*HolderInfo, 0)
	newInst.utxoMap = make(map[string]map[uint64]int64, 0)
	newInst.utxoDeleted = make(map[string]map[uint64]bool, len(p.utxoDeleted))
	for ticker, deleted := range p.utxoDeleted {
		copyDeleted := make(map[uint64]bool, len(deleted))
		for utxoID := range deleted {
			copyDeleted[utxoID] = true
		}
		newInst.utxoDeleted[ticker] = copyDeleted
	}
	newInst.holderBalanceTouched = make(map[string]int64, len(p.holderBalanceTouched))
	for key, amount := range p.holderBalanceTouched {
		newInst.holderBalanceTouched[key] = amount
	}

	for _, action := range p.holderActionList {
		if action.Action > 0 {
			value, ok := p.holderInfo[action.UtxoId]
			if ok {
				newInst.holderInfo[action.UtxoId] = value.Clone()
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
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for name, ticker := range another.tickerAdded {
		n, ok := p.tickerAdded[name]
		if ok && n.TotalMinted == ticker.TotalMinted {
			delete(p.tickerAdded, name)
		}
	}

	p.holderActionList = append([]*HolderAction(nil), p.holderActionList[len(another.holderActionList):]...)

	for key, value := range another.tickerMap {
		ticker, ok := p.tickerMap[key]
		if ok {
			ticker.MintAdded = append([]*common.Mint(nil), ticker.MintAdded[len(value.MintAdded):]...)
		}
	}

	for ticker, flushed := range another.utxoMap {
		current := p.utxoMap[ticker]
		for utxoID, amount := range flushed {
			if currentAmount, ok := current[utxoID]; ok && currentAmount == amount {
				delete(current, utxoID)
			}
		}
		if len(current) == 0 {
			delete(p.utxoMap, ticker)
		}
	}
	for ticker, flushed := range another.utxoDeleted {
		current := p.utxoDeleted[ticker]
		for utxoID := range flushed {
			delete(current, utxoID)
		}
		if len(current) == 0 {
			delete(p.utxoDeleted, ticker)
		}
	}
	for key, amount := range another.holderBalanceTouched {
		if current, ok := p.holderBalanceTouched[key]; ok && current == amount {
			delete(p.holderBalanceTouched, key)
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
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	startTime := time.Now()
	for name, tickInfo := range p.tickerMap {
		if tickInfo == nil || tickInfo.Ticker == nil {
			common.Log.Errorf("ExoticIndexer ticker %s metadata missing", name)
			return false
		}

		holders := p.getTickerHolderAmounts(name)
		var holderAmount int64
		for _, amount := range holders {
			holderAmount += amount
		}
		if holderAmount != tickInfo.Ticker.TotalMinted {
			common.Log.Errorf(
				"ExoticIndexer ticker %s holder amount incorrect: minted=%d holders=%d",
				name, tickInfo.Ticker.TotalMinted, holderAmount,
			)
			return false
		}

		utxos := p.getTickerUtxos(name)
		var utxoAmount int64
		for utxoID, amount := range utxos {
			utxoAmount += amount
			holder := p.holderInfoForRead(utxoID)
			if holder == nil {
				common.Log.Errorf("ExoticIndexer ticker %s UTXO %d has no holder record", name, utxoID)
				return false
			}
			asset := holder.Tickers[name]
			if asset == nil {
				common.Log.Errorf("ExoticIndexer ticker %s UTXO %d missing asset detail", name, utxoID)
				return false
			}
			if asset.AssetAmt() != amount {
				common.Log.Errorf(
					"ExoticIndexer ticker %s UTXO %d amount differs: index=%d holder=%d",
					name, utxoID, amount, asset.AssetAmt(),
				)
				return false
			}
		}
		if utxoAmount != holderAmount {
			common.Log.Errorf(
				"ExoticIndexer ticker %s UTXO total %d != holder total %d",
				name, utxoAmount, holderAmount,
			)
			return false
		}
		common.Log.Infof("ExoticIndexer %s amount: %d, holders: %d, utxos: %d", name, holderAmount, len(holders), len(utxos))
	}

	p.setDBVersion()
	common.Log.Infof("ExoticIndexer CheckSelf took %v.", time.Since(startTime))
	return true
}

func PrintHoldersWithMap(holders map[uint64]int64, baseIndexer *base.BaseIndexer) {
	var total int64
	type pair struct {
		addressId uint64
		amt       int64
	}
	mid := make([]*pair, 0)
	for addressId, amt := range holders {
		//common.Log.Infof("%x: %s", addressId, amt.String())
		total += amt
		mid = append(mid, &pair{
			addressId: addressId,
			amt:       amt,
		})
	}
	sort.Slice(mid, func(i, j int) bool {
		return mid[i].amt > mid[j].amt
	})
	limit := 10 //len(mid) // 40
	for i, item := range mid {
		if i > limit {
			break
		}
		if item.amt == 0 {
			continue
		}
		address, err := baseIndexer.GetAddressByID(item.addressId)
		if err != nil {
			common.Log.Panicf("printHoldersWithMap GetAddressByID %x failed, %v", item.addressId, err)
			address = "-\t"
		}
		common.Log.Printf("%d: %x %s: %d", i, item.addressId, address, item.amt)
	}
	common.Log.Infof("total in holders: %d", total)
}

func (p *ExoticIndexer) printHolders(name string) {
	holdermap := p.GetHolderAndAmountWithTick(name)
	common.Log.Infof("holders from holder DB")
	PrintHoldersWithMap(holdermap, p.baseIndexer)
}
