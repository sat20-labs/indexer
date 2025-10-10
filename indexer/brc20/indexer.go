package brc20

import (
	"strings"
	"sync"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
	"github.com/sat20-labs/indexer/indexer/nft"
)

type BRC20TickInfo struct {
	Id             uint64
	Name           string
	InscriptionMap map[string]*common.BRC20MintAbbrInfo // key: inscriptionId
	MintAdded      []*common.BRC20Mint
	Ticker         *common.BRC20Ticker
}

type HolderAction struct {
	Height   int
	Utxo     string
	NftId    int64
	FromAddr uint64
	ToAddr   uint64

	Ticker string
	Amount common.Decimal

	Action int // 0: inscribe-mint  1: inscribe-transfer  2: transfer
}

const (
	// 0: inscribe-mint  1: inscribe-transfer  2: transfer
	Action_InScribe_Mint int = iota
	Action_InScribe_Transfer
	Action_Transfer
	// Action_Transfer_Send
	// Action_TRansfer_Receive
)

type HolderInfo struct {
	AddressId uint64
	Tickers   map[string]*common.BRC20TickAbbrInfo // key: ticker, 小写
}

type TransferNftInfo struct {
	AddressId   uint64
	Index       int
	Ticker      string
	TransferNft *common.TransferNFT
}

type BRC20Indexer struct {
	db         common.KVDB
	nftIndexer *nft.NftIndexer

	// 所有必要数据都保存在这几个数据结构中，任何查找数据的行为，必须先通过这几个数据结构查找，再去数据库中读其他数据
	// 禁止直接对外暴露这几个结构的数据，防止被不小心修改
	// 禁止直接遍历holderInfo和utxoMap，因为数据量太大（ord有亿级数据）
	mutex             sync.RWMutex                // 只保护这几个结构
	tickerMap         map[string]*BRC20TickInfo   // ticker -> TickerInfo.  name 小写。 数据由mint数据构造
	holderMap         map[uint64]*HolderInfo      // addrId -> holder 用于动态更新ticker的holder数据，需要备份到数据库
	tickerToHolderMap map[string]map[uint64]bool  // ticker -> addrId. 动态数据，跟随Holder变更，内存数据。
	transferNftMap    map[uint64]*TransferNftInfo // utxoId -> HolderInfo中的TransferableData的Nft

	// 其他辅助信息
	holderActionList []*HolderAction                // 在同一个block中，状态变迁需要按顺序执行
	tickerAdded      map[string]*common.BRC20Ticker // key: ticker
	tickerUpdated    map[string]*common.BRC20Ticker // key: ticker
}

func NewIndexer(db common.KVDB) *BRC20Indexer {
	return &BRC20Indexer{
		db: db,
	}
}

func (s *BRC20Indexer) setDBVersion() {
	err := db.SetRawValueToDB([]byte(BRC20_DB_VER_KEY), []byte(BRC20_DB_VERSION), s.db)
	if err != nil {
		common.Log.Panicf("SetRawValueToDB failed %v", err)
	}
}

func (s *BRC20Indexer) GetDBVersion() string {
	value, err := db.GetRawValueFromDB([]byte(BRC20_DB_VER_KEY), s.db)
	if err != nil {
		common.Log.Errorf("GetRawValueFromDB failed %v", err)
		return ""
	}

	return string(value)
}

// 只保存UpdateDB需要用的数据
func (s *BRC20Indexer) Clone() *BRC20Indexer {
	newInst := NewIndexer(s.db)

	newInst.holderActionList = make([]*HolderAction, len(s.holderActionList))
	copy(newInst.holderActionList, s.holderActionList)

	newInst.tickerAdded = make(map[string]*common.BRC20Ticker, 0)
	for key, value := range s.tickerAdded {
		newInst.tickerAdded[key] = value
	}

	newInst.tickerUpdated = make(map[string]*common.BRC20Ticker, 0)
	for key, value := range s.tickerUpdated {
		newInst.tickerUpdated[key] = value
	}

	newInst.tickerMap = make(map[string]*BRC20TickInfo, 0)
	for key, value := range s.tickerMap {
		tick := BRC20TickInfo{}
		tick.Id = value.Id
		tick.Name = value.Name
		tick.Ticker = value.Ticker
		tick.MintAdded = make([]*common.BRC20Mint, len(value.MintAdded))
		copy(tick.MintAdded, value.MintAdded)

		tick.InscriptionMap = make(map[string]*common.BRC20MintAbbrInfo, 0)
		for inscriptionId, mintAbbrInfo := range value.InscriptionMap {
			tick.InscriptionMap[inscriptionId] = mintAbbrInfo
		}
		newInst.tickerMap[key] = &tick
	}

	// 保存holderActionList对应的数据，更新数据库需要
	newInst.holderMap = make(map[uint64]*HolderInfo, 0)
	newInst.tickerToHolderMap = make(map[string]map[uint64]bool, 0)
	for _, action := range s.holderActionList {

		value, ok := s.holderMap[action.FromAddr]
		if ok {
			info := HolderInfo{AddressId: value.AddressId, Tickers: value.Tickers}
			newInst.holderMap[action.FromAddr] = &info
		}

		value, ok = s.holderMap[action.ToAddr]
		if ok {
			info := HolderInfo{AddressId: value.AddressId, Tickers: value.Tickers}
			newInst.holderMap[action.ToAddr] = &info
		}

		holders, ok := s.tickerToHolderMap[action.Ticker]
		if ok {
			newInst.tickerToHolderMap[action.Ticker] = holders
		}
	}

	for key, value := range s.transferNftMap {
		newInst.transferNftMap[key] = value
	}
	return newInst
}

