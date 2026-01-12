package runes

import (

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/base"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"github.com/sat20-labs/indexer/indexer/runes/table"
	"lukechampine.com/uint128"
)

type Indexer struct {
	dbWrite                    *store.DbWrite
	baseIndexer                *base.BaseIndexer
	chaincfgParam              *chaincfg.Params
	enableHeight               int
	height                     int
	blockTime                  uint64
	Status                     *table.RunesStatus
	minimumRune                *runestone.Rune
	
	idToEntryTbl               *table.RuneIdToEntryTable           // RuneId->RuneEntry
	runeToIdTbl                *table.RuneToIdTable                // Rune->RuneId
	outpointToBalancesTbl      *table.OutpointToBalancesTable      // utxoId->该utxo包含的所有符文资产数据
	runeIdAddressToBalanceTbl  *table.RuneIdAddressToBalanceTable  // RuneId+AddressId -> Balance
	runeIdOutpointToBalanceTbl *table.RuneIdOutpointToBalanceTable // RuneId+utxoId -> 该utxo包含该符文的资产数量
	runeIdToMintHistoryTbl     *table.RuneToMintHistoryTable       // runeId+addressId -> utxoId + amount
	runeIdAddressToCountTbl    *table.RuneIdAddressToCountTable    // runeId+addressId -> utxo的数量

	//addressOutpointToBalancesTbl  *table.AddressOutpointToBalancesTable // addressId+utxoId -> runeId+balance  TODO 这个没用

	// transferUpdate 临时使用
	burnedMap                  table.RuneIdLotMap
	HolderUpdateCount          int
	HolderRemoveCount          int

	// checkpoint 临时使用
	holderMapInPrevBlock map[uint64]*common.Decimal
}

func NewIndexer(db common.KVDB, param *chaincfg.Params, bCheckValidateFile bool) *Indexer {
	logs := cmap.New[*store.DbLog]()
	dbWrite := store.NewDbWrite(db, &logs)
	enableHeight := 840000
	if !common.IsMainnet() {
		enableHeight = 30562
	}
	_enable_checking_more_files = bCheckValidateFile
	return &Indexer{
		dbWrite:                    dbWrite,
		chaincfgParam:              param,
		enableHeight:               enableHeight,
		burnedMap:                  nil,
		Status:                     table.NewRunesStatus(store.NewCache[pb.RunesStatus](dbWrite)),
		idToEntryTbl:               table.NewRuneIdToEntryTable(store.NewCache[pb.RuneEntry](dbWrite)),
		runeToIdTbl:                table.NewRuneToIdTable(store.NewCache[pb.RuneId](dbWrite)),
		outpointToBalancesTbl:      table.NewOutpointToBalancesTable(store.NewCache[pb.OutpointToBalances](dbWrite)),
		runeIdAddressToBalanceTbl:  table.NewRuneIdAddressToBalanceTable(store.NewCache[pb.RuneIdAddressToBalance](dbWrite)),
		runeIdOutpointToBalanceTbl: table.NewRuneIdOutpointToBalancesTable(store.NewCache[pb.RuneBalance](dbWrite)),
		//addressOutpointToBalancesTbl:  table.NewAddressOutpointToBalancesTable(store.NewCache[pb.AddressOutpointToBalance](dbWrite)),
		runeIdAddressToCountTbl: table.NewRuneIdAddressToCountTable(store.NewCache[pb.RuneIdAddressToCount](dbWrite)),
		runeIdToMintHistoryTbl:  table.NewRuneIdToMintHistoryTable(store.NewCache[pb.RuneIdToMintHistory](dbWrite)),
	}
}

func (s *Indexer) setDefaultRune() {
	firstRuneValue, err := uint128.FromString("2055900680524219742")
	if err != nil {
		common.Log.Panicf("RuneIndexer.Init-> uint128.FromString(2055900680524219742) err: %v", err)
	}
	r := runestone.Rune{
		Value: firstRuneValue,
	}
	id := &runestone.RuneId{Block: 1, Tx: 0}
	etching := "0000000000000000000000000000000000000000000000000000000000000000"
	s.runeToIdTbl.SetToDB(&r, id)

	symbol := defaultRuneSymbol
	startHeight := uint64(runestone.SUBSIDY_HALVING_INTERVAL * 4)
	endHeight := uint64(runestone.SUBSIDY_HALVING_INTERVAL * 5)
	s.idToEntryTbl.SetToDB(id, &runestone.RuneEntry{
		RuneId:       *id,
		Burned:       uint128.Uint128{},
		Divisibility: 0,
		Etching:      etching,
		Parent:       nil,
		Terms: &runestone.Terms{
			Amount: &uint128.Uint128{Hi: 0, Lo: 1},
			Cap:    &uint128.Max,
			Height: [2]*uint64{&startHeight, &endHeight},
			Offset: [2]*uint64{nil, nil},
		},
		Mints:      uint128.Uint128{},
		Number:     0,
		Premine:    uint128.Uint128{},
		SpacedRune: *runestone.NewSpacedRune(r, 128),
		Symbol:     &symbol,
		Timestamp:  0,
		Turbo:      true,
	})
}

