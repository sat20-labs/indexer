package brc20

import (
	"bufio"
	"embed"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
	"github.com/sat20-labs/indexer/indexer/nft"
	"github.com/sat20-labs/indexer/share/base_indexer"
)

type BRC20TickInfo struct {
	Id             uint64
	Name           string
	InscriptionMap map[string]*common.BRC20MintAbbrInfo // key: inscriptionId
	MintAdded      []*common.BRC20Mint
	Ticker         *common.BRC20Ticker
}

type HolderAction struct {
	Height int
	// Utxo     string
	UtxoId   uint64
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
	// AddressId uint64
	Tickers map[string]*common.BRC20TickAbbrInfo // key: ticker, 小写
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
			info := HolderInfo{ /*AddressId: value.AddressId,*/ Tickers: value.Tickers}
			newInst.holderMap[action.FromAddr] = &info
		}

		value, ok = s.holderMap[action.ToAddr]
		if ok {
			info := HolderInfo{ /*AddressId: value.AddressId,*/ Tickers: value.Tickers}
			newInst.holderMap[action.ToAddr] = &info
		}

		holders, ok := s.tickerToHolderMap[action.Ticker]
		if ok {
			newInst.tickerToHolderMap[action.Ticker] = holders
		}
	}

	for key, value := range s.transferNftMap {
		if newInst.transferNftMap == nil {
			newInst.transferNftMap = make(map[uint64]*TransferNftInfo)
		}
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
	version := s.GetDBVersion()
	if s.nftIndexer.GetBaseIndexer().IsMainnet() && version == "" {
		s.initCursorInscriptionsDB()
	}
	common.Log.Infof("brc20 db version: %s", version)
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

	var specialTickers [1]TickerInfo
	var checkHeight int
	if s.nftIndexer.GetBaseIndexer().IsMainnet() {
		checkHeight = 921406
		tickerInfo1 := TickerInfo{
			Name:               "sats",
			InscriptionId:      "9b664bdd6f5ed80d8d88957b63364c41f3ad4efb8eee11366aa16435974d9333i0",
			Max:                "2100000000000000",
			Minted:             "2100000000000000",
			Limit:              "100000000",
			Decimal:            18,
			SelfMint:           false,
			DeployAddress:      "bc1prtawdt82wfgrujx6d0heu0smxt4yykq440t447wan88csf3mc7csm3ulcn",
			DeployTime:         "2023-03-09 13:32:14",
			CompletedTime:      "2023-09-24 18:52:12",
			StartInscriptionId: "9b664bdd6f5ed80d8d88957b63364c41f3ad4efb8eee11366aa16435974d9333i0",
			EndInscriptionId:   "5d417bdd264635c441a4327711f4635c085092aa359b5a03dde4b16687fe8dadi0",
			HolderCount:        54377,
			TransactionCount:   21867973,
			Top10Holders: map[string]map[string]string{
				"bc1p8w6zr5e2q60s0r8al4tvmsfer77c0eqc8j55gk8r7hzv39zhs2lqa8p0k6": {
					"GDP ":  "103000000000000000",
					"F财":    "88888888888888901",
					"vitas": "10000000000000000",
					"sats":  "972376769384044",
					// ...
				},
				"bc1qggf48ykykz996uv5vsp5p9m9zwetzq9run6s64hm6uqfn33nhq0ql9t85q": {
					"FC2 ": "666666666666666666",
					"GDP ": "101000000000000000",
					"S@AI": "99999999999999000",
					"sats": "402841653528669.71358",
					// ...
				},
				"bc1qn2cpj0hrl37wqh5q94kwrlhtj2lx8ahtw7ef5rg35tswxsqtvufqfmmrq2": {
					"sats": "68084860391688.3432",
					"ordi": "992.5413",
					"WPCD": "5",
					// ...
				},
			},
		}
		specialTickers[0] = tickerInfo1
	} else {
		checkHeight = 108237
		tickerInfo1 := TickerInfo{
			Name:               "ordi",
			InscriptionId:      "3b84bfba456be05287c0888bcbf5df778c8946ff6b057fd0836cc65c12546f12i0",
			Max:                "2400000000",
			Minted:             "1211700992", // unisat: 1211670992
			Limit:              "10000",
			Decimal:            18,
			SelfMint:           false,
			DeployAddress:      "tb1pmm586mlhs35e8ns08trdejpzv02rupx0hp9j8arumg5c29dyrfnq2trqcw",
			DeployTime:         "2024-06-06 14:43:56",
			CompletedTime:      "",
			StartInscriptionId: "3b84bfba456be05287c0888bcbf5df778c8946ff6b057fd0836cc65c12546f12i0",
			EndInscriptionId:   "",
			HolderCount:        142,    // unisat: 138
			TransactionCount:   121633, // unisat: 121636
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
	}

	if checkHeight <= height {
		for _, specialTicker := range specialTickers {
			ticker := s.GetTicker(specialTicker.Name)
			if specialTicker.InscriptionId != ticker.Nft.Base.InscriptionId {
				return false
			}
			if specialTicker.TransactionCount != ticker.TransactionCount {
				return false
			}
			if specialTicker.Max != ticker.Max.String() {
				return false
			}
			if specialTicker.Minted != ticker.Minted.String() {
				return false
			}
			if specialTicker.Limit != ticker.Limit.String() {
				return false
			}
			if specialTicker.Decimal != ticker.Decimal {
				return false
			}
			if specialTicker.SelfMint != ticker.SelfMint {
				return false
			}

			startNftInfo := base_indexer.ShareBaseIndexer.GetNftInfoWithInscriptionId(ticker.StartInscriptionId)
			deployAddress := base_indexer.ShareBaseIndexer.GetAddressById(startNftInfo.OwnerAddressId)
			if specialTicker.DeployAddress != deployAddress {
				return false
			}
			deployTime := time.Unix(ticker.DeployTime, 0).Format("2006-01-02 15:04:05")
			if specialTicker.DeployTime != deployTime {
				return false
			}

			endNftInfo := base_indexer.ShareBaseIndexer.GetNftInfoWithInscriptionId(ticker.EndInscriptionId)
			if endNftInfo != nil {
				completedTime := time.Unix(endNftInfo.Base.BlockTime, 0).Format("2006-01-02 15:04:05")
				if specialTicker.CompletedTime != completedTime {
					return false
				}
			}

			if specialTicker.StartInscriptionId != ticker.StartInscriptionId {
				return false
			}
			if specialTicker.EndInscriptionId != ticker.EndInscriptionId {
				return false
			}
			if specialTicker.HolderCount != ticker.HolderCount {
				return false
			}
			if specialTicker.TransactionCount != ticker.TransactionCount {
				return false
			}

			for address, holder := range specialTicker.Top10Holders {
				addressId := s.nftIndexer.GetBaseIndexer().GetAddressId(address)
				assertSummarys := s.GetAssetSummaryByAddress(addressId)
				for tickerName, amt := range assertSummarys {
					if holder[tickerName] != amt.String() {
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

//go:embed brc20_curse.txt
var brc20Fs embed.FS

func (s *BRC20Indexer) initCursorInscriptionsDB() {
	// first brc inscriptin_number = 348020, cursor end block height = 837090 / last inescription number = 66799147
	inputPath := filepath.Join("", "brc20_curse.txt")
	input, err := brc20Fs.ReadFile(inputPath)
	if err != nil {
		common.Log.Panicf("Error reading brc20_curse: %v", err)
	}
	reader := strings.NewReader(string(input))
	regex := regexp.MustCompile(`id:([a-z0-9]+)`)
	scanner := bufio.NewScanner(reader)

	wb := s.db.NewWriteBatch()
	defer wb.Close()

	for scanner.Scan() {
		line := scanner.Text()
		submatches := regex.FindStringSubmatch(line)
		if len(submatches) != 2 {
			common.Log.Panicf("Error parsing brc20_curse: %s", line)
		}
		id := submatches[1]

		key := GetCurseInscriptionKey(id)
		err := wb.Put([]byte(key), nil)
		if err != nil {
			common.Log.Panicf("Error setting %s in db %v", key, err)
		}
	}
	wb.Flush()
}

func (s *BRC20Indexer) IsExistCursorInscriptionInDB(inscriptionId string) bool {
	key := GetCurseInscriptionKey(inscriptionId)
	_, err := s.db.Read([]byte(key))
	return err == nil
}
