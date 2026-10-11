package base

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
)

func newSPTestState(t *testing.T) *silentPayState {
	t.Helper()
	kv := indexdb.NewKVDBWithCache(filepath.Join(t.TempDir(), "base"), 1)
	if kv == nil {
		t.Fatal("open temporary Base database")
	}
	s := newSilentPayState(kv, &chaincfg.RegressionNetParams)
	t.Cleanup(func() {
		s.views.close()
		if err := kv.Close(); err != nil {
			t.Error(err)
		}
	})
	return s
}

func spTestHash(n byte) string { return (chainhash.Hash{n}).String() }

func spTestCandidate(height int, n byte) *common.Block {
	secret := make([]byte, 32)
	secret[31] = n
	_, pub := btcec.PrivKeyFromBytes(secret)
	key := pub.SerializeCompressed()
	previous := common.NewTxOutput(2000)
	previous.OutValue.PkScript = append([]byte{0, 20}, btcutil.Hash160(key)...)
	input := &common.TxInput{
		TxOutputV2: common.TxOutputV2{TxOutput: *previous},
		TxId: spTestHash(n + 100),
		Witness: wire.TxWitness{[]byte{1}, key},
	}
	outputs := make([]*common.TxOutputV2, 2)
	for i := range outputs {
		out := common.NewTxOutput(1000)
		out.OutValue.PkScript = append([]byte{0x51, 0x20}, key[1:]...)
		outputs[i] = &common.TxOutputV2{TxOutput: *out, OutHeight: height, OutTxIndex: 1, TxOutIndex: i * 2}
	}
	return &common.Block{Height: height, Hash: spTestHash(n + 30), Transactions: []*common.Transaction{
		{TxId: spTestHash(n + 60)},
		{TxId: spTestHash(n), Inputs: []*common.TxInput{input}, Outputs: outputs},
	}}
}

func spTestSpend(origin *common.Block, height, output int) *common.Block {
	parent := origin.Transactions[1]
	in := &common.TxInput{TxOutputV2: *parent.Outputs[output], TxId: parent.TxId, Witness: wire.TxWitness{[]byte{1}}}
	return &common.Block{Height: height, Hash: spTestHash(byte(height + 200)), Transactions: []*common.Transaction{
		{}, {TxId: spTestHash(byte(height + 150)), Inputs: []*common.TxInput{in}},
	}}
}

// This helper tests SP persistence in isolation. The end-to-end suite separately
// drives the real Base UpdateDB and Bitcoin transaction parser.
func spCommitState(t *testing.T, s *silentPayState, anchor SPAnchor) {
	t.Helper()
	if err := s.beginCommit(anchor.Height, anchor.BlockHash); err != nil {
		t.Fatal(err)
	}
	wb := s.db.NewWriteBatch()
	defer wb.Close()
	if err := s.writeTo(wb); err != nil {
		t.Fatal(err)
	}
	if err := indexdb.SetDB([]byte(SyncStatsKey), &SyncStats{SyncHeight: anchor.Height, SyncBlockHash: anchor.BlockHash}, wb); err != nil {
		t.Fatal(err)
	}
	if err := wb.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := s.finishCommit(); err != nil {
		t.Fatal(err)
	}
}

func TestSilentPayCloneSubtractPreservesLaterSpend(t *testing.T) {
	s := newSPTestState(t)
	origin := spTestCandidate(0, 7)
	if err := s.process(origin); err != nil {
		t.Fatal(err)
	}
	frozen := s.clone()
	if err := s.process(spTestSpend(origin, 1, 0)); err != nil {
		t.Fatal(err)
	}
	s.subtract(frozen)
	key := string(spPositionKey(spLivePrefix, 0, 1))
	if !bytes.Equal(frozen.live[key].bitmap, []byte{3}) || !bytes.Equal(s.live[key].bitmap, []byte{2}) {
		t.Fatal("frozen bitmap mutated or later spend removed by Subtract")
	}
	spCommitState(t, frozen, SPAnchor{Height: 0, BlockHash: origin.Hash})
	snapshot, err := s.createSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	spCommitState(t, s, SPAnchor{Height: 1, BlockHash: spTestHash(201)})
	page, err := s.scan(context.Background(), SPScanRequest{Mode: "live", SnapshotID: snapshot.SnapshotID, ToHeight: -1})
	if err != nil || !page.Done || len(page.Blocks) != 1 || !bytes.Equal(page.Blocks[0].Transactions[0].UnspentBitmap, []byte{3}) {
		t.Fatalf("old snapshot drifted: %+v / %v", page, err)
	}
	newSnapshot, err := s.createSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	page, err = s.scan(context.Background(), SPScanRequest{Mode: "live", SnapshotID: newSnapshot.SnapshotID, ToHeight: -1})
	if err != nil || len(page.Blocks) != 1 || !bytes.Equal(page.Blocks[0].Transactions[0].UnspentBitmap, []byte{2}) || len(page.Blocks[0].Transactions[0].P2TROutputs) != 2 {
		t.Fatalf("Live must retain the spent k=0 matching data: %+v / %v", page, err)
	}
	if err := s.process(spTestSpend(origin, 2, 1)); err != nil {
		t.Fatal(err)
	}
	spCommitState(t, s, SPAnchor{Height: 2, BlockHash: spTestHash(202)})
	if _, err := s.db.Read([]byte(key)); !errors.Is(err, common.ErrKeyNotFound) {
		t.Fatalf("fully spent transaction remained Live: %v", err)
	}
	if _, err := s.db.Read(spPositionKey(spArchivePrefix, 0, 1)); err != nil {
		t.Fatalf("fully spent Archive was deleted: %v", err)
	}
}

