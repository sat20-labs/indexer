package indexer

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
	baseindex "github.com/sat20-labs/indexer/indexer/base"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
)

func newExactOutputTestManager(t *testing.T) *IndexerMgr {
	t.Helper()
	kv := indexdb.NewKVDBWithCache(filepath.Join(t.TempDir(), "base"), 1)
	if kv == nil {
		t.Fatal("open temporary Pebble Base database")
	}
	base := baseindex.NewBaseIndexer(kv, &chaincfg.RegressionNetParams, 0, 12)
	base.Init()
	t.Cleanup(func() {
		base.CloseSilentPayViews()
		if err := kv.Close(); err != nil {
			t.Errorf("close temporary Base database: %v", err)
		}
	})
	return &IndexerMgr{
		base:        base,
		rpcService:  baseindex.NewRpcIndexer(base),
		miniMempool: NewMiniMemPool(),
	}
}

func commitExactPlainTestOutput(pool *MiniMemPool, seed byte, value int64) (string, *wire.MsgTx) {
	tx := makeMempoolTestTx(wire.OutPoint{Hash: chainhash.Hash{seed}, Index: 0}, value)
	pool.mutex.Lock()
	pool.admitTransactionLocked(tx)
	pool.mutex.Unlock()

	output := common.NewTxOutput(value)
	output.OutValue.PkScript = append([]byte(nil), tx.TxOut[0].PkScript...)
	pool.commitMempoolOutputs(tx, []*common.TxOutput{output}, []bool{false})
	return tx.TxID() + ":0", tx
}

func TestGetTxOutputWithUtxoV3UsesOnlyCachedClassifiedPlainOutput(t *testing.T) {
	mgr := newExactOutputTestManager(t)

	t.Run("reads exact classified 600 sat plain output", func(t *testing.T) {
		pool := NewMiniMemPool()
		mgr.miniMempool = pool
		outpoint, _ := commitExactPlainTestOutput(pool, 1, 600)

		got := mgr.GetTxOutputWithUtxoV3(outpoint, false)
		if got == nil {
			t.Fatal("V3 missed a cached, classified plain mempool output")
		}
		if got.OutPoint != outpoint || got.Value != 600 || !bytes.Equal(got.PkScript, []byte{0x51}) || len(got.Assets) != 0 {
			t.Fatalf("unexpected exact plain output: %+v", got)
		}
	})

	t.Run("unknown output is rejected", func(t *testing.T) {
		pool := NewMiniMemPool()
		mgr.miniMempool = pool
		tx := makeMempoolTestTx(wire.OutPoint{Hash: chainhash.Hash{2}, Index: 0}, 600)
		pool.mutex.Lock()
		pool.admitTransactionLocked(tx)
		pool.mutex.Unlock()

		if got := mgr.GetTxOutputWithUtxoV3(tx.TxID()+":0", false); got != nil {
			t.Fatalf("unknown output was exposed: %+v", got)
		}
	})

	t.Run("known non-plain output is rejected", func(t *testing.T) {
		pool := NewMiniMemPool()
		mgr.miniMempool = pool
		tx := makeMempoolTestTx(wire.OutPoint{Hash: chainhash.Hash{3}, Index: 0}, 600)
		pool.mutex.Lock()
		pool.admitTransactionLocked(tx)
		pool.mutex.Unlock()
		output := common.NewTxOutput(600)
		output.OutValue.PkScript = append([]byte(nil), tx.TxOut[0].PkScript...)
		pool.commitMempoolOutputs(tx, []*common.TxOutput{output}, []bool{true})

		if got := mgr.GetTxOutputWithUtxoV3(tx.TxID()+":0", false); got != nil {
			t.Fatalf("known non-plain output was exposed: %+v", got)
		}
	})

	t.Run("spent cached plain output is rejected", func(t *testing.T) {
		pool := NewMiniMemPool()
		mgr.miniMempool = pool
		outpoint, parent := commitExactPlainTestOutput(pool, 4, 600)
		child := makeMempoolTestTx(wire.OutPoint{Hash: parent.TxHash(), Index: 0}, 500)
		pool.mutex.Lock()
		pool.admitTransactionLocked(child)
		pool.mutex.Unlock()

		if got := mgr.GetTxOutputWithUtxoV3(outpoint, false); got != nil {
			t.Fatalf("spent cached plain output was exposed: %+v", got)
		}
	})

	t.Run("confirmed-spent cached plain output is rejected", func(t *testing.T) {
		pool := NewMiniMemPool()
		mgr.miniMempool = pool
		outpoint, parent := commitExactPlainTestOutput(pool, 5, 600)
		child := makeMempoolTestTx(wire.OutPoint{Hash: parent.TxHash(), Index: 0}, 500)
		pool.mutex.Lock()
		pool.admitTransactionLocked(child)
		pool.confirmTransactionLocked(child)
		pool.mutex.Unlock()

		if got := mgr.GetTxOutputWithUtxoV3(outpoint, false); got != nil {
			t.Fatalf("confirmed-spent cached plain output was exposed: %+v", got)
		}
	})

	t.Run("returned output does not alias cached script", func(t *testing.T) {
		pool := NewMiniMemPool()
		mgr.miniMempool = pool
		outpoint, _ := commitExactPlainTestOutput(pool, 6, 600)

		first := mgr.GetTxOutputWithUtxoV3(outpoint, false)
		if first == nil {
			t.Fatal("V3 missed cached output before clone check")
		}
		first.PkScript[0] = 0x00
		second := mgr.GetTxOutputWithUtxoV3(outpoint, false)
		if second == nil || !bytes.Equal(second.PkScript, []byte{0x51}) {
			t.Fatalf("caller mutation leaked into cached output: got %+v", second)
		}
	})
}