func (s *Indexer) Init(baseIndexer *base.BaseIndexer) {
	s.baseIndexer = baseIndexer
	isExist := s.Status.Init()
	if !isExist && s.chaincfgParam.Net == wire.MainNet {
		s.setDefaultRune()

		s.Status.Number = 1
		s.Status.UpdateDb()
	}
	s.minimumRune = runestone.MinimumAtHeight(s.chaincfgParam.Net, uint64(s.Status.Height))

	s.height = s.Status.Height

	if s.chaincfgParam.Net == wire.MainNet {
		id := &runestone.RuneId{Block: 1, Tx: 0}
		rune := s.idToEntryTbl.Get(id)
		if rune == nil {
			s.setDefaultRune()
			rune = s.idToEntryTbl.Get(id)
			if rune == nil {
				common.Log.Panicf("rune %s is not existing", id.String())
			}
		}
	}
}

func (s *Indexer) Clone(baseIndexer *base.BaseIndexer) *Indexer {
	cloneIndex := NewIndexer(s.dbWrite.Db, s.chaincfgParam, _enable_checking_more_files)
	cloneIndex.height = s.height
	cloneIndex.Status.Version = s.Status.Version
	cloneIndex.Status.Height = s.Status.Height
	cloneIndex.Status.Number = s.Status.Number
	cloneIndex.Status.ReservedRunes = s.Status.ReservedRunes

	cloneIndex.minimumRune = &runestone.Rune{
		Value: s.minimumRune.Value,
	}

	s.dbWrite.Clone(cloneIndex.dbWrite)

	return cloneIndex
}

func (s *Indexer) Subtract(backupIndexer *Indexer) {
	backupIndexer.dbWrite.Subtract(s.dbWrite)
}

