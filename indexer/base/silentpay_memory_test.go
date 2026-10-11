package base

import (
	"bytes"
	"context"
	"runtime"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/indexer/common"
)

// Reuse the existing state fixture; only the number of public outputs changes.
func spMemoryCandidate(height, count int) *common.Block {
	block := spTestCandidate(height, 57)
	template := block.Transactions[1].Outputs[0]
	outputs := make([]*common.TxOutputV2, count)
	for i := range outputs {
		out := *template
		out.TxOutIndex = i * 2
		out.OutValue.PkScript = append([]byte(nil), template.OutValue.PkScript...)
		outputs[i] = &out
	}
	block.Transactions[1].Outputs = outputs
	return block
}

func TestSilentPayCloneDoesNotAllocatePerOutput(t *testing.T) {
	s := newSPTestState(t)
	if err := s.process(spMemoryCandidate(0, 4096)); err != nil {
		t.Fatal(err)
	}
	allocs := testing.AllocsPerRun(3, func() {
		frozen := s.clone()
		if len(frozen.blocks) != 1 || len(frozen.live) != 1 {
			t.Fatal("clone lost pending data")
		}
		runtime.KeepAlive(frozen)
	})
	t.Logf("clone allocations for 4096 outputs: %.0f", allocs)
	if allocs > 128 {
		t.Fatalf("clone expanded/copied immutable outputs: %.0f allocations", allocs)
	}
}

func TestSilentPaySparseSpendDoesNotExpandAllSourceOutputs(t *testing.T) {
	s := newSPTestState(t)
	origin := spMemoryCandidate(0, 4096)
	if err := s.process(origin); err != nil {
		t.Fatal(err)
	}
	spCommitState(t, s, SPAnchor{Height: 0, BlockHash: origin.Hash})
	spend := spTestSpend(origin, 1, 0)
	spend.Transactions = append(spend.Transactions, spTestSpend(origin, 1, 4095).Transactions[1])
	liveKey := string(spPositionKey(spLivePrefix, 0, 1))
	counter := &spArchiveReadCounter{KVDB: s.db}
	s.db = counter
	allocs := testing.AllocsPerRun(3, func() {
		counter.reads = 0
		if err := s.process(spend); err != nil {
			t.Fatal(err)
		}
		if counter.reads != 1 {
			t.Fatalf("one source was read %d times", counter.reads)
		}
		bits := s.live[liveKey].bitmap
		if len(bits) != 512 || bits[0] != 0xfe || bits[511] != 0x7f {
			t.Fatal("sparse spend did not update exactly the requested outputs")
		}
		// Each measurement starts from the same committed source, not a
		// double spend. No production records or real chain data are touched.
		delete(s.blocks, 1)
		delete(s.live, liveKey)
	})
	t.Logf("sparse spend allocations for 4096 source outputs: %.0f", allocs)
	if allocs > 256 {
		t.Fatalf("sparse spend materialized unrelated outputs: %.0f allocations", allocs)
	}
}

func TestSilentPayRPCCloneDoesNotRetainPendingRecords(t *testing.T) {
	s := newSPTestState(t)
	b := NewBaseIndexer(s.db, &chaincfg.RegressionNetParams, 0, 10)
	b.Init()
	t.Cleanup(b.CloseSilentPayViews)
	if err := b.silentpay.process(spMemoryCandidate(0, 1024)); err != nil {
		t.Fatal(err)
	}
	rpc := NewRpcIndexer(b)
	if rpc.silentpay == nil || rpc.silentpay.views != b.silentpay.views {
		t.Fatal("RPC clone lost shared committed-view support")
	}
	if len(rpc.silentpay.blocks) != 0 || len(rpc.silentpay.live) != 0 {
		t.Fatal("RPC-only clone pins pending Archive/Live memory")
	}
	if len(b.silentpay.blocks) != 1 || len(b.silentpay.live) != 1 {
		t.Fatal("discarding RPC deltas modified the compiling Base")
	}
}

func TestSilentPayCompactRecordsSurviveCommitAndRelease(t *testing.T) {
	s := newSPTestState(t)
	origin := spMemoryCandidate(0, 4096)
	candidate, err := spCandidate(origin.Transactions[1], 1)
	if err != nil || candidate == nil {
		t.Fatalf("candidate: %v", err)
	}
	want, err := spEncodeTransaction(*candidate)
	if err != nil {
		t.Fatal(err)
	}
	// Existing v1 encoding: version + txid + T + CompactSize count,
	// then one-byte vout deltas and full 32-byte keys for this fixture.
	if len(want) != 1+32+33+3+4096*33 {
		t.Fatalf("unexpected record expansion: %d bytes", len(want))
	}
	if err := s.process(origin); err != nil {
		t.Fatal(err)
	}
	frozen := s.clone()
	if err := s.process(spTestSpend(origin, 1, 0)); err != nil {
		t.Fatal(err)
	}
	s.subtract(frozen)
	spCommitState(t, frozen, SPAnchor{Height: 0, BlockHash: origin.Hash})
	if len(frozen.blocks) != 0 || len(frozen.live) != 0 || len(s.blocks) != 1 {
		t.Fatal("committed deltas retained or newer block was discarded")
	}
	spCommitState(t, s, SPAnchor{Height: 1, BlockHash: spTestHash(201)})
	if len(s.blocks) != 0 || len(s.live) != 0 {
		t.Fatal("successful commit retained pending records")
	}
	got, err := s.db.Read(spPositionKey(spArchivePrefix, 0, 1))
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("archive byte format changed: %v", err)
	}
	meta, err := s.createSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	page, err := s.scan(context.Background(), SPScanRequest{Mode: "live", SnapshotID: meta.SnapshotID, ToHeight: -1})
	if err != nil || !page.Done || len(page.Blocks) != 1 {
		t.Fatalf("scan after release: %+v %v", page, err)
	}
	tx := page.Blocks[0].Transactions[0]
	if len(tx.P2TROutputs) != 4096 || tx.UnspentBitmap[0] != 0xfe {
		t.Fatal("release truncated historical matching outputs or lost Live state")
	}
}
