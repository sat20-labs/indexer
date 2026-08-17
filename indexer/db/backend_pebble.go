//go:build !badger

package db

import "github.com/sat20-labs/indexer/common"

func newKVDB(path string) common.KVDB {
	return NewPebbleDB(path, 0)
}

func newKVDBWithCache(path string, cacheSizeMB int) common.KVDB {
	return NewPebbleDB(path, cacheSizeMB)
}
