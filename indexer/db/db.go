package db

import (
	"errors"

	"github.com/sat20-labs/indexer/common"
)

var ErrGCUnsupported = errors.New("database backend does not support online GC")
var ErrFinalizeUnsupported = errors.New("database backend does not support finalize compaction")

type OpenOptions struct {
	BlockCacheMB  int
	IndexCacheMB  int
	NumCompactors int
}

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
	return newKVDB(path)
	//return NewLMDB(path)
	//return NewBoltDB(path)
}

func NewKVDBWithCache(path string, cacheSizeMB int) common.KVDB {
	return newKVDBWithCache(path, cacheSizeMB)
}

func NewKVDBWithOptions(path string, options OpenOptions) common.KVDB {
	return newKVDBWithOptions(path, options)
}

type finalizer interface {
	Finalize(workers int) error
}

func FinalizeDB(kvdb common.KVDB, workers int) error {
	if kvdb == nil {
		return ErrFinalizeUnsupported
	}
	f, ok := kvdb.(finalizer)
	if !ok {
		return ErrFinalizeUnsupported
	}
	return f.Finalize(workers)
}
