package ft

import (
	"sync"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
	"github.com/sat20-labs/indexer/indexer/exotic"
	"github.com/sat20-labs/indexer/indexer/nft"
)

type TickInfo = exotic.TickInfo
type HolderInfo = exotic.HolderInfo

type HolderAction = exotic.HolderAction

type ActionInfo struct {
	Action int
	Input  *common.TxInput
	Info   any
}

// TODO 加载所有数据，太耗时间和内存，需要优化，参考nft和brc20模块 (目前数据量少，问题不大)
type FTIndexer struct {
	db           common.KVDB
	nftIndexer   *nft.NftIndexer
	enableHeight int

	// 所有必要数据都保存在这几个数据结构中，任何查找数据的行为，必须先通过这几个数据结构查找，再去数据库中读其他数据
	// 禁止直接对外暴露这几个结构的数据，防止被不小心修改
	// 禁止直接遍历holderInfo和utxoMap，因为数据量太大（ord有亿级数据）
	mutex      sync.RWMutex                // 只保护这几个结构
	tickerMap  map[string]*TickInfo        // ticker -> TickerInfo.  name 小写。
	holderInfo map[uint64]*HolderInfo      // utxoId -> holder 用于动态更新ticker的holder数据，需要备份到数据库
	utxoMap    map[string]map[uint64]int64 // ticker -> utxoId -> 资产数量. 动态数据，跟随Holder变更，需要保存在数据库中。

	// 其他辅助信息
	holderActionList []*HolderAction           // 在同一个block中，状态变迁需要按顺序执行，因为一个utxo会很快被消费掉，变成新的utxo
	tickerAdded      map[string]*common.Ticker // key: ticker

	// 一个区块的缓存数据，不需要备份
	actionBufferMap map[uint64][]*ActionInfo                  // 当前块内待并入输入UTXO的 deploy/mint 增量
	unbindHistory   []*common.UnbindHistory                   // 当前块内新增的 unbind 历史，供 UpdateDB 增量落库
	freezeHistory   []*common.FreezeHistory                   // 当前块内新增的 freeze/unfreeze 历史，供 UpdateDB 增量落库
	freezeStates    map[string]map[uint64]*common.FreezeState // 当前生效的冻结状态: ticker -> addressId -> state
	freezeTouched   map[string]*common.FreezeState            // 当前块内新增/更新的冻结状态，供 UpdateDB 增量落库
	freezeDeleted   map[string]*common.FreezeState            // 当前块内删除的冻结状态，供 UpdateDB 增量删库

	// 校验数据，不需要保存
	holderMapInPrevBlock map[uint64]int64

	// 冻结相关的临时控制状态：
	// 1. freezeAuthoritySnapshot 保存“当前正在处理的 freezeHeight-1”时刻谁有权限发起 freeze/unfreeze
	// 2. pendingHistoricalFreezes/pendingHistoricalKeys 用于历史编译模式下的 2-block lookahead 注入
	// 3. reloadFreezeDirectives/reloadRequestHeight 仅用于链头实时模式下 lookahead 不足时的非 reorg reload 兜底
	freezeAuthoritySnapshot  map[string]uint64
	pendingHistoricalFreezes map[int][]*common.FreezeDirective
	pendingHistoricalKeys    map[string]bool
	reloadFreezeDirectives   map[string]*common.FreezeDirective
	reloadRequestHeight      int
}

func NewOrdxIndexer(db common.KVDB) *FTIndexer {
	enableHeight := 827307
	if !common.IsMainnet() {
		enableHeight = 28883
	}
	return &FTIndexer{
		db:           db,
		enableHeight: enableHeight,
	}
}

func (s *FTIndexer) setDBVersion() {
	err := db.SetRawValueToDB([]byte(ORDX_DB_VER_KEY), []byte(ORDX_DB_VERSION), s.db)
	if err != nil {
		common.Log.Panicf("SetRawValueToDB failed %v", err)
	}
}

func (s *FTIndexer) GetDBVersion() string {
	value, err := db.GetRawValueFromDB([]byte(ORDX_DB_VER_KEY), s.db)
	if err != nil {
		common.Log.Errorf("GetRawValueFromDB failed %v", err)
		return ""
	}

	return string(value)
}

