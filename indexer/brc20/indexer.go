package brc20

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/base"
	"github.com/sat20-labs/indexer/indexer/brc20/validate"
	"github.com/sat20-labs/indexer/indexer/db"
	"github.com/sat20-labs/indexer/indexer/nft"
	"github.com/sat20-labs/indexer/share/base_indexer"
)

type BRC20TickInfo struct {
	Name string
	//InscriptionMap map[string]*common.BRC20MintAbbrInfo // key: inscriptionId
	MintAdded []*common.BRC20Mint
	Ticker    *common.BRC20Ticker
}

type ActionInfo struct {
	Action int
	Info   any
}

type HolderAction = common.BRC20ActionHistory

type HolderInfo struct {
	// AddressId uint64
	FreshTime int64
	Tickers map[string]*common.BRC20TickAbbrInfo // key: ticker, 小写
}

func NewHolderInfo() *HolderInfo {
	return &HolderInfo{
		FreshTime: 0,
		Tickers: make(map[string]*common.BRC20TickAbbrInfo),
	}
}

func (p *HolderInfo) Clone() *HolderInfo {
	n := &HolderInfo{
		FreshTime: p.FreshTime,
		Tickers: make(map[string]*common.BRC20TickAbbrInfo),
	}
	for name, info := range p.Tickers {
		newInfo := &common.BRC20TickAbbrInfo{
			AvailableBalance:    info.AvailableBalance.Clone(),
			TransferableBalance: info.TransferableBalance.Clone(),
			TransferableData:    make(map[uint64]*common.TransferNFT),
		}
		for k, v := range info.TransferableData {
			newInfo.TransferableData[k] = v.Clone()
		}
		n.Tickers[name] = newInfo
	}
	return n
}

func (p *HolderInfo) Updated() {
	p.FreshTime++
}

type TransferNftInfo struct {
	TxInIndex   int    // 在当前交易中，作为第几个输入进入TX
	AddressId   uint64 // 当前地址
	UtxoId      uint64 // 当前utxo
	Ticker      string
	TransferNft *common.TransferNFT // 有可能多个transfer nft在转移时，输出到同一个utxo中，这个时候直接修改Amount
}

func (p *TransferNftInfo) Clone() *TransferNftInfo {
	return &TransferNftInfo{
		TxInIndex:   p.TxInIndex,
		AddressId:   p.AddressId,
		UtxoId:      p.UtxoId,
		Ticker:      p.Ticker,
		TransferNft: p.TransferNft.Clone(),
	}
}

type BRC20Indexer struct {
	db           common.KVDB
	nftIndexer   *nft.NftIndexer
	status       *common.BRC20Status
	enableHeight int

	// 缓存数据，非全量数据
	mutex          sync.RWMutex                // 只保护这几个结构
	tickerMap      map[string]*BRC20TickInfo   // ticker -> TickerInfo，只保存近期几个区块的铸造数据，非全量
	holderMap      map[uint64]*HolderInfo      // addrId -> holder 用于动态更新ticker的holder数据，需要备份到数据库
	transferNftMap map[uint64]*TransferNftInfo // utxoId -> HolderInfo中的TransferableData的Nft，当前区块所需数据
	//tickerToHolderMap map[string]map[uint64]bool  // ticker -> addrId. 动态数据，跟随Holder变更，当前区块所需数据

	holderActionList []*HolderAction // 在同一个block中，状态变迁需要按顺序执行
	tickerAdded      []*common.BRC20Ticker
	tickerUpdated    map[string]*common.BRC20Ticker // key: ticker

	// 其他辅助信息，不需要clone
	actionBufferMap map[uint64]*ActionInfo // key: input的utxoId，保存一个区块

	// checkpoint 临时使用
	holderMapInPrevBlock map[uint64]*common.Decimal
}

