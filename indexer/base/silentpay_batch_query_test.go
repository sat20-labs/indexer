package base

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/sat20-labs/indexer/common"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
)

// Count real snapshot reads (not direct KVDB.Read calls). These wrappers use
// exactly the same Pebble/Badger read views as the public SP query entrypoints.
type spBatchReadCounts struct {
	sync.Mutex
	archive, directory, live, utxo, address int
}

func (c *spBatchReadCounts) reset() {
	c.Lock()
	defer c.Unlock()
	c.archive, c.directory, c.live, c.utxo, c.address = 0, 0, 0, 0, 0
}

func (c *spBatchReadCounts) values() (archive, directory, live int) {
	c.Lock()
	defer c.Unlock()
	return c.archive, c.directory, c.live
}

type spBatchCountDB struct {
	common.KVDB
	counts *spBatchReadCounts
}

func (d *spBatchCountDB) NewReadView() (common.DBReadView, error) {
	view, err := d.KVDB.(common.SnapshotDB).NewReadView()
	if err != nil {
		return nil, err
	}
	return &spBatchCountView{DBReadView: view, counts: d.counts}, nil
}

func (d *spBatchCountDB) Sync() error {
	return d.KVDB.(common.SnapshotDB).Sync()
}

type spBatchCountView struct {
	common.DBReadView
	counts *spBatchReadCounts
}

func (v *spBatchCountView) Read(key []byte) ([]byte, error) {
	v.counts.Lock()
	switch {
	case bytes.HasPrefix(key, spArchivePrefix):
		v.counts.archive++
	case bytes.HasPrefix(key, spBlockPrefix):
		v.counts.directory++
	case bytes.HasPrefix(key, spLivePrefix):
		v.counts.live++
	case bytes.HasPrefix(key, []byte(common.DB_KEY_UTXO)):
		v.counts.utxo++
	case bytes.HasPrefix(key, []byte(common.DB_KEY_ADDRESSID)):
		v.counts.address++
	}
	v.counts.Unlock()
	return v.DBReadView.Read(key)
}

func spBatchQueryFixture(t *testing.T) (*silentPayState, *common.Block, *common.Block, *spBatchReadCounts) {
	t.Helper()
	s := newSPTestState(t)
	first := spMemoryCandidate(0, 4096)
	second := spTestCandidate(1, 31)
	if err := s.process(first); err != nil {
		t.Fatal(err)
	}
	if err := s.process(second); err != nil {
		t.Fatal(err)
	}
	spCommitState(t, s, SPAnchor{Height: 1, BlockHash: second.Hash})

	spend := spTestSpend(first, 2, 0)
	if err := s.process(spend); err != nil {
		t.Fatal(err)
	}
	spCommitState(t, s, SPAnchor{Height: 2, BlockHash: spend.Hash})

	// This is a read-path unit fixture, not a fake full Base E2E: seed
	// actual encoded Base UTXO rows for precisely the queried unspent outputs.
	wb := s.db.NewWriteBatch()
	for _, output := range []struct {
		block *common.Block
		vout  uint32
	}{
		{first, 2}, {first, 8190}, {second, 0},
	} {
		txid := output.block.Transactions[1].TxId
		id := common.ToUtxoId(output.block.Height, 1, int(output.vout))
		if err := indexdb.SetDBWithProto3(
			indexdb.GetUTXODBKey(txid+":"+spVoutString(output.vout)),
			&common.UtxoValueInDB{UtxoId: id, Value: 1000}, wb,
		); err != nil {
			t.Fatal(err)
		}
	}
	if err := wb.Flush(); err != nil {
		t.Fatal(err)
	}
	wb.Close()

	counts := &spBatchReadCounts{}
	s.db = &spBatchCountDB{KVDB: s.db, counts: counts}
	return s, first, second, counts
}

