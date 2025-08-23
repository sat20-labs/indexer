package db

// import (
// 	"bytes"
// 	"encoding/gob"
// 	"errors"
// 	"io"
// 	"os"

// 	"github.com/dgraph-io/badger/v4"
// 	"github.com/sat20-labs/indexer/common"
// )

// type badgerDB struct {
// 	path string
// 	db   *badger.DB
// }

// func openBadgerDB(filepath string) (*badger.DB, error) {
	
// 	opt := badger.DefaultOptions(filepath).
// 		WithBlockCacheSize(3000 << 20).
// 		WithDir(filepath).
// 		WithValueDir(filepath).
// 		WithLoggingLevel(badger.WARNING)
	
// 	return badger.Open(opt)
// }

// func NewBadgerDB(path string) common.KVDB {
// 	db, err := initBadgerDB(path)
// 	if err != nil {
// 		common.Log.Errorf("initBadgerDB failed, %v", err)
// 		return nil
// 	}
// 	return &badgerDB{path: path, db: db}
// }

// func initBadgerDB(path string) (*badger.DB, error) {
// 	if path == "" {
// 		path = "./data/db"
// 	}
// 	return openBadgerDB(path)
// }

// func (b *badgerDB) get(key []byte) ([]byte, error) {
// 	var valCopy []byte
// 	err := b.db.View(func(txn *badger.Txn) error {
// 		item, err := txn.Get(key)
// 		if err != nil {
// 			if errors.Is(err, badger.ErrKeyNotFound) {
// 				return common.ErrKeyNotFound
// 			}
// 			return err
// 		}
// 		return item.Value(func(val []byte) error {
// 			valCopy = append([]byte{}, val...)
// 			return nil
// 		})
// 	})
// 	return valCopy, err
// }

// func (b *badgerDB) put(key, value []byte) error {
// 	return b.db.Update(func(txn *badger.Txn) error {
// 		return txn.Set(key, value)
// 	})
// }

// func (b *badgerDB) remove(key []byte) error {
// 	return b.db.Update(func(txn *badger.Txn) error {
// 		return txn.Delete(key)
// 	})
// }

// func (b *badgerDB) close() error {
// 	return b.db.Close()
// }

// func (b *badgerDB) commit() error {
// 	// Badger 写事务自动 commit，这里保持接口一致
// 	return nil
// }

// func (b *badgerDB) Read(key []byte) ([]byte, error) {
// 	return b.get(key)
// }

// func (b *badgerDB) Write(key, value []byte) error {
// 	return b.put(key, value)
// }

// func (b *badgerDB) Delete(key []byte) error {
// 	return b.remove(key)
// }

// func (b *badgerDB) DropPrefix(prefix []byte) error {
// 	return b.db.DropPrefix(prefix)
// }

// func (b *badgerDB) DropAll() error {
// 	return b.db.DropAll()
// }

// func (b *badgerDB) Close() error {
// 	return b.close()
// }

// func (b *badgerDB) iter(prefix, start []byte, reverse bool, r func(k, v []byte) error) error {
// 	opt := badger.DefaultIteratorOptions
// 	opt.PrefetchValues = true
// 	opt.Reverse = reverse

// 	return b.db.View(func(txn *badger.Txn) error {
// 		it := txn.NewIterator(opt)
// 		defer it.Close()

// 		var seekKey []byte
// 		if len(start) > 0 {
// 			seekKey = start
// 		} else if len(prefix) > 0 {
// 			seekKey = prefix
// 		}

// 		if seekKey != nil {
// 			it.Seek(seekKey)
// 		} else {
// 			if reverse {
// 				it.Rewind()
// 				if !it.Valid() {
// 					return nil
// 				}
// 				it.Seek([]byte{0xFF, 0xFF, 0xFF, 0xFF})
// 			} else {
// 				it.Rewind()
// 			}
// 		}

// 		for ; it.Valid(); it.Next() {
// 			item := it.Item()
// 			k := item.Key()
// 			if len(prefix) > 0 && !bytes.HasPrefix(k, prefix) {
// 				break
// 			}
// 			err := item.Value(func(v []byte) error {
// 				return r(append([]byte{}, k...), append([]byte{}, v...))
// 			})
// 			if err != nil {
// 				return err
// 			}
// 		}
// 		return nil
// 	})
// }

// func (b *badgerDB) BatchRead(prefix []byte, reverse bool, r func(k, v []byte) error) error {
// 	return b.iter(prefix, nil, reverse, r)
// }