// 只保存UpdateDB需要用的数据
func (s *FTIndexer) Clone(nftIndexer *nft.NftIndexer) *FTIndexer {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	newInst := NewOrdxIndexer(s.db)
	newInst.nftIndexer = nftIndexer

	newInst.holderActionList = make([]*HolderAction, len(s.holderActionList))
	copy(newInst.holderActionList, s.holderActionList)

	newInst.unbindHistory = make([]*common.UnbindHistory, len(s.unbindHistory))
	for i, item := range s.unbindHistory {
		newInst.unbindHistory[i] = item.Clone()
	}
	newInst.freezeHistory = make([]*common.FreezeHistory, len(s.freezeHistory))
	for i, item := range s.freezeHistory {
		newInst.freezeHistory[i] = item.Clone()
	}
	newInst.freezeTouched = make(map[string]*common.FreezeState, len(s.freezeTouched))
	for key, value := range s.freezeTouched {
		state := *value
		newInst.freezeTouched[key] = &state
	}
	newInst.freezeDeleted = make(map[string]*common.FreezeState, len(s.freezeDeleted))
	for key, value := range s.freezeDeleted {
		state := *value
		newInst.freezeDeleted[key] = &state
	}
	newInst.freezeStates = make(map[string]map[uint64]*common.FreezeState, len(s.freezeStates))
	for ticker, stateMap := range s.freezeStates {
		cloned := make(map[uint64]*common.FreezeState, len(stateMap))
		for addressId, value := range stateMap {
			state := *value
			cloned[addressId] = &state
		}
		newInst.freezeStates[ticker] = cloned
	}
	newInst.pendingHistoricalFreezes = make(map[int][]*common.FreezeDirective, len(s.pendingHistoricalFreezes))
	for height, directives := range s.pendingHistoricalFreezes {
		cloned := make([]*common.FreezeDirective, len(directives))
		for i, item := range directives {
			d := *item
			cloned[i] = &d
		}
		newInst.pendingHistoricalFreezes[height] = cloned
	}
	newInst.pendingHistoricalKeys = make(map[string]bool, len(s.pendingHistoricalKeys))
	for key, value := range s.pendingHistoricalKeys {
		newInst.pendingHistoricalKeys[key] = value
	}
	newInst.reloadFreezeDirectives = make(map[string]*common.FreezeDirective, len(s.reloadFreezeDirectives))
	for key, value := range s.reloadFreezeDirectives {
		d := *value
		newInst.reloadFreezeDirectives[key] = &d
	}
	newInst.freezeAuthoritySnapshot = make(map[string]uint64, len(s.freezeAuthoritySnapshot))
	for key, value := range s.freezeAuthoritySnapshot {
		newInst.freezeAuthoritySnapshot[key] = value
	}
	newInst.reloadRequestHeight = s.reloadRequestHeight

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
func (s *FTIndexer) Subtract(another *FTIndexer) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	//s.holderActionList = s.holderActionList[len(another.holderActionList):]
	s.holderActionList = append([]*HolderAction(nil), s.holderActionList[len(another.holderActionList):]...)

	for key := range another.tickerAdded {
		delete(s.tickerAdded, key)
	}

	for key, value := range another.tickerMap {
		ticker, ok := s.tickerMap[key]
		if ok {
			//ticker.MintAdded = ticker.MintAdded[len(value.MintAdded):]
			ticker.MintAdded = append([]*common.Mint(nil), ticker.MintAdded[len(value.MintAdded):]...)
		}
	}

	s.unbindHistory = append([]*common.UnbindHistory(nil), s.unbindHistory[len(another.unbindHistory):]...)
	s.freezeHistory = append([]*common.FreezeHistory(nil), s.freezeHistory[len(another.freezeHistory):]...)
	s.freezeTouched = make(map[string]*common.FreezeState)
	s.freezeDeleted = make(map[string]*common.FreezeState)
	s.reloadFreezeDirectives = make(map[string]*common.FreezeDirective)
	s.freezeAuthoritySnapshot = make(map[string]uint64)
	s.reloadRequestHeight = 0

	// 不需要更新 holderInfo 和 utxoMap
}

// 在系统初始化时调用一次，如果有历史数据的话。一般在NewSatIndex之后调用。
func (s *FTIndexer) Init(nftIndexer *nft.NftIndexer) {

	s.nftIndexer = nftIndexer

	startTime := time.Now()
	common.Log.Infof("ordx db version: %s", s.GetDBVersion())
	common.Log.Info("InitOrdxIndexerFromDB ...")

	ticks := s.getTickListFromDB()
	if true {
		s.mutex.Lock()

		s.tickerMap = make(map[string]*TickInfo, 0)
		for _, ticker := range ticks {
			s.tickerMap[ticker] = s.initTickInfoFromDB(ticker)
		}

		s.holderInfo = s.loadHolderInfoFromDB()
		s.utxoMap = s.loadUtxoMapFromDB()
		s.freezeStates = s.loadFreezeStateFromDB()
		// TODO utxoMap 可以从 holderInfo 中构建出来

		s.holderActionList = make([]*HolderAction, 0)
		s.tickerAdded = make(map[string]*common.Ticker, 0)
		s.unbindHistory = make([]*common.UnbindHistory, 0)
		s.freezeHistory = make([]*common.FreezeHistory, 0)
		s.freezeTouched = make(map[string]*common.FreezeState)
		s.freezeDeleted = make(map[string]*common.FreezeState)
		s.freezeAuthoritySnapshot = make(map[string]uint64)
		s.pendingHistoricalFreezes = make(map[int][]*common.FreezeDirective)
		s.pendingHistoricalKeys = make(map[string]bool)
		s.reloadFreezeDirectives = make(map[string]*common.FreezeDirective)
		s.reloadRequestHeight = 0

		s.mutex.Unlock()
	}

	s.CheckSelf()

	elapsed := time.Since(startTime).Milliseconds()
	common.Log.Infof("InitSatIndexFromDB %d ms\n", elapsed)
}

