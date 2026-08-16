package indexer

import (
	"testing"
	"time"

	"github.com/sat20-labs/indexer/common"
)

type testGCRunnerDB struct {
	common.KVDB
	calls int
}

func (d *testGCRunnerDB) RunGC() error {
	d.calls++
	return nil
}

func TestRunDBGCIntervalAndForce(t *testing.T) {
	mgr := &IndexerMgr{}
	first := time.Unix(1_000, 0)

	mgr.runDBGC(first, false)
	if !mgr.lastDBGCAttempt.Equal(first) {
		t.Fatalf("first GC attempt = %v, want %v", mgr.lastDBGCAttempt, first)
	}
	if !mgr.lastDBGC.IsZero() {
		t.Fatalf("unsupported backend must not advance successful GC time: %v", mgr.lastDBGC)
	}

	mgr.runDBGC(first.Add(dbGCInterval-time.Second), false)
	if !mgr.lastDBGCAttempt.Equal(first) {
		t.Fatalf("GC attempted before interval: got %v, want %v", mgr.lastDBGCAttempt, first)
	}

	second := first.Add(dbGCInterval)
	mgr.runDBGC(second, false)
	if !mgr.lastDBGCAttempt.Equal(second) {
		t.Fatalf("second GC attempt = %v, want %v", mgr.lastDBGCAttempt, second)
	}

	forced := second.Add(time.Second)
	mgr.runDBGC(forced, true)
	if !mgr.lastDBGCAttempt.Equal(forced) {
		t.Fatalf("forced GC attempt = %v, want %v", mgr.lastDBGCAttempt, forced)
	}
}

func TestRunDBGCSuccessAdvancesLastSuccessfulTime(t *testing.T) {
	database := &testGCRunnerDB{}
	mgr := &IndexerMgr{baseDB: database}
	now := time.Unix(2_000, 0)

	mgr.runDBGC(now, false)
	if database.calls != 1 {
		t.Fatalf("RunGC calls = %d, want 1", database.calls)
	}
	if !mgr.lastDBGC.Equal(now) {
		t.Fatalf("successful GC time = %v, want %v", mgr.lastDBGC, now)
	}
}
