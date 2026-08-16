package db

import (
	"errors"

	"github.com/sat20-labs/indexer/common"
)

var ErrGCUnsupported = errors.New("database backend does not support online GC")

type gcRunner interface {
	RunGC() error
}

func RunDBGC(kvdb common.KVDB) error {
	if kvdb == nil {
		return ErrGCUnsupported
	}
	runner, ok := kvdb.(gcRunner)
	if !ok {
		return ErrGCUnsupported
	}
	return runner.RunGC()
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
