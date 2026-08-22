package common

import "errors"

var (
	ErrKeyNotFound = errors.New("key not found")
	// ErrStopScan lets a scan callback stop successfully without converting
	// normal control flow into a database error.
	ErrStopScan = errors.New("stop scan")
)

type ReadBatch interface {
	Get(key []byte) ([]byte, error)    // 获得数据的新copy
	GetRef(key []byte) ([]byte, error) // 数据的引用，不能持久使用
}

// BulkWriteBatch is optimized for throughput. A backend may split one logical
// batch into multiple physical transactions. Flush guarantees that queued
// operations have completed, but it does not promise rollback of the entire
// logical batch after a partial failure. Indexer failures invalidate the DB and
// require a rebuild.
type BulkWriteBatch interface {
	Put(key, value []byte) error
	Delete(key []byte) error
	Flush() error
	Close()
}

// WriteBatch remains as a source-compatible name for existing indexers. New
// storage code should use BulkWriteBatch so the non-atomic semantics are clear.
type WriteBatch = BulkWriteBatch

type ScanOptions struct {
	Prefix         []byte
	Start          []byte
	Reverse        bool
	StartInclusive bool
	Limit          int
	KeysOnly       bool
	CopyKey        bool
	CopyValue      bool
	PrefetchSize   int
}

// 每个调用都是完整的transaction
type KVDB interface {
	DropAll() error
	DropPrefix([]byte) error

	Read(key []byte) ([]byte, error)
	Write(key, value []byte) error
	Delete(key []byte) error
	Close() error

	NewWriteBatch() WriteBatch

	// Scan is the canonical iterator contract. Unless CopyKey/CopyValue are
	// requested, key/value bytes are valid only during the callback.
	Scan(options ScanOptions, r func(k, v []byte) error) error

	// Compatibility wrappers. New code should use Scan.
	BatchRead(prefix []byte, reverse bool, r func(k, v []byte) error) error
	BatchReadV2(prefix, seekKey []byte, reverse bool, r func(k, v []byte) error) error

	// 随机读
	View(func(ReadBatch) error) error
}