// update之后，删除原来instance中的数据
func (s *BRC20Indexer) Subtract(another *BRC20Indexer) {

	//s.holderActionList = s.holderActionList[len(another.holderActionList):]
	s.holderActionList = append([]*HolderAction(nil), s.holderActionList[len(another.holderActionList):]...)

	for key := range another.tickerAdded {
		delete(s.tickerAdded, key)
	}

	for key := range another.tickerUpdated {
		delete(s.tickerUpdated, key)
	}

	for key, value := range another.tickerMap {
		ticker, ok := s.tickerMap[key]
		if ok {
			//ticker.MintAdded = ticker.MintAdded[len(value.MintAdded):]
			ticker.MintAdded = append([]*common.BRC20Mint(nil), ticker.MintAdded[len(value.MintAdded):]...)
		}
	}

	// 不需要更新 holderInfo 和 utxoMap
}

// 在系统初始化时调用一次，如果有历史数据的话。一般在NewSatIndex之后调用。
func (s *BRC20Indexer) InitIndexer(nftIndexer *nft.NftIndexer) {

	s.nftIndexer = nftIndexer

	startTime := time.Now()
	common.Log.Infof("brc20 db version: %s", s.GetDBVersion())
	common.Log.Info("InitIndexer ...")

	ticks := s.getTickListFromDB()
	if true {
		s.mutex.Lock()

		s.tickerMap = make(map[string]*BRC20TickInfo, 0)
		for _, ticker := range ticks {
			s.tickerMap[strings.ToLower(ticker)] = s.initTickInfoFromDB(ticker)
		}

		s.loadHolderInfoFromDB()

		s.holderActionList = make([]*HolderAction, 0)
		s.tickerAdded = make(map[string]*common.BRC20Ticker, 0)
		s.tickerUpdated = make(map[string]*common.BRC20Ticker, 0)

		s.mutex.Unlock()
	}

	//height := nftIndexer.GetBaseIndexer().GetSyncHeight()
	//s.CheckSelf(height)

	elapsed := time.Since(startTime).Milliseconds()
	common.Log.Infof("InitIndexer %d ms\n", elapsed)
}

// 自检。如果错误，将停机
func (s *BRC20Indexer) CheckSelf(height int) bool {

	common.Log.Infof("BRC20Indexer->CheckSelf ...")
	startTime := time.Now()
	for name := range s.tickerMap {
		common.Log.Infof("checking ticker %s", name)
		holdermap := s.GetHoldersWithTick(name)
		var holderAmount *common.Decimal
		for _, amt := range holdermap {
			holderAmount = holderAmount.Add(amt)
		}
		mintAmount, _ := s.GetMintAmount(name)
		common.Log.Infof("ticker %s, minted %s", name, mintAmount.String())
		if holderAmount.Cmp(mintAmount) != 0 {
			common.Log.Errorf("ticker %s amount incorrect. %d %d", name, mintAmount, holderAmount)
			return false
		}
	}
	common.Log.Infof("total tickers %d", len(s.tickerMap))

	// 需要高度到达一定高度才需要检查
	if (s.nftIndexer.GetBaseIndexer().IsMainnet() && height == 828800) || 
	(!s.nftIndexer.GetBaseIndexer().IsMainnet() && height == 28865) {
		// 需要区分主网和测试网
		name := "ordi"
		ticker := s.GetTicker(name)
		if ticker == nil {
			common.Log.Errorf("can't find %s in db", name)
			return false
		}

		holdermap := s.GetHoldersWithTick(name)
		var holderAmount common.Decimal
		for _, amt := range holdermap {
			holderAmount = *holderAmount.Add(amt)
		}

		mintAmount, _ := s.GetMintAmount(name)
		if holderAmount.Cmp(mintAmount) != 0 {
			common.Log.Errorf("ticker amount incorrect. %d %d", mintAmount, holderAmount)
			return false
		}

		// if holderAmount != 156271012 {
		// 	common.Log.Panicf("%s amount incorrect. %d", name, holderAmount)
		// }

	}

	// 检查holderinfo？
	// for utxo, holderInfo := range s.holderInfo {

	// }

	// 最后才设置dbver
	s.setDBVersion()
	common.Log.Infof("BRC20Indexer->CheckSelf took %v.", time.Since(startTime))

	return true
}
