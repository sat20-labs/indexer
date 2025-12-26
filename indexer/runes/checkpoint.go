package runes

import (
	"time"

	"github.com/sat20-labs/indexer/common"
)

type CheckPoint struct {
	Height      int
	TickerCount int
	Tickers     map[string]*TickerStatus
}

type TickerStatus struct {
	Name        string
	DeployHeight int
	Max         string
	Minted      string
	MintCount   int64
	HolderCount int
	Holders     map[string]string
}

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
				HolderCount: 250903,
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
				HolderCount: 90868, // 90867
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

}

func (p *Indexer) CheckPointWithBlockHeight(height int) {

	startTime := time.Now()
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
		tickers := p.GetAllRuneIds()
		if checkpoint.TickerCount != 0 && len(tickers) != checkpoint.TickerCount {
			common.Log.Panicf("ticker count different")
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
		if tickerStatus.Max != "" && ticker.Max().String() != tickerStatus.Max {
			common.Log.Panicf("%s Max different, %s %s", name, ticker.MaxSupply.String(), tickerStatus.Max)
		}
		if tickerStatus.Minted != "" && ticker.TotalMinted().String() != tickerStatus.Minted {
			common.Log.Panicf("%s Minted different, %s %s", name, ticker.TotalMinted().String(), tickerStatus.Minted)
		}

		if tickerStatus.MintCount != 0 {
			if ticker.MintInfo != nil {
				mintCount := ticker.MintInfo.Mints.Big().Int64()
				if mintCount != tickerStatus.MintCount {
					common.Log.Panicf("%s MinteMintCountd different, %d %d", name, mintCount, tickerStatus.MintCount)
				}
			}
		}

		if tickerStatus.HolderCount != 0 {
			if ticker.HolderCount != uint64(tickerStatus.HolderCount) {
				common.Log.Panicf("%s HolderCount different, %d %d", name, ticker.HolderCount, tickerStatus.HolderCount)
			}
		}

		for address, amt := range tickerStatus.Holders {
			addressId := baseIndexer.GetAddressIdFromDB(address)
			if addressId == common.INVALID_ID {
				common.Log.Panicf("%s GetAddressIdFromDB %s failed", name, address)
			}
			d := p.GetAddressAssetWithName(addressId, name)
			if d.String() != amt {
				common.Log.Panicf("%s holder %s amt different, %s %s", name, address, d, amt)
			}
		}

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
	common.Log.Infof("Indexer.CheckPointWithBlockHeight %d checked, takes %v", height, time.Since(startTime))
}

func (s *Indexer) printHistoryWithAddress(name string, addressId uint64) {
}
