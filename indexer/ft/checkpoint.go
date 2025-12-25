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
			"RarePizza": {DeployHeight: 31894},
			"dogcoin": {DeployHeight: 60886}, // 每个区块，如果没有其他检查，就默认检查该资产的holder和minted是否匹配
			"fail02": {DeployHeight: 29008},
			"fair01": {DeployHeight: 28883},
			"pizzaTest": {DeployHeight: 32026},
		},
	},

	36344: {
		Tickers: map[string]*TickerStatus{
			"pizzaTest": {
				Minted:      1237986, 
			},
		},
	},

	36213: {
		Tickers: map[string]*TickerStatus{
			"RarePizza": {
				Minted:      997000, 
			},
		},
	},

	114380: {
		Tickers: map[string]*TickerStatus{
			"dogcoin": {
				Minted:      1800100, 
				MintCount:   1810,
				HolderCount: 78,
				Holders: map[string]int64{
					"tb1p7pnu75ln3evuyw26sdnl5kq6xxmlpqs5x7l6c6dqy5nqy5nz46cqvc7g8n": 886000,
					"tb1qw86hsm7etf4jcqqg556x94s6ska9z0239ahl0tslsuvr5t5kd0nq7vh40m": 454262,
					"tb1p8dklwgn9zsm3cezmtl6rdtygcllf649pm75ua33hacnfpd7t4dqqxmcvfr": 100000,
					"tb1p6jcfgfwyfw2nhd948c3sq8cyuevcfjm9u5p8zjrh80kldxsruw2su0432a": 63500,
					"tb1pfgrteggdxrejy5xafgsc0l00uset0wxvgq29ug8s0enkmhyt9zcqetp23z": 330,
					"tb1phv466d2pzf2fat7rc06hlecvuvk4wwa8v6mudw93fwdeauctujlqrxkcm0": 100,
				},
			},

			// 114964
			"RarePizza": {
				Minted:      997000, 
				MintCount:   997,
				HolderCount: 23,
				Holders: map[string]int64{
					"tb1prcc8rp5wn0y9vp434kchl3aag8r8hz699006ufvczwnneuqx0wdsfmvq4y": 495000,
					"tb1p6jcfgfwyfw2nhd948c3sq8cyuevcfjm9u5p8zjrh80kldxsruw2su0432a": 457300,
					"tb1pt9c60e43sxcvksr7arx9qvczj0w9sqjellk6xg9chw2d5pv7ax4sdy5r7n": 7000,
					"tb1qw86hsm7etf4jcqqg556x94s6ska9z0239ahl0tslsuvr5t5kd0nq7vh40m": 6482,
					"tb1qsp335pjzpzmddh6txa30t8gjlv8kurephdtnwz42f7yxd7afrrfs3gs0x7": 5990,
					"tb1prnftyn4t8wz0rcxhw0emqa0r7lwh228nexmwwadnndxyz2dd3eksuqqlps": 7,
				},
			},

			"pizzaTest": {
				Minted:      1237986, 
				MintCount:   1241,
				HolderCount: 34,
				Holders: map[string]int64{
					"tb1p6jcfgfwyfw2nhd948c3sq8cyuevcfjm9u5p8zjrh80kldxsruw2su0432a": 543886,
					"tb1p62gjhywssq42tp85erlnvnumkt267ypndrl0f3s4sje578cgr79sekhsua": 454328,
					"tb1pmzc7yvc7jt9t7vwv8xaqqchqlskg7g4e2ylfu9z4wrk37czjmweqpgq4zd": 100000,
					"tb1q6l2zctpxqvwvkf73fnxqewezk3txflzw3se9h82ux9arksekcrss5cpzj2": 81,
					"tb1p7q2t454hg6r0scdaphud3rdhtc7ghav9gne09hj203ck2q2hjphqcs8vuz": 15,
					"tb1p30x9tc93c3rlfwxv5h5mcpxpwnxx0rvjt5x2f9ef9uukd9l98y4slp2ukr": 8,
				},
			},
		},
	},
}

var mainnet_checkpoint = map[int]*CheckPoint{
	0: {
		Tickers: map[string]*TickerStatus{
			"pearl": {DeployHeight: 827307}, // 每个区块，如果没有其他检查，就默认检查该资产的holder和minted是否匹配
			"jades": {DeployHeight: 830973},
			"rarepizza": {DeployHeight: 850282},
		},
	},

	828800: {
		Tickers: map[string]*TickerStatus{
			"pearl": {
				Minted:      156271012, 
			},
		},
	},

	853358: {
		Tickers: map[string]*TickerStatus{
			"rarepizza": {
				Minted:      99099456, 
			},
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
				Minted:      99099456,
				MintCount:   99226,
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
	
	baseIndexer := p.nftIndexer.GetBaseIndexer()
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
