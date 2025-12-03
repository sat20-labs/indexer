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
	AddressId   uint64 // 当前地址
	UtxoId      uint64 // 当前utxo
	Ticker      string
	TransferNft *common.TransferNFT // 有可能多个transfer nft在转移时，输出到同一个utxo中，这个时候直接修改Amount
}

type BRC20Indexer struct {
	db         common.KVDB
	nftIndexer *nft.NftIndexer
	enableHeight int

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
	enableHeight := 779832
	if !common.IsMainnet() {
		enableHeight = 27228
	}
	return &BRC20Indexer{
		db: db,
		enableHeight: enableHeight,
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

	newInst.transferNftMap = make(map[uint64]*TransferNftInfo)
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
	if (s.nftIndexer.GetBaseIndexer().IsMainnet() && height >= 828800) ||
		(!s.nftIndexer.GetBaseIndexer().IsMainnet() && height >= 28865) {
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
	if s.nftIndexer.GetBaseIndexer().IsMainnet() {
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