func (s *Indexer) CheckSelf() bool {

	var firstRuneName = ""
	switch s.chaincfgParam.Net {
	case wire.TestNet4:
		firstRuneName = "BESTINSLOT•XYZ"
		if s.height < 30562 {
			return true
		}
	case wire.MainNet:
		firstRuneName = "UNCOMMON•GOODS"
		if s.height < 840000 {
			return true
		}
	default:
		common.Log.Panicf("RuneIndexer.CheckSelf-> unknown net:%d", s.chaincfgParam.Net)
	}
	runeId, err := s.GetRuneIdWithName(firstRuneName)
	if err != nil {
		common.Log.Panicf("GetRuneIdWithName err:%s", err.Error())
	}
	common.Log.Debugf("rune: %s\n", firstRuneName)

	runeInfo := s.GetRuneInfoWithId(runeId.String())
	_, total := s.GetAllAddressBalances(runeId.String(), 0, 1)
	addressBalances, _ := s.GetAllAddressBalances(runeId.String(), 0, total)
	var addressBalance uint128.Uint128
	for _, v := range addressBalances {
		addressBalance = v.Balance.Add(addressBalance)
	}

	totalAddressBalance := addressBalance.Add(runeInfo.Burned)
	if addressBalance.Add(runeInfo.Burned).Cmp(totalAddressBalance) != 0 {
		common.Log.Errorf("all address(%d)'s total balance(%s) + burned is not equal to supply(%s)", total, totalAddressBalance.String(), runeInfo.Supply.String())
		return false
	}

	_, total = s.GetAllUtxoBalances(runeId.String(), 0, 0)
	utxoBalances, _ := s.GetAllUtxoBalances(runeId.String(), 0, total)
	totalUtxoBalance := utxoBalances.Total.Add(runeInfo.Burned)
	if utxoBalances.Total.Add(runeInfo.Burned).Cmp(totalUtxoBalance) != 0 {
		common.Log.Errorf("all utxo(%d)'s total balance(%s) + burned is not equal to supply(%s)", total, totalUtxoBalance.String(), runeInfo.Supply.String())
		return false
	}

	checkHolders := func(name string) bool {

		rune := s.GetRuneInfo(name)

		holdermap := s.GetHoldersWithTick(rune.Id)
		//common.Log.Infof("GetHoldersWithTick %s took %v.", rune.Id, time.Since(startTime2))
		var holderAmount *common.Decimal
		for _, amt := range holdermap {
			holderAmount = holderAmount.Add(amt)
		}
		if rune.HolderCount != uint64(len(holdermap)) {
			common.Log.Errorf("rune ticker %s holder count different. %d %d", rune.Name, rune.HolderCount, len(holdermap))
			return false
		}
		if rune.TotalHolderAmt().Cmp(holderAmount) != 0 {
			common.Log.Errorf("rune ticker %s holder amount different. %s %s", rune.Name, rune.TotalHolderAmt(), holderAmount)
			return false
		}

		if rune.Number < 10 {
			common.Log.Infof("rune %s amount: %s, holders: %d", rune.Name, holderAmount.String(), len(holdermap))
		} 

		//startTime2 = time.Now()
		_, total := s.GetAllUtxoBalances(rune.Id, 0, 0)
		//common.Log.Infof("GetAllUtxoBalances %s took %v.", rune.Id, time.Since(startTime2))
		if total == 0 {
			if holderAmount.Sign() != 0 {
				common.Log.Errorf("rune ticker %s GetAllUtxoBalances failed", rune.Name)
				return false
			}
		} else {
			//startTime2 = time.Now()
			utxos, _ := s.GetAllUtxoBalances(rune.Id, 0, total)
			//common.Log.Infof("GetAllUtxoBalances %s took %v.", rune.Id, time.Since(startTime2))
			var amontInUtxos uint128.Uint128
			for _, balance := range utxos.Balances {
				amontInUtxos = amontInUtxos.Add(balance.Balance)
			}
			amt := common.NewDecimalFromUint128(amontInUtxos, int(rune.Divisibility))

			if amt.Cmp(holderAmount) != 0 {
				common.Log.Errorf("rune ticker %s amount in utoxs incorrect. %s %s", rune.Name, holderAmount.String(), amt.String())
				return false
			}
		}
		return true
	}

	checkAmount := func(address string, expectedmap map[string]string) bool {
		addressId, utxos := s.baseIndexer.GetUTXOsWithAddress(address)
		assets := s.GetAddressAssets(addressId, utxos)

		if len(assets) != len(expectedmap) {
			common.Log.Errorf("assets count different, have %d but expected %d", len(assets), len(expectedmap))
			return false
		}
		for r, b := range expectedmap {
			asset, ok := assets[r]
			if !ok {
				common.Log.Errorf("can't find rune %s", r)
				return false
			}
			amt := common.NewDecimalFromUint128(asset.Balance, int(asset.Divisibility))
			if amt.String() != b {
				common.Log.Errorf("rune %s amount %s incorrect. expected %s", r, asset.Balance.String(), b)
				return false
			}

			if !checkHolders(r) {
				common.Log.Errorf("rune %s checkHolders failed", r)
				return false
			}

			common.Log.Infof("runes %s checked.", r)
		}
		return true
	}	

	if s.chaincfgParam.Net == wire.MainNet && s.height == 919482 {
		expectedmap1 := map[string]string{
			// bc1p50n9sksy5gwe6fgrxxsqfcp6ndsfjhykjqef64m8067hfadd9efqrhpp9k DOG
			"USDT•TETHER":                      "1000000",
			"LOBO•THE•WOLF":                    "20000000",
			"WE•ALL•LOVE•PI":                   "62831853071795",
			"WZRD•BITCOIN":                     "19999",
			"BITCOIN•PENIS":                    "9999999.999",
			"BTC•NINJA•SATS":                   "1",
			"CZ•CZ•CZ•CZ•CZ•CZ":                "1999",
			"DOBERMAN•COIN":                    "44.444",
			"DOG•OF•BITCOIN":                   "2000",
			"FLOKI•INU•COIN":                   "3333333333.333",
			"GANGSTERS•CAT":                    "1000000",
			"HACHIKO•RUNES":                    "8888",
			"MOON•CAT•RUNES":                   "333.333",
			"ORDI•CATS•ARMY":                   "33.333",
			"ORDI•ETHEREUM":                    "10",
			"ORDI•GOLD•COIN":                   "750",
			"RESTAURANTES":                     "8",
			"SATOSHI•RUNES":                    "10",
			"SHIBA•FRACTAL":                    "4444444.444",
			"SHIBA•ON•OPNET":                   "1777",
			"SHIB•SHIBA•INU":                   "9999999.999",
			"STACKING•SATS":                    "2100",
			"TOTOMATO•MEME":                    "210000000",
			"AMERICAN•RUNES":                   "210000",
			"BABY•SHIBA•COIN":                  "555.555",
			"BITCOIN•ESTATE":                   "100",
			"BTC•TRUMP•RUNES":                  "888",
			"BTICH•GO•TO•MARS":                 "100000",
			"DOGECOIN•CHAIN":                   "8888888.888",
			"DOGE•TROLL•COIN":                  "1333.332",
			"MACKEREL•PACKS":                   "100000",
			"MOTOSWAP•RUNES":                   "100",
			"OFFICIAL•TRAMP":                   "1000",
			"ORDI•ALIEN•COIN":                  "22.222",
			"PAPA•RAZZI•MEME":                  "1200000",
			"RUNES•X•BITCOIN":                  "156544705",
			"SSSSS•BTC•SSSSS":                  "299.999",
			"BABY•DOGE•CRYPTO":                 "222.222",
			"BITCOIN•IN•GREEN":                 "2000000",
			"BITCOIN•RHODIUM":                  "100",
			"BITCOIN•SWAP•NET":                 "900",
			"DOG•FRACTAL•COIN":                 "2111.111",
			"DOG•GO•TO•THE•MOON":               "4320806228.54329",
			"JOKER•RUNES•COIN":                 "21000",
			"LIQUIDIUM•TOKEN":                  "53941.88",
			"NICOLAS•PI•RUNES":                 "1300",
			"NIKOLA•TESLA•GOD":                 "20",
			"OKX•NETWORK•SATS":                 "4444444.444",
			"PI•PROTOCOL•SATS":                 "39876543210.987",
			"SATOSHI•FRACTAL":                  "1188.888",
			"THE•BITCOIN•CHIP":                 "6900",
			"ANONYMOUS•PLANET":                 "10000000",
			"ANSEM•WIF•NO•HANDS":               "1000",
			"BULLX•BULLX•BULLX":                "1400",
			"DOGE•BILLIONAIRE":                 "1000000",
			"DOGS•DAO•MEMECOIN":                "1000000",
			"FIRST•CAT•ON•CHAIN":               "1000",
			"FISH•FRACTAL•COIN":                "8888.888",
			"GOLD•BITCOIN•COIN":                "111.111",
			"MOONVEMBER•TRUMP":                 "1000000",
			"PROTO•RUNES•STATE":                "6000",
			"RSIC•GENESIS•RUNE":                "888",
			"WUKONG•MAGIC•BOOK":                "1.111",
			"ZBIT•BLUE•BITCOIN":                "1",
			"BITCOIN•PALLADIUM":                "100",
			"BITCOINS•IN•CHARGE":               "500000",
			"BLACKGOO•BLACKGOO":                "99999",
			"EPIC•EPIC•EPIC•EPIC":              "2000",
			"FIRA•ROBOWORLD•CUP":               "8888888.888",
			"FRACTAL•BITCOIN•PI":               "26179938780",
			"MAGIC•EDEN•NETWORK":               "85555555.554",
			"ORDI•OXBT•SATS•RATS":              "1000000",
			"PIXEL•UNDERGROUND":                "5555",
			"WONDERLANDWABBIT":                 "25000",
			"DOGETOSHI•PEPEMOTO":               "1000000",
			"FOUR•HUNDRED•TWENTY":              "10",
			"FRACTAL•BITCOIN•MAP":              "4444444.444",
			"OKX•METAVERSE•RUNES":              "150",
			"OKX•NETWORK•BITCOIN":              "12",
			"SATOSHI•BITCOIN•SAT":              "1000000000",
			"THE•MILLIONARE•COIN":              "10",
			"UNISAT•NETWORK•DOGS":              "333.333",
			"UNISAT•NETWORK•SATS":              "42222.222",
			"BASED•INTERNET•PANDA":             "5000",
			"BITCOIN•FRACTAL•COIN":             "0.00001",
			"DOGE•FRACTAL•BITCOIN":             "515151.515",
			"FRACTAL•BITCOIN•SATS":             "11111.111",
			"BITCOIN•RUNESTONE•DOG":            "100000",
			"FRACTAL•BITCOIN•RUNES":            "900000",
			"FROM•BITCOIN•WITH•LOVE":           "888",
			"TROLL•TRUMP•PRESIDENT":            "1777777777.776",
			"FRACTAL•PROTOCOL•RUNES":           "444.444",
			"INTERGALACTIC•BITCOIN":            "1",
			"CATS•MEMECOIN•ON•BITCOIN":         "493547142.85715",
			"FRANKLIN•TEMPLETON•RUNES":         "7777777777.777",
			"FRACTAL•BITCOIN•TO•THE•MOON":      "10101010.101",
			"TWO•IN•THE•PINK•ONE•IN•THE•STINK": "743906250",
		}

		// bc1qvdnml9vy93twyhpz23dcdscey44d0w242zvfzg UG
		expectedmap2 := map[string]string{
			"KANGAROON":                    "12500000",
			"APE•SEASON•COIN":              "3000000",
			"BITCOIN•ROBOTS":               "150000000000",
			"DOPE•ASS•TICKER":              "240807000",
			"HYPER•AGI•AGENT":              "750000000",
			"THE•PROTO•RUNES":              "4000",
			"UNCOMMON•GOODS":               "2475686",
			"UNCOMMON•RUNES":               "1500000",
			"WARREN•BUFFETT":               "750000",
			"BTC•RUNES•UP•UP•UP":           "200",
			"DOG•LOOKING•DOWN":             "777",
			"GREED•FRAGMENTS":              "1846",
			"NICOLAS•PI•RUNES":             "300",
			"NON•STOP•NYAN•CAT":            "476700000000",
			"SATOSHI•IS•A•CHAD":            "750000",
			"SGN•SHO•GA•NAI•SGN":           "27500000",
			"TAPROOT•WIZORDS":              "2902000000",
			"THE•BILL•CLINTON":             "151515151",
			"THE•DOLAND•TREMP":             "1000",
			"DOGS•DAO•MEMECOIN":            "3472000000",
			"DOPE•ASS•RUNE•COIN":           "189000",
			"MR•MONOPOLY•COINS":            "1000",
			"REAL•DONALD•TRUMP":            "100000",
			"RSIC•GENESIS•RUNE":            "8880",
			"YOU•MAY•ETCH•MY•ASS":          "750000",
			"EPIC•EPIC•EPIC•EPIC":          "58500",
			"JESUS•IS•MY•SAVIOUR":          "750000",
			"QUANTUM•CAT•WIF•CAP":          "1050000000000",
			"RUNESTONE•WIZARDS":            "375000",
			"THE•CASEY•RODARMOR":           "75000000",
			"WONDERLANDWABBIT":             "25000",
			"BUSH•DID•NINE•ELEVEN":         "91100000",
			"FAKE•INTERNET•MONEY":          "2976000",
			"WHERE•LAMBO•GORILLA":          "1000000",
			"MAGA•THE•DONALD•TRUMP":        "1108",
			"AMAZING•PYRAMID•NUMBER":       "13614",
			"SHADOW•WIZARD•MONEY•GANG":     "3570",
			"DEGENERATE•ORDINALS•GAMBLERS": "42075000",
		}

		address1 := "bc1p50n9sksy5gwe6fgrxxsqfcp6ndsfjhykjqef64m8067hfadd9efqrhpp9k"
		if !checkAmount(address1, expectedmap1) {
			return false
		}
		common.Log.Infof("address %s checked!", address1)

		address2 := "bc1qvdnml9vy93twyhpz23dcdscey44d0w242zvfzg"
		if !checkAmount(address2, expectedmap2) {
			return false
		}
		common.Log.Infof("address %s checked!", address2)
	}

	if s.chaincfgParam.Net == wire.TestNet4 && s.height >= 74056 {
		expectedmap1 := map[string]string{
			// tb1p425q0pyngj5hcge7pu9krhpcu50p5hrpy93ajqt97a3xvzy0lzpsg9v22z
			// "BESTINSLOT•XYZ": "80643.79667061",
			// "BTC•PEPE•MATRIX": "13992.64702742",
			// "CAT•CAT•CUTE•UTE": "152147.49",
			// "NO•AMOUNT•RUNES": "2443",
			// "OPEN•YOUR•MOUTH": "195",
			// "STA•STA•TEST•ONE": "12951.51",
			// "STA•STA•TEST•TWO": "70435.6803",
			// "BITCOIN•TESTNET": "80156",
			// "CAT•CAT•CAT•HELLO": "1622.09",
			// "DOTSWAP•DOTSWAP": "10354",
			// "THIRD•IS•TEST•CAT": "1883.6376",
			// "THIRD•IS•TEST•DOG": "27874.45",
			// "XV•TEST•THIRD•TWO": "135794.2295",
			// "GOD•GOD•GOD•GOD•GOD": "24853.963",
			// "HAVE•HAVE•TESTSIA": "90258.09",
			// "HAVE•HAVE•TESTSIB": "10009.43",
			// "SHE•SHE•SHE•SHE•SHE": "97146",
			// "THIRD•IS•TEST•BIRD": "62533.612268",
			// "XV•TEST•THIED•FOUR": "928515.06",
			// "YKO•DDD•DDD•DDD•DDD": "3463.722",
			// "YKO•KKK•KKK•KKK•KKK": "3797.06607392735198461365",
			// "YKO•KOT•GGG•GGG•GGG": "729.21",
			// "THANKS•FOR•YOUR•HELP": "6532",
			// "THIRD•THIRD•TEST•TEO": "200522.39632",
			// "HAPPY•HAPPY•FAMILY•WU": "329886.03",
			// "SATRUN•GIVE•ME•CHANCE": "34",
			// "YKO•MARCH•BBBBBBBBBB": "1987",
			// "YKO•MATCH•LOGO•AAAAAAAA": "202934.48",
			// "JASON•HAPPY•HAPPY•ACE•ACE": "2357.8367",

			// tb1pa3usf65w59zu4g6m264kadzuj38atzwvmgrz3kkdrckt8eq6aexqrckesw
			"THE•BEST•RUNE":             "100",
			"ABSTRACT•PENGU":            "10",
			"A•CHILL•IGBO•GUY":          "4014690.361835",
			"ARCH•MEMECOINS":            "1118063.204118",
			"BADAITOKENSSS":             "131583676.373756",
			"BCVDSF•FEWQF•EE":           "3439175.555718",
			"BESTINSLOT•XYZ":            "72222.20424707",
			"BITCOIN•PIZZAS":            "130030643784792",
			"BTC•PEPE•MATRIX":           "1000",
			"BULLL•RUN•COINS":           "63130529.385838",
			"CONANGTOCNGAN":             "4147728.554485",
			"DOGE•MEME•RUNES":           "28000",
			"HOT•SEX•JIIIIII":           "1708031.262792",
			"LUCKY•CAT•LUCKY":           "3000",
			"MEMO•SMALL•FISH":           "59990.828322",
			"MYFUNNYDOGSSS":             "10",
			"NIPHERMEDAVEE":             "124978015.93439",
			"OCUMPYZXUZBUK":             "846677880979761810.3",
			"PIZZA•NINJA•CAT":           "0.00325",
			"REAL•TRUMP•COIN":           "75955.557441",
			"RRRQQQEEE•SSSS":            "183.41",
			"RUNE•X•BTC•X•RUNE":         "0.00448",
			"SMARTMONEYDAY":             "4681769.245579",
			"TEST•RUNES•TEST":           "1236",
			"TIMISLATTTWTF":             "950",
			"AKANMU•MEMECOIN":           "7889926.299141",
			"BITCOIN•TESTNET":           "592464960349789",
			"COIN•APACHE•COIN":          "14535720.358472",
			"FLOWEBRS•BTC•WEB":          "68384.899373",
			"JENNIELOVELOVE":            "5763426.908152",
			"MARK•GOONER•BERG":          "36625406.269916",
			"MMMJJJLLL•TESTA":           "1394.16",
			"ORDINAL•MAXI•BIZ":          "85994586.88366",
			"PEPE•GOOD•LOVELY":          "52199.705069",
			"PUPS•WORLD•PEACE":          "6408413.483880729890266286",
			"TESTNETRNES•BBB":           "619.51",
			"TESTNETRNES•CCC":           "2198",
			"THE•DONALD•TRUMP":          "83867462.419347",
			"TWT•COIN•HUPPPPP":          "58851825.458757",
			"YONNNAAAYONNAA":            "111530.340927",
			"ZEENCRYPTO•COIN":           "16742755.007761",
			"ARCH•NETWORK•COIN":         "23",
			"ARSENAL•BEST•TEAM":         "1000",
			"BASED•ANGELS•RUNE":         "104962.4",
			"BULL•RUN•FUN•TOKEN":        "10760440.26101",
			"CATTLEYCATCATTY":           "1294118.810031",
			"CHACHING•BTC•BANK":         "32930",
			"CHIBI•LOVELY•GIRL":         "843115.427765",
			"CHIMERA•PROTOCOL":          "500000",
			"DONALD•TRUMP•MAGA":         "99000",
			"FARMKRU•MEMECOIN":          "3033157.73444",
			"FIRST•SMART•NIGGA":         "500",
			"GDGDGDGDGDGDFGD":           "24215",
			"KOLOBOK•MEMECOIN":          "3868091.083352",
			"LIL•BABY•MEME•COIN":        "15388.684779",
			"LOBO•THE•WOLF•RUNE":        "78405",
			"MAGIC•AGENT•MONEY":         "20001",
			"MELANIA•OFFICIAL":          "21111444.766182",
			"SATURN•TEST•TOKEN":         "554194",
			"TESTNET•RUNE•BUSD":         "300412.656813",
			"THEGREATGANOFUI":           "22910642.778334",
			"THIS•IS•FIRST•RUNE":        "2915.7",
			"THIS•IS•THIRD•RUNE":        "5247.2",
			"TRUE•OGFUNYKYBIT":          "2859271.341631",
			"TRUMP•AND•MELANIA":         "12",
			"BILLION•DOLLAR•CAT":        "2608356",
			"BTCS•BTCS•BTCS•BTCS":       "86373909",
			"BURNING•LIQUIDITY":         "1",
			"DSOTSWAPTEST•TEST":         "1492",
			"DUOLINGO•MEMECOIN":         "77904823.450582",
			"EVERY•THING•NA•TIME":       "71720.205681",
			"GOD•OVER•ALL•POWERS":       "14602900.820283",
			"NIGGA•COIN•FOR•IYOU":       "439100.72662",
			"PAINTBALL•FOREVER":         "3342423",
			"STAR•OF•THE•SEA•COIN":      "1",
			"THIS•IS•SECOND•RUNE":       "1766.2",
			"FUNKY•BIT•CASINO•TWO":      "48123.562002",
			"GRASSOFFICIALCOIN":         "2.569859",
			"MULTICOLORED•STONE":        "20000",
			"TETETETETETETTCCC":         "1213",
			"FRACTAL•TESTNET•FOUR":      "145487125621",
			"FUNKYBIT•BITCOIN•DOG":      "3661487.034886",
			"MEME•BACKED•CURRENCY":      "1",
			"NYARURIAITHE•FUTURE":       "33527990.901646",
			"SATOSHI•NAKAMOTO•HAT":      "41552859.314773",
			"YOU•CANT•STOP•FREEDOM":     "1233677.41367248",
			"ZEOS•WANTS•TRUMP•DOWN":     "236293863.587324",
			"ARCH•FORSWAP•MEMECOIN":     "43387.041771",
			"ELITECHADONLYFORLORD":      "2315021.88401",
			"OKAYYYY•OKAYYYY•OKAYYYY":   "30.09",
			"STRATEGIC•BITCOIN•RESERVE": "500",
		}

		expectedmap2 := map[string]string{
			// tb1p9hz7n8w66hzgyn5yaefunm6ah2cqxv8dfyvg0mt26wsv345g4c5sfw3w7r
			"BIMA•USBD•BOOM":                   "10000",
			"ADURAGBEMMMMY":                    "7536550.777762",
			"BBW•ART•PEGED•VB":                 "120083.718079",
			"BERA•TO•THE•MOON":                 "14159708.466192",
			"BESTINSLOT•XYZ":                   "257030.71004138",
			"BITCOIN•PIZZAS":                   "286522261380320",
			"BTC•PEPE•MATRIX":                  "893971.03021618",
			"DOGE•MEME•RUNES":                  "192425",
			"DRACORAYYYYYY":                    "825000",
			"ELON•TRUMP•COIN":                  "10574352.231961",
			"JUSTBULLIOOON":                    "500",
			"MELANIATRUUMP":                    "6821916.177824",
			"MEMEBTCGSDOIT":                    "114216.281003",
			"MONAD•OFFICIAL":                   "10000",
			"OCUMPYZXUZBUK":                    "242525025613467397",
			"OFFICIAL•CRASH":                   "100000",
			"PIZZA•NINJA•CAT":                  "46285.480124",
			"PRINT•THE•MONEY":                  "147",
			"RUNE•X•BTC•X•RUNE":                "43175.4076031",
			"SOSOSOSOSCATT":                    "6146533.701417",
			"TEST•RUNES•TEST":                  "5319",
			"BAILEYOQ•OGO•OMO":                 "4810113.300165",
			"BITCOIN•TESTNET":                  "2413604042942496",
			"BURNA•BOY•ODOGWU":                 "200",
			"CRYPTOWORLDSSS":                   "1000",
			"GCHO•VHJ•HGCKVGH":                 "106942950.826744",
			"HEMAN•TO•THE•MOON":                "414826",
			"ICE•AGE•NUT•SCRAT":                "8098595.110416",
			"LILILITESTAAAA":                   "165.15",
			"MOON•ALIEN•TOKEN":                 "74093.194272",
			"MYMUMISTHEBEST":                   "1",
			"NISHANTRAJPOOT":                   "972330.373758",
			"NULESMALLZCOIN":                   "150000000",
			"OLAJUWONGEORGE":                   "25878267.13472",
			"PUPS•WORLD•PEACE":                 "2543075.933021217080210508",
			"RAHUL•KUMAR•JAIN":                 "1224818.850663",
			"SI•TITID•KOINMUA":                 "16705.02989",
			"TESTNETRNES•BBB":                  "283.22",
			"TESTNETRNES•CCC":                  "1179",
			"THE•FUNKYBIT•DOG":                 "3910648.293622",
			"BASED•ANGELS•RUNE":                "208000",
			"BERAARCHNETWORK":                  "16632964.08638",
			"BIG•MONEY•AIRDROP":                "2779.389303",
			"BTC•PUNK•MEMECOIN":                "16735170.919915",
			"CHACHING•BTC•BANK":                "1451543",
			"CHIMERA•PROTOCOL":                 "333865",
			"CLOWN•CLOWN•CLOWN":                "5238864.855103",
			"DEVIL•KING•TRENER":                "3967257.785371",
			"FOX•UTRFUTFUYJHF":                 "655381.316812",
			"GDGDGDGDGDGDFGD":                  "5183",
			"HAVE•HAVE•TESTSIA":                "10307.64",
			"HAVE•HAVE•TESTSIC":                "8377024.19",
			"JFMJFMJFMHHHAAA":                  "0.44",
			"LOBO•THE•WOLF•RUNE":               "120721.31",
			"MAGIC•AGENT•MONEY":                "555345.288",
			"OFFICIAL•ZACHXBT":                 "82014.108046",
			"PIGGY•PIGGY•PIGGY":                "22078895.823769",
			"SHE•SHE•SHE•SHE•SHE":              "10647",
			"SOLANA•ON•BITCOIN":                "310000",
			"TESTNET•RUNE•BUSD":                "94359.437791",
			"THIS•IS•FIRST•RUNE":               "2984.8",
			"THIS•IS•THIRD•RUNE":               "3775.4",
			"WOLRDWOLFAPTAPT":                  "100000",
			"BILLION•DOLLAR•CAT":               "644573",
			"BITCOIN•CAT•BITCAT":               "1456375.383132",
			"BTCS•BTCS•BTCS•BTCS":              "17307068",
			"CAT•CAT•CAT•CAT•CATA":             "10680430.954335",
			"DSOTSWAPTEST•TEST":                "1293",
			"DUOLINGO•MEMECOIN":                "371946.534984",
			"OFFICIAL•CREO•COIN":               "100182806.093868",
			"OFFICIAL•ELONMUSK":                "50000",
			"ORDINALS•BANK•BANK":               "17408.11264",
			"SUBMITMERIGHTNOW":                 "5781271.329454",
			"THIS•IS•SECOND•RUNE":              "1678.2",
			"GORILLA•MOVERZ•LABS":              "14108.384222",
			"OZIRTSANIYEVSVETY":                "12630619.074722",
			"PIZZA•PET•ON•BITCOIN":             "5392097.479065",
			"PUDGY•PENGUINS•MEME":              "98737625.647675",
			"TETETETETETETTCCC":                "331",
			"VIKINGS•IN•THE•HOUSE":             "400000",
			"FRACTAL•TESTNET•FOUR":             "59644416923",
			"KENECHUKWUTHEGREAT":               "16277.954755",
			"MEME•BACKED•CURRENCY":             "148831.9844",
			"SONICPEPEGRINPOWER":               "166928.030751",
			"YOU•CANT•STOP•FREEDOM":            "1474646.19180866",
			"SMATOKENDEVELOPMENT":              "24898.981422",
			"TRUMP•ELON•VANCE•CHARLIE":         "134454.705701",
			"COSMOCOSMOS•COSMO•COSMOS":         "9978215.8148",
			"DO•NOT•TELL•ME•SO•MUCH•THING":     "215",
			"DOG•GO•TO•THE•MOON•SATOSHINET•TE": "51215",
		}

		// address := "tb1pa3usf65w59zu4g6m264kadzuj38atzwvmgrz3kkdrckt8eq6aexqrckesw" // 70499
		//address := "tb1p9hz7n8w66hzgyn5yaefunm6ah2cqxv8dfyvg0mt26wsv345g4c5sfw3w7r" // 74056
		//address := "tb1p425q0pyngj5hcge7pu9krhpcu50p5hrpy93ajqt97a3xvzy0lzpsg9v22z" // 104225

		address1 := "tb1pa3usf65w59zu4g6m264kadzuj38atzwvmgrz3kkdrckt8eq6aexqrckesw" // 70499
		if !checkAmount(address1, expectedmap1) {
			return false
		}
		common.Log.Infof("address %s checked!", address1)

		address2 := "tb1p9hz7n8w66hzgyn5yaefunm6ah2cqxv8dfyvg0mt26wsv345g4c5sfw3w7r" // 74056
		if !checkAmount(address2, expectedmap2) {
			return false
		}
		common.Log.Infof("address %s checked!", address2)
	}

	// 下面这个方式极慢，需要参考nft模块的方案 TODO
	// check all runes minted amount
	allRunes := s.GetAllRuneInfos()
	common.Log.Infof("total runes: %d", len(allRunes))
	// startTime := time.Now()
	// for _, rune := range allRunes {
	// 	if !checkHolders(rune.Name) {
	// 		common.Log.Errorf("rune %s checkHolders failed", rune.Name)
	// 		return false
	// 	}
	// }
	// common.Log.Infof("rune check amount took %v.", time.Since(startTime))

	common.Log.Infof("runes checked.")
	return true
}