func TestSilentPaySnapshotPaginationAndExpiry(t *testing.T) {
	s := newSPTestState(t)
	var last *common.Block
	for h := 0; h < 3; h++ {
		last = spTestCandidate(h, byte(h + 1))
		if err := s.process(last); err != nil {
			t.Fatal(err)
		}
	}
	spCommitState(t, s, SPAnchor{Height: 2, BlockHash: last.Hash})
	meta, err := s.createSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	request := SPScanRequest{Mode: "live", SnapshotID: meta.SnapshotID, ToHeight: -1, Limit: 1}
	seen := make(map[int]bool)
	for n := 0; n < 4; n++ {
		page, err := s.scan(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		for _, b := range page.Blocks {
			if seen[b.Height] {
				t.Fatal("keyset pagination repeated a transaction")
			}
			seen[b.Height] = true
		}
		if page.Done {
			break
		}
		if page.NextCursor == "" {
			t.Fatal("incomplete page without cursor")
		}
		request.Cursor = page.NextCursor
	}
	if len(seen) != 3 {
		t.Fatalf("pagination lost rows: %v", seen)
	}
	request.FromHeight = 1
	if _, err := s.scan(context.Background(), request); !errors.Is(err, ErrSPInvalidRequest) {
		t.Fatalf("cursor accepted under a different scope: %v", err)
	}
	s.views.expire(meta.SnapshotID)
	request.Cursor = ""
	if _, err := s.scan(context.Background(), request); !errors.Is(err, ErrSPSnapshotExpired) {
		t.Fatalf("expired view reused: %v", err)
	}
}

func TestSilentPayMissingDataAndCommitFenceFailClosed(t *testing.T) {
	s := newSPTestState(t)
	block := spTestCandidate(0, 9)
	if err := s.process(block); err != nil {
		t.Fatal(err)
	}
	anchor := SPAnchor{Height: 0, BlockHash: block.Hash}
	spCommitState(t, s, anchor)
	archiveKey := spPositionKey(spArchivePrefix, 0, 1)
	raw, err := s.db.Read(archiveKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.db.Delete(archiveKey); err != nil {
		t.Fatal(err)
	}
	if _, err := s.scan(context.Background(), SPScanRequest{Mode: "archive", ToHeight: 0}); err == nil {
		t.Fatal("missing Archive was treated as an empty successful scan")
	}
	if err := s.db.Write(archiveKey, raw); err != nil {
		t.Fatal(err)
	}
	if err := s.beginCommit(1, spTestHash(44)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.freshView(context.Background()); !errors.Is(err, ErrSPIncompleteCommit) {
		t.Fatalf("partial commit published: %v", err)
	}
	reloaded := newSilentPayState(s.db, &chaincfg.RegressionNetParams)
	defer reloaded.views.close()
	if err := reloaded.validateInitialState(&SyncStats{SyncHeight: 0, SyncBlockHash: block.Hash}); !errors.Is(err, ErrSPIncompleteCommit) {
		t.Fatalf("restart ignored unfinished Base write: %v", err)
	}
}

func TestSilentPayCodecRejectsMalformedRecords(t *testing.T) {
	tx, err := spCandidate(spTestCandidate(0, 4).Transactions[1], 1)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := spEncodeTransaction(*tx)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := spDecodeTransaction(raw, 1)
	if err != nil || !spEqualTransactions(*tx, decoded) {
		t.Fatalf("round trip: %v", err)
	}
	for _, corrupt := range [][]byte{nil, raw[:len(raw)-1], append(append([]byte{}, raw...), 0)} {
		if _, err := spDecodeTransaction(corrupt, 1); err == nil {
			t.Fatal("accepted malformed record")
		}
	}
	for _, bits := range [][]byte{nil, {0}, {4}, {3, 0}} {
		if err := spValidateBitmap(bits, 2, false); err == nil {
			t.Fatalf("accepted invalid bitmap %x", bits)
		}
	}
}


func TestSilentPayPublishedSchemaLossPanics(t *testing.T) {
	s := newSPTestState(t)
	block := spTestCandidate(0, 41)
	if err := s.process(block); err != nil { t.Fatal(err) }
	spCommitState(t, s, SPAnchor{Height: 0, BlockHash: block.Hash})
	if err := s.db.Delete(spSchemaKey); err != nil { t.Fatal(err) }

	b := NewBaseIndexer(s.db, &chaincfg.RegressionNetParams, 0, 10)
	b.silentpay = s
	b.stats.SyncHeight, b.stats.SyncBlockHash = 0, block.Hash
	panicked := false
	func() {
		defer func() { panicked = recover() != nil }()
		_, _ = b.SilentPayStatus(context.Background())
	}()
	if !panicked { t.Fatal("committed SP schema loss returned not_ready instead of panicking") }
}


func TestSilentPayArchiveBeforeCoverageMustNotAdvanceCursor(t *testing.T) {
	s := newSPTestState(t)
	s.startHeight = 5
	block := spTestCandidate(5, 52)
	if err := s.process(block); err != nil { t.Fatal(err) }
	spCommitState(t, s, SPAnchor{Height: 5, BlockHash: block.Hash})

	page, err := s.scan(context.Background(), SPScanRequest{
		Mode: "archive", FromHeight: 0, ToHeight: 4,
	})
	if !errors.Is(err, ErrSPInvalidRequest) {
		t.Fatalf("pre-coverage scan must reject an unrepresentable anchor; page=%+v err=%v", page, err)
	}
}

func TestSilentPaySpendingRejectsMissingArchiveWithDirectory(t *testing.T) {
	s := newSPTestState(t)
	origin := spTestCandidate(0, 53)
	if err := s.process(origin); err != nil { t.Fatal(err) }
	spCommitState(t, s, SPAnchor{Height: 0, BlockHash: origin.Hash})
	if err := s.db.Delete(spPositionKey(spArchivePrefix, 0, 1)); err != nil { t.Fatal(err) }

	if err := s.process(spTestSpend(origin, 1, 0)); !errors.Is(err, ErrSPCorrupt) {
		t.Fatalf("missing mandatory Archive entry was treated as noncandidate: %v", err)
	}
}

type spArchiveReadCounter struct {
	common.KVDB
	reads int
}
func (c *spArchiveReadCounter) Read(key []byte) ([]byte, error) {
	if bytes.HasPrefix(key, spArchivePrefix) { c.reads++ }
	return c.KVDB.Read(key)
}

func TestSilentPayBlockRepeatedSpendReadsOriginArchiveOnce(t *testing.T) {
	s := newSPTestState(t)
	origin := spTestCandidate(0, 54)
	if err := s.process(origin); err != nil { t.Fatal(err) }
	spCommitState(t, s, SPAnchor{Height: 0, BlockHash: origin.Hash})

	spends := spTestSpend(origin, 1, 0)
	second := spTestSpend(origin, 1, 1)
	spends.Transactions[1].Inputs = append(spends.Transactions[1].Inputs, second.Transactions[1].Inputs[0])
	counter := &spArchiveReadCounter{KVDB: s.db}
	s.db = counter
	if err := s.process(spends); err != nil { t.Fatal(err) }
	if counter.reads != 1 {
		t.Fatalf("same source transaction Archive re-decoded %d times, want 1", counter.reads)
	}
}

func TestSilentPaySpendingSkipsPersistedNoncandidate(t *testing.T) {
	s := newSPTestState(t)
	key := make([]byte, 32)
	key[31] = 55
	_, pub := btcec.PrivKeyFromBytes(key)
	output := common.NewTxOutput(500)
	output.OutValue.PkScript = append([]byte{0x51, 0x20}, pub.SerializeCompressed()[1:]...)
	noCandidate := &common.Block{Height: 0, Hash: spTestHash(81), Transactions: []*common.Transaction{
		{TxId: spTestHash(80)},
		{TxId: spTestHash(79), Outputs: []*common.TxOutputV2{{TxOutput: *output, OutHeight: 0, OutTxIndex: 1, TxOutIndex: 0}}},
	}}
	if err := s.process(noCandidate); err != nil { t.Fatal(err) }
	spCommitState(t, s, SPAnchor{Height: 0, BlockHash: noCandidate.Hash})
	spend := spTestSpend(noCandidate, 1, 0)
	if err := s.process(spend); err != nil {
		t.Fatalf("noncandidate output rejected: %v", err)
	}
}
