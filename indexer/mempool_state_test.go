package indexer

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

func makeMempoolTestTx(previous wire.OutPoint, value int64) *wire.MsgTx {
	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxIn(wire.NewTxIn(&previous, nil, nil))
	tx.AddTxOut(wire.NewTxOut(value, []byte{0x51}))
	return tx
}

func TestMempoolConflictReplacementEvictsDescendants(t *testing.T) {
	pool := NewMiniMemPool()
	root := wire.OutPoint{Hash: chainhash.Hash{1}, Index: 0}

	first := makeMempoolTestTx(root, 1_000)
	child := makeMempoolTestTx(wire.OutPoint{Hash: first.TxHash(), Index: 0}, 900)
	grandchild := makeMempoolTestTx(wire.OutPoint{Hash: child.TxHash(), Index: 0}, 800)
	replacement := makeMempoolTestTx(root, 700)

	pool.mutex.Lock()
	pool.admitTransactionLocked(first)
	pool.admitTransactionLocked(child)
	pool.admitTransactionLocked(grandchild)
	pool.admitTransactionLocked(replacement)

	rootKey := root.String()
	owner := pool.spentByOutpoint[rootKey]
	_, firstExists := pool.txMap[first.TxID()]
	_, childExists := pool.txMap[child.TxID()]
	_, grandchildExists := pool.txMap[grandchild.TxID()]
	_, replacementExists := pool.txMap[replacement.TxID()]
	pool.mutex.Unlock()

	if owner != replacement.TxID() {
		t.Fatalf("root owner = %s, want replacement %s", owner, replacement.TxID())
	}
	if firstExists || childExists || grandchildExists {
		t.Fatalf("replacement left invalid branch in mempool: first=%v child=%v grandchild=%v", firstExists, childExists, grandchildExists)
	}
	if !replacementExists {
		t.Fatal("replacement transaction was not admitted")
	}
}

func TestMempoolConfirmedConflictEvictsLocalBranch(t *testing.T) {
	pool := NewMiniMemPool()
	root := wire.OutPoint{Hash: chainhash.Hash{2}, Index: 1}

	local := makeMempoolTestTx(root, 1_000)
	child := makeMempoolTestTx(wire.OutPoint{Hash: local.TxHash(), Index: 0}, 900)
	confirmed := makeMempoolTestTx(root, 600)

	pool.mutex.Lock()
	pool.admitTransactionLocked(local)
	pool.admitTransactionLocked(child)
	pool.confirmTransactionLocked(confirmed)

	_, localExists := pool.txMap[local.TxID()]
	_, childExists := pool.txMap[child.TxID()]
	owner := pool.spentByOutpoint[root.String()]
	_, confirmedSpent := pool.confirmedSpent[root.String()]
	pool.mutex.Unlock()

	if localExists || childExists {
		t.Fatalf("confirmed conflict left invalid branch: local=%v child=%v", localExists, childExists)
	}
	if owner != "" {
		t.Fatalf("confirmed input still owned by mempool tx %s", owner)
	}
	if !confirmedSpent {
		t.Fatal("confirmed input was not retained as spent until index catch-up")
	}
	if !pool.IsSpent(root.String()) {
		t.Fatal("confirmed-but-not-indexed input must remain unavailable")
	}
}

func TestMempoolRemovalDoesNotReleaseReplacementOwner(t *testing.T) {
	pool := NewMiniMemPool()
	root := wire.OutPoint{Hash: chainhash.Hash{3}, Index: 2}

	first := makeMempoolTestTx(root, 1_000)
	replacement := makeMempoolTestTx(root, 900)

	pool.mutex.Lock()
	pool.admitTransactionLocked(first)
	pool.admitTransactionLocked(replacement)
	pool.removeTransactionLocked(first.TxID(), true, true)
	owner := pool.spentByOutpoint[root.String()]
	pool.mutex.Unlock()

	if owner != replacement.TxID() {
		t.Fatalf("removing stale tx released replacement ownership: got %s want %s", owner, replacement.TxID())
	}
}
