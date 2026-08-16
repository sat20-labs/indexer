package indexer

import "testing"

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
