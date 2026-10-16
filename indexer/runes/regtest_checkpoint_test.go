package runes

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/indexer/indexer/base"
	"github.com/stretchr/testify/require"
)

func TestRegtestSkipsPublicCheckpoints(t *testing.T) {
	for _, params := range []*chaincfg.Params{&chaincfg.RegressionNetParams, &chaincfg.MainNetParams, &chaincfg.TestNet4Params} {
		t.Run(params.Name, func(t *testing.T) {
			idx, kv := newHolderCheckIndexer(t)
			idx.chaincfgParam = params
			idx.baseIndexer = base.NewBaseIndexer(kv, params, 0, 12)
			check := func() {
				if params == &chaincfg.MainNetParams {
					idx.CheckPointWithBlockHeight(840000)
				} else {
					idx.CheckPointWithBlockHeight(30562)
				}
			}
			if params == &chaincfg.RegressionNetParams {
				require.NotPanics(t, check)
			} else {
				require.Panics(t, check, "public checkpoints must still require their reference rune")
			}
		})
	}
}
