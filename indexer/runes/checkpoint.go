package runes

import (
	"time"

	"github.com/sat20-labs/indexer/common"
	indexerCommon "github.com/sat20-labs/indexer/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/validate"
)

// TODO 编译数据时将开关打开，可以对brc20的数据进行校验
var _enable_checking_more_files = false

type CheckPoint struct {
	Height      int
	TickerCount uint64
	CheckHolder bool
	Tickers     map[string]*TickerStatus
}

type TickerStatus struct {
	Number      uint64
	Name        string
	DeployHeight int
	Max         string
	Minted      string
	MintCount   int64
	HolderCount int
	Holders     map[string]string
}

var _holderStartHeight, _holderEndHeight int
var _heightToHolderRecords map[int]map[string]map[string]*validate.HolderCSVRecord


var testnet4_checkpoint = map[int]*CheckPoint{
	0: {
		Tickers: map[string]*TickerStatus{
			"BESTINSLOT•XYZ": {DeployHeight: 30562}, // 每个区块，如果没有其他检查，就默认检查该资产的holder和minted是否匹配
		},
	},

	50000: {
		Tickers: map[string]*TickerStatus{
			"BESTINSLOT•XYZ": {
				Minted:      "16",
				HolderCount: 4,
				Holders: map[string]string{
					"tb1pudsspvxgvclznfu5lkxezexvta48pgnu407gw4fce0t9yawaqm6s39ycrz": "10",
					"tb1p9ts4eu2s4adgjwumdmcu9qfw0hcavrh8m54tyrd39lkk7h4940yq2dmvzx": "4",
					"tb1pn9dzakm6egrv90c9gsgs63axvmn6ydwemrpuwljnmz9qdk38ueqsqae936": "1",
					"tb1qvhl8k0xu0stk956tjqx6q5ujp6tdyh24xcz0qq": "1",
				},
			},
		},
	},

	106881: {
		TickerCount: 23573,
	},

	114881: {
		Tickers: map[string]*TickerStatus{
			"BESTINSLOT•XYZ": {
				Minted:      "2914637",
				MintCount:   2914636,
				HolderCount: 53059,
				Holders: map[string]string{
					"tb1p3xqsaj90q9y62m659n2803u4u7p27f40vma56ln69t34vv8rydjqafz2kp": "832271.65893581",
					"tb1pf0tcpgxr30kqhh2gn3pgvr4qyachzm0xxydnfxth5cxlkjl37ugspxp7tm": "381899.23634302",
					"tb1pzn4sw97wmlagctjtr3jktgqjaksgugc9ja7vhlawj89u0p098s9qafsrql": "370144.33818536",
					"tb1p9hz7n8w66hzgyn5yaefunm6ah2cqxv8dfyvg0mt26wsv345g4c5sfw3w7r": "257030.71004138",
					"tb1q0qsga3frh6ktux6c7g9n9yzsqgkzqac5spfutm":                     "388",
				},
			},
		},
	},
}