// func (b *badgerDB) BatchReadV2(prefix, seekKey []byte, reverse bool, r func(k, v []byte) error) error {
// 	return b.iter(prefix, seekKey, reverse, r)
// }

// type badgerReadBatch struct {
// 	txn *badger.Txn
// }

// func (br *badgerReadBatch) Get(key []byte) ([]byte, error) {
// 	item, err := br.txn.Get(key)
// 	if err != nil {
// 		if errors.Is(err, badger.ErrKeyNotFound) {
// 			return nil, common.ErrKeyNotFound
// 		}
// 		return nil, err
// 	}
// 	var valCopy []byte
// 	err = item.Value(func(val []byte) error {
// 		valCopy = append([]byte{}, val...)
// 		return nil
// 	})
// 	return valCopy, err
// }

// func (br *badgerReadBatch) GetRef(key []byte) ([]byte, error) {
// 	item, err := br.txn.Get(key)
// 	if err != nil {
// 		if errors.Is(err, badger.ErrKeyNotFound) {
// 			return nil, common.ErrKeyNotFound
// 		}
// 		return nil, err
// 	}
// 	var val []byte
// 	err = item.Value(func(v []byte) error {
// 		val = v
// 		return nil
// 	})
// 	return val, err
// }

// func (br *badgerReadBatch) MultiGet(keys [][]byte) ([][]byte, error) {
// 	results := make([][]byte, len(keys))
// 	for i, k := range keys {
// 		val, err := br.Get(k)
// 		if err != nil {
// 			results[i] = nil
// 		} else {
// 			results[i] = val
// 		}
// 	}
// 	return results, nil
// }

// func (br *badgerReadBatch) MultiGetSorted(keys [][]byte) (map[string][]byte, error) {
// 	result := make(map[string][]byte, len(keys))
// 	for _, k := range keys {
// 		v, _ := br.Get(k)
// 		if v != nil {
// 			result[string(k)] = v
// 		}
// 	}
// 	return result, nil
// }

// func (b *badgerDB) View(fn func(txn common.ReadBatch) error) error {
// 	return b.db.View(func(txn *badger.Txn) error {
// 		return fn(&badgerReadBatch{txn: txn})
// 	})
// }

// func (b *badgerDB) Update(fn func(any) error) error {
// 	return b.db.Update(func(txn *badger.Txn) error {
// 		return fn(txn)
// 	})
// }

// func (b *badgerDB) BackupToFile(fname string) error {
// 	f, err := os.Create(fname)
// 	if err != nil {
// 		return err
// 	}
// 	defer f.Close()

// 	enc := gob.NewEncoder(f)
// 	return b.db.View(func(txn *badger.Txn) error {
// 		it := txn.NewIterator(badger.DefaultIteratorOptions)
// 		defer it.Close()
// 		total := 0
// 		for it.Rewind(); it.Valid(); it.Next() {
// 			item := it.Item()
// 			k := item.KeyCopy(nil)
// 			err := item.Value(func(v []byte) error {
// 				total++
// 				return enc.Encode([2][]byte{k, append([]byte{}, v...)})
// 			})
// 			if err != nil {
// 				return err
// 			}
// 		}
// 		common.Log.Infof("BackupToFile %s succeed, total %d", fname, total)
// 		return nil
// 	})
// }

// func (b *badgerDB) RestoreFromFile(fname string) error {
// 	f, err := os.Open(fname)
// 	if err != nil {
// 		return err
// 	}
// 	defer f.Close()
// 	dec := gob.NewDecoder(f)
// 	return b.db.Update(func(txn *badger.Txn) error {
// 		for {
// 			var kv [2][]byte
// 			if err := dec.Decode(&kv); err != nil {
// 				if err == io.EOF {
// 					break
// 				}
// 				return err
// 			}
// 			if err := txn.Set(kv[0], kv[1]); err != nil {
// 				return err
// 			}
// 		}
// 		return nil
// 	})
// }

// type badgerWriteBatch struct {
// 	wb *badger.WriteBatch
// }

// func (bw *badgerWriteBatch) Put(key, value []byte) error {
// 	return bw.wb.Set(key, value)
// }

// func (bw *badgerWriteBatch) Delete(key []byte) error {
// 	return bw.wb.Delete(key)
// }

// func (bw *badgerWriteBatch) Flush() error {
// 	return bw.wb.Flush()
// }

// func (bw *badgerWriteBatch) Close() {
// 	bw.wb.Cancel()
// }

// func (b *badgerDB) NewWriteBatch() common.WriteBatch {
// 	return &badgerWriteBatch{wb: b.db.NewWriteBatch()}
// }
