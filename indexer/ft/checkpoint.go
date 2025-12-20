package ft

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
	Max         int64
	Minted      int64
	MintCount   int64
	HolderCount int
	Holders     map[string]int64
}

var testnet4_checkpoint = map[int]*CheckPoint{
	0: {
		Tickers: map[string]*TickerStatus{
			"dogecoin": {}, // 每个区块，如果没有其他检查，就默认检查该资产的holder和minted是否匹配
		},
	},

	114380: {
		Tickers: map[string]*TickerStatus{
			"dogecoin": {
				Minted:      156271012,
				MintCount:   15638,
				HolderCount: 2083,
				Holders: map[string]int64{
					"bc1qmgactfmdfympq5tqld7rc53y4dphvdyqnmtuuv9jwgpn7hqwr2kss26dls": 18436116,
					"bc1pae8vkxlfa6aeefswxjnaxupm90zjc56y924jr4uhwkzv7aldhdysrc45cr": 3004146,
					"bc1qfsvzq8wysyzxdmt982sq4pkpc9xpd4nlku7tjr":                     2667024,
					"bc1pgn2jfve3n7hhmua4966aa57ggckpp67qm34hj6hmtjm82qdmukyq29rau7": 36,
					"bc1qtz5ymtyycg0dyrz2y6zlm7rc46wl9nmy49jen3":                     1,
				},
			},
		},
	},
}

var mainnet_checkpoint = map[int]*CheckPoint{
	0: {
		Tickers: map[string]*TickerStatus{
			"pearl": {}, // 每个区块，如果没有其他检查，就默认检查该资产的holder和minted是否匹配
		},
	},

	928648: {
		Tickers: map[string]*TickerStatus{
			"pearl": {
				Minted:      156271012,
				MintCount:   15638,
				HolderCount: 2083,
				Holders: map[string]int64{
					"bc1qmgactfmdfympq5tqld7rc53y4dphvdyqnmtuuv9jwgpn7hqwr2kss26dls": 18436116,
					"bc1pae8vkxlfa6aeefswxjnaxupm90zjc56y924jr4uhwkzv7aldhdysrc45cr": 3004146,
					"bc1qfsvzq8wysyzxdmt982sq4pkpc9xpd4nlku7tjr":                     2667024,
					"bc1pgn2jfve3n7hhmua4966aa57ggckpp67qm34hj6hmtjm82qdmukyq29rau7": 36,
					"bc1qtz5ymtyycg0dyrz2y6zlm7rc46wl9nmy49jen3":                     1,
				},
			},

			"jades": {
				Minted:      115,
				MintCount:   115,
				HolderCount: 11,
				Holders: map[string]int64{
					"bc1pmc00se8pu07lna0qvnmward3nt83lpcnnnjyv2wj72adx3ghdr2sd3ghrk": 36,
					"bc1pdyml6m53ga9s9zd2tjwdg8ht30r3qel2glw3952834emah0tw96q5d2qj2": 26,
					"bc1pd3xa9mjlpx08v5ertdm8uu48vlfa4deu82llcec0ga5jw69zghas2uje4x": 22,
					"bc1pmufun7s7qnwckpf9yx4kr5ez4wk9qxx66ap89pn5jp4hyjtuv53sfh6vrf": 20,
					"bc1p0klza3g0p3rknwlx4s66uckw8nljetptqmt3jd5twfltdru3g9mssqkxxx": 1,
				},
			},
		},
	},

	928649: {
		Tickers: map[string]*TickerStatus{
			"rarepizza": {
				Minted:      99106856,
				MintCount:   99234,
				HolderCount: 6078,
				Holders: map[string]int64{
					"bc1p7jmfgmlecgp8c3j8260rpm3g2dlvey45kvhjd0hh0wn9zlwfr32qus90r3": 6896927,
					"bc1qmgactfmdfympq5tqld7rc53y4dphvdyqnmtuuv9jwgpn7hqwr2kss26dls": 2423634,
					"bc1pzdzs9xlw76llpes6p4tps4dxvsw7jwvdt4hadxl9dxuazrauj7fspmpted": 2000000,
					"bc1pefe3ay7wcntwg45l90rwd9jj7lgvdeh4mflh8nctxzfrlhd6xhmquca7aq": 1670967,
					"bc1qlaal3drvueaxndam7p5kxa9epl2g7rshqphd0t":                     111,
					"bc1qv4m0g775jucmh5ttht7nst6nq88r2sjfyrmdr7":                     1,
				},
			},
		},
	},
}

func (p *FTIndexer) CheckPointWithBlockHeight(height int) {

	startTime := time.Now()
	var checkpoint *CheckPoint
	matchHeight := height
	isMainnet := p.nftIndexer.GetBaseIndexer().IsMainnet()
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
	// 太花时间
	//rpc := base.NewRpcIndexer(p.nftIndexer.GetBaseIndexer())
	baseIndexer := p.nftIndexer.GetBaseIndexer()
	for name, tickerStatus := range checkpoint.Tickers {
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

		if tickerStatus.MintCount != 0 {
			_, mintCount := p.getMintAmount(name)
			if mintCount != tickerStatus.MintCount {
				common.Log.Panicf("%s MinteMintCountd different, %d %d", name, mintCount, tickerStatus.MintCount)
			}
		}

		if tickerStatus.HolderCount != 0 {
			holders := p.getHoldersWithTick(name)
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

			printAddress := make(map[uint64]bool)
			for k, v := range holdermap {
				old, ok := p.holderMapInPrevBlock[k]
				if ok {
					if old != v {
						common.Log.Infof("%x changed %d -> %d", k, old, v)
						printAddress[k] = true
					}
				} else {
					common.Log.Infof("%x added %d -> %d", k, old, v)
					printAddress[k] = true
				}
			}
			for k := range printAddress {
				p.printHistoryWithAddress(name, k)
			}

			//p.printHistory(name)
			//p.printHistoryWithAddress(name, 0x52b1777c)
			common.Log.Panicf("%s amount different %d %d", name, ticker.TotalMinted, holderAmount)
		}
		p.holderMapInPrevBlock = holdermap

	}
	common.Log.Infof("CheckPointWithBlockHeight %d checked, takes %v", height, time.Since(startTime))
}

func (s *FTIndexer) printHistoryWithAddress(name string, addressId uint64) {
}
