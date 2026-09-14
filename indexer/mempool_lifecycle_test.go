package indexer

import (
	"testing"
	"time"

	"github.com/btcsuite/btcd/wire"
)

func TestMiniMempoolStopWaitsForOwnedWorkers(t *testing.T) {
	pool := NewMiniMemPool()
	stop := make(chan struct{})
	workerExited := make(chan struct{})

	pool.lifecycleMutex.Lock()
	pool.running = true
	pool.syncing = true
	pool.stopChan = stop
	pool.workerWG.Add(1)
	pool.lifecycleMutex.Unlock()

	go func() {
		defer pool.workerWG.Done()
		<-stop
		close(workerExited)
	}()

	pool.Stop()

	select {
	case <-workerExited:
	default:
		t.Fatal("Stop returned before the owned worker exited")
	}

	pool.lifecycleMutex.Lock()
	defer pool.lifecycleMutex.Unlock()
	if pool.running {
		t.Fatal("mempool remained running after Stop")
	}
	if pool.syncing {
		t.Fatal("mempool remained marked syncing after Stop")
	}
	if pool.stopChan != nil {
		t.Fatal("mempool retained the stopped lifecycle channel")
	}
}

func TestMiniMempoolStopDrainsAdmittedPeerCallback(t *testing.T) {
	pool := NewMiniMemPool()
	stop := make(chan struct{})

	pool.lifecycleMutex.Lock()
	pool.running = true
	pool.stopChan = stop
	pool.lifecycleMutex.Unlock()

	if !pool.beginPeerCallback(stop) {
		t.Fatal("peer callback was not admitted while mempool was running")
	}

	releaseCallback := make(chan struct{})
	callbackDone := make(chan struct{})
	go func() {
		<-releaseCallback
		pool.endPeerCallback()
		close(callbackDone)
	}()

	stopDone := make(chan struct{})
	go func() {
		pool.Stop()
		close(stopDone)
	}()

	deadline := time.Now().Add(time.Second)
	for {
		pool.lifecycleMutex.Lock()
		running := pool.running
		pool.lifecycleMutex.Unlock()
		if !running {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Stop did not close peer callback admission")
		}
		time.Sleep(time.Millisecond)
	}

	select {
	case <-stopDone:
		t.Fatal("Stop returned before an admitted peer callback exited")
	default:
	}

	close(releaseCallback)
	select {
	case <-callbackDone:
	case <-time.After(time.Second):
		t.Fatal("admitted peer callback did not exit")
	}
	select {
	case <-stopDone:
	case <-time.After(time.Second):
		t.Fatal("Stop did not return after the admitted peer callback exited")
	}

	if pool.beginPeerCallback(stop) {
		pool.endPeerCallback()
		t.Fatal("stale peer callback was admitted after Stop")
	}
}

func TestMiniMempoolTxBroadcastedTakesIndexerReadBeforeProcessingMutex(t *testing.T) {
	pool := NewMiniMemPool()
	tx := wire.NewMsgTx(wire.TxVersion)
	txID := tx.TxID()

	// Make txBroadcasted return immediately after it acquires its locks, so the
	// test only observes lock ordering and never needs a live indexer instance.
	pool.mutex.Lock()
	pool.txMap[txID] = tx
	pool.classifiedTxMap[txID] = true
	pool.mutex.Unlock()

	// Hold processingMutex. With the required ordering, txBroadcasted must
	// acquire indexerReadBarrier.RLock before it blocks on processingMutex.
	pool.processingMutex.Lock()
	txDone := make(chan struct{})
	go func() {
		pool.txBroadcasted(tx)
		close(txDone)
	}()

	readLockObserved := false
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if !pool.indexerReadBarrier.TryLock() {
			readLockObserved = true
			break
		}
		pool.indexerReadBarrier.Unlock()
		time.Sleep(time.Millisecond)
	}

	pool.processingMutex.Unlock()
	if !readLockObserved {
		select {
		case <-txDone:
		case <-time.After(time.Second):
		}
		t.Fatal("txBroadcasted waited on processingMutex before acquiring the indexer read barrier")
	}

	select {
	case <-txDone:
	case <-time.After(time.Second):
		t.Fatal("txBroadcasted did not complete after processingMutex was released")
	}
}
