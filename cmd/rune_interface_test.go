package main

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/indexer/indexer"
	"github.com/sat20-labs/indexer/indexer/runes"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/share/base_indexer"
	"lukechampine.com/uint128"
)

var firstRuneName = "BESTINSLOT•XYZ"

var runesIndexer *runes.Indexer

func InitRuneTester() {
	if runesIndexer == nil {
		dbdir := "../db/testnet/"
		indexerMgr := indexer.NewIndexerMgr(dbdir, &chaincfg.TestNet4Params, 61680, 20)
		base_indexer.InitBaseIndexer(indexerMgr)
		indexerMgr.Init()
		runesIndexer = indexerMgr.RunesIndexer
		runestone.IsLessStorage = false
	}
}

func TestInterfaceRune(t *testing.T) {
	InitRuneTester()
	// 0
	runeIdStr := "61721_61"
	runeInfo := runesIndexer.GetRuneInfoWithId(runeIdStr)
	t.Logf("GetRuneInfoWithId return: %+v\n", runeInfo)
	// 1
	runeInfo = runesIndexer.GetRuneInfoWithName(firstRuneName)
	// common.Log.Infof("GetRuneInfo return: %+v\n", runeInfo)
	t.Logf("GetRuneInfo return: %+v\n", runeInfo)
	// 2
	isExistRune := runesIndexer.IsExistRuneWithName(firstRuneName)
	t.Logf("IsExistRune return: %+v\n", isExistRune)
	// 3
	runeInfos, total := runesIndexer.GetRuneInfos(0, 1000)
	t.Logf("GetRuneInfos return runeInfo total count: %d\n", total)
	for i, v := range runeInfos {
		t.Logf("GetRuneInfos return runeInfo %d: %+v\n", i, v)
	}
}

func TestGetHoldersWithTicks(t *testing.T) {
	InitRuneTester()
	// 11
	runeId, err := runesIndexer.GetRuneIdWithName(firstRuneName)
	if err != nil {
		t.Fatalf("GetRuneIdWithName err:%s", err.Error())
	}
	holders := runesIndexer.GetHoldersWithTick(runeId.String())
	t.Logf("GetHoldersWithTicks return holders total count: %d\n", len(holders))
	for i, v := range holders {
		t.Logf("GetHoldersWithTicks return holders, addressId: %d, value: %s\n", i, v.String())
	}
}

func TestGetAllAddressBalances(t *testing.T) {
	InitRuneTester()
	// 4
	runeId, err := runesIndexer.GetRuneIdWithName(firstRuneName)
	if err != nil {
		t.Fatalf("GetRuneIdWithName err:%s", err.Error())
	}
	addressBalance, total := runesIndexer.GetAllAddressBalances(runeId.String(), 0, 10)
	t.Logf("GetAllAddressBalances return addressBalance total count: %d\n", total)
	for i, v := range addressBalance {
		t.Logf("GetAllAddressBalances return addressBalance %d: addressId: %d, balance: %s\n", i, v.AddressId, v.Balance.String())
	}
}

func TestGetAllUtxoBalances(t *testing.T) {
	InitRuneTester()
	runeId, err := runesIndexer.GetRuneIdWithName(firstRuneName)
	if err != nil {
		t.Fatalf("GetRuneIdWithName err:%s", err.Error())
	}
	// 5
	allUtxoBalances1, total1 := runesIndexer.GetAllUtxoBalances(runeId.String(), 0, 10)
	t.Logf("GetAllUtxoBalances return utxoBalance total count: %d\n", total1)
	for i, v := range allUtxoBalances1.Balances {
		t.Logf("GetAllUtxoBalances return utxoBalance %d: %+v\n", i, v)
	}
}