var mainnet_checkpoint = map[int]*CheckPoint{
	0: {
		Tickers: map[string]*TickerStatus{
			"UNCOMMON•GOODS": {DeployHeight: 1}, // 每个区块，如果没有其他检查，就默认检查该资产的holder和minted是否匹配
			"DOG•GO•TO•THE•MOON": {DeployHeight: 840000},

		},
	},

	929124: {
		Tickers: map[string]*TickerStatus{
			"UNCOMMON•GOODS": {
				Minted:      "69032629",
				MintCount:   69032629,
				HolderCount: 250904, // 250903 TODO 
				Holders: map[string]string{
					"bc1qvdnml9vy93twyhpz23dcdscey44d0w242zvfzg": "1608512",
					"bc1q62gk9dawqz3ne2wksyc72zwpw85t4na3p92n9w": "1584152",
					"bc1ptg6thewzpqqvchy9eu7462ah4829yptyqy600nxdljmvkhtgu53s4gjynw": "1575978",
					"bc1pzkyqtdj3atxtym57ud44wugtvwzkrufl5hazmw9u2kej2n7t92qs0237ey": "1479968",
				},
			},

			"DOG•GO•TO•THE•MOON": {
				Minted:      "100000000000",
				MintCount:   0,
				HolderCount: 90868, // 90867 // TODO 
				Holders: map[string]string{
					"bc1p50n9sksy5gwe6fgrxxsqfcp6ndsfjhykjqef64m8067hfadd9efqrhpp9k": "3957388135.36401",
					"bc1pk8g4rztfkxs2q9c40g6keeknjw6aadx3kzu4suzlll0remfw7xxs5x9ctv": "3756442080.45356",
					"bc1qj7dam98j6ktjcp320qu77y2vrylv49c2k2hkmu": "2399737912.52754",
					"3G7gSaxPY7BhbEASd2pnZY5cg7uEQMQvd8": "1946635349.37447",
					"bc1p38d6mfutw5h6gx46c7334uxtsf5ey5l7xqfeg36gyc4q83plmwwqsf9wxd": "1816300050.097",
				},
			},
		},
	},

	841000: {
		TickerCount: 20805,
		Tickers: map[string]*TickerStatus{
			"PUNK•TWO•EIGH•NINE•THREE": {
				Number: 	 20245,
			},
			"MERLINSTONE•THE•CODE•IS•LAW": {
				Number: 	 20804,
			},
		},
	},
	841500: {
		TickerCount: 63401,
		Tickers: map[string]*TickerStatus{
			"BIRD•ONE•ZERO•FIVE•EIGHT": {
				Number: 	 63384,
			},
			"BIRD•SEVEN•ONE•SIX•THREE": {
				Number: 	 63400,
			},
		},
	},
	845000: {
		TickerCount: 70922,
		Tickers: map[string]*TickerStatus{
			"RATS•IN•THE•SEWER": {
				Number: 	 70921,
			},
		},
	},
	850000: {
		TickerCount: 82163,
		Tickers: map[string]*TickerStatus{
			"BLOCKS•OF•BITCOIN": {
				Number: 	 82135,
			},
			"OCTOGLYPH•RARAMIPA": {
				Number: 	 82162,
			},
		},
	},
	860000: {
		TickerCount: 82163,
		Tickers: map[string]*TickerStatus{
			"BLOCKS•OF•BITCOIN": {
				Number: 	 82135,
			},
			"OCTOGLYPH•RARAMIPA": {
				Number: 	 82162,
			},
		},
	},
	870002: {
		TickerCount: 143418,
		Tickers: map[string]*TickerStatus{
			"BITCOIN•BRO•BEARS": {
				Number: 	 143417,
			},
		},
	},
	880000: {
		TickerCount: 168568,
		Tickers: map[string]*TickerStatus{
			"GEGTHRQYNLBW": {
				Number: 	 168567,
			},
		},
	},
	890009: {
		TickerCount: 176535,
		Tickers: map[string]*TickerStatus{
			"MUPPET•COIN": {
				Number: 	 176534,
			},
		},
	},
	900000: {
		TickerCount: 178862,
		Tickers: map[string]*TickerStatus{
			"NIKOLA•TESLA•GODS": {
				Number: 	 178861,
			},
		},
	},
	910000: {
		TickerCount: 181633,
		Tickers: map[string]*TickerStatus{
			"AAAAAAAAAAAAAAABBRWCDRDAGNG": {
				Number: 	 181632,
			},
		},
	},
	920019: {
		TickerCount: 208203,
		Tickers: map[string]*TickerStatus{
			"VIKING•ID•CSZH•ODIN": {
				Number: 	 208202,
			},
		},
	},
	930003: {
		TickerCount: 208203,
		Tickers: map[string]*TickerStatus{
			"VIKING•ID•CSZH•ODIN": {
				Number: 	 208202,
			},
		},
	},
	931977: {
		TickerCount: 209424,
		Tickers: map[string]*TickerStatus{
			"LUNAR•RECORDS•FUND": {
				Number: 	 209423,
			},
		},
	},

}

