package base

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/sat20-labs/indexer/common"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
)

// Use real SP processing/persistence and the existing Base row encoders. Some
// outputs are then marked spent in both tables, so both streams cross pages at
// different positions. Full Base processing is covered by the existing E2Es.
func spAuditMergeFixture(t *testing.T) (*silentPayState, SPAnchor, *spBatchReadCounts) {
	t.Helper()
	s := newSPTestState(t)
	block := spTestCandidate(0, 17)
	template := block.Transactions[1]
	block.Transactions = block.Transactions[:1]
	const count = 2500
	for i := 1; i <= count; i++ {
		tx := *template
		var hash chainhash.Hash
		binary.LittleEndian.PutUint32(hash[:], uint32(i))
		tx.TxId = hash.String()
		block.Transactions = append(block.Transactions, &tx)
	}
	if err := s.process(block); err != nil {
		t.Fatal(err)
	}
	anchor := SPAnchor{Height: 0, BlockHash: block.Hash}
	spCommitState(t, s, anchor)
	wb := s.db.NewWriteBatch()
	defer wb.Close()
	put := func(key, value []byte) {
		if err := wb.Put(key, value); err != nil {
			t.Fatal(err)
		}
	}
	// The address decoder also supports raw-script base64, independent of the
	// process-global network selected by other tests.
	put(indexdb.GetAddressIdKey(1), []byte(base64.StdEncoding.EncodeToString(template.Outputs[0].OutValue.PkScript)))
	for i := 1; i <= count; i++ {
		liveKey := spPositionKey(spLivePrefix, 0, uint32(i))
		if i%3 == 0 {
			if err := wb.Delete(liveKey); err != nil {
				t.Fatal(err)
			}
			continue
		}
		bitmap := byte(3)
		if i%3 == 1 {
			bitmap = 2
		}
		put(liveKey, []byte{bitmap})
		for ordinal, vout := range []int{0, 2} {
			if bitmap&(1<<ordinal) == 0 {
				continue
			}
			if err := indexdb.SetDBWithProto3(
				indexdb.GetUTXODBKey(block.Transactions[i].TxId+":"+spVoutString(uint32(vout))),
				&common.UtxoValueInDB{UtxoId: common.ToUtxoId(0, i, vout), AddressId: 1, Value: 1000}, wb,
			); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := wb.Flush(); err != nil {
		t.Fatal(err)
	}
	counts := &spBatchReadCounts{}
	s.db = &spBatchCountDB{KVDB: s.db, counts: counts}
	return s, anchor, counts
}

func TestSilentPayDBAuditMergeAvoidsLivePointReads(t *testing.T) {
	s, anchor, counts := spAuditMergeFixture(t)
	if err := s.audit(context.Background(), anchor); err != nil {
		t.Fatal(err)
	}
	_, _, live := counts.values()
	if live != 0 {
		t.Fatalf("audit performed %d Live point reads; want zero", live)
	}
}

func TestSilentPayDBAuditUTXOKeyEncodingUnchanged(t *testing.T) {
	for _, txid := range []string{
		"0000000000000000000000000000000000000000000000000000000000000000",
		"0123456789abcdef1032547698badcfeffeeddccbbaa99887766554433221100",
		"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	} {
		hash, err := chainhash.NewHashFromStr(txid)
		if err != nil {
			t.Fatal(err)
		}
		prefix := spAuditUTXOPrefix(hash[:])
		// Include decimal/CompactSize boundaries and decreasing lengths to
		// catch stale suffix bytes when the same prefix buffer is reused.
		for _, vout := range []uint32{0, 9, 10, 99, 100, 252, 253, 65535, 65536, 0x3ffff, 2, 0} {
			got := strconv.AppendUint(prefix, uint64(vout), 10)
			want := indexdb.GetUTXODBKey(txid + ":" + strconv.FormatUint(uint64(vout), 10))
			if !bytes.Equal(got, want) {
				t.Fatalf("changed key for %s:%d: %x != %x", txid, vout, got, want)
			}
		}
	}
}

func TestSilentPayDBAuditMergeEmptyArchiveRejectsLive(t *testing.T) {
	s := newSPTestState(t)
	block := spTestCandidate(0, 17)
	block.Transactions = block.Transactions[:1]
	if err := s.process(block); err != nil {
		t.Fatal(err)
	}
	anchor := SPAnchor{Height: 0, BlockHash: block.Hash}
	spCommitState(t, s, anchor)
	if err := s.audit(context.Background(), anchor); err != nil {
		t.Fatal(err)
	}
	if err := s.db.Write(spPositionKey(spLivePrefix, 0, 1), []byte{1}); err != nil {
		t.Fatal(err)
	}
	if err := s.audit(context.Background(), anchor); !errors.Is(err, ErrSPCorrupt) {
		t.Fatalf("orphan in empty Archive: %v", err)
	}
}

func TestSilentPayDBAuditMergeRejectsCorruption(t *testing.T) {
	s, anchor, _ := spAuditMergeFixture(t)
	if err := s.audit(context.Background(), anchor); err != nil {
		t.Fatal(err)
	}
	archiveKey := spPositionKey(spArchivePrefix, 0, 1024)
	archive, err := s.db.Read(archiveKey)
	if err != nil {
		t.Fatal(err)
	}
	badPoint := append([]byte(nil), archive...)
	// Correct compressed length/prefix, but x is outside the field.
	for i := 34; i < 66; i++ {
		badPoint[i] = 0xff
	}
	tests := []struct {
		name       string
		key, value []byte
		remove     bool
	}{
		{"missing_live_page_boundary", spPositionKey(spLivePrefix, 0, 1024), nil, true},
		{"missing_live_later_page", spPositionKey(spLivePrefix, 0, 1537), nil, true},
		{"wrong_bitmap", spPositionKey(spLivePrefix, 0, 1024), []byte{1}, false},
		{"empty_bitmap", spPositionKey(spLivePrefix, 0, 1024), []byte{}, false},
		{"zero_bitmap", spPositionKey(spLivePrefix, 0, 1024), []byte{0}, false},
		{"extra_bitmap_byte", spPositionKey(spLivePrefix, 0, 1024), []byte{2, 0}, false},
		{"unused_bitmap_bit", spPositionKey(spLivePrefix, 0, 1024), []byte{0x82}, false},
		{"spent_live", spPositionKey(spLivePrefix, 0, 1023), []byte{1}, false},
		{"spent_empty_live", spPositionKey(spLivePrefix, 0, 1023), []byte{}, false},
		{"orphan_before", spPositionKey(spLivePrefix, 0, 0), []byte{1}, false},
		{"orphan_after", spPositionKey(spLivePrefix, 0, 2501), []byte{1}, false},
		{"outside_coverage", spPositionKey(spLivePrefix, 1, 1), []byte{1}, false},
		{"invalid_live_key", append(spPositionKey(spLivePrefix, 0, 1024), 0), []byte{1}, false},
		{"invalid_curve_point", archiveKey, badPoint, false},
		{"truncated_archive", archiveKey, archive[:len(archive)-1], false},
		{"trailing_archive_bytes", archiveKey, append(append([]byte(nil), archive...), 0), false},
		{"missing_archive", archiveKey, nil, true},
		{"missing_directory", spBlockKey(0), nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			old, readErr := s.db.Read(tc.key)
			if readErr != nil && !errors.Is(readErr, common.ErrKeyNotFound) {
				t.Fatal(readErr)
			}
			if tc.remove {
				err = s.db.Delete(tc.key)
			} else {
				err = s.db.Write(tc.key, tc.value)
			}
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if readErr == nil {
					err = s.db.Write(tc.key, old)
				} else {
					err = s.db.Delete(tc.key)
				}
				if err != nil {
					t.Fatal(err)
				}
			}()
			if err := s.audit(context.Background(), anchor); err == nil {
				t.Fatal("audit accepted corrupt data")
			}
			// A failed audit must not repair or delete the offending record.
			after, afterErr := s.db.Read(tc.key)
			if tc.remove {
				if !errors.Is(afterErr, common.ErrKeyNotFound) {
					t.Fatalf("audit rewrote missing row: %v", afterErr)
				}
			} else if afterErr != nil || !bytes.Equal(after, tc.value) {
				t.Fatalf("audit changed corrupt row: %v", afterErr)
			}
		})
	}
	if err := s.audit(context.Background(), anchor); err != nil {
		t.Fatalf("restored audit: %v", err)
	}
}

// Real backend benchmark: page copying/iteration only, with no per-row DB
// lookups or cryptography to hide allocation changes in the pager itself.
func BenchmarkSilentPayDBAuditPaging(b *testing.B) {
	kv := indexdb.NewKVDBWithCache(filepath.Join(b.TempDir(), "base"), 1)
	if kv == nil {
		b.Fatal("open DB")
	}
	defer kv.Close()
	wb := kv.NewWriteBatch()
	for i := 0; i < 4096; i++ {
		if err := wb.Put(spPositionKey(spArchivePrefix, 0, uint32(i+1)), bytes.Repeat([]byte{byte(i)}, 100)); err != nil {
			b.Fatal(err)
		}
	}
	if err := wb.Flush(); err != nil {
		b.Fatal(err)
	}
	wb.Close()
	view, err := kv.(common.SnapshotDB).NewReadView()
	if err != nil {
		b.Fatal(err)
	}
	defer view.Close()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := 0
		if err := spAuditEach(context.Background(), view, spArchivePrefix, func(_, _ []byte) error { count++; return nil }); err != nil {
			b.Fatal(err)
		}
		if count != 4096 {
			b.Fatal(count)
		}
	}
}
