package base

import (
	"bytes"
	"errors"
	"testing"

	"github.com/sat20-labs/indexer/common"
)

func TestSilentPayCompactEncodingKeepsVoutVarintBoundaries(t *testing.T) {
	block := spMemoryCandidate(0, 4)
	vouts := []int{0, 252, 505, 66042}
	for i, vout := range vouts {
		block.Transactions[1].Outputs[i].TxOutIndex = vout
	}
	raw, count, err := spCandidateRecord(block.Transactions[1], 1)
	if err != nil || count != len(vouts) {
		t.Fatalf("encode candidate: count=%d err=%v", count, err)
	}
	// Existing v1 CompactSize deltas are 0, 252, 253 and 65537, followed
	// by the original full public key. Do not substitute Go Uvarint here.
	want := append([]byte(nil), raw[:66]...)
	want = append(want, 4)
	deltas := [][]byte{{0}, {252}, {253, 253, 0}, {254, 1, 0, 1, 0}}
	for i, delta := range deltas {
		want = append(want, delta...)
		want = append(want, block.Transactions[1].Outputs[i].OutValue.PkScript[2:]...)
	}
	if !bytes.Equal(raw, want) {
		t.Fatal("compact candidate changed the v1 output encoding")
	}
	tx, err := spDecodeTransaction(raw, 1)
	if err != nil {
		t.Fatal(err)
	}
	for i, vout := range vouts {
		if tx.P2TROutputs[i].Vout != uint32(vout) || cap(tx.P2TROutputs[i].OutputKey) != 32 {
			t.Fatalf("decoded output %d has wrong position or append capacity", i)
		}
	}
	for _, invalid := range [][]byte{raw[:len(raw)-1], append(append([]byte(nil), raw...), 0)} {
		if _, err := spDecodeTransaction(invalid, 1); !errors.Is(err, ErrSPCorrupt) {
			t.Fatalf("accepted incomplete/trailing record: %v", err)
		}
	}
}

func TestSilentPayGroupedSparseSpendsKeepOriginalInputOrder(t *testing.T) {
	s := newSPTestState(t)
	const sourceCount = 8
	origins := make([]*common.Block, sourceCount)
	for h := range origins {
		origin := spMemoryCandidate(h, 1024)
		origin.Transactions[1].TxId = spTestHash(byte(90 + h))
		origin.Hash = spTestHash(byte(110 + h))
		origins[h] = origin
		if err := s.process(origin); err != nil {
			t.Fatal(err)
		}
	}
	spCommitState(t, s, SPAnchor{Height: sourceCount - 1, BlockHash: origins[sourceCount-1].Hash})
	// Interleave unrelated sources in reverse order, and consume two distant
	// outputs from each source. Only input references may be sorted internally.
	block := &common.Block{Height: sourceCount, Hash: spTestHash(150), Transactions: []*common.Transaction{{}, {}}}
	for _, output := range []int{1023, 0} {
		for h := sourceCount - 1; h >= 0; h-- {
			spend := spTestSpend(origins[h], sourceCount, output)
			block.Transactions[1].Inputs = append(block.Transactions[1].Inputs, spend.Transactions[1].Inputs[0])
		}
	}
	before := append([]*common.TxInput(nil), block.Transactions[1].Inputs...)
	counter := &spArchiveReadCounter{KVDB: s.db}
	s.db = counter
	if err := s.process(block); err != nil {
		t.Fatal(err)
	}
	if counter.reads != sourceCount {
		t.Fatalf("loaded source Archive %d times, want %d", counter.reads, sourceCount)
	}
	for i, in := range before {
		if block.Transactions[1].Inputs[i] != in {
			t.Fatal("SP grouped processing reordered a Base transaction")
		}
	}
	for h := range origins {
		bits := s.live[string(spPositionKey(spLivePrefix, h, 1))].bitmap
		if len(bits) != 128 || bits[0] != 0xfe || bits[127] != 0x7f {
			t.Fatalf("source %d lost sparse Live changes", h)
		}
	}
}

func TestSilentPayGroupedSpendRejectsDuplicates(t *testing.T) {
	s := newSPTestState(t)
	origin := spTestCandidate(0, 58)
	if err := s.process(origin); err != nil {
		t.Fatal(err)
	}
	spCommitState(t, s, SPAnchor{Height: 0, BlockHash: origin.Hash})
	block := spTestSpend(origin, 1, 0)
	block.Transactions[1].Inputs = append(block.Transactions[1].Inputs, block.Transactions[1].Inputs[0])
	if err := s.process(block); !errors.Is(err, ErrSPCorrupt) {
		t.Fatalf("grouping hid a duplicate confirmed spend: %v", err)
	}
}