func (p *Indexer) CheckPointWithBlockHeight(height int) {

	startTime := time.Now()
	p.validateHolderData(height)

	var checkpoint *CheckPoint
	matchHeight := height
	isMainnet := p.baseIndexer.IsMainnet()
	if isMainnet {
		checkpoint = mainnet_checkpoint[height]
		if checkpoint == nil {
			matchHeight = 0
			checkpoint = mainnet_checkpoint[0]
		}
	} else {
		checkpoint = testnet4_checkpoint[height]
		if checkpoint == nil {
			matchHeight = 0
			checkpoint = testnet4_checkpoint[0]
		}
	}
	if checkpoint == nil {
		return
	}

	if matchHeight != 0 {
		if checkpoint.TickerCount != 0 {
			if p.Status.Number != checkpoint.TickerCount {
				common.Log.Panicf("ticker count different, %d %d", checkpoint.TickerCount, p.Status.Number)
			}
		}
	}
	
	baseIndexer := p.baseIndexer
	for name, tickerStatus := range checkpoint.Tickers {
		if tickerStatus.DeployHeight != 0 {
			if height < tickerStatus.DeployHeight {
				continue
			}
		}
	
		ticker := p.GetRuneInfo(name)
		if ticker == nil {
			common.Log.Panicf("CheckPointWithBlockHeight can't find ticker %s", name)
		}
		if tickerStatus.Number != 0 && ticker.Number != tickerStatus.Number {
			common.Log.Panicf("%s number different, %d %d", name, tickerStatus.Number, ticker.Number)
		}
		if tickerStatus.Max != "" && ticker.Max().String() != tickerStatus.Max {
			common.Log.Panicf("%s Max different, %s %s", name, tickerStatus.Max, ticker.MaxSupply.String())
		}
		if tickerStatus.Minted != "" && ticker.TotalMinted().String() != tickerStatus.Minted {
			common.Log.Panicf("%s Minted different, %s %s", name, tickerStatus.Minted, ticker.TotalMinted().String())
		}

		if tickerStatus.MintCount != 0 {
			if ticker.MintInfo != nil {
				mintCount := ticker.MintInfo.Mints.Big().Int64()
				if mintCount != tickerStatus.MintCount {
					common.Log.Panicf("%s MinteMintCountd different, %d %d", name, tickerStatus.MintCount, mintCount)
				}
			}
		}

		if tickerStatus.HolderCount != 0 {
			if ticker.HolderCount != uint64(tickerStatus.HolderCount) {
				common.Log.Panicf("%s HolderCount different, %d %d", name, tickerStatus.HolderCount, ticker.HolderCount)
			}
		}

		for address, amt := range tickerStatus.Holders {
			addressId := baseIndexer.GetAddressIdFromDB(address)
			if addressId == common.INVALID_ID {
				common.Log.Panicf("%s GetAddressIdFromDB %s failed", name, address)
			}
			d := p.GetAddressAssetWithName(addressId, name)
			if d.String() != amt {
				common.Log.Panicf("%s holder %s amt different, %s %s", name, address, amt, d)
			}
		}

		if checkpoint.CheckHolder {
			holdermap := p.GetHoldersWithTick(name)
			var holderAmount *common.Decimal
			for _, amt := range holdermap {
				holderAmount = holderAmount.Add(amt)
			}
			if holderAmount.String() != ticker.TotalHolderAmt().String() {
				common.Log.Infof("block %d, ticker %s, asset amount different %s %s",
					height, name, ticker.TotalHolderAmt().String(), holderAmount)

				// printAddress := make(map[uint64]bool)
				// for k, v := range holdermap {
				// 	old, ok := p.holderMapInPrevBlock[k]
				// 	if ok {
				// 		if old != v {
				// 			common.Log.Infof("%x changed %s -> %s", k, old.String(), v.String())
				// 			printAddress[k] = true
				// 		}
				// 	} else {
				// 		common.Log.Infof("%x added %d -> %d", k, old, v)
				// 		printAddress[k] = true
				// 	}
				// }
				// for k := range printAddress {
				// 	p.printHistoryWithAddress(name, k)
				// }

				//p.printHistory(name)
				//p.printHistoryWithAddress(name, 0x52b1777c)
				common.Log.Panicf("%s holders amount different %s %s", name, ticker.TotalHolderAmt().String(), holderAmount)
			}
			p.holderMapInPrevBlock = holdermap
		}
	}
	common.Log.Infof("Indexer.CheckPointWithBlockHeight %d checked, takes %v", height, time.Since(startTime))
}


