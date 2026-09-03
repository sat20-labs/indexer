//go:build !pebble

package db

import "testing"

func TestNormalizeBadgerOpenOptionsKeepsIndexCacheBounded(t *testing.T) {
	got := normalizeBadgerOpenOptions(OpenOptions{BlockCacheMB: 0, IndexCacheMB: 0})
	if got.IndexCacheMB <= 0 {
		t.Fatalf("IndexCacheMB=%d; Badger zero means unbounded", got.IndexCacheMB)
	}
	if got.NumCompactors <= 0 {
		t.Fatalf("NumCompactors=%d, want positive", got.NumCompactors)
	}
}
