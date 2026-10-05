//go:build badger

package db

import "testing"

func TestNormalizeBadgerOpenOptionsKeepsIndexCacheBounded(t *testing.T) {
	for _, blockCacheMB := range []int{0, 1, 2048} {
		got := normalizeBadgerOpenOptions(OpenOptions{BlockCacheMB: blockCacheMB, IndexCacheMB: 0})
		if got.IndexCacheMB != 1 {
			t.Fatalf("block cache=%dMB, index cache=%dMB, want 1MB", blockCacheMB, got.IndexCacheMB)
		}
		if got.NumCompactors <= 0 {
			t.Fatalf("NumCompactors=%d, want positive", got.NumCompactors)
		}
	}
}

func TestNormalizeBadgerOpenOptionsPreservesExplicitIndexCache(t *testing.T) {
	got := normalizeBadgerOpenOptions(OpenOptions{BlockCacheMB: 2048, IndexCacheMB: 64})
	if got.IndexCacheMB != 64 {
		t.Fatalf("index cache=%dMB, want 64MB", got.IndexCacheMB)
	}
}
