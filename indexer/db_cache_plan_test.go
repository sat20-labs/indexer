package indexer

import "testing"

func TestAllocateBadgerBlockCacheUsesExactProcessBudget(t *testing.T) {
	const total = 1024
	plan := allocateBadgerBlockCache(total)
	got := 0
	for _, value := range plan {
		got += value
		if value < 0 {
			t.Fatalf("negative cache allocation: %v", plan)
		}
	}
	if got != total {
		t.Fatalf("allocated %dMB, want %dMB: %v", got, total, plan)
	}
	if plan["atom"] != 0 {
		t.Fatalf("disabled atom cache=%dMB, want 0", plan["atom"])
	}
	if plan["base"] <= plan["nft"] || plan["nft"] <= plan["exotic"] {
		t.Fatalf("unexpected cache priorities: %v", plan)
	}
}

func TestAllocateBadgerBlockCacheCanDisableBlockCache(t *testing.T) {
	plan := allocateBadgerBlockCache(0)
	for name, value := range plan {
		if value != 0 {
			t.Fatalf("%s cache=%dMB, want 0", name, value)
		}
	}
}

func TestConfiguredBadgerBlockCacheUsesYAMLValue(t *testing.T) {
	t.Setenv("INDEXER_BADGER_BLOCK_CACHE_TOTAL_MB", "")
	t.Setenv("INDEXER_DB_CACHE_TOTAL_MB", "")
	if got := configuredBadgerBlockCacheTotalMB(48 * 1024); got != 48*1024 {
		t.Fatalf("configured cache=%dMB, want %dMB", got, 48*1024)
	}
}

func TestConfiguredBadgerBlockCacheEnvironmentOverridesYAML(t *testing.T) {
	t.Setenv("INDEXER_BADGER_BLOCK_CACHE_TOTAL_MB", "65536")
	t.Setenv("INDEXER_DB_CACHE_TOTAL_MB", "")
	if got := configuredBadgerBlockCacheTotalMB(32 * 1024); got != 65536 {
		t.Fatalf("configured cache=%dMB, want env override 65536MB", got)
	}
}

func TestConfiguredBadgerBlockCacheEnvironmentCanDisableCache(t *testing.T) {
	t.Setenv("INDEXER_BADGER_BLOCK_CACHE_TOTAL_MB", "0")
	t.Setenv("INDEXER_DB_CACHE_TOTAL_MB", "")
	if got := configuredBadgerBlockCacheTotalMB(32 * 1024); got != 0 {
		t.Fatalf("configured cache=%dMB, want env override 0MB", got)
	}
}

func TestConfiguredBadgerBlockCacheDefaultsTo32GiB(t *testing.T) {
	t.Setenv("INDEXER_BADGER_BLOCK_CACHE_TOTAL_MB", "")
	t.Setenv("INDEXER_DB_CACHE_TOTAL_MB", "")
	if got := configuredBadgerBlockCacheTotalMB(0); got != 32*1024 {
		t.Fatalf("configured cache=%dMB, want default %dMB", got, 32*1024)
	}
}
