package exotic

import (
	"strings"
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
	Max         int64
	Minted      int64
	MintCount   int64
	HolderCount int
	Holders     map[string]int64
}

var testnet4_checkpoint = map[int]*CheckPoint{
	0: {
		Tickers: map[string]*TickerStatus{
			Pizza: {DeployHeight: PIZZA_HEIGHT},
			Uncommon: {DeployHeight: 0}, 
			Rare: {DeployHeight: 0}, 
			Epic: {DeployHeight: 0}, 
			Block9: {DeployHeight: 9},
			Block78: {DeployHeight: 78},
			Nakamoto: {DeployHeight: 9},
		},
	},

	10000: {
		Tickers: map[string]*TickerStatus{
			Uncommon: {
				Minted: 9996,
				HolderCount: 43,
				Holders: map[string]int64{
					"tb1q548z58kqvwyjqwy8vc2ntmg33d7s2wyfv7ukq4": 3947,
					"mjP97q5BWtdpdsJLkEJvQWgLe9zw4MMVU6": 2508,
					"mjcNxNEUrMs29U3wSdd7UZ54KGweZAehn6": 2491,
					"tb1qmzwkn0gp0lec9233pf0ymft8k6a2qsx3ymfe6e": 361,
					"tb1qnjkdfejgc2t9q24qp7d2ssmfwjmsdfyx4fvpst": 187,
					"tb1qujqjh59haxnu2uf6d6f6e7gla3fn8phkak8mxj": 122,
					"tb1q2z4gwfdv6jzsh2mjt8ydw4cdxps4lcjy37zvay": 81,
					"mgTgHVFXFdMEJiMmLhGrxu75waDYjCjDvN": 73,
					"tb1qg8zlznrvns9u46muxamxjh7sa8wry3vutzaujm": 64,
					"tb1qmf7xdqc5nvzhturuzc46qtq5kywdf3p76cpq53": 26,
					"tb1qhk95yuyzkv67v0d73u68vr7ug5684shqzeqnlm": 25,
				},
			},
			Rare: {
				Minted: 4,
				HolderCount: 3,
				Holders: map[string]int64{
					"mjP97q5BWtdpdsJLkEJvQWgLe9zw4MMVU6": 2,
					"mjcNxNEUrMs29U3wSdd7UZ54KGweZAehn6": 1,
					"tb1q548z58kqvwyjqwy8vc2ntmg33d7s2wyfv7ukq4": 1,
				},
			},
			FirstTransaction: {
				Minted: 2814999961782,
				HolderCount: 9,
				Holders: map[string]int64{
					"tb1qmzwkn0gp0lec9233pf0ymft8k6a2qsx3ymfe6e": 1804999960099,
					"tb1qnjkdfejgc2t9q24qp7d2ssmfwjmsdfyx4fvpst": 935699899400,
					"tb1pvsxkcr6qsl5pmehg9hcfgd0mn4rz3xva8rqjf64wejv0zewg3xcqvcg25n": 64299899785,
					"tb1qx4csx5gzsnev56ayzc4tyayyk307n56apqfjsh": 10000000000,
					"tb1pkm89jv7x3qntkfsluuc0f7983w9flzycu888jntxf2ls49y543vsncnats": 100000,
					"tb1qe2wz6lyu0qan7apxljr44ltvlrvddsywu27s7e": 89580,
					"tb1q8p9nvh892fau0ln2lz4wpfnh04ewnyxsnv76c4": 10000,
					"tb1q548z58kqvwyjqwy8vc2ntmg33d7s2wyfv7ukq4": 2703,
					"mjP97q5BWtdpdsJLkEJvQWgLe9zw4MMVU6": 215,
				},
			},
			Vintage: {
				Minted: 5005000000000,
				HolderCount: 13,
				Holders: map[string]int64{
					"tb1qmzwkn0gp0lec9233pf0ymft8k6a2qsx3ymfe6e": 1804999960099,
					"tb1q548z58kqvwyjqwy8vc2ntmg33d7s2wyfv7ukq4": 1800000240921,
					"tb1qnjkdfejgc2t9q24qp7d2ssmfwjmsdfyx4fvpst": 935699899400,
					"mgTgHVFXFdMEJiMmLhGrxu75waDYjCjDvN": 365000000000,
					"tb1pvsxkcr6qsl5pmehg9hcfgd0mn4rz3xva8rqjf64wejv0zewg3xcqvcg25n": 64299899785,
					"tb1q32ct2uc2n5x44uq73cprqlf3qrp04r63rz766j": 14999800000,
					"tb1qx4csx5gzsnev56ayzc4tyayyk307n56apqfjsh": 10000000000,
					"tb1qhw67jcsumk7899260up2at86q5xza0frvzrp49": 5000000000,
					"IQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAKw=": 5000000000,
					"tb1pkm89jv7x3qntkfsluuc0f7983w9flzycu888jntxf2ls49y543vsncnats": 100000,
					"tb1qe2wz6lyu0qan7apxljr44ltvlrvddsywu27s7e": 89580,
				},
			},
		},
	},

	100000: {
		Tickers: map[string]*TickerStatus{
			"RarePizza": {
				Minted:      997000, 
			},
		},
	},

}

