package base

import (
	"context"
	"errors"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/indexer/common"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
)

// A Base database indexed before the SP feature was deployed must remain
// usable on software upgrade, without silently creating incomplete SP data.
func TestSilentPayLegacyBaseContinuesWithoutInitializingSP(t *testing.T) {
	s := newSPTestState(t)
	const legacyHeight = 100
	legacyHash := spTestHash(71)
	wb := s.db.NewWriteBatch()
	if err := indexdb.SetDB([]byte(SyncStatsKey), &SyncStats{
		SyncHeight: legacyHeight, SyncBlockHash: legacyHash,
	}, wb); err != nil {
		t.Fatal(err)
	}
	if err := wb.Flush(); err != nil {
		t.Fatal(err)
	}
	wb.Close()

	b := NewBaseIndexer(s.db, &chaincfg.RegressionNetParams, 0, 10)
	b.Init()
	if b.silentpay != nil {
		t.Fatal("SP should be disabled for an existing Base DB without SP history")
	}
	if err := b.processSilentPay(spTestCandidate(101, 17)); err != nil {
		t.Fatalf("disabled SP interrupted ordinary Base block processing: %v", err)
	}
	b.lastHeight, b.lastHash = legacyHeight+1, spTestHash(72)
	b.UpdateDB()

	if b.stats.SyncHeight != legacyHeight+1 || b.stats.SyncBlockHash != b.lastHash {
		t.Fatalf("Base DB did not advance independently of SP: %+v", b.stats)
	}
	for _, key := range [][]byte{spSchemaKey, spPendingKey, spBlockKey(101)} {
		if _, err := s.db.Read(key); !errors.Is(err, common.ErrKeyNotFound) {
			t.Fatalf("legacy DB unexpectedly created SP data %q: %v", key, err)
		}
	}

	// On restart the old database must not begin building an incomplete SP
	// history from its already-advanced height.
	b.Init()
	if b.silentpay != nil {
		t.Fatal("restarting legacy database automatically enabled partial SP indexing")
	}
	if _, err := b.SilentPayStatus(context.Background()); !errors.Is(err, ErrSPNotInitialized) {
		t.Fatalf("SP status did not explain missing initialization: %v", err)
	}
	if _, err := b.SilentPayCreateSnapshot(context.Background()); !errors.Is(err, ErrSPNotInitialized) {
		t.Fatalf("legacy SP snapshot succeeded: %v", err)
	}
	if _, err := b.SilentPayScan(context.Background(), SPScanRequest{Mode: "archive"}); !errors.Is(err, ErrSPNotInitialized) {
		t.Fatalf("legacy SP scan succeeded: %v", err)
	}
	if _, err := b.SilentPayLookup(context.Background(), []SPPosition{{Height: legacyHeight}}); !errors.Is(err, ErrSPNotInitialized) {
		t.Fatalf("legacy SP position lookup succeeded: %v", err)
	}
	if _, _, err := b.SilentPayOutpoints(context.Background(), "snapshot", []SPPosition{{Height: legacyHeight}}); !errors.Is(err, ErrSPNotInitialized) {
		t.Fatalf("legacy SP outpoint query succeeded: %v", err)
	}
}

func TestSilentPayLegacyBaseBeforeActivationIsNotSilentlyBackfilled(t *testing.T) {
	s := newSPTestState(t)
	wb := s.db.NewWriteBatch()
	if err := indexdb.SetDB([]byte(SyncStatsKey), &SyncStats{
		SyncHeight: 400000, SyncBlockHash: spTestHash(81),
	}, wb); err != nil {
		t.Fatal(err)
	}
	if err := wb.Flush(); err != nil {
		t.Fatal(err)
	}
	wb.Close()

	b := NewBaseIndexer(s.db, &chaincfg.MainNetParams, 0, 10)
	b.Init()
	if b.silentpay != nil {
		t.Fatal("pre-Taproot legacy Base must not auto-enable partial SP history")
	}
}

func TestSilentPayMissingSchemaWithExistingSPStateRemainsFatal(t *testing.T) {
	for _, test := range []struct {
		name string
		key []byte
		value []byte
	}{
		{"pending_commit", spPendingKey, []byte("incomplete")},
		{"block_directory", spBlockKey(100), []byte("unexpected")},
		{"archived_record", spPositionKey(spArchivePrefix, 99, 1), []byte("unexpected")},
		{"live_record", spPositionKey(spLivePrefix, 99, 1), []byte("unexpected")},
		{"wrong_schema", spSchemaKey, []byte("1:mainnet")},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := newSPTestState(t)
			wb := s.db.NewWriteBatch()
			if err := indexdb.SetDB([]byte(SyncStatsKey), &SyncStats{
				SyncHeight: 100, SyncBlockHash: spTestHash(92),
			}, wb); err != nil {
				t.Fatal(err)
			}
			if err := wb.Put(test.key, test.value); err != nil {
				t.Fatal(err)
			}
			if err := wb.Flush(); err != nil {
				t.Fatal(err)
			}
			wb.Close()
			b := NewBaseIndexer(s.db, &chaincfg.RegressionNetParams, 0, 10)
			defer func() {
				if recover() == nil {
					t.Fatal("missing or incorrect SP schema with existing SP records was silently ignored")
				}
			}()
			b.Init()
		})
	}
}

func TestSilentPayFreshBaseStillInitializesSP(t *testing.T) {
	s := newSPTestState(t)
	b := NewBaseIndexer(s.db, &chaincfg.RegressionNetParams, 0, 10)
	b.Init()
	if b.silentpay == nil {
		t.Fatal("fresh Base database must enable SP and build complete history")
	}
	b.CloseSilentPayViews()
}
