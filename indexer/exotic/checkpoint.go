package exotic

import (
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
)

type CheckPoint struct {
	Height      int
	TickerCount int
	CheckHolder bool
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
			Uncommon: {DeployHeight: 1}, 
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

	50000: {
		Tickers: map[string]*TickerStatus{
			Uncommon: {
				Minted: 49976,
				HolderCount: 1327,
				Holders: map[string]int64{
					"mkq9gRbqQCPqhS8AdjSfQzCgvDiSJTrrvg": 18965,
					"tb1q548z58kqvwyjqwy8vc2ntmg33d7s2wyfv7ukq4": 4577,
					"mjP97q5BWtdpdsJLkEJvQWgLe9zw4MMVU6": 2508,
					"tb1pq3qun4r880v5k4g80xzgjmzspunw7r4d5exfsa457fqzd7yqy9vs0drlxu": 2005,
					"tb1pmlne4hgps990s3ygfyza89mjdzzxvzcgg7rjw6h83kn0th5cmquspcpy7r": 1956,
					"tb1qm3lcnz58f5398spu4rvr6tk2l8as3sun8h7rn9": 1911,
					"tb1q3u8f5899ymkatx69h0n3sw0qpalgwdmrcj80dm": 1744,
					"tb1pvetwfcsrse4apac588sf86ecw8z0gfh4nlgy442lec6qe3m84w4su8c27x": 1598,
					"tb1q2dsc94zq40nwnz27w5rxljwllutnwjtlxk44fz": 1005,
					"tb1ppr7hkyz0ug004rq85u4rkjzc7vz5zp0zakmp59ddljtmgg5x7h7sz74psv": 1000,
					"tb1q0dzcgv7scppjxsnwlzpkt02vlmc5rtr40wyjgr": 974,
				},
			},
			Rare: {
				Minted: 24,
				HolderCount: 12,
				Holders: map[string]int64{
					"mkq9gRbqQCPqhS8AdjSfQzCgvDiSJTrrvg": 10,
					"tb1pvetwfcsrse4apac588sf86ecw8z0gfh4nlgy442lec6qe3m84w4su8c27x": 2,
					"mjP97q5BWtdpdsJLkEJvQWgLe9zw4MMVU6": 2,
					"tb1pmlne4hgps990s3ygfyza89mjdzzxvzcgg7rjw6h83kn0th5cmquspcpy7r": 2,
					"tb1q548z58kqvwyjqwy8vc2ntmg33d7s2wyfv7ukq4": 1,
				},
			},
			FirstTransaction: {
				Minted: 2814999961782,
				HolderCount: 3915,
				Holders: map[string]int64{
					"tb1q0dzcgv7scppjxsnwlzpkt02vlmc5rtr40wyjgr": 1803513911357,
					"tb1qnjkdfejgc2t9q24qp7d2ssmfwjmsdfyx4fvpst": 935699899400,
					"tb1p3hfnwp2nt6t6de9crmpwhfqkk64dak62d00hau5npp3c6r5mj8aqxnl8vz": 64299820374,
					"tb1qx4csx5gzsnev56ayzc4tyayyk307n56apqfjsh": 10000000000,
					"tb1pqj42j9m0cu6lr50t2dnx8223aklntsys7fa7xu6j7qnphgmrydsswwetek": 50427827,
					"tb1qw5z7ulpjjp6gwu45kh0jmg5yvyjkc5ffu9l5kd": 20821150,
					"tb1pethmmnjaunvyjsfn4ykv9q32k0k6xqh6ejacavpwjmea3qwg8yuqgn9duf": 17226767,
					"tb1qr0h6wrdrfsdnk5ulerhhr9958pkv5ks9p5em98": 13843059,
					"tb1qlcztpurpevvt58vlrg8nv7kv73xl7k0e6f8vcu": 9499145,
					"tb1py9vv6zfq72mdkqelenfdy0el8wqvuc4488mvmt6mzvacnmt9023shtk6s3": 8499844,
					"tb1q39v6jl4leqv6hc53zlay02a3r5jk4h4z6zrjq9": 8499072,
				},
			},
			Vintage: {
				Minted: 5005000000000,
				HolderCount: 3920,
				Holders: map[string]int64{
					"tb1q0dzcgv7scppjxsnwlzpkt02vlmc5rtr40wyjgr": 1803513911357,
					"tb1q548z58kqvwyjqwy8vc2ntmg33d7s2wyfv7ukq4": 1800001097250,
					"tb1qnjkdfejgc2t9q24qp7d2ssmfwjmsdfyx4fvpst": 935699899400,
					"mgTgHVFXFdMEJiMmLhGrxu75waDYjCjDvN": 365000000000,
					"tb1p3hfnwp2nt6t6de9crmpwhfqkk64dak62d00hau5npp3c6r5mj8aqxnl8vz": 64299820374,
					"tb1qfcsg40qn8tcl990vph9vjm27wvy3l5lryl4t5c": 14999730397,
					"tb1qx4csx5gzsnev56ayzc4tyayyk307n56apqfjsh": 10000000000,
					"tb1qhw67jcsumk7899260up2at86q5xza0frvzrp49": 5000000000,
					"IQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAKw=": 5000000000,
					"tb1pqj42j9m0cu6lr50t2dnx8223aklntsys7fa7xu6j7qnphgmrydsswwetek": 50427827,
					"tb1qw5z7ulpjjp6gwu45kh0jmg5yvyjkc5ffu9l5kd": 20821150,
				},
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
		if checkpoint.TickerCount != 0 {
			tickers := p.getAllTickers()
			if len(tickers) != checkpoint.TickerCount {
				common.Log.Panicf("ticker count different")
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

		if checkpoint.CheckHolder {
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
	}
	common.Log.Infof("ExoticIndexer.CheckPointWithBlockHeight %d checked, takes %v", height, time.Since(startTime))
}

func (s *ExoticIndexer) printHistoryWithAddress(name string, addressId uint64) {
}
