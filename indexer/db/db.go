package db

import (
	"github.com/sat20-labs/indexer/common"
)

func RunDBGC(db common.KVDB) {
	//RunBadgerGC(db)
}

func NewKVDB(path string) common.KVDB {
	//return NewLevelDB(path)
	return NewPebbleDB(path, 0)
	//return NewBadgerDB(path)
	//return NewLMDB(path)
	//return NewBoltDB(path)
}

func NewKVDBWithCache(path string, cacheSizeMB int) common.KVDB {
	return NewPebbleDB(path, cacheSizeMB)
}