func TestSilentPayBatchOutpointsGroupedSnapshotReads(t *testing.T) {
	s, first, second, counts := spBatchQueryFixture(t)
	ctx := context.Background()
	snap, err := s.createSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	positions := []SPPosition{
		{Height: 0, BlockHash: first.Hash, TxIndex: 1, Vout: 2},
		{Height: 0, BlockHash: first.Hash, TxIndex: 1, Vout: 0},
		{Height: 1, BlockHash: second.Hash, TxIndex: 1, Vout: 0},
		{Height: 0, BlockHash: first.Hash, TxIndex: 1, Vout: 8190},
		{Height: 0, BlockHash: first.Hash, TxIndex: 1, Vout: 10001},
		{Height: 0, BlockHash: first.Hash, TxIndex: 1, Vout: 2},
		{Height: 0, BlockHash: first.Hash, TxIndex: 1, Vout: 2, ExpectedTxID: spTestHash(1)},
		{Height: 0, BlockHash: second.Hash, TxIndex: 1, Vout: 2},
		{Height: 9, BlockHash: first.Hash, TxIndex: 1, Vout: 2},
		{Height: 0, BlockHash: first.Hash, TxIndex: 3, Vout: 2},
	}
	counts.reset()
	anchor, results, err := s.snapshotOutpoints(ctx, snap.SnapshotID, positions)
	if err != nil {
		t.Fatal(err)
	}
	if anchor != snap.Anchor || len(results) != len(positions) {
		t.Fatalf("snapshot shape mismatch: anchor=%+v results=%d", anchor, len(results))
	}
	for i := range results {
		if results[i].SPPosition != positions[i] {
			t.Fatalf("response order changed at %d", i)
		}
	}
	for i, want := range map[int]string{0: "unspent", 1: "spent", 2: "unspent", 3: "unspent", 5: "unspent"} {
		if results[i].State != want || results[i].TxID == "" || results[i].Error != "" {
			t.Fatalf("position %d state=%+v, want %s", i, results[i], want)
		}
	}
	for i, msg := range map[int]string{4: "not_candidate_output", 6: "transaction_identity_mismatch", 7: "snapshot_expired", 8: "invalid_silentpay_request", 9: "not_candidate_transaction"} {
		if results[i].State != "unknown" || !strings.Contains(results[i].Error, msg) {
			t.Fatalf("position %d state=%+v, want error %s", i, results[i], msg)
		}
	}
	archive, directory, live := counts.values()
	if archive != 2 || live != 2 || directory > 3 {
		t.Fatalf("outpoint batch repeated source reads: Archive=%d directory=%d Live=%d, want <=2 sources", archive, directory, live)
	}
}

func TestSilentPayTxIDLookupDeduplicatesCompactSources(t *testing.T) {
	s, first, second, counts := spBatchQueryFixture(t)
	ctx := context.Background()
	positions := make([]SPPosition, 0, 200)
	for i := 0; i < 200; i++ {
		block := first
		if i%4 == 0 {
			block = second
		}
		positions = append(positions, SPPosition{Height: block.Height, BlockHash: block.Hash, TxIndex: 1, Vout: uint32(i * 2)})
	}
	counts.reset()
	results, err := s.lookup(ctx, positions)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != len(positions) {
		t.Fatalf("lookup returned %d, want %d", len(results), len(positions))
	}
	for i, item := range results {
		want := first.Transactions[1].TxId
		if i%4 == 0 {
			want = second.Transactions[1].TxId
		}
		if item.SPPosition != positions[i] || item.TxID != want || item.Error != "" {
			t.Fatalf("txid lookup mismatch at %d: %+v", i, item)
		}
	}
	archive, directory, _ := counts.values()
	if archive != 2 || directory > 3 {
		t.Fatalf("lookup repeated compact Archive decode: Archive=%d directory=%d", archive, directory)
	}
}

func TestSilentPayBatchOutpointsCorruptRecordFailsClosed(t *testing.T) {
	s, first, _, _ := spBatchQueryFixture(t)
	// The compressed record must still be validated to its end, even when a
	// requested vout is near the start and no other vout is requested.
	raw, err := s.db.Read(spPositionKey(spArchivePrefix, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.db.Write(spPositionKey(spArchivePrefix, 0, 1), append(raw, 0xff)); err != nil {
		t.Fatal(err)
	}
	snap, err := s.createSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = s.snapshotOutpoints(context.Background(), snap.SnapshotID, []SPPosition{{
		Height: 0, BlockHash: first.Hash, TxIndex: 1, Vout: 2,
	}})
	if !errors.Is(err, ErrSPCorrupt) {
		t.Fatalf("record tail corruption was ignored: %v", err)
	}
	if _, err := s.lookup(context.Background(), []SPPosition{{
		Height: 0, BlockHash: first.Hash, TxIndex: 1,
	}}); !errors.Is(err, ErrSPCorrupt) {
		t.Fatalf("lookup ignored full compact record corruption: %v", err)
	}
}
