package runes

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"lukechampine.com/uint128"
)

func TestRegtestRunesActivationPreservesPublicNetworks(t *testing.T) {
	old := common.CHAIN
	t.Cleanup(func() { common.CHAIN = old })
	for _, tc := range []struct {
		chain  string
		params *chaincfg.Params
		height int
	}{
		{"regtest", &chaincfg.RegressionNetParams, 0},
		{"mainnet", &chaincfg.MainNetParams, 840000},
		{"testnet", &chaincfg.TestNet4Params, 30562},
	} {
		common.CHAIN = tc.chain
		if got := NewIndexer(nil, tc.params, false).enableHeight; got != tc.height {
			t.Errorf("%s activation=%d want=%d", tc.chain, got, tc.height)
		}
	}
}

func TestRegtestRuneSelfCheckKeepsHolderValidation(t *testing.T) {
	idx, _ := newHolderCheckIndexer(t)
	idx.chaincfgParam = &chaincfg.RegressionNetParams
	idx.height = 10
	id := &runestone.RuneId{Block: 1, Tx: 1}
	rune := runestone.Rune{Value: uint128.From64(42)}
	entry := &runestone.RuneEntry{RuneId: *id, Number: 1, Premine: uint128.From64(10), HolderCount: 1, SpacedRune: *runestone.NewSpacedRune(rune, 0)}
	idx.runeToIdTbl.SetToDB(&rune, id)
	idx.idToEntryTbl.Insert(id, entry)
	idx.runeIdAddressToBalanceTbl.Insert(holderRow(id, 1, 10))
	if !idx.CheckSelf() {
		t.Fatal("consistent regtest rune was rejected")
	}
	idx.runeIdAddressToBalanceTbl.Insert(holderRow(id, 1, 9))
	if idx.CheckSelf() {
		t.Fatal("regtest accepted inconsistent holder balances")
	}
}
