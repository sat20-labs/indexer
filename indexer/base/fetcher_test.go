package base

import (
	"sync"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/indexer/common"
)

func TestSendBlockOrStopDeliversBlock(t *testing.T) {
	blocks := make(chan *common.Block, 1)
	stop := make(chan struct{})
	block := &common.Block{Height: 123}
	if !sendBlockOrStop(blocks, block, stop) {
		t.Fatal("send was cancelled unexpectedly")
	}
	if got := <-blocks; got != block {
		t.Fatalf("delivered block=%p, want %p", got, block)
	}
}

func TestSendBlockOrStopCancelsBlockedSend(t *testing.T) {
	blocks := make(chan *common.Block)
	stop := make(chan struct{})
	done := make(chan bool, 1)
	go func() {
		done <- sendBlockOrStop(blocks, &common.Block{Height: 123}, stop)
	}()

	select {
	case <-done:
		t.Fatal("send returned before delivery or cancellation")
	case <-time.After(30 * time.Millisecond):
	}
	close(stop)
	select {
	case delivered := <-done:
		if delivered {
			t.Fatal("blocked send reported successful delivery after cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("blocked send did not observe cancellation")
	}
}

func TestPrefetchedBlockCacheLifecycle(t *testing.T) {
	indexer := NewBaseIndexer(nil, &chaincfg.TestNet4Params, 0, 10)
	block := &common.Block{Height: 321}

	indexer.setPrefetchedBlock(block)
	if got := indexer.GetPrefetchedBlock(block.Height); got != block {
		t.Fatalf("prefetched block=%p, want %p", got, block)
	}
	indexer.removePrefetchedBlock(block.Height)
	if got := indexer.GetPrefetchedBlock(block.Height); got != nil {
		t.Fatalf("removed block still cached: %p", got)
	}
}

func TestDrainBlocksChanClearsPrefetchedIndex(t *testing.T) {
	indexer := NewBaseIndexer(nil, &chaincfg.TestNet4Params, 0, 10)
	blocks := []*common.Block{{Height: 10}, {Height: 11}}
	for _, block := range blocks {
		indexer.setPrefetchedBlock(block)
		indexer.blocksChan <- block
	}

	indexer.drainBlocksChan()
	if len(indexer.blocksChan) != 0 {
		t.Fatalf("blocksChan len=%d, want 0", len(indexer.blocksChan))
	}
	for _, block := range blocks {
		if got := indexer.GetPrefetchedBlock(block.Height); got != nil {
			t.Fatalf("height %d remains in prefetched index", block.Height)
		}
	}
}

func TestPrefetchedBlockCacheConcurrentAccess(t *testing.T) {
	indexer := NewBaseIndexer(nil, &chaincfg.TestNet4Params, 0, 10)
	const goroutines = 8
	const iterations = 500
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(base int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				height := base*iterations + i
				block := &common.Block{Height: height}
				indexer.setPrefetchedBlock(block)
				_ = indexer.GetPrefetchedBlock(height)
				indexer.removePrefetchedBlock(height)
			}
		}(g)
	}
	wg.Wait()
}
