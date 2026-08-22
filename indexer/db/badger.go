//go:build !pebble

package db

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dgraph-io/badger/v4"
	badgerOptions "github.com/dgraph-io/badger/v4/options"
	"github.com/sat20-labs/indexer/common"
)

const defaultBadgerBlockCacheMB = 2048

type badgerDB struct {
	path string
	db   *badger.DB
}

func openBadgerDB(path string, cacheSizeMB int) (*badger.DB, error) {
	if path == "" {
		path = "./data/db"
	}
	if cacheSizeMB < 0 {
		cacheSizeMB = defaultBadgerBlockCacheMB
	}

	cacheBytes := int64(cacheSizeMB) << 20
	indexCacheMB := cacheSizeMB / 4
	if indexCacheMB < 1 {
		indexCacheMB = 1
	}
	indexCacheBytes := int64(indexCacheMB) << 20

	opt := badger.DefaultOptions(path).
		WithDir(path).
		WithValueDir(path).
		WithBlockCacheSize(cacheBytes).
		WithIndexCacheSize(indexCacheBytes).
		WithLoggingLevel(badger.WARNING)
	if cacheSizeMB == 0 {
		// Badger requires a block cache when compression is enabled. A DB
		// with zero cache budget therefore writes new tables uncompressed.
		opt = opt.WithCompression(badgerOptions.None)
	}

	common.Log.Infof(
		"badger cache capacity: path=%s block=%dMB index=%dMB",
		path, cacheSizeMB, indexCacheMB,
	)

	return badger.Open(opt)
}

func NewBadgerDB(path string) common.KVDB {
	return NewBadgerDBWithCache(path, defaultBadgerBlockCacheMB)
}

func NewBadgerDBWithCache(path string, cacheSizeMB int) common.KVDB {
	bdb, err := openBadgerDB(path, cacheSizeMB)
	if err != nil {
		common.Log.Errorf("openBadgerDB %s failed: %v", path, err)
		return nil
	}

	return &badgerDB{
		path: path,
		db:   bdb,
	}
}

func (b *badgerDB) RunGC() error {
	if b == nil || b.db == nil || b.db.IsClosed() {
		return nil
	}

	const discardRatio = 0.5
	start := time.Now()
	rewrites := 0
	for {
		err := b.db.RunValueLogGC(discardRatio)
		if errors.Is(err, badger.ErrNoRewrite) {
			break
		}
		if err != nil {
			return fmt.Errorf("badger value log GC %s: %w", b.path, err)
		}
		rewrites++
	}

	if err := b.db.Sync(); err != nil {
		return fmt.Errorf("sync Badger DB after GC %s: %w", b.path, err)
	}
	common.Log.Infof(
		"badger value log GC completed: path=%s rewrites=%d elapsed=%v",
		b.path, rewrites, time.Since(start),
	)
	return nil
}

func (b *badgerDB) get(key []byte) ([]byte, error) {
	var valCopy []byte
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return common.ErrKeyNotFound
			}
			return err
		}
		return item.Value(func(val []byte) error {
			valCopy = append([]byte{}, val...)
			return nil
		})
	})
	return valCopy, err
}

func (b *badgerDB) put(key, value []byte) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

func (b *badgerDB) remove(key []byte) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

func (b *badgerDB) close() error {
	return b.db.Close()
}

func (b *badgerDB) Read(key []byte) ([]byte, error) {
	return b.get(key)
}

func (b *badgerDB) Write(key, value []byte) error {
	return b.put(key, value)
}

func (b *badgerDB) Delete(key []byte) error {
	return b.remove(key)
}

func (b *badgerDB) DropPrefix(prefix []byte) error {
	return b.db.DropPrefix(prefix)
}

func (b *badgerDB) DropAll() error {
	return b.db.DropAll()
}

func (b *badgerDB) Close() error {
	return b.close()
}

func badgerPrefixSuccessor(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	result := append([]byte(nil), prefix...)
	for i := len(result) - 1; i >= 0; i-- {
		if result[i] != 0xff {
			result[i]++
			return result[:i+1]
		}
	}
	return nil
}

func scanCallbackResult(err error) error {
	if errors.Is(err, common.ErrStopScan) {
		return nil
	}
	return err
}