// 自检。如果错误，将停机
func (s *FTIndexer) CheckSelf() bool {

	height := s.nftIndexer.GetBaseIndexer().GetHeight()

	//common.Log.Infof("OrdxIndexer->CheckSelf ...")
	startTime := time.Now()
	allHolders := make(map[uint64]bool)
	allTickers := s.GetAllTickers()
	for _, name := range allTickers {
		//common.Log.Infof("checking ticker %s", name)
		holdermap := s.GetHolderAndAmountWithTick(name)
		holderAmount := int64(0)
		for u, amt := range holdermap {
			holderAmount += amt
			allHolders[u] = true
		}

		mintAmount, _ := s.GetMintAmount(name)
		if holderAmount != mintAmount {
			common.Log.Errorf("FTIndexer ticker %s amount incorrect. %d %d", name, mintAmount, holderAmount)
			return false
		}

		common.Log.Infof("FTIndexer %s amount: %d, holders: %d", name, mintAmount, len(holdermap))

		utxos, ok := s.utxoMap[name]
		if !ok {
			if holderAmount != 0 {
				common.Log.Errorf("FTIndexer ticker %s has no asset utxos", name)
				return false
			}
		} else {
			amontInUtxos := int64(0)
			for utxo, amoutInUtxo := range utxos {
				amontInUtxos += amoutInUtxo

				holderInfo, ok := s.holderInfo[utxo]
				if !ok {
					common.Log.Errorf("FTIndexer ticker %s's utxo %d not in holdermap", name, utxo)
					return false
				}
				tickAssetInfo, ok := holderInfo.Tickers[name]
				if !ok {
					common.Log.Errorf("FTIndexer ticker %s's utxo %d not in holders", name, utxo)
					return false
				}

				amountInHolder := tickAssetInfo.AssetAmt()
				if amountInHolder != amoutInUtxo {
					common.Log.Errorf("FTIndexer ticker %s's utxo %d assets %d and %d different", name, utxo, amoutInUtxo, amountInHolder)
					return false
				}
			}
		}
	}
	common.Log.Infof("ordx has %d holders", len(allHolders))
	allHolders = nil

	// 需要高度到达一定高度才需要检查
	if s.nftIndexer.GetBaseIndexer().IsMainnet() && height > 828800 {
		// 需要区分主网和测试网
		name := "pearl"
		ticker := s.GetTicker(name)
		if ticker == nil {
			common.Log.Errorf("FTIndexer can't find %s in db", name)
			return false
		}

		holdermap := s.GetHolderAndAmountWithTick(name)
		holderAmount := int64(0)
		for _, amt := range holdermap {
			holderAmount += amt
		}

		mintAmount, _ := s.GetMintAmount(name)
		if holderAmount != mintAmount {
			common.Log.Errorf("FTIndexer ticker amount incorrect. %d %d", mintAmount, holderAmount)
			return false
		}

		// 1.2.0 版本升级后，pearl的数量增加了105张。原因是之前铸造时，部分输出少于amt的铸造，被错误的识别为无效的铸造。
		// 但实际上，这些铸造是有效的，铸造时已经提供了大于10000的聪，只是大部分铸造出来的pearl，都给了矿工，只有546或者330留在铸造者手里
		// 比如： 5647d570edcbe45d4953915f7b9063e9b39b83432ae2ae13fdbd5283abb83367i0 等
		if ticker.BlockStart == 828200 {
			if holderAmount != 156271012 {
				common.Log.Errorf("FTIndexer %s amount incorrect. %d", name, holderAmount)
				return false
			}
		} else {
			common.Log.Errorf("FTIndexer Incorrect %s", name)
			return false
		}
	}

	// 检查holderinfo？
	// for utxo, holderInfo := range s.holderInfo {

	// }

	// 最后才设置dbver
	s.setDBVersion()
	common.Log.Infof("FTIndexer CheckSelf took %v.", time.Since(startTime))

	return true
}