func NewIndexer(db common.KVDB, bCheckValidateFiles bool) *BRC20Indexer {
	enableHeight := 779832
	if !common.IsMainnet() {
		enableHeight = 27228
	}
	_enable_checking_more_files = bCheckValidateFiles
	return &BRC20Indexer{
		db:              db,
		enableHeight:    enableHeight,
		tickerMap:       make(map[string]*BRC20TickInfo),
		holderMap:       make(map[uint64]*HolderInfo),
		transferNftMap:  make(map[uint64]*TransferNftInfo),
		actionBufferMap: make(map[uint64]*ActionInfo),
		tickerUpdated:   make(map[string]*common.BRC20Ticker),
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

func (s *BRC20Indexer) Repair() map[string]string {

	// tickers := s.loadTickListFromDBv2()
	// wb := s.db.NewWriteBatch()
	// defer wb.Close()

	// for id, name := range tickers {
	// 	key := GetTickerIdKey(id)
	// 	err := db.SetDB([]byte(key), name, wb)
	// 	if err != nil {
	// 		common.Log.Panicf("Error setting %s in db %v", key, err)
	// 	}
	// }

	// err := wb.Flush()
	// if err != nil {
	// 	common.Log.Panicf("Error ordxwb flushing writes to db %v", err)
	// }
	// common.Log.Infof("BRC20Indexer repair done, write items %d", len(tickers))
	// return true


	tickerAll, err := validate.ReadBRC20TickersCSV("./indexer/brc20/validate/ticker_all.csv")
	if err != nil {
		common.Log.Panicf("ReadBRC20TickersCSV failed, %v", err)
	}

	result := make(map[string]string)
	tickersInDB := s.loadTickListFromDBv2()
	for id, ticker := range tickerAll {
		_, ok := tickersInDB[id]
		if !ok {
			if ticker.DeployHeight > 931900 {
				continue
			}
			result[id] = ticker.Ticker
			common.Log.Infof("ticker %s is missing, %v", ticker.Ticker, ticker)
		}
	}
	return result

	//return false
}

// 只保存UpdateDB需要用的数据
func (s *BRC20Indexer) Clone(nftIndexer *nft.NftIndexer) *BRC20Indexer {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	newInst := NewIndexer(s.db, _enable_checking_more_files)
	newInst.nftIndexer = nftIndexer

	newInst.tickerMap = make(map[string]*BRC20TickInfo, 0)
	for key, value := range s.tickerMap {
		tick := BRC20TickInfo{}
		tick.Name = value.Name
		tick.Ticker = value.Ticker.Clone()
		tick.MintAdded = make([]*common.BRC20Mint, len(value.MintAdded))
		copy(tick.MintAdded, value.MintAdded)

		// tick.InscriptionMap = make(map[string]*common.BRC20MintAbbrInfo, 0)
		// for inscriptionId, mintAbbrInfo := range value.InscriptionMap {
		// 	tick.InscriptionMap[inscriptionId] = mintAbbrInfo
		// }
		newInst.tickerMap[key] = &tick
	}

	newInst.holderMap = make(map[uint64]*HolderInfo)
	for addressId, holder := range s.holderMap {
		newInst.holderMap[addressId] = holder.Clone()
	}

	newInst.transferNftMap = make(map[uint64]*TransferNftInfo)
	for utxoId, transfer := range s.transferNftMap {
		newInst.transferNftMap[utxoId] = transfer.Clone()
	}

	// 保存holderActionList对应的数据，更新数据库需要
	newInst.holderActionList = make([]*HolderAction, len(s.holderActionList))
	copy(newInst.holderActionList, s.holderActionList)

	newInst.tickerAdded = make([]*common.BRC20Ticker, len(s.tickerAdded))
	for i, ticker := range s.tickerAdded {
		newInst.tickerAdded[i] = ticker.Clone()
	}

	newInst.tickerUpdated = make(map[string]*common.BRC20Ticker, 0)
	for key, value := range s.tickerUpdated {
		newInst.tickerUpdated[key] = value.Clone()
	}

	newInst.status = s.status.Clone()

	return newInst
}

// update之后，删除原来instance中的数据
func (s *BRC20Indexer) Subtract(another *BRC20Indexer) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for key, value := range another.tickerMap {
		ticker, ok := s.tickerMap[key]
		if ok {
			ticker.MintAdded = append([]*common.BRC20Mint(nil), ticker.MintAdded[len(value.MintAdded):]...)
		}
		if ticker.Ticker.TransactionCount == value.Ticker.TransactionCount {
			// 没有更多交易
			delete(s.tickerMap, key)
		}
	}

	for addressId, old := range another.holderMap {
		n, ok := s.holderMap[addressId]
		if !ok {
			continue
		}
		if n.FreshTime == old.FreshTime {
			delete(s.holderMap, addressId)
		}
	}

	for utxoId := range another.transferNftMap {
		delete(s.transferNftMap, utxoId)
	}

	s.holderActionList = append([]*HolderAction(nil), s.holderActionList[len(another.holderActionList):]...)
	s.tickerAdded = append([]*common.BRC20Ticker(nil), s.tickerAdded[len(another.tickerAdded):]...)

	for name, old := range another.tickerUpdated {
		n, ok := s.tickerUpdated[name]
		if !ok {
			continue
		}
		if n.TransactionCount == old.TransactionCount {
			delete(s.tickerUpdated, name)
		}
	}
}

// 在系统初始化时调用一次，如果有历史数据的话。一般在NewSatIndex之后调用。
func (s *BRC20Indexer) Init(nftIndexer *nft.NftIndexer) {

	s.nftIndexer = nftIndexer

	startTime := time.Now()
	version := s.GetDBVersion()

	s.status = initStatusFromDB(s.db)
	common.Log.Infof("brc20 db version: %s", version)
	common.Log.Info("Init ...")

	elapsed := time.Since(startTime).Milliseconds()
	common.Log.Infof("Init %d ms", elapsed)

	//s.validateAllHistory("cats", "./indexer/brc20/validate/cats_records.csv")
	//s.printTickerHistoryWithHeight("cats", 814163)
	//s.printLatestTickerHistory("cats", 100)
	//s.printTickerHistoryWithHeight("mask", 885497)
	//s.printLatestTickerHistory("𝛑", 100)
	//common.Log.Panicf("")
	// s.printHistoryWithAddress("benz", 0x6a0acc6b)
	// info := s.getHolderAbbrInfo(0x6a0acc6b, "benz")
	// common.Log.Infof("%v", info)

	common.Log.Infof("height = %d, total tickers %d", nftIndexer.GetBaseIndexer().GetSyncHeight(), s.status.TickerCount)
}

func (s *BRC20Indexer) printHistoryWithAddress(name string, addressId uint64) {
	history := s.loadTickerHistoryWithHolder(name, addressId)
	var total, available, transferrable *common.Decimal
	var count int
	rpc := base.NewRpcIndexer(s.nftIndexer.GetBaseIndexer())
	address, _ := rpc.GetAddressByID(addressId)
	common.Log.Infof("address %x %s", addressId, address)
	common.Log.Infof("data from history")
	for _, item := range history {
		if item.Action == common.BRC20_Action_Transfer_Spent {
			continue
		}
		flag := "+"
		var method string
		switch item.Action {
		case common.BRC20_Action_InScribe_Mint:
			total = total.Add(&item.Amount)
			available = available.Add(&item.Amount)
			method = "insribe-mint"

		case common.BRC20_Action_InScribe_Transfer:
			flag = ""
			transferrable = transferrable.Add(&item.Amount)
			available = available.Sub(&item.Amount)
			method = "inscribe-transfer"

		case common.BRC20_Action_Transfer:
			if addressId == item.FromAddr {
				flag = "-"
				total = total.Sub(&item.Amount)
				transferrable = transferrable.Sub(&item.Amount)
			}
			if addressId == item.ToAddr {
				total = total.Add(&item.Amount)
				available = available.Add(&item.Amount)
			}
			if item.FromAddr == item.ToAddr {
				flag = ""
			}
			method = "transfer"

		}

		var from string
		if item.FromAddr == common.INVALID_ID {
			from = "-\t"
		} else {
			from = fmt.Sprintf("%x", item.FromAddr)
		}

		var to string
		if item.ToAddr == common.INVALID_ID {
			to = "-\t"
		} else {
			to = fmt.Sprintf("%x", item.ToAddr)
		}
		// h, i, j := common.FromUtxoId(item.ToUtxoId)
		// common.Log.Infof("%d %d %d: %d %s -> %s, %s%s, total = %s (%s, %s), %s",
		// 	h, i, j, item.NftId, from, to, flag, item.Amount.String(), total.String(),
		// 	available.String(), transferrable.String(), method)

		nft := s.nftIndexer.GetNftWithId(item.NftId)
		common.Log.Infof("%d %s: %s -> %s, %s%s, total = %s (%s, %s), %s",
			nft.Base.Id, nft.Base.InscriptionId, from, to, flag, item.Amount.String(), total.String(),
			available.String(), transferrable.String(), method)

		count++
		if count%20 == 0 {
			common.Log.Infof("")
		}
	}
	common.Log.Infof("total: %s", total.String())

	abbrInfo := s.getHolderAbbrInfo(addressId, name)
	if abbrInfo == nil {
		common.Log.Infof("GetHolderAbbrInfo failed")
		return
	}
	common.Log.Infof("data from GetHolderAbbrInfo")
	common.Log.Infof("asset = %s (%s, %s)", abbrInfo.AssetAmt().String(),
		abbrInfo.AvailableBalance.String(), abbrInfo.TransferableBalance.String())
}

func (s *BRC20Indexer) printHistory(history []*common.BRC20ActionHistory) map[uint64]*common.Decimal {
	holders := make(map[uint64]*common.Decimal)
	var total *common.Decimal
	var count int
	baseIndexer := s.nftIndexer.GetBaseIndexer()
	for _, item := range history {
		flag := ""
		if item.Action == common.BRC20_Action_InScribe_Mint {
			holders[item.ToAddr] = holders[item.ToAddr].Add(&item.Amount)
			total = total.Add(&item.Amount)
			flag = "+"
		}
		if item.Action == common.BRC20_Action_Transfer {
			holders[item.FromAddr] = holders[item.FromAddr].Sub(&item.Amount)
			holders[item.ToAddr] = holders[item.ToAddr].Add(&item.Amount)
		}

		var from string
		if item.FromAddr == common.INVALID_ID {
			from = "-\t"
		} else {
			addr, _ := baseIndexer.GetAddressByID(item.FromAddr)
			from = fmt.Sprintf("%s(0x%x)", addr, item.FromAddr)
		}

		var to string
		if item.ToAddr == common.INVALID_ID {
			to = "-\t"
		} else {
			addr, _ := baseIndexer.GetAddressByID(item.ToAddr)
			to = fmt.Sprintf("%s(0x%x)", addr, item.ToAddr)
		}
		// h, i, j := common.FromUtxoId(item.ToUtxoId)
		// common.Log.Infof("%d %d %d: %d %s -> %s, %s%s, %s, total = %s, %d",
		// 	h, i, j, item.NftId, from, to, flag, item.Amount.String(), holders[item.ToAddr].String(), total.String(),
		// 	item.Action)
		nft := s.nftIndexer.GetNftWithId(item.NftId)
		common.Log.Infof("%d %s: %s -> %s, %s%s, %s, total = %s, %d",
			nft.Base.Id, nft.Base.InscriptionId, from, to, flag, item.Amount.String(), holders[item.ToAddr].String(), total.String(),
			item.Action)

		count++
		if count%20 == 0 {
			common.Log.Infof("")
		}
	}
	common.Log.Infof("total in mint: %s", total.String())
	common.Log.Infof("holders from history")
	return holders
}

func (s *BRC20Indexer) printTickerHistory(name string) {
	history := s.loadTickerHistory(name)
	common.Log.Infof("ticker %s history:", name)
	holders := s.printHistory(history)
	s.printHoldersWithMap(holders)
}


func (s *BRC20Indexer) printTickerHistoryWithHeight(name string, height int) {
	history := s.loadTickerHistoryWithHeight(name, height)
	common.Log.Infof("ticker %s history in height %d:", name, height)
	holders := s.printHistory(history)
	s.printHoldersWithMap(holders)
}

// 无效
func (s *BRC20Indexer) printLatestTickerHistory(name string, limit int) {
	history := s.loadLatestTransferHistoryFromDB(name, limit)
	common.Log.Infof("ticker %s latest history %d:", name, len(history))
	holders := s.printHistory(history)
	s.printHoldersWithMap(holders)
}

func (s *BRC20Indexer) printHoldersWithMap(holders map[uint64]*common.Decimal) {
	var total *common.Decimal
	type pair struct {
		addressId uint64
		amt       *common.Decimal
	}
	mid := make([]*pair, 0)
	for addressId, amt := range holders {
		//common.Log.Infof("%x: %s", addressId, amt.String())
		total = total.Add(amt)
		mid = append(mid, &pair{
			addressId: addressId,
			amt:       amt,
		})
	}
	sort.Slice(mid, func(i, j int) bool {
		return mid[i].amt.Cmp(mid[j].amt) > 0
	})
	limit := 20 //len(mid) // 40
	baseIndexer := (s.nftIndexer.GetBaseIndexer())
	for i, item := range mid {
		if i > limit {
			break
		}
		if item.amt.Sign() == 0 {
			continue
		}
		address, err := baseIndexer.GetAddressByID(item.addressId)
		if err != nil {
			common.Log.Panicf("printHoldersWithMap GetAddressByID %x failed, %v", item.addressId, err)
			address = "-\t"
		}
		common.Log.Infof("%d: %x %s: %s", i, item.addressId, address, item.amt.String())
	}
	for i, item := range mid {
		if i > limit {
			break
		}
		address, err := baseIndexer.GetAddressByID(item.addressId)
		if err != nil {
			common.Log.Panicf("printHoldersWithMap GetAddressByID %x failed, %v", item.addressId, err)
			address = "-\t"
		}
		fmt.Printf("\"%s\": \"%s\",\n", address, item.amt.String())
	}
	common.Log.Infof("total in holders: %s", total.String())
}

func (s *BRC20Indexer) printHolders(name string) {
	holdermap := s.GetHoldersWithTick(name)
	common.Log.Infof("holders from holder DB")
	s.printHoldersWithMap(holdermap)
}

func (s *BRC20Indexer) printTicker(name string) {
	ticker := s.GetTicker(name)
	if ticker == nil {
		common.Log.Infof("can't find brc20 ticker %s", name)
		return
	}
	common.Log.Infof("Ticker: %s", ticker.Name)
	common.Log.Infof("self mint: %v", ticker.SelfMint)
	common.Log.Infof("Limit: %s", ticker.Limit.String())
	common.Log.Infof("Max: %s", ticker.Max.String())
	common.Log.Infof("Divisibility: %d", ticker.Decimal)
	common.Log.Infof("Minted: %s", ticker.Minted.String())
	common.Log.Infof("MintCount: %d", ticker.MintCount)
	common.Log.Infof("StartInscription: %s", ticker.StartInscriptionId)
	common.Log.Infof("EndInscription: %s", ticker.EndInscriptionId)
	common.Log.Infof("HolderCount: %d", ticker.HolderCount)
	common.Log.Infof("TX Count: %d", ticker.TransactionCount)
	common.Log.Infof("%d %s", ticker.Nft.Base.Id, ticker.Nft.Base.InscriptionId)
}

// 自检。如果错误，将停机
func (s *BRC20Indexer) CheckSelf() bool {
	common.Log.Infof("BRC20Indexer->CheckSelf ...")
	common.Log.Infof("stats: %v", s.status)
	height := s.nftIndexer.GetBaseIndexer().GetHeight()

	isMainnet := s.nftIndexer.GetBaseIndexer().IsMainnet()
	//var names []string

	//s.CheckPointWithBlockHeight(height)

	// holdermap := s.GetHoldersWithTick("meme")
	// for k, v := range holdermap {
	// 	old, ok := s.holdermap[k]
	// 	if ok {
	// 		if old.Cmp(v) != 0 {
	// 			common.Log.Infof("%x changed %s -> %s", k, old.String(), v.String())
	// 		}
	// 	} else {
	// 		common.Log.Infof("%x added %s -> %s", k, old.String(), v.String())
	// 	}
	// }

	// if isMainnet {
		// names = []string{
		// 	"ordi",
		// 	"sats",
		// 	"doge",
		// 	"rats",
		// 	"𝛑",
		// 	"pizza",
		// 	"ligo",
		// 	"piin",
		// 	"benz",
		// 	"mask",
		// 	"eorb",
		// 	"mmss",
		// 	"mice",
		// 	"bear",
		// 	"dior",
		// 	"safe",
		// 	"scat",
		// }
	// } else {
	// 	names = []string{
	// 		"ordi",
	// 		"usdt",
	// 		"test",
	// 		"husk",
	// 		"gc  ",
	// 		"ttt3",
	// 		"doge",
	// 		"rats",
	// 		"ttt3",
	// 		"tbtc",
	// 		"brc20",
	// 		"sats",
	// 		"bfun ",
	// 		"⚽ ",
	// 	}
	// }
	// for _, name := range names {
	// 	s.printTicker(name)
	// 	//s.printHistory(name)
	// 	s.printHolders(name)
	// 	//s.printHistoryWithAddress(name, 0x51cd94cd)
	// 	//s.printHistoryWithAddress(name, 0x306ce3)
	// 	// s.printHistoryWithAddress(name, 0x38815d)
	// 	// s.printHistoryWithAddress(name, 0x3b37a3)
	// 	// s.printHistoryWithAddress(name, 0x3ff5fe)
	// 	//s.printHistoryWithAddress(name, 0x1569f9)
	// 	//s.printHistoryWithAddress(name, 0x3b0cee)
	// }

	// 下面这个方式遍历所有ticker极慢，需要参考nft模块的方案 TODO
	startTime := time.Now()
	allTickers := s.GetAllTickers()
	allHolders := make(map[uint64]bool)
	for _, name := range allTickers {
	//for _, name := range names {
		//common.Log.Infof("checking ticker %s", name)

		//name := "benz"
		ticker := s.GetTicker(name)

		holdermap := s.GetHoldersWithTick(name)
		var holderAmount *common.Decimal
		for u, amt := range holdermap {
			holderAmount = holderAmount.Add(amt)
			allHolders[u] = true
		}

		// if name == "benz" {
		// 	info1 := s.getHolderAbbrInfo(0x6a913542, "benz")
		// 	info2 := holdermap[0x6a913542]
		// 	if info2.Cmp(info1.AssetAmt()) != 0 {
		// 		common.Log.Panic("")
		// 	}
		// }

		// if name == "42-c" {
		// 	common.Log.Info("")
		// }
		mintAmount, _ := s.GetMintAmount(name)
		if ticker.Id < 10 {
			common.Log.Infof("ticker %s, minted %s, holders %d, TxCount %d", name, mintAmount.String(), ticker.HolderCount, ticker.TransactionCount)
			//s.printHistory(name)
			//s.printHolders(name)
		}
		//fmt.Printf("\"%s\": {Minted: \"%s\", HolderCount: %d, TxCount: %d},\n", name, mintAmount.String(), ticker.HolderCount, ticker.TransactionCount)
		if holderAmount.Cmp(mintAmount) != 0 {
			common.Log.Errorf("ticker %s amount incorrect. %s %s", name, mintAmount.String(), holderAmount.String())
			//s.printTickerHistory(name)
			//s.printHolders(name)
			return false
		}
		//s.printTicker(name)
		//s.printHolders(name)
		//common.Log.Info("")
	}
	common.Log.Infof("total tickers %d", len(allTickers))
	common.Log.Infof("brc20 has %d holders", len(allHolders))
	allHolders = nil

	// 需要高度到达一定高度才需要检查
	if (isMainnet && height >= 828800) ||
		(!isMainnet && height >= 28865) {
		// 需要区分主网和测试网
		name := "ordi"
		ticker := s.GetTicker(name)
		if ticker == nil {
			common.Log.Errorf("can't find %s in db", name)
			return false
		}

		holdermap := s.GetHoldersWithTick(name)
		var holderAmount *common.Decimal
		for _, amt := range holdermap {
			holderAmount = holderAmount.Add(amt)
		}

		mintAmount, _ := s.GetMintAmount(name)
		if holderAmount.Cmp(mintAmount) != 0 {
			common.Log.Errorf("ticker amount incorrect. %d %d", mintAmount, holderAmount)
			return false
		}
	}

	// special tickers
	type TickerInfo struct {
		Name               string
		InscriptionId      string
		Max                string
		Minted             string
		Limit              string
		Decimal            uint8
		SelfMint           bool
		DeployAddress      string
		DeployTime         string
		CompletedTime      string
		StartInscriptionId string
		EndInscriptionId   string
		HolderCount        uint64
		TransactionCount   uint64
		Top10Holders       map[string]map[string]string // address -> ticker -> balance
	}

	var specialTickers [1]*TickerInfo
	var checkHeight int
	if isMainnet {
		checkHeight = 923306
		tickerInfo1 := TickerInfo{
			Name:               "ordi",
			InscriptionId:      "b61b0172d95e266c18aea0c624db987e971a5d6d4ebc2aaed85da4642d635735i0",
			Max:                "21000000",
			Minted:             "21000000",
			Limit:              "1000",
			Decimal:            18,
			SelfMint:           false,
			DeployAddress:      "bc1pxaneaf3w4d27hl2y93fuft2xk6m4u3wc4rafevc6slgd7f5tq2dqyfgy06",
			DeployTime:         "2023-03-08 12:16:31",
			CompletedTime:      "2023-03-10 07:23:15",
			StartInscriptionId: "b61b0172d95e266c18aea0c624db987e971a5d6d4ebc2aaed85da4642d635735i0",
			EndInscriptionId:   "17352fd494b0cd70f0a835575178bdbaeca789fa2fd49c4c552bc9abfdb96b5bi0",
			HolderCount:        27469,
			TransactionCount:   412663,
			Top10Holders: map[string]map[string]string{
				// 目标ticker： ordi, sats, rats，mask，ligo，mmss，pizza, (再补充几个ticker，按照该ticker的transaction数量，找出最大交易量的前十个ticker)
				"bc1p8w6zr5e2q60s0r8al4tvmsfer77c0eqc8j55gk8r7hzv39zhs2lqa8p0k6": {
					"ordi":  "7962666.2",
					"sats":  "967676769384044",
					"rats":  "0",
					"mask":  "10",
					"ligo":  "0",
					"mmss":  "0",
					"pizza": "0",
					"nenk":  "1377995411110924.568",
					"GDP ":  "103000000000000000",
					"F财":    "88888888888888901",
					"vitas": "10000000000000000",
					"$🐀":    "4379476432206120.446",
					"℡:":    "3999999999999990",
				},
				"bc1qggf48ykykz996uv5vsp5p9m9zwetzq9run6s64hm6uqfn33nhq0ql9t85q": {
					"ordi":  "1675393.4579653",
					"sats":  "402841653528669.71358",
					"rats":  "0",
					"mask":  "0",
					"ligo":  "100000",
					"mmss":  "0",
					"pizza": "0",
					"F财":    "88888888888888900",
					"FC2 ":  "666666666666666666",
					"GDP ":  "101000000000000000",
				},
				"1GrwDkr33gT6LuumniYjKEGjTLhsL5kmqC": {
					"ordi":  "1182993.79442012",
					"sats":  "26434461284591.70974878",
					"rats":  "145073767753.54206146",
					"mask":  "0",
					"ligo":  "0",
					"mmss":  "0",
					"pizza": "0",
					"GDP ":  "100000000000000000",
					"F财":    "88888888888888900",
					"FC2 ":  "41115347698724264",
				},
				"bc1qqd72vtqlw0nugqmzrx398x8gj03z8aqr79aexrncezqaw74dtu4qxjydq3": {
					"ordi":  "989780.51420967",
					"sats":  "0",
					"rats":  "0",
					"mask":  "0",
					"ligo":  "100000",
					"mmss":  "0",
					"pizza": "0",
					"GDP ":  "101,000,000,000,000,000",
					"vitas": "10,000,000,000,000,000",
					"F财":    "9,999,999,999,999,999",
				},
				"bc1qz7rw2atrt3e8jrywva2y8xmka8lewalx8qazlxaq8xkn2xke0yyqvpel3e": {
					"ordi":  "650111.63640285",
					"sats":  "0",
					"rats":  "0",
					"mask":  "0",
					"ligo":  "100000",
					"mmss":  "0",
					"pizza": "0",
					"GDP ":  "100,000,000,000,000,000",
					"F财":    "9,999,999,999,999,999",
					"vitas": "6,000,000,000,000,000",
				},
				"bc1q8u9thhxvkjw9t8tf0sj6k0vwmk7jstc9z0f3at0r5xunxxp9f0pqmetg7x": {
					"ordi":  "612,586.44263859",
					"sats":  "0",
					"rats":  "0",
					"mask":  "0",
					"ligo":  "100000",
					"mmss":  "0",
					"pizza": "0",
					"XOKX":  "1111111111111111111",
					"GDP ":  "100,000,000,000,000,000",
					"F财":    "9999999999999999",
				},
				"bc1qm64dsdz853ntzwleqsrdt5p53w75zfrtnmyzcx": {
					"ordi":  "401,961.33365660",
					"sats":  "51,813,398,915,940.04846520",
					"rats":  "6,575,223,084.11387540",
					"mask":  "980",
					"ligo":  "0",
					"mmss":  "573,528.4645034",
					"pizza": "0",
					"FC2 ":  "888,888,888,888,888,888",
					"GDP ":  "100,000,000,000,000,000",
					"F财":    "88,888,888,888,888,900",
				},
				"bc1pxl55h9yhj6v3uuwx7njp3gyqdd8fv0erya8qfj5dnuuy92jdzmmsjjjl6w": {
					"ordi":  "332,819.11426585",
					"sats":  "16,187,214,938,276.18321805",
					"rats":  "27,504,282,046.32612588",
					"mask":  "0",
					"ligo":  "0",
					"mmss":  "321,777.08702964",
					"pizza": "0",
					"F财":    "9,999,999,999,999,999",
					"X@AI":  "210,000,000,000,000",
					"GDP ":  "9,999,999,999,999",
				},
				"bc1qzy2hg9aup0vnt3cnetlpc8h7eytqveqxk36rjfsd8dy8kfyg29yqg29swh": {
					"ordi":  "298,461.89126292",
					"sats":  "0",
					"rats":  "0",
					"mask":  "0",
					"ligo":  "0",
					"mmss":  "0",
					"pizza": "0",
					"OROK":  "100",
					"WPCD":  "5",
					"BCOF":  "5000000",
				},
				"bc1qvf3hhl2jj75tq834yrrud3tj5ltrsqzsgevyhadfytar5depvlgqpfvpau": {
					"ordi":  "236103.82747003",
					"sats":  "0",
					"rats":  "0",
					"mask":  "0",
					"ligo":  "0",
					"mmss":  "0",
					"pizza": "0",
					"BCOF":  "5000000",
					"WPCD":  "5",
					"hoe.":  "0.1",
				},
			},
		}
		specialTickers[0] = &tickerInfo1
	} else {
		checkHeight = 108237
		tickerInfo1 := &TickerInfo{
			Name:               "ordi",
			InscriptionId:      "3b84bfba456be05287c0888bcbf5df778c8946ff6b057fd0836cc65c12546f12i0",
			Max:                "2400000000",
			Minted:             "1211730992", // unisat: 1211670992
			Limit:              "10000",
			Decimal:            18,
			SelfMint:           false,
			DeployAddress:      "tb1pmm586mlhs35e8ns08trdejpzv02rupx0hp9j8arumg5c29dyrfnq2trqcw",
			DeployTime:         "2024-06-06 14:43:56",
			CompletedTime:      "",
			StartInscriptionId: "3b84bfba456be05287c0888bcbf5df778c8946ff6b057fd0836cc65c12546f12i0",
			EndInscriptionId:   "",
			HolderCount:        141,    // unisat: 138
			TransactionCount:   121638, // unisat: 121636
			Top10Holders: map[string]map[string]string{
				"tb1p6eahny66039p30ntrp9ke0qpyyffgnkekf69js6d2qcjf8cdmu0shx273f": {
					"ordi": "230000000",
					"Usdt": "4239000",
					// "pizza5": "0",
				},
				"tb1pgw439hxzr7vj0gzfqx69wl3plem4ne26kj7ktnuzj3lkpw5mmp3qhz7yv4": {
					"ordi": "230000000",
					"Usdt": "4302000",
					"Test": "2000000",
					"GC  ": "100000",
				},
				"tb1pc2nqm8k0kwnctkr2amchtcys4fq4elkq8ezhtsrntlkfc92z5tssh68xzl": {
					"ordi": "190000000",
				},
				"tb1qy6zm520mnla9894t4jqvwe9s2sjsn2sfude0r0": {
					"ordi": "50260000",
				},
				"tb1plzvdzn3sagtlavxsrdv9kp65empk80j0ksmazzqdc6nqkarj238s4r5qwx": {
					"⚽ ":   "98010000", // unisat: 98020000
					"ordi": "50000000",
					"sats": "42000000",
					"CTRA": "12400000",
					"cats": "2000000",
					"Test": "2000000",
					"doge": "1000000",
					"rats": "412000",
				},
				"tb1p5cymzvgf87fgeuzfexwxgvlmuuq309gegfh4q6np8g4qq6lnlk3qpzf2rs": {
					"brc20": "1113000000",
					"ordi":  "50000000",
					"Usdt":  "7910000",
					"Test":  "3000000",
					"GC  ":  "9200",
				},
				"tb1qmtlvgn8fl8ug2kgu26r6j9gykxm90tv5v4f6zx": {
					"ordi": "40000000",
				},
				"tb1qn5pvsgw32gshn365n93wzw606hfy9k6cuvkxmn": {
					"ordi": "30000000",
				},
				"tb1qw3qp3d0m0ykl2v7yj4uvrp4gsw8pwqmghul8w8": {
					"ordi": "30000000",
				},
				"tb1qw65mlex2hpv2py2pucysfrfe59h3acde3vtya9": {
					"ordi": "20260000",
				},
			},
		}
		specialTickers[0] = tickerInfo1
		specialTickers[0] = nil
	}

	if checkHeight == height {
		for _, specialTicker := range specialTickers {
			if specialTicker == nil {
				continue
			}
			ticker := s.GetTicker(specialTicker.Name)
			if specialTicker.InscriptionId != ticker.Nft.Base.InscriptionId {
				common.Log.Errorf("ticker InscriptionId incorrect")
				return false
			}
			if specialTicker.TransactionCount != ticker.TransactionCount {
				common.Log.Errorf("ticker TransactionCount incorrect")
				return false
			}
			if specialTicker.Max != ticker.Max.String() {
				common.Log.Errorf("ticker Max incorrect")
				return false
			}
			minted := ticker.Minted.String()
			if specialTicker.Minted != minted {
				common.Log.Errorf("ticker Minted incorrect")
				return false
			}
			if specialTicker.Limit != ticker.Limit.String() {
				common.Log.Errorf("ticker Limit incorrect")
				return false
			}
			if specialTicker.Decimal != ticker.Decimal {
				common.Log.Errorf("ticker Decimal incorrect")
				return false
			}
			if specialTicker.SelfMint != ticker.SelfMint {
				common.Log.Errorf("ticker SelfMint incorrect")
				return false
			}

			startNftInfo := base_indexer.ShareBaseIndexer.GetNftInfoWithInscriptionId(ticker.StartInscriptionId)
			deployAddress := base_indexer.ShareBaseIndexer.GetAddressById(startNftInfo.OwnerAddressId)
			if specialTicker.DeployAddress != deployAddress {
				common.Log.Errorf("ticker DeployAddress incorrect")
				return false
			}
			deployTime := time.Unix(ticker.DeployTime, 0).Format("2006-01-02 15:04:05")
			if specialTicker.DeployTime != deployTime {
				common.Log.Errorf("ticker DeployTime incorrect")
				return false
			}

			endNftInfo := base_indexer.ShareBaseIndexer.GetNftInfoWithInscriptionId(ticker.EndInscriptionId)
			if endNftInfo != nil {
				completedTime := time.Unix(endNftInfo.Base.BlockTime, 0).Format("2006-01-02 15:04:05")
				if specialTicker.CompletedTime != completedTime {
					common.Log.Errorf("ticker CompletedTime incorrect")
					return false
				}
			}

			if specialTicker.StartInscriptionId != ticker.StartInscriptionId {
				common.Log.Errorf("ticker StartInscriptionId incorrect")
				return false
			}
			if specialTicker.EndInscriptionId != ticker.EndInscriptionId {
				common.Log.Errorf("ticker EndInscriptionId incorrect")
				return false
			}
			if specialTicker.HolderCount != ticker.HolderCount {
				common.Log.Errorf("ticker HolderCount incorrect")
				return false
			}

			for address, holder := range specialTicker.Top10Holders {
				addressId := s.nftIndexer.GetBaseIndexer().GetAddressId(address)
				assertSummarys := s.GetAssetSummaryByAddress(addressId)
				for tickerName, amt := range assertSummarys {
					if holder[tickerName] != amt.String() {
						common.Log.Errorf("ticker amt incorrect")
						return false
					}
				}
			}
		}
	}

	// 最后才设置dbver
	s.setDBVersion()
	common.Log.Infof("BRC20Indexer->CheckSelf took %v.", time.Since(startTime))

	return true
}


func (s *BRC20Indexer) loadTickInfo(name string) *BRC20TickInfo {
	ret := s.tickerMap[name]
	if ret != nil {
		return ret
	}

	ticker := s.loadTickerFromDB(name)
	if ticker == nil {
		return nil
	}

	info := &BRC20TickInfo{
		Name:   name,
		Ticker: ticker,
	}
	s.tickerMap[name] = info

	return info
}

// 仅加载需要的ticker数据，需要加写锁，确保返回值都非nil
func (s *BRC20Indexer) loadHolderInfo(addressId uint64, name string) (*HolderInfo, *common.BRC20TickAbbrInfo) {
	holder := s.holderMap[addressId]
	if holder == nil {
		holder = NewHolderInfo()
		s.holderMap[addressId] = holder
	}

	info, ok := holder.Tickers[name]
	if !ok {
		info = s.loadTickAbbrInfoFromDB(addressId, name) // 虽然预加载过，但可能中间有问题，多加载一次，确保数据跟数据库一致
		if info == nil {
			info = common.NewBRC20TickAbbrInfo()
		}
		holder.Tickers[name] = info
	}

	return holder, info
}

func (s *BRC20Indexer) loadTransferNft(utxoId uint64) *TransferNftInfo {
	transfer := s.transferNftMap[utxoId]
	if transfer == nil {
		transfer = s.loadTransferFromDB(utxoId)
		if transfer != nil {
			s.transferNftMap[utxoId] = transfer
		}
	}

	return transfer
}

func (s *BRC20Indexer) loadTickerHistory(name string) []*common.BRC20ActionHistory {
	history := s.loadTransferHistoryFromDB(name)
	for _, item := range s.holderActionList {
		if item.Ticker == name {
			history = append(history, item)
		}
	}
	return history
}

func (s *BRC20Indexer) loadTickerHistoryWithHolder(name string, addressId uint64) []*common.BRC20ActionHistory {
	history := s.loadTransferHistoryWithHolderFromDB(name, addressId)
	for _, item := range s.holderActionList {
		if item.Ticker == name && (item.FromAddr == addressId || item.ToAddr == addressId) {
			history = append(history, item)
		}
	}
	return history
}

func (s *BRC20Indexer) loadTickerHistoryWithHeight(name string, height int) []*common.BRC20ActionHistory {
	history := s.loadTransferHistoryWithHeightFromDB(name, height)
	for _, item := range s.holderActionList {
		if item.Ticker == name && item.Height == height {
			history = append(history, item)
		}
	}
	return history
}

func (s *BRC20Indexer) CheckHolderExisting(addrs []uint64) []uint64 {
	sort.Slice(addrs, func(i, j int) bool {
		return addrs[i] < addrs[j]
	})

	existingAddress := make([]uint64, 0)
	for _, addressId := range addrs {
		_, ok := s.holderMap[addressId]
		if ok {
			existingAddress = append(existingAddress, addressId)
			continue
		}
		if s.checkHolderExistingFromDB(addressId) {
			existingAddress = append(existingAddress, addressId)
			continue
		}
	}
	return existingAddress
}
