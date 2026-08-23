package common

import (
	"bytes"
	"errors"
)

var (
	ErrKeyNotFound = errors.New("key not found")
	ErrStopScan    = errors.New("stop scan")
)

type ScanOptions struct {
	Prefix         []byte
	Start          []byte
	StartInclusive bool
	Reverse        bool
	Limit          int
	KeysOnly       bool
	CopyKey        bool
	CopyValue      bool
}

type ReadBatch interface {
	Get(key []byte) ([]byte, error)    // 获得数据的新copy
	GetRef(key []byte) ([]byte, error) // 数据的引用，不能持久使用
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
	Scan(options ScanOptions, r func(k, v []byte) error) error

	// 遍历读
	BatchRead(prefix []byte, reverse bool, r func(k, v []byte) error) error
	BatchReadV2(prefix, seekKey []byte, reverse bool, r func(k, v []byte) error) error // 只用于非客户端模式下

	// 随机读
	View(func(ReadBatch) error) error
}

func ScanWithBatchRead(
	options ScanOptions,
	batchRead func(prefix []byte, reverse bool, r func(k, v []byte) error) error,
	r func(k, v []byte) error,
) error {
	count := 0
	err := batchRead(options.Prefix, options.Reverse, func(k, v []byte) error {
		if len(options.Start) > 0 {
			cmp := bytes.Compare(k, options.Start)
			if (!options.Reverse && (cmp < 0 || (cmp == 0 && !options.StartInclusive))) ||
				(options.Reverse && (cmp > 0 || (cmp == 0 && !options.StartInclusive))) {
				return nil
			}
		}

		if options.CopyKey {
			k = append([]byte(nil), k...)
		}
		if options.KeysOnly {
			v = nil
		} else if options.CopyValue {
			v = append([]byte(nil), v...)
		}
		if err := r(k, v); err != nil {
			return err
		}
		count++
		if options.Limit > 0 && count >= options.Limit {
			return ErrStopScan
		}
		return nil
	})
	if errors.Is(err, ErrStopScan) {
		return nil
	}
	return err
}