var mainnet_checkpoint = map[int]*CheckPoint{
	0: {
		Tickers: map[string]*TickerStatus{
			Pizza: {DeployHeight: PIZZA_HEIGHT},
			Uncommon: {DeployHeight: 0}, 
			Rare: {DeployHeight: 0}, 
			Epic: {DeployHeight: 0}, 
			Block9: {DeployHeight: 9},
			Block78: {DeployHeight: 78},
			Nakamoto: {DeployHeight: 9},
		},
	},

	10000: {
		Tickers: map[string]*TickerStatus{
			Uncommon: {
				Minted: 9996,
				HolderCount: 43,
				Holders: map[string]int64{
					
				},
			},
			Rare: {
				Minted: 4,
				HolderCount: 3,
				Holders: map[string]int64{
					
				},
			},
			FirstTransaction: {
				Minted: 2814999961782,
				HolderCount: 9,
				Holders: map[string]int64{
					
				},
			},
			Vintage: {
				Minted: 5005000000000,
				HolderCount: 13,
				Holders: map[string]int64{
					
				},
			},
		},
	},

	100000: {
		Tickers: map[string]*TickerStatus{
			"RarePizza": {
				Minted:      997000, 
			},
		},
	},
}

func (p *ExoticIndexer) CheckPointWithBlockHeight(height int) {

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
		tickers := p.getAllTickers()
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
		name = strings.ToLower(name)
		ticker := p.getTicker(name)
		if ticker == nil {
			common.Log.Panicf("CheckPointWithBlockHeight can't find ticker %s", name)
		}
		if tickerStatus.Max != 0 && ticker.Max != tickerStatus.Max {
			common.Log.Panicf("%s Max different, %d %d", name, ticker.Max, tickerStatus.Max)
		}
		if tickerStatus.Minted != 0 && ticker.TotalMinted != tickerStatus.Minted {
			common.Log.Panicf("%s Minted different, %d %d", name, ticker.TotalMinted, tickerStatus.Minted)
		}


		if tickerStatus.HolderCount != 0 {
			holders := p.getHolderAndAmountWithTick(name)
			if len(holders) != tickerStatus.HolderCount {
				common.Log.Panicf("%s HolderCount different, %d %d", name, len(holders), tickerStatus.HolderCount)
			}
		}

		for address, amt := range tickerStatus.Holders {
			addressId := baseIndexer.GetAddressIdFromDB(address)
			if addressId == common.INVALID_ID {
				common.Log.Panicf("%s GetAddressIdFromDB %s failed", name, address)
			}
			d := p.getAssetAmtByAddress(addressId, name)
			if d != amt {
				common.Log.Panicf("%s holder %s amt different, %d %d", name, address, d, amt)
			}
		}

		holdermap := p.getHolderAndAmountWithTick(name)
		var holderAmount int64
		for _, amt := range holdermap {
			holderAmount += amt
		}
		if holderAmount != ticker.TotalMinted {
			common.Log.Infof("block %d, ticker %s, asset amount different %d %d",
				height, name, ticker.TotalMinted, holderAmount)

			//p.printHistory(name)
			//p.printHistoryWithAddress(name, 0x52b1777c)
			common.Log.Panicf("%s amount different %d %d", name, ticker.TotalMinted, holderAmount)
		}

	}
	common.Log.Infof("CheckPointWithBlockHeight %d checked, takes %v", height, time.Since(startTime))
}

func (s *ExoticIndexer) printHistoryWithAddress(name string, addressId uint64) {
}