func readHolderDataToMap(dir string) (int, int) {
	validateHolderData, start, end, err := validate.ReadCSVDir(dir)
	if err != nil {
		common.Log.Panicf("ReadHolderCSVDir %s failed, %v", dir, err)
	}

	_heightToHolderRecords = make(map[int]map[string]map[string]*validate.HolderCSVRecord)
	for _, record := range validateHolderData {
		tickerToHolders, ok := _heightToHolderRecords[record.Height]
		if !ok {
			tickerToHolders = make(map[string]map[string]*validate.HolderCSVRecord)
			_heightToHolderRecords[record.Height] = tickerToHolders
		}
		holders, ok := tickerToHolders[record.Ticker]
		if !ok {
			holders = make(map[string]*validate.HolderCSVRecord)
			tickerToHolders[record.Ticker] = holders
		}
		holders[record.Address] = record
	}
	common.Log.Infof("readHolderDataToMap height %d %d, records %d", start, end, len(validateHolderData))
	return start, end
}

func addFailedItem(failed map[string]map[string]*common.Decimal, ticker, address string, amt *common.Decimal) {
	holders, ok := failed[ticker]
	if !ok {
		holders = make(map[string]*common.Decimal)
		failed[ticker] = holders
	}
	holders[address] = amt
}

// 逐个区块对比某个brc20 ticker的相关事件，效率很低，只适合开发阶段做数据的校验，后续要关闭该校验
func (p *Indexer) validateHolderData(height int) {
	
	if _heightToHolderRecords == nil {
		if _enable_checking_more_files {
			_holderStartHeight, _holderEndHeight = readHolderDataToMap("./indexer/runes/validate/holders")
		}
	}
	if len(_heightToHolderRecords) == 0 {
		return
	}
	if height < _holderStartHeight || height > _holderEndHeight {
		return
	}

	tickerToHolders, ok := _heightToHolderRecords[height]
	if !ok {
		return
	}

	// 执行验证
	baseIndexer := p.baseIndexer
	failed := make(map[string]map[string]*common.Decimal)
	for ticker, holders := range tickerToHolders {
		verified := true
		for address, record := range holders {
			addressId := baseIndexer.GetAddressIdFromDB(address)
			if addressId == common.INVALID_ID {
				common.Log.Errorf("validateHolderData GetAddressIdFromDB %s failed", address)
				addFailedItem(failed, ticker, address, nil)
				verified = false
				continue
			}
			info := p.GetAddressAssetWithName(addressId, record.Ticker)
			if info == nil {
				// p.printTicker(ticker)
				// p.printHolders(ticker)
				// p.printHistoryWithAddress(ticker, addressId)
				common.Log.Errorf("validateHolderData getHolderAbbrInfo %s %s failed", address, record.Ticker)
				addFailedItem(failed, ticker, address, nil)
				verified = false
				continue
			}
			if info.String() != record.Balance {
				// record.balance是不考虑精度的整数
				if !indexerCommon.CompareForRunes(info.Value, record.Balance) {
					//p.printHistoryWithAddress(ticker, addressId)
					common.Log.Errorf("validateHolderData %s %s available balance different %s %s",
						address, record.Ticker, record.Balance, info.Value.String())
					addFailedItem(failed, ticker, address, info)
					verified = false
					continue
				}
			}
		}
		if verified {
			common.Log.Infof("runes validateHolderData %s %d check succeeded.", ticker, len(holders))
		} else {
			common.Log.Infof("runes validateHolderData %s check failed.", ticker)
		}
	}

	if len(failed) > 0 {
		common.Log.Panicf("check %v holders failed", failed)
	}
}
