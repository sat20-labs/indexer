package ft

import (
	"github.com/sat20-labs/indexer/common"
	"testing"
)

func TestRegtestORDXActivationPreservesPublicNetworks(t *testing.T) {
	old := common.CHAIN
	t.Cleanup(func() { common.CHAIN = old })
	for _, tc := range []struct {
		chain  string
		height int
	}{{"regtest", 0}, {"mainnet", 827307}, {"testnet", 28883}} {
		common.CHAIN = tc.chain
		if got := NewOrdxIndexer(nil).enableHeight; got != tc.height {
			t.Errorf("%s activation=%d want=%d", tc.chain, got, tc.height)
		}
	}
}
