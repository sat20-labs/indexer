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

func NewRuneTester() *RuneTester {
	dbdir := "/Users/chenwenjie/test/indexer/db/testnet-61680/" + "runes"
	indexerMgr := indexer.NewIndexerMgr(dbdir, &chaincfg.TestNet4Params, 61680, 20)
	base_indexer.InitBaseIndexer(indexerMgr)
	indexerMgr.Init()

	ret := &RuneTester{
		indexerMgr: indexerMgr,
	}
	return ret
}

func TestInterface1(t *testing.T) {
	// runeTester := NewRuneTester()
	firstRuneName := "BESTINSLOT•XYZ"
	var runesIndexer *runes.Indexer
	// runesIndexer = runeTester.indexerMgr.RunesIndexer
	// 1
	runeInfo := runesIndexer.GetRuneInfo(firstRuneName)
	t.Logf("GetRuneInfo return: %+v\n", runeInfo)
	// 2
	isExistRune := runesIndexer.IsExistRune(firstRuneName)
	t.Logf("IsExistRune return: %+v\n", isExistRune)
	// 3
	runeInfos, total := runesIndexer.GetRuneInfos(0, 10)
	t.Logf("GetRuneInfos return runeInfo total count: %d\n", total)
	for i, v := range runeInfos {
		t.Logf("GetRuneInfos return runeInfo %d: %+v\n", i, v)
	}
	// 4
	addressBalance, total := runesIndexer.GetAllAddressBalances(firstRuneName, 0, 10)
	t.Logf("GetAllAddressBalances return addressBalance total count: %d\n", total)
	for i, v := range addressBalance {
		t.Logf("GetAllAddressBalances return addressBalance %d: %+v\n", i, v)
	}
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
	utxoAssets, total := runesIndexer.GetUtxoAssets(firstRuneName, 0, 10)
	t.Logf("GetUtxoAssets return utxoAssets total count: %d\n", total)
	for i, v := range utxoAssets {
		t.Logf("GetUtxoAssets return utxoAssets %d: %+v\n", i, v)
	}
	// 8
	isExistAsset := runesIndexer.IsExistAsset(firstRuneName)
	t.Logf("IsExistAsset return: %+v\n", isExistAsset)
	// 9
	txids, total := runesIndexer.GetMintHistory(firstRuneName, 0, 10)
	t.Logf("GetMintHistory return txids total count: %d\n", total)
	for i, v := range txids {
		t.Logf("GetMintHistory return txids %d: %+v\n", i, v)
	}
	// 10
	txids, total = runesIndexer.GetAddressMintHistory(firstRuneName, firstRuneAddress, 0, 10)
	t.Logf("GetAddressMintHistory return txids total count: %d\n", total)
	for i, v := range txids {
		t.Logf("GetAddressMintHistory return txids %d: %+v\n", i, v)
	}
}
