package common

import "errors"


var (
	ErrKeyNotFound = errors.New("Key not found")
)

type ReadBatch interface {
	Get(key []byte) ([]byte, error)    // 获得数据的新copy
	GetRef(key []byte) ([]byte, error) // 数据的引用，不能持久使用
	MultiGet(keys [][]byte) ([][]byte, error) // 需要key排序才能提高性能，获得数据的新copy
	MultiGetSorted(keys [][]byte) (map[string][]byte, error) // 获得数据的新copy
}

type WriteBatch interface {
	Put(key, value []byte) error
	Delete(key []byte) error
	Flush() error
	Close()
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
	
	// 遍历读
	BatchRead(prefix []byte, reverse bool, r func(k, v []byte) error) error
	BatchReadV2(prefix, seekKey []byte, reverse bool, r func(k, v []byte) error) error  // 只用于非客户端模式下

	// 多次读
	View(func(ReadBatch) error) error
}
