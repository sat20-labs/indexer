package exotic

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/indexer/indexer/base"
	"github.com/stretchr/testify/require"
)

func TestRegtestSkipsPublicCheckpoints(t *testing.T) {
	for _, params := range []*chaincfg.Params{&chaincfg.RegressionNetParams, &chaincfg.MainNetParams, &chaincfg.TestNet4Params} {
		t.Run(params.Name, func(t *testing.T) {
			kv := openExoticStateTestDB(t)
			idx := newExoticStateTestIndexer(kv)
			idx.baseIndexer = base.NewBaseIndexer(kv, params, 0, 12)
			check := func() { idx.CheckPointWithBlockHeight(10000) }
			if params == &chaincfg.RegressionNetParams {
				require.NotPanics(t, check)
			} else {
				require.Panics(t, check, "public checkpoints must still require their reference assets")
			}
		})
	}
}
