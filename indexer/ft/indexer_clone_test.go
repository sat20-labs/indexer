package ft

import (
	"testing"

	"github.com/sat20-labs/indexer/common"
)

func newCloneTestFTIndexer(ticker *common.Ticker) *FTIndexer {
	indexer := NewOrdxIndexer(nil)
	indexer.tickerAdded = map[string]*common.Ticker{ticker.Name: ticker}
	indexer.tickerMap = make(map[string]*TickInfo)
	indexer.holderInfo = make(map[uint64]*HolderInfo)
	indexer.utxoMap = make(map[string]map[uint64]int64)
	indexer.holderActionList = nil
	indexer.unbindHistory = nil
	indexer.freezeHistory = nil
	indexer.freezeTouched = make(map[string]*common.FreezeState)
	indexer.freezeDeleted = make(map[string]*common.FreezeState)
	indexer.freezeStates = make(map[string]map[uint64]*common.FreezeState)
	indexer.pendingHistoricalFreezes = make(map[int][]*common.FreezeDirective)
	indexer.pendingHistoricalKeys = make(map[string]bool)
	indexer.reloadFreezeDirectives = make(map[string]*common.FreezeDirective)
	indexer.freezeAuthoritySnapshot = make(map[string]uint64)
	return indexer
}

func TestCloneOwnsTickerAddedState(t *testing.T) {
	ticker := &common.Ticker{
		Name:          "asset",
		TotalMinted:   100,
		TotalUnbound:  10,
		TotalFrozen:   20,
		TotalUnfrozen: 5,
		TotalBurned:   2,
	}
	source := newCloneTestFTIndexer(ticker)
	clone := source.Clone(nil)

	ticker.TotalMinted = 200
	ticker.TotalUnbound = 30
	ticker.TotalFrozen = 40
	ticker.TotalUnfrozen = 15
	ticker.TotalBurned = 12

	got := clone.tickerAdded["asset"]
	if got == ticker {
		t.Fatal("snapshot shares tickerAdded pointer with live indexer")
	}
	if got.TotalMinted != 100 || got.TotalUnbound != 10 || got.TotalFrozen != 20 ||
		got.TotalUnfrozen != 5 || got.TotalBurned != 2 {
		t.Fatalf("snapshot ticker changed after live mutation: %+v", got)
	}
}

func TestSubtractPreservesTickerUpdatedAfterSnapshot(t *testing.T) {
	ticker := &common.Ticker{Name: "asset", TotalMinted: 100}
	source := newCloneTestFTIndexer(ticker)
	snapshot := source.Clone(nil)

	ticker.TotalMinted = 200
	source.tickerAdded["asset"] = ticker
	source.Subtract(snapshot)

	got := source.tickerAdded["asset"]
	if got == nil {
		t.Fatal("Subtract removed a ticker changed after the snapshot")
	}
	if got.TotalMinted != 200 {
		t.Fatalf("live TotalMinted = %d, want 200", got.TotalMinted)
	}
}

func TestSubtractRemovesUnchangedTickerSnapshot(t *testing.T) {
	ticker := &common.Ticker{Name: "asset", TotalMinted: 100}
	source := newCloneTestFTIndexer(ticker)
	snapshot := source.Clone(nil)

	source.Subtract(snapshot)
	if _, ok := source.tickerAdded["asset"]; ok {
		t.Fatal("Subtract retained an unchanged flushed ticker")
	}
}
