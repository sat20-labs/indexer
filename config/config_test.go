package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfigFixture(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "indexer.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}
	return path
}

func TestLoadYamlConfDefaultsBadgerBlockCacheTo32GiB(t *testing.T) {
	path := writeConfigFixture(t, `chain: testnet4
db:
  path: ./db/test
log:
  level: info
  path: ./log
basic_index:
  period_flush_to_db: 10
`)
	cfg, err := LoadYamlConf(path)
	if err != nil {
		t.Fatalf("LoadYamlConf: %v", err)
	}
	if cfg.DB.BadgerBlockCacheTotalMB != 32*1024 {
		t.Fatalf("badger block cache=%dMB, want %dMB", cfg.DB.BadgerBlockCacheTotalMB, 32*1024)
	}
	if cfg.DB.BadgerIndexCacheTotalMB != 4*1024 {
		t.Fatalf("badger index cache=%dMB, want %dMB", cfg.DB.BadgerIndexCacheTotalMB, 4*1024)
	}
	if cfg.DB.BadgerNumCompactors != 8 {
		t.Fatalf("badger compactors=%d, want 8", cfg.DB.BadgerNumCompactors)
	}
	if cfg.DB.BadgerFlattenWorkers != 4 {
		t.Fatalf("badger flatten workers=%d, want 4", cfg.DB.BadgerFlattenWorkers)
	}
	if !cfg.DB.ShouldFlattenBadgerOnFinalize() {
		t.Fatal("Badger finalize flatten should default to enabled")
	}
}

func TestLoadYamlConfUsesConfiguredBadgerBlockCache(t *testing.T) {
	path := writeConfigFixture(t, `chain: testnet4
db:
  path: ./db/test
  badger_block_cache_total_mb: 49152
log:
  level: info
  path: ./log
basic_index:
  period_flush_to_db: 10
`)
	cfg, err := LoadYamlConf(path)
	if err != nil {
		t.Fatalf("LoadYamlConf: %v", err)
	}
	if cfg.DB.BadgerBlockCacheTotalMB != 49152 {
		t.Fatalf("badger cache=%dMB, want 49152MB", cfg.DB.BadgerBlockCacheTotalMB)
	}
}

func TestLoadYamlConfUsesConfiguredBadgerMemoryAndCompaction(t *testing.T) {
	path := writeConfigFixture(t, `chain: testnet4
db:
  path: ./db/test
  badger_block_cache_total_mb: 49152
  badger_index_cache_total_mb: 6144
  badger_num_compactors: 6
  badger_flatten_on_finalize: false
  badger_flatten_workers: 3
log:
  level: info
  path: ./log
basic_index:
  period_flush_to_db: 10
`)
	cfg, err := LoadYamlConf(path)
	if err != nil {
		t.Fatalf("LoadYamlConf: %v", err)
	}
	if cfg.DB.BadgerBlockCacheTotalMB != 49152 || cfg.DB.BadgerIndexCacheTotalMB != 6144 {
		t.Fatalf("cache config = block:%d index:%d", cfg.DB.BadgerBlockCacheTotalMB, cfg.DB.BadgerIndexCacheTotalMB)
	}
	if cfg.DB.BadgerNumCompactors != 6 || cfg.DB.BadgerFlattenWorkers != 3 {
		t.Fatalf("compaction config = compactors:%d workers:%d", cfg.DB.BadgerNumCompactors, cfg.DB.BadgerFlattenWorkers)
	}
	if cfg.DB.ShouldFlattenBadgerOnFinalize() {
		t.Fatal("explicit badger_flatten_on_finalize=false ignored")
	}
}
