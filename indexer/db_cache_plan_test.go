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
