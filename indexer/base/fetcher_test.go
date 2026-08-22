package base

import (
	"testing"
	"time"

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
