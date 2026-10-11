package common

// DBReadView pins one immutable database version. A caller must Close it before
// closing the owning database. Read returns owned bytes; Scan callback bytes
// are borrowed unless CopyKey/CopyValue are requested. Implementations serialize
// Close against active reads and must report iterator/storage errors.
type DBReadView interface {
	Read(key []byte) ([]byte, error)
	Scan(options ScanOptions, visit func(key, value []byte) error) error
	Close() error
}

// SnapshotDB is an optional capability, not a replacement for KVDB. It is used
// by the public, bounded SilentPay read views and their durable commit fence.
// Sync must make all preceding successful writes durable before returning.
type SnapshotDB interface {
	NewReadView() (DBReadView, error)
	Sync() error
}
