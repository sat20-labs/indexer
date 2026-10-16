package indexer

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/config"
)

func TestRegtestIndexerUsesActualNetwork(t *testing.T) {
	previousInstance, previousChain := instance, common.CHAIN
	previousJubilee, previousSelfMint := common.Jubilee_Height, common.SELFMINT_ENABLE_HEIGHT
	t.Cleanup(func() {
		instance = previousInstance
		common.CHAIN = previousChain
		common.Jubilee_Height = previousJubilee
		common.SELFMINT_ENABLE_HEIGHT = previousSelfMint
	})
	for _, tc := range []struct {
		chain  string
		params *chaincfg.Params
	}{
		{"regtest", &chaincfg.RegressionNetParams},
		{"mainnet", &chaincfg.MainNetParams},
		{"testnet", &chaincfg.TestNet4Params},
		{"testnet4", &chaincfg.TestNet4Params},
	} {
		t.Run(tc.chain, func(t *testing.T) {
			instance = nil
			common.CHAIN = "mainnet"
			manager := NewIndexerMgr(&config.YamlConf{Chain: tc.chain, DB: config.DB{Path: t.TempDir()}})
			if manager.GetChainParam() != tc.params {
				t.Fatalf("chain %s selected %s", tc.chain, manager.GetChainParam().Name)
			}
			if tc.chain == "regtest" && (common.CHAIN != "regtest" || manager.ordFirstHeight != 0 || manager.ordxFirstHeight != 0) {
				t.Fatalf("regtest protocol classification=%s ord=%d ordx=%d", common.CHAIN, manager.ordFirstHeight, manager.ordxFirstHeight)
			}
		})
	}
}
