package main

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/config"
	"github.com/sat20-labs/indexer/indexer"
	"github.com/sat20-labs/indexer/indexer/brc20"
	"github.com/sat20-labs/indexer/share/base_indexer"
)

var firstTestnet4Brc20Name = "box1"

var tickerName = "GC  " // box6
var brc20Indexer *brc20.BRC20Indexer
var indexerMgr *indexer.IndexerMgr

func InitBrc20Tester() {
	if brc20Indexer == nil {
		dbdir := "../db/testnet/"
		yamlcfg := config.YamlConf{
			Chain: "testnet",
			DB: config.DB{
				Path: dbdir,
			},
			BasicIndex: config.BasicIndex{
				MaxIndexHeight:  61680,
				PeriodFlushToDB: 20,
			},
		}
		indexerMgr = indexer.NewIndexerMgr(&yamlcfg)
		base_indexer.InitBaseIndexer(indexerMgr)
		indexerMgr.Init()
		brc20Indexer = indexerMgr.GetBrc20Indexer()
	}
}

func printBrc20Ticker(t *testing.T, ticker *common.BRC20Ticker) {
	format := "print brc20 ticker:\nName: “%s”\n"
	format += "inscription: %s\nSupply: %s\nMinted: %s\nLimit per mint: %s\nDecimal: %d\nSelf-issuance: %v\nDeploy By: %s\n"
	format += "Deploy Time: %s\nCompleted Time: %s\nInscription Number Start: %v\nInscription Number End: %v\n"
	format += "Holders: %d\nTotal Transactions: %d\n"

	nftInfo := indexerMgr.GetNftInfoWithInscriptionId(ticker.StartInscriptionId)
	deployAddress := indexerMgr.GetAddressById(nftInfo.OwnerAddressId)
	deployTime := time.Unix(int64(ticker.DeployTime), 0).Format("2006-01-02 15:04:05")

	completedTime := ""
	if ticker.EndInscriptionId != "" {
		nftInfo = indexerMgr.GetNftInfoWithInscriptionId(ticker.EndInscriptionId)
		completedTime = time.Unix(int64(nftInfo.Base.BlockTime), 0).Format("2006-01-02 15:04:05")
	}

	t.Logf(format, ticker.Name,
		ticker.Nft.Base.InscriptionId, ticker.Max.String(), ticker.Minted.String(), ticker.Limit.String(), ticker.Decimal, ticker.SelfMint, deployAddress,
		deployTime, completedTime, ticker.StartInscriptionId, ticker.EndInscriptionId,
		ticker.HolderCount, ticker.TransactionCount,
	)

	// t.Logf("brc20 ticker %+v\n", brc20Ticker.Limit.String())
}

func TestInterfaceBrc20(t *testing.T) {
	InitBrc20Tester()
	// 1
	ticker := brc20Indexer.GetTicker(firstTestnet4Brc20Name)
	printBrc20Ticker(t, ticker)
	// 2
	isExist := brc20Indexer.TickExisted(firstTestnet4Brc20Name)
	t.Logf("brc20 ticker IsExist: %+v\n", isExist)
	// 3
	// tickers, total := brc20Indexer.GetTickers(0, 1000, brc20.BRC20_TICKER_ORDER_DEPLOYTIME_DESC)
	// t.Logf("brc20Indexer return brc20Info total count: %d\n", total)
	// for i, v := range tickers {
	// 	t.Logf("brc20Indexer return brc20Info %d: %+v\n", i, v.Name)
	// }
}

type HolderBalance struct {
	Address string
	Balance *common.Decimal
}

func getHolderBalances(tickerName string, start, limit uint64) (ret []*HolderBalance, total uint64) {
	holders := brc20Indexer.GetHoldersWithTick(tickerName)
	for addressId, balance := range holders {
		address := indexerMgr.GetAddressById(addressId)
		ret = append(ret, &HolderBalance{
			Address: address,
			Balance: balance,
		})
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Balance.Cmp(ret[j].Balance) > 0
	})

	total = uint64(len(ret))
	end := total
	if start >= end {
		return nil, 0
	}
	if start+limit < end {
		end = start + limit
	}
	ret = ret[start:end]
	return
}

func TestGetBrc20HoldersBalance(t *testing.T) {
	InitBrc20Tester()
	// 4
	holders, total := getHolderBalances(firstTestnet4Brc20Name, 0, 1000)
	t.Logf("GetHoldersWithTicks return holders total count: %d\n", total)
	for _, v := range holders {
		fmt.Printf("holder %s: %s\n", v.Address, v.Balance.String())
	}
}

func TestBrc20GetAddressMintHistory(t *testing.T) {
	InitBrc20Tester()
	// 10
	// firstRuneAddress := "tb1pfu2ff6ycy99t02zteumkm2jtk3uwm4skp50m7tevapcpkm8vaqqq73vxqr"
	// runeId, err := brc20Indexer.GetRuneIdWithName(firstTestnet4Brc20Name)
	// if err != nil {
	// 	t.Fatalf("GetRuneIdWithName err:%s", err.Error())
	// }
	// addressId := brc20Indexer.RpcService.GetAddressId(firstRuneAddress)
	// mintHistorys, total := brc20Indexer.GetAddressMintHistory(runeId.Hex(), addressId, 0, 10)
	// t.Logf("GetAddressMintHistory return txids total count: %d\n", total)
	// for i, v := range mintHistorys {
	// 	t.Logf("GetAddressMintHistory return txids %d: %+v\n", i, v)
	// }
}
