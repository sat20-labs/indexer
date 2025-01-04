package main

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/indexer/indexer"
	"github.com/sat20-labs/indexer/indexer/runes"
	"github.com/sat20-labs/indexer/share/base_indexer"
)

const firstRuneName = "BESTINSLOT•XYZ"

var runesIndexer *runes.Indexer

func InitRuneTester() {
	if runesIndexer == nil {
		dbdir := "../db/testnet/"
		indexerMgr := indexer.NewIndexerMgr(dbdir, &chaincfg.TestNet4Params, 61680, 20)
		base_indexer.InitBaseIndexer(indexerMgr)
		indexerMgr.Init()
		runesIndexer = indexerMgr.RunesIndexer
	}
}

func TestInterfaceRune(t *testing.T) {
	InitRuneTester()
	// 1
	runeInfo := runesIndexer.GetRuneInfoWithName(firstRuneName)
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

func TestGetAllAddressBalances(t *testing.T) {
	InitRuneTester()
	// 4
	runeId, err := runesIndexer.GetRuneIdWithName(firstRuneName)
	if err != nil {
		t.Fatalf("GetRuneIdWithName err:%s", err.Error())
	}
	// TODO optimize
	addressBalance, total := runesIndexer.GetAllAddressBalances(runeId.String(), 0, 10)
	t.Logf("GetAllAddressBalances return addressBalance total count: %d\n", total)
	for i, v := range addressBalance {
		t.Logf("GetAllAddressBalances return addressBalance %d: address: %s, balance: %s\n", i, v.Address, v.Pile.String())
	}
}

func TestInterfaceAsset(t *testing.T) {
	InitRuneTester()
	// 5
	runeId, err := runesIndexer.GetRuneIdWithName(firstRuneName)
	if err != nil {
		t.Fatalf("GetRuneIdWithName err:%s", err.Error())
	}
	allUtxoBalances, total := runesIndexer.GetAllUtxoBalances(runeId.String(), 0, 10)
	t.Logf("GetAllUtxoBalances return utxoBalance total count: %d\n", total)
	for i, v := range allUtxoBalances.Balances {
		t.Logf("GetAllUtxoBalances return utxoBalance %d: %+v\n", i, v)
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
	mintHistorys, total := runesIndexer.GetMintHistory(runeId.String(), 0, 10)
	t.Logf("GetMintHistory return txids total count: %d\n", total)
	for i, v := range mintHistorys {
		t.Logf("GetMintHistory return txids %d: %+v\n", i, v)
	}
}

func TestGetAddressMintHistory(t *testing.T) {
	InitRuneTester()
	// 10
	firstRuneAddress := "tb1pfu2ff6ycy99t02zteumkm2jtk3uwm4skp50m7tevapcpkm8vaqqq73vxqr"
	addressId := runesIndexer.RpcService.GetAddressId(firstRuneAddress)
	mintHistorys, total := runesIndexer.GetAddressMintHistory(firstRuneName, addressId, 0, 10)
	t.Logf("GetAddressMintHistory return txids total count: %d\n", total)
	for i, v := range mintHistorys {
		t.Logf("GetAddressMintHistory return txids %d: %+v\n", i, v)
	}
}
