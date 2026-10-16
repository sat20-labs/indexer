package ft

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/indexer/indexer/base"
	"github.com/sat20-labs/indexer/indexer/db"
	"github.com/sat20-labs/indexer/indexer/nft"
	"github.com/stretchr/testify/require"
)

func TestRegtestSkipsPublicCheckpoints(t *testing.T) {
	for _, params := range []*chaincfg.Params{&chaincfg.RegressionNetParams, &chaincfg.MainNetParams, &chaincfg.TestNet4Params} {
		t.Run(params.Name, func(t *testing.T) {
			kv := db.NewKVDBWithCache(t.TempDir(), 1)
			require.NotNil(t, kv)
			t.Cleanup(func() { require.NoError(t, kv.Close()) })
			n := nft.NewNftIndexer(kv)
			n.Init(base.NewBaseIndexer(kv, params, 0, 12), nil)
			idx := newTestFTIndexer()
			idx.db, idx.nftIndexer = kv, n
			check := func() {
				if params == &chaincfg.MainNetParams {
					idx.CheckPointWithBlockHeight(900000)
				} else {
					idx.CheckPointWithBlockHeight(28883)
				}
			}
			if params == &chaincfg.RegressionNetParams {
				require.NotPanics(t, check)
			} else {
				require.Panics(t, check, "public checkpoints must still require their reference ticker")
			}
		})
	}
}
