package brc20

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/base"
	"github.com/sat20-labs/indexer/indexer/db"
	"github.com/sat20-labs/indexer/indexer/nft"
	"github.com/stretchr/testify/require"
)

func TestRegtestSkipsPublicCheckpoints(t *testing.T) {
	for _, params := range []*chaincfg.Params{&chaincfg.RegressionNetParams, &chaincfg.MainNetParams, &chaincfg.TestNet4Params} {
		t.Run(params.Name, func(t *testing.T) {
			idx := newRegtestCheckpointIndexer(t, params)
			check := func() {
				if params == &chaincfg.MainNetParams {
					idx.CheckPointWithBlockHeight(900000)
				} else {
					idx.CheckPointWithBlockHeight(30000)
				}
			}
			if params == &chaincfg.RegressionNetParams {
				require.NotPanics(t, check)
			} else {
				require.Panics(t, check, "public checkpoints must still reject missing tickers")
			}
		})
	}
}

func newRegtestCheckpointIndexer(t *testing.T, params *chaincfg.Params) *BRC20Indexer {
	t.Helper()
	kv := db.NewKVDBWithCache(t.TempDir(), 1)
	require.NotNil(t, kv)
	t.Cleanup(func() { require.NoError(t, kv.Close()) })
	wb := kv.NewWriteBatch()
	require.NoError(t, db.SetDB([]byte(base.SyncStatsKey), &base.SyncStats{SyncHeight: 108237}, wb))
	require.NoError(t, wb.Flush())
	wb.Close()
	b := base.NewBaseIndexer(kv, params, 0, 12)
	b.Init()
	n := nft.NewNftIndexer(kv)
	n.Init(b, nil)
	idx := NewIndexer(kv, false)
	idx.Init(n)
	return idx
}

func TestRegtestSelfCheckKeepsBalanceValidation(t *testing.T) {
	idx := newRegtestCheckpointIndexer(t, &chaincfg.RegressionNetParams)
	require.True(t, idx.CheckSelf(), "an empty regtest does not require the public ordi ticker")
	ticker := &common.BRC20Ticker{Name: "local", Minted: *common.NewDecimal(1, 0)}
	idx.status.TickerCount = 1
	idx.tickerAdded = []*common.BRC20Ticker{ticker}
	idx.tickerMap = map[string]*BRC20TickInfo{"local": {Ticker: ticker}}
	require.False(t, idx.CheckSelf(), "minted supply without holder balances must still fail")
}
