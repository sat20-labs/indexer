package indexer

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestDBBufferReaderBarrierDrainsRPCAndMempoolReaders(t *testing.T) {
	mempool := NewMiniMemPool()
	mgr := &IndexerMgr{miniMempool: mempool}

	// Keep one mempool reader and one RPC reader active. The barrier must
	// first drain the mempool reader before it closes RPC admission, then
	// drain the already-active RPC reader before entering the update.
	mempool.enterIndexerRead()
	atomic.StoreInt32(&mgr.rpcProcessing, 1)

	updateEntered := make(chan struct{})
	barrierDone := make(chan struct{})
	go func() {
		mgr.withDBBufferReaderBarrier(func() {
			close(updateEntered)
		})
		close(barrierDone)
	}()

	select {
	case <-updateEntered:
		t.Fatal("update entered while a mempool indexer reader was active")
	case <-time.After(50 * time.Millisecond):
	}

	// Once the mempool reader drains, the barrier may close RPC admission.
	mempool.leaveIndexerRead()
	deadline := time.After(time.Second)
	for atomic.LoadInt32(&mgr.reloading) == 0 {
		select {
		case <-deadline:
			t.Fatal("barrier did not block new RPC readers after mempool readers drained")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	select {
	case <-updateEntered:
		t.Fatal("update entered while an RPC reader was active")
	case <-time.After(50 * time.Millisecond):
	}

	atomic.StoreInt32(&mgr.rpcProcessing, 0)
	select {
	case <-barrierDone:
	case <-time.After(time.Second):
		t.Fatal("barrier did not proceed after existing readers drained")
	}

	if got := atomic.LoadInt32(&mgr.reloading); got != 0 {
		t.Fatalf("reloading = %d after barrier, want 0", got)
	}

	readerEntered := make(chan struct{})
	go func() {
		mempool.enterIndexerRead()
		close(readerEntered)
		mempool.leaveIndexerRead()
	}()
	select {
	case <-readerEntered:
	case <-time.After(time.Second):
		t.Fatal("mempool readers remained paused after barrier")
	}
}

func TestDBBufferReaderBarrierDoesNotDeadlockMempoolRPCRead(t *testing.T) {
	mempool := NewMiniMemPool()
	mgr := &IndexerMgr{miniMempool: mempool}

	// Reproduce the production lock dependency:
	// txBroadcasted holds indexerReadBarrier.RLock while rebuildTxOutput may
	// enter an RPC-gated indexer read. If withDBBufferReaderBarrier sets
	// reloading before waiting for the write lock, the two goroutines wait on
	// each other forever.
	readerHasBarrier := make(chan struct{})
	allowRPCRead := make(chan struct{})
	readerDone := make(chan struct{})
	go func() {
		mempool.enterIndexerRead()
		defer mempool.leaveIndexerRead()
		close(readerHasBarrier)
		<-allowRPCRead
		mgr.rpcEnter()
		mgr.rpcLeft()
		close(readerDone)
	}()

	select {
	case <-readerHasBarrier:
	case <-time.After(time.Second):
		t.Fatal("mempool reader did not acquire reader barrier")
	}

	updateEntered := make(chan struct{})
	barrierDone := make(chan struct{})
	go func() {
		mgr.withDBBufferReaderBarrier(func() {
			close(updateEntered)
		})
		close(barrierDone)
	}()

	// Give the writer time to contend on the reader barrier. With the broken
	// ordering, reloading is already set here and the following rpcEnter will
	// deadlock. With the fixed ordering, reloading is still zero until this
	// reader finishes.
	time.Sleep(20 * time.Millisecond)
	close(allowRPCRead)

	select {
	case <-readerDone:
	case <-time.After(time.Second):
		t.Fatal("mempool reader deadlocked entering RPC read while DB barrier was pending")
	}

	select {
	case <-updateEntered:
	case <-time.After(time.Second):
		t.Fatal("DB barrier did not enter update after mempool reader completed")
	}

	select {
	case <-barrierDone:
	case <-time.After(time.Second):
		t.Fatal("DB barrier did not exit")
	}

	if got := atomic.LoadInt32(&mgr.reloading); got != 0 {
		t.Fatalf("reloading = %d after barrier, want 0", got)
	}
}