func (b *badgerDB) Scan(options common.ScanOptions, r func(k, v []byte) error) error {
	opt := badger.DefaultIteratorOptions
	opt.PrefetchValues = !options.KeysOnly
	if options.PrefetchSize > 0 {
		opt.PrefetchSize = options.PrefetchSize
	}
	opt.Reverse = options.Reverse
	opt.Prefix = options.Prefix

	return b.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(opt)
		defer it.Close()

		if len(options.Start) > 0 {
			it.Seek(options.Start)
			if it.Valid() && !options.StartInclusive && bytes.Equal(it.Item().Key(), options.Start) {
				it.Next()
			}
		} else if options.Reverse && len(options.Prefix) > 0 {
			if upper := badgerPrefixSuccessor(options.Prefix); upper != nil {
				it.Seek(upper)
			} else {
				it.Rewind()
			}
			if it.Valid() && !bytes.HasPrefix(it.Item().Key(), options.Prefix) {
				it.Next()
			}
		} else {
			it.Rewind()
		}

		count := 0
		for ; it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()
			if len(options.Prefix) > 0 && !bytes.HasPrefix(key, options.Prefix) {
				break
			}
			if options.CopyKey {
				key = item.KeyCopy(nil)
			}

			var err error
			if options.KeysOnly {
				err = r(key, nil)
			} else {
				err = item.Value(func(value []byte) error {
					if options.CopyValue {
						value = append([]byte(nil), value...)
					}
					return r(key, value)
				})
			}
			if err != nil {
				return scanCallbackResult(err)
			}

			count++
			if options.Limit > 0 && count >= options.Limit {
				break
			}
		}
		return nil
	})
}

func (b *badgerDB) BatchRead(prefix []byte, reverse bool, r func(k, v []byte) error) error {
	return b.Scan(common.ScanOptions{
		Prefix:       prefix,
		Reverse:      reverse,
		CopyKey:      true,
		CopyValue:    true,
		PrefetchSize: badger.DefaultIteratorOptions.PrefetchSize,
	}, r)
}

func (b *badgerDB) BatchReadV2(prefix, seekKey []byte, reverse bool, r func(k, v []byte) error) error {
	return b.Scan(common.ScanOptions{
		Prefix:         prefix,
		Start:          seekKey,
		Reverse:        reverse,
		StartInclusive: true,
		CopyKey:        true,
		CopyValue:      true,
		PrefetchSize:   badger.DefaultIteratorOptions.PrefetchSize,
	}, r)
}

type badgerReadBatch struct {
	txn *badger.Txn
}

func (br *badgerReadBatch) Get(key []byte) ([]byte, error) {
	item, err := br.txn.Get(key)
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, common.ErrKeyNotFound
		}
		return nil, err
	}
	var valCopy []byte
	err = item.Value(func(val []byte) error {
		valCopy = append([]byte{}, val...)
		return nil
	})
	return valCopy, err
}

func (br *badgerReadBatch) GetRef(key []byte) ([]byte, error) {
	item, err := br.txn.Get(key)
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, common.ErrKeyNotFound
		}
		return nil, err
	}

	// Badger only guarantees Item.Value bytes during its callback. Returning
	// that slice would expose a buffer whose lifetime has already ended.
	return item.ValueCopy(nil)
}

func (b *badgerDB) View(fn func(txn common.ReadBatch) error) error {
	return b.db.View(func(txn *badger.Txn) error {
		return fn(&badgerReadBatch{txn: txn})
	})
}

func (b *badgerDB) Update(fn func(any) error) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return fn(txn)
	})
}

func (b *badgerDB) BackupToFile(fname string) error {
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	return b.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		total := 0
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.KeyCopy(nil)
			err := item.Value(func(v []byte) error {
				total++
				return enc.Encode([2][]byte{k, append([]byte{}, v...)})
			})
			if err != nil {
				return err
			}
		}
		common.Log.Infof("BackupToFile %s succeed, total %d", fname, total)
		return nil
	})
}

func (b *badgerDB) RestoreFromFile(fname string) error {
	f, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	wb := b.db.NewWriteBatch()
	defer wb.Cancel()
	for {
		var kv [2][]byte
		if err := dec.Decode(&kv); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if err := wb.Set(kv[0], kv[1]); err != nil {
			return err
		}
	}
	return wb.Flush()
}

type badgerWriteBatch struct {
	wb *badger.WriteBatch
}

func (bw *badgerWriteBatch) Put(key, value []byte) error {
	return bw.wb.Set(key, value)
}

func (bw *badgerWriteBatch) Delete(key []byte) error {
	return bw.wb.Delete(key)
}

func (bw *badgerWriteBatch) Flush() error {
	return bw.wb.Flush()
}

func (bw *badgerWriteBatch) Close() {
	bw.wb.Cancel()
}

func (b *badgerDB) NewWriteBatch() common.WriteBatch {
	return &badgerWriteBatch{wb: b.db.NewWriteBatch()}
}
