package indexer

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDBBufferReaderBarrierDrainsReadersAndBlocksAdmission(t *testing.T) {
	mempool := NewMiniMemPool()
	mgr := &IndexerMgr{miniMempool: mempool}

	mempool.enterIndexerRead()
	mgr.rpcEnter()

	updateEntered := make(chan struct{})
	releaseUpdate := make(chan struct{})
	barrierDone := make(chan struct{})
	go func() {
		mgr.withDBBufferReaderBarrier(func() {
			close(updateEntered)
			<-releaseUpdate
		})
		close(barrierDone)
	}()

	select {
	case <-updateEntered:
		t.Fatal("update entered while mempool and RPC readers were active")
	case <-time.After(50 * time.Millisecond):
	}

	mempool.leaveIndexerRead()
	select {
	case <-updateEntered:
		t.Fatal("update entered while an RPC reader was active")
	case <-time.After(50 * time.Millisecond):
	}

	mgr.rpcLeft()
	select {
	case <-updateEntered:
	case <-time.After(time.Second):
		t.Fatal("barrier did not enter after existing readers drained")
	}

	newReaderEntered := make(chan struct{})
	go func() {
		mgr.rpcEnter()
		close(newReaderEntered)
		mgr.rpcLeft()
	}()
	select {
	case <-newReaderEntered:
		t.Fatal("new RPC reader entered while update held the write barrier")
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseUpdate)
	select {
	case <-barrierDone:
	case <-time.After(time.Second):
		t.Fatal("barrier did not exit")
	}
	select {
	case <-newReaderEntered:
	case <-time.After(time.Second):
		t.Fatal("RPC admission did not reopen after barrier")
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

	// The writer must wait for the mempool reader before acquiring the RPC
	// write lock, so the reader may safely perform an RPC-gated read.
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
		t.Fatal("DB barrier did not enter after mempool reader completed")
	}
	select {
	case <-barrierDone:
	case <-time.After(time.Second):
		t.Fatal("DB barrier did not exit")
	}
}

func TestRPCBarrierStressNeverOverlapsWriterAndReaders(t *testing.T) {
	mgr := &IndexerMgr{miniMempool: NewMiniMemPool()}
	var activeReaders int32
	var writerActive int32
	var failed int32

	var readers sync.WaitGroup
	for i := 0; i < 8; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for j := 0; j < 200; j++ {
				mgr.rpcEnter()
				atomic.AddInt32(&activeReaders, 1)
				if atomic.LoadInt32(&writerActive) != 0 {
					atomic.StoreInt32(&failed, 1)
				}
				atomic.AddInt32(&activeReaders, -1)
				mgr.rpcLeft()
			}
		}()
	}

	for i := 0; i < 40; i++ {
		mgr.withDBBufferReaderBarrier(func() {
			atomic.StoreInt32(&writerActive, 1)
			if atomic.LoadInt32(&activeReaders) != 0 {
				atomic.StoreInt32(&failed, 1)
			}
			atomic.StoreInt32(&writerActive, 0)
		})
	}
	readers.Wait()
	if atomic.LoadInt32(&failed) != 0 {
		t.Fatal("RPC reader overlapped an indexer writer")
	}
}
