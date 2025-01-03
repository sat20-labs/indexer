package main

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/indexer/indexer"
	"github.com/sat20-labs/indexer/indexer/runes"
	"github.com/sat20-labs/indexer/share/base_indexer"
)

type RuneTester struct {
	indexerMgr *indexer.IndexerMgr
}

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
	runeInfo := runesIndexer.GetRuneInfo(firstRuneName)
	// common.Log.Infof("GetRuneInfo return: %+v\n", runeInfo)
	t.Logf("GetRuneInfo return: %+v\n", runeInfo)
	// 2
	isExistRune := runesIndexer.IsExistRune(firstRuneName)
	t.Logf("IsExistRune return: %+v\n", isExistRune)
	// 3
	runeInfos, total := runesIndexer.GetRuneInfos(0, 1000)
	t.Logf("GetRuneInfos return runeInfo total count: %d\n", total)
	for i, v := range runeInfos {
		t.Logf("GetRuneInfos return runeInfo %d: %+v\n", i, v)
	}
}

func TestInterfaceAsset(t *testing.T) {
	InitRuneTester()
	// 4
	// addressBalance, total := runesIndexer.GetAllAddressBalances(firstRuneName, 0, 10)
	// common.Log.Infof("GetAllAddressBalances return addressBalance total count: %d\n", total)
	// for i, v := range addressBalance {
	// 	common.Log.Infof("GetAllAddressBalances return addressBalance %d: address: %s, balance: %s\n", i, v.Address, v.Pile.String())
	// }

	// t.Logf("GetAllAddressBalances return addressBalance total count: %d\n", total)
	// for i, v := range addressBalance {
	// 	t.Logf("GetAllAddressBalances return addressBalance %d: address: %s, balance: %s\n", i, v.Address, v.Pile.String())
	// }

	// 5
	allUtxoBalances, total := runesIndexer.GetAllUtxoBalances(firstRuneName, 0, 10)
	t.Logf("GetAllUtxoBalances return utxoBalance total count: %d\n", total)
	for i, v := range allUtxoBalances.Balances {
		t.Logf("GetAllUtxoBalances return utxoBalance %d: %+v\n", i, v)
	}

	// 6
	firstRuneAddress := "tb1pn9dzakm6egrv90c9gsgs63axvmn6ydwemrpuwljnmz9qdk38ueqsqae936"
	addressAssets, total := runesIndexer.GetAddressAssets(firstRuneAddress, 0, 10)
	t.Logf("GetAddressAssets return addressAssets total count: %d\n", total)
	for i, v := range addressAssets {
		t.Logf("GetAddressAssets return addressAssets %d: %+v\n", i, v)
	}

	// 7
	utxo := "d2f8fe663c83550fee4039027fc4d5053066c10b638180137f43b997cc427108:0"
	utxoAssets := runesIndexer.GetUtxoAssets(utxo, 0, 10)
	t.Logf("GetUtxoAssets return utxoAssets total count: %d\n", len(utxoAssets))
	for i, v := range utxoAssets {
		t.Logf("GetUtxoAssets return utxoAssets %d: %+v\n", i, v)
	}

	// 8
	isExistAsset := runesIndexer.IsExistAsset(utxo)
	t.Logf("IsExistAsset return: %+v\n", isExistAsset)

	// 9
	mintHistorys, total := runesIndexer.GetMintHistory(firstRuneName, 0, 10)
	t.Logf("GetMintHistory return txids total count: %d\n", total)
	for i, v := range mintHistorys {
		t.Logf("GetMintHistory return txids %d: %+v\n", i, v)
	}

	// 10
	firstRuneAddress = "tb1pfu2ff6ycy99t02zteumkm2jtk3uwm4skp50m7tevapcpkm8vaqqq73vxqr"
	mintHistorys, total = runesIndexer.GetAddressMintHistory(firstRuneName, firstRuneAddress, 0, 10)
	t.Logf("GetAddressMintHistory return txids total count: %d\n", total)
	for i, v := range mintHistorys {
		t.Logf("GetAddressMintHistory return txids %d: %+v\n", i, v)
	}
}