func TestInterfaceAsset(t *testing.T) {
	InitRuneTester()
	runeId, err := runesIndexer.GetRuneIdWithName(firstRuneName)
	if err != nil {
		t.Fatalf("GetRuneIdWithName err:%s", err.Error())
	}
	// 6
	firstRuneAddress := "tb1pn9dzakm6egrv90c9gsgs63axvmn6ydwemrpuwljnmz9qdk38ueqsqae936"
	addressId := runesIndexer.RpcService.GetAddressId(firstRuneAddress)
	addressAssets := runesIndexer.GetAddressAssets(addressId)
	for i, v := range addressAssets {
		t.Logf("GetAddressAssets return addressAssets %d: %+v\n", i, v)
	}

	// 7
	utxo := "d2f8fe663c83550fee4039027fc4d5053066c10b638180137f43b997cc427108:0"
	utxoInfo, err := runesIndexer.RpcService.GetUtxoInfo(utxo)
	if err != nil {
		t.Errorf("RpcService.GetUtxoInfo error: %s", err.Error())
	}
	utxoAssets := runesIndexer.GetUtxoAssets(utxoInfo.UtxoId)
	for i, v := range utxoAssets {
		t.Logf("GetUtxoAssets return utxoAssets %d: %+v\n", i, v)
	}

	// 8
	isExistAsset := runesIndexer.IsExistAsset(utxoInfo.UtxoId)
	t.Logf("IsExistAsset return: %+v\n", isExistAsset)

	// 9
	mintHistorys, total := runesIndexer.GetMintHistory(runeId.Hex(), 0, 10)
	t.Logf("GetMintHistory return txids total count: %d\n", total)
	for i, v := range mintHistorys {
		t.Logf("GetMintHistory return txids %d: %+v\n", i, v)
	}
}

func TestGetAddressMintHistory(t *testing.T) {
	InitRuneTester()
	// 10
	firstRuneAddress := "tb1pfu2ff6ycy99t02zteumkm2jtk3uwm4skp50m7tevapcpkm8vaqqq73vxqr"
	runeId, err := runesIndexer.GetRuneIdWithName(firstRuneName)
	if err != nil {
		t.Fatalf("GetRuneIdWithName err:%s", err.Error())
	}
	addressId := runesIndexer.RpcService.GetAddressId(firstRuneAddress)
	mintHistorys, total := runesIndexer.GetAddressMintHistory(runeId.Hex(), addressId, 0, 10)
	t.Logf("GetAddressMintHistory return txids total count: %d\n", total)
	for i, v := range mintHistorys {
		t.Logf("GetAddressMintHistory return txids %d: %+v\n", i, v)
	}
}

func TestCheckRunesSummary(t *testing.T) {
	InitRuneTester()
	runeId, err := runesIndexer.GetRuneIdWithName(firstRuneName)
	if err != nil {
		t.Fatalf("GetRuneIdWithName err:%s", err.Error())
	}
	t.Logf("rune: %s\n", firstRuneName)

	runeInfo := runesIndexer.GetRuneInfoWithId(runeId.String())
	_, total := runesIndexer.GetAllAddressBalances(runeId.String(), 0, 1)
	addressBalances, _ := runesIndexer.GetAllAddressBalances(runeId.String(), 0, total)
	var addressBalance uint128.Uint128
	for _, v := range addressBalances {
		addressBalance = v.Balance.Add(addressBalance)
	}

	totalAddressBalance := addressBalance.Add(runeInfo.Burned)
	if addressBalance.Add(runeInfo.Burned).Cmp(totalAddressBalance) != 0 {
		t.Errorf("all address(%d)'s total balance(%s) + burned is not equal to supply(%s)", total, totalAddressBalance.String(), runeInfo.Supply.String())
	} else {
		t.Logf("all address(%d)'s total balance(%s) + burned is equal to supply(%s)", total, totalAddressBalance.String(), runeInfo.Supply.String())
	}

	_, total = runesIndexer.GetAllUtxoBalances(runeId.String(), 0, 1)
	utxoBalances, _ := runesIndexer.GetAllUtxoBalances(runeId.String(), 0, total)
	totalUtxoBalance := utxoBalances.Total.Add(runeInfo.Burned)
	if utxoBalances.Total.Add(runeInfo.Burned).Cmp(totalUtxoBalance) != 0 {
		t.Errorf("all utxo(%d)'s total balance(%s) + burned is not equal to supply(%s)", total, totalUtxoBalance.String(), runeInfo.Supply.String())
	} else {
		t.Logf("all utxo(%d)'s total balance(%s) + burned is equal to supply(%s)", total, totalUtxoBalance.String(), runeInfo.Supply.String())
	}
}
