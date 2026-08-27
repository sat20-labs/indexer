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
		t.Fatalf("badger cache=%dMB, want %dMB", cfg.DB.BadgerBlockCacheTotalMB, 32*1024)
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
