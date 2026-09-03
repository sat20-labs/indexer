package db

import (
	"errors"
	"testing"

	"github.com/sat20-labs/indexer/common"
)

type fakeGCDB struct {
	common.KVDB
	calls int
	err   error
}

func (f *fakeGCDB) RunGC() error {
	f.calls++
	return f.err
}

func TestRunDBGCDispatchesToBackend(t *testing.T) {
	database := &fakeGCDB{}
	if err := RunDBGC(database); err != nil {
		t.Fatalf("RunDBGC returned error: %v", err)
	}
	if database.calls != 1 {
		t.Fatalf("RunGC calls = %d, want 1", database.calls)
	}
}

func TestRunDBGCPropagatesBackendError(t *testing.T) {
	want := errors.New("gc failed")
	database := &fakeGCDB{err: want}
	if err := RunDBGC(database); !errors.Is(err, want) {
		t.Fatalf("RunDBGC error = %v, want %v", err, want)
	}
}

func TestRunDBGCReportsUnsupportedBackend(t *testing.T) {
	if err := RunDBGC(nil); !errors.Is(err, ErrGCUnsupported) {
		t.Fatalf("RunDBGC(nil) error = %v, want ErrGCUnsupported", err)
	}

	unsupported := &fakeUnsupportedDB{}
	if err := RunDBGC(unsupported); !errors.Is(err, ErrGCUnsupported) {
		t.Fatalf("RunDBGC(unsupported) error = %v, want ErrGCUnsupported", err)
	}
}

type fakeUnsupportedDB struct {
	common.KVDB
}

type fakeFinalizeDB struct {
	common.KVDB
	workers int
	err     error
}

func (f *fakeFinalizeDB) Finalize(workers int) error {
	f.workers = workers
	return f.err
}

func TestFinalizeDBDispatchesToBackend(t *testing.T) {
	database := &fakeFinalizeDB{}
	if err := FinalizeDB(database, 3); err != nil {
		t.Fatalf("FinalizeDB returned error: %v", err)
	}
	if database.workers != 3 {
		t.Fatalf("workers=%d, want 3", database.workers)
	}
}

func TestFinalizeDBReportsUnsupportedBackend(t *testing.T) {
	if err := FinalizeDB(&fakeUnsupportedDB{}, 1); !errors.Is(err, ErrFinalizeUnsupported) {
		t.Fatalf("FinalizeDB error=%v, want ErrFinalizeUnsupported", err)
	}
}
