//go:build pebble

package db

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/cockroachdb/pebble"
	"github.com/sat20-labs/indexer/common"
)

func newPebbleScanTestDB(t *testing.T) *pebbleDB {
	t.Helper()
	raw, err := pebble.Open(t.TempDir(), &pebble.Options{})
	if err != nil {
		t.Fatalf("open Pebble: %v", err)
	}
	database := &pebbleDB{db: raw}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("close Pebble: %v", err)
		}
	})
	for _, key := range []string{"a-3", "b-1", "a-1", "a-2"} {
		if err := database.Write([]byte(key), []byte("value-"+key)); err != nil {
			t.Fatalf("write %s: %v", key, err)
		}
	}
	return database
}

func TestPebbleScanContract(t *testing.T) {
	database := newPebbleScanTestDB(t)
	tests := []struct {
		name    string
		options common.ScanOptions
		want    []string
	}{
		{name: "forward prefix", options: common.ScanOptions{Prefix: []byte("a-")}, want: []string{"a-1", "a-2", "a-3"}},
		{name: "reverse prefix", options: common.ScanOptions{Prefix: []byte("a-"), Reverse: true}, want: []string{"a-3", "a-2", "a-1"}},
		{name: "forward inclusive", options: common.ScanOptions{Prefix: []byte("a-"), Start: []byte("a-2"), StartInclusive: true}, want: []string{"a-2", "a-3"}},
		{name: "forward exclusive", options: common.ScanOptions{Prefix: []byte("a-"), Start: []byte("a-2")}, want: []string{"a-3"}},
		{name: "reverse inclusive", options: common.ScanOptions{Prefix: []byte("a-"), Start: []byte("a-2"), StartInclusive: true, Reverse: true}, want: []string{"a-2", "a-1"}},
		{name: "reverse exclusive", options: common.ScanOptions{Prefix: []byte("a-"), Start: []byte("a-2"), Reverse: true}, want: []string{"a-1"}},
		{name: "limit", options: common.ScanOptions{Prefix: []byte("a-"), Limit: 2}, want: []string{"a-1", "a-2"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got []string
			err := database.Scan(test.options, func(k, _ []byte) error {
				got = append(got, string(k))
				return nil
			})
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("keys=%v, want %v", got, test.want)
			}
		})
	}
}

func TestPebbleScanKeysOnlyStopErrorAndOwnership(t *testing.T) {
	database := newPebbleScanTestDB(t)
	var firstKey, firstValue []byte
	count := 0
	err := database.Scan(common.ScanOptions{
		Prefix: []byte("a-"), CopyKey: true, CopyValue: true,
	}, func(k, v []byte) error {
		if count == 0 {
			firstKey, firstValue = k, v
		}
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("owned Scan: %v", err)
	}
	if !bytes.Equal(firstKey, []byte("a-1")) || !bytes.Equal(firstValue, []byte("value-a-1")) {
		t.Fatalf("owned bytes changed: key=%q value=%q", firstKey, firstValue)
	}

	count = 0
	err = database.Scan(common.ScanOptions{Prefix: []byte("a-"), KeysOnly: true}, func(_ []byte, v []byte) error {
		if v != nil {
			t.Fatalf("KeysOnly value=%q, want nil", v)
		}
		count++
		return common.ErrStopScan
	})
	if err != nil || count != 1 {
		t.Fatalf("stop result: count=%d err=%v", count, err)
	}

	wantErr := errors.New("callback failed")
	err = database.Scan(common.ScanOptions{Prefix: []byte("a-")}, func(_, _ []byte) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("callback error=%v, want %v", err, wantErr)
	}
}
