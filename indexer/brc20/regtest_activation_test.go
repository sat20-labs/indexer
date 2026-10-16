package brc20

import (
	"github.com/sat20-labs/indexer/common"
	"testing"
)

func TestRegtestBRC20ActivationPreservesPublicNetworks(t *testing.T) {
	old := common.CHAIN
	t.Cleanup(func() { common.CHAIN = old })
	for _, tc := range []struct {
		chain  string
		height int
	}{{"regtest", 0}, {"mainnet", 779832}, {"testnet", 27228}} {
		common.CHAIN = tc.chain
		if got := NewIndexer(nil, false).enableHeight; got != tc.height {
			t.Errorf("%s activation=%d want=%d", tc.chain, got, tc.height)
		}
	}
}
