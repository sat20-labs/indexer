//go:build !pebble

package db

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/sat20-labs/indexer/common"
)

func newBadgerScanTestDB(t *testing.T) *badgerDB {
	t.Helper()
	path := t.TempDir()
	raw, err := openBadgerDB(path, 0)
	if err != nil {
		t.Fatalf("open Badger: %v", err)
	}
	database := &badgerDB{path: path, db: raw}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("close Badger: %v", err)
		}
	})
	for _, key := range []string{"a-1", "a-2", "a-3", "b-1"} {
		if err := database.Write([]byte(key), []byte("value-"+key)); err != nil {
			t.Fatalf("write %s: %v", key, err)
		}
	}
	return database
}

func scanKeys(t *testing.T, database common.KVDB, options common.ScanOptions) []string {
	t.Helper()
	var result []string
	if err := database.Scan(options, func(k, v []byte) error {
		result = append(result, string(k))
		return nil
	}); err != nil {
		t.Fatalf("scan: %v", err)
	}
	return result
}

func TestBadgerScanContract(t *testing.T) {
	database := newBadgerScanTestDB(t)

	tests := []struct {
		name    string
		options common.ScanOptions
		want    []string
	}{
		{
			name:    "forward prefix",
			options: common.ScanOptions{Prefix: []byte("a-")},
			want:    []string{"a-1", "a-2", "a-3"},
		},
		{
			name:    "reverse prefix",
			options: common.ScanOptions{Prefix: []byte("a-"), Reverse: true},
			want:    []string{"a-3", "a-2", "a-1"},
		},
		{
			name: "forward inclusive",
			options: common.ScanOptions{
				Prefix: []byte("a-"), Start: []byte("a-2"), StartInclusive: true,
			},
			want: []string{"a-2", "a-3"},
		},
		{
			name: "forward exclusive",
			options: common.ScanOptions{
				Prefix: []byte("a-"), Start: []byte("a-2"), StartInclusive: false,
			},
			want: []string{"a-3"},
		},
		{
			name: "reverse inclusive",
			options: common.ScanOptions{
				Prefix: []byte("a-"), Start: []byte("a-2"), Reverse: true, StartInclusive: true,
			},
			want: []string{"a-2", "a-1"},
		},
		{
			name: "reverse exclusive",
			options: common.ScanOptions{
				Prefix: []byte("a-"), Start: []byte("a-2"), Reverse: true, StartInclusive: false,
			},
			want: []string{"a-1"},
		},
		{
			name:    "limit",
			options: common.ScanOptions{Prefix: []byte("a-"), Limit: 2},
			want:    []string{"a-1", "a-2"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := scanKeys(t, database, test.options); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("keys=%v, want %v", got, test.want)
			}
		})
	}
}

func TestBadgerScanKeysOnlyAndEarlyStop(t *testing.T) {
	database := newBadgerScanTestDB(t)
	count := 0
	err := database.Scan(common.ScanOptions{
		Prefix:   []byte("a-"),
		KeysOnly: true,
	}, func(k, v []byte) error {
		if v != nil {
			t.Fatalf("keys-only scan returned value %q", v)
		}
		count++
		return common.ErrStopScan
	})
	if err != nil {
		t.Fatalf("early stop returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("callback count=%d, want 1", count)
	}
}

func TestBadgerBatchReadPreservesOwnedBytes(t *testing.T) {
	database := newBadgerScanTestDB(t)
	var firstKey, firstValue []byte
	if err := database.BatchRead([]byte("a-"), false, func(k, v []byte) error {
		if firstKey == nil {
			firstKey = k
			firstValue = v
		}
		return nil
	}); err != nil {
		t.Fatalf("BatchRead: %v", err)
	}
	if !bytes.Equal(firstKey, []byte("a-1")) || !bytes.Equal(firstValue, []byte("value-a-1")) {
		t.Fatalf("owned bytes changed: key=%q value=%q", firstKey, firstValue)
	}
}
