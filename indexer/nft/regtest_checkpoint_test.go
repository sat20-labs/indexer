package nft

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/indexer/indexer/base"
	"github.com/stretchr/testify/require"
)

func TestRegtestSkipsPublicCheckpoints(t *testing.T) {
	for _, params := range []*chaincfg.Params{&chaincfg.RegressionNetParams, &chaincfg.MainNetParams, &chaincfg.TestNet4Params} {
		t.Run(params.Name, func(t *testing.T) {
			kv := openNftPersistenceTestDB(t)
			idx := NewNftIndexer(kv)
			idx.Init(base.NewBaseIndexer(kv, params, 0, 12), nil)
			check := func() {
				if params == &chaincfg.MainNetParams {
					idx.CheckPointWithBlockHeight(780000)
				} else {
					idx.CheckPointWithBlockHeight(30000)
				}
			}
			if params == &chaincfg.RegressionNetParams {
				require.NotPanics(t, check)
			} else {
				require.Panics(t, check, "public checkpoints must still reject missing inscriptions")
			}
		})
	}
}
