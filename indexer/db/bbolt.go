package db

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sat20-labs/indexer/common"
	bolt "go.etcd.io/bbolt"
)

var defaultBucket = []byte("main")

type bboltDB struct {
	path string
	db   *bolt.DB
}

func openBoltDB(dbPath string, opt *bolt.Options) (*bolt.DB, error) {
	if err := os.MkdirAll(dbPath, 0700); err != nil {
		return nil, err
	}
	fullPath := filepath.Join(dbPath, "bolt.db")
	if opt == nil {
		opt = &bolt.Options{Timeout: 1 * time.Second}
	}
	db, err := bolt.Open(fullPath, 0600, opt)
	if err != nil {
		return nil, err
	}
	// 确保默认 bucket 存在
	err = db.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists(defaultBucket)
		return e
	})
	return db, err
}

func NewBoltDB(path string) common.KVDB {
	db, err := initBoltDB(path)
	if err != nil {
		common.Log.Errorf("initBoltDB failed, %v", err)
		return nil
	}
	return &bboltDB{path: path, db: db}
}

func initBoltDB(path string) (*bolt.DB, error) {
	if path == "" {
		path = "./data/db"
	}
	return openBoltDB(path, nil)
}

func (b *bboltDB) get(key []byte) ([]byte, error) {
	var valCopy []byte
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(defaultBucket)
		if bucket == nil {
			return common.ErrKeyNotFound
		}
		val := bucket.Get(key)
		if val == nil {
			return common.ErrKeyNotFound
		}
		valCopy = append([]byte{}, val...)
		return nil
	})
	return valCopy, err
}

func (b *bboltDB) put(key, value []byte) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(defaultBucket).Put(key, value)
	})
}

func (b *bboltDB) remove(key []byte) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(defaultBucket).Delete(key)
	})
}

func (b *bboltDB) close() error {
	return b.db.Close()
}

func (b *bboltDB) commit() error { return nil }

func (b *bboltDB) Read(key []byte) ([]byte, error) {
	return b.get(key)
}

func (b *bboltDB) Write(key, value []byte) error {
	return b.put(key, value)
}

func (b *bboltDB) Delete(key []byte) error {
	return b.remove(key)
}

func (b *bboltDB) DropPrefix(prefix []byte) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(defaultBucket)
		if bucket == nil {
			return nil
		}
		c := bucket.Cursor()
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			if err := c.Delete(); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *bboltDB) DropAll() error {
	return b.db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket(defaultBucket)
		if err != nil && !errors.Is(err, bolt.ErrBucketNotFound) {
			return err
		}
		_, err = tx.CreateBucket(defaultBucket)
		return err
	})
}

func (b *bboltDB) Close() error {
	return b.close()
}

func (b *bboltDB) iter(prefix, start []byte, reverse bool, r func(k, v []byte) error) error {
	return b.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(defaultBucket).Cursor()
		var k, v []byte
		if reverse {
			if start != nil {
				k, v = c.Seek(start)
				if k == nil || bytes.Compare(k, start) > 0 {
					k, v = c.Prev()
				}
			} else {
				k, v = c.Last()
			}
			for ; k != nil; k, v = c.Prev() {
				if len(prefix) > 0 && !bytes.HasPrefix(k, prefix) {
					continue
				}
				if err := r(append([]byte{}, k...), append([]byte{}, v...)); err != nil {
					return err
				}
			}
		} else {
			if start != nil {
				k, v = c.Seek(start)
			} else {
				k, v = c.First()
			}
			for ; k != nil; k, v = c.Next() {
				if len(prefix) > 0 && !bytes.HasPrefix(k, prefix) {
					continue
				}
				if err := r(append([]byte{}, k...), append([]byte{}, v...)); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (b *bboltDB) BatchRead(prefix []byte, reverse bool, r func(k, v []byte) error) error {
	return b.iter(prefix, nil, reverse, r)
}

func (b *bboltDB) BatchReadV2(prefix, seekKey []byte, reverse bool, r func(k, v []byte) error) error {
	return b.iter(prefix, seekKey, reverse, r)
}

type boltReadBatch struct {
	tx *bolt.Tx
}

func (br *boltReadBatch) Get(key []byte) ([]byte, error) {
	bucket := br.tx.Bucket(defaultBucket)
	if bucket == nil {
		return nil, common.ErrKeyNotFound
	}
	val := bucket.Get(key)
	if val == nil {
		return nil, common.ErrKeyNotFound
	}
	return append([]byte{}, val...), nil
}

// 注意：GetRef 返回的数据仅在 View 回调函数内部有效，事务结束后失效！
func (br *boltReadBatch) GetRef(key []byte) ([]byte, error) {
	bucket := br.tx.Bucket(defaultBucket)
	if bucket == nil {
		return nil, common.ErrKeyNotFound
	}
	val := bucket.Get(key)
	if val == nil {
		return nil, common.ErrKeyNotFound
	}
	return val, nil
}

func (b *bboltDB) View(fn func(txn common.ReadBatch) error) error {
	return b.db.View(func(tx *bolt.Tx) error {
		return fn(&boltReadBatch{tx: tx})
	})
}

// Update：提供直接的事务访问
func (b *bboltDB) Update(fn func(tx *bolt.Tx) error) error {
	return b.db.Update(fn)
}

func (b *bboltDB) BackupToFile(fname string) error {
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	total := 0
	err = b.BatchRead(nil, false, func(k, v []byte) error {
		total++
		return enc.Encode([2][]byte{k, v})
	})
	if err != nil {
		common.Log.Errorf("BackupToFile %s failed, %v", fname, err)
		return err
	}
	common.Log.Infof("BackupToFile %s succeed, total %d", fname, total)
	return nil
}

func (b *bboltDB) RestoreFromFile(backupFile string) error {
	f, err := os.Open(backupFile)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(defaultBucket)
		for {
			var kv [2][]byte
			if err := dec.Decode(&kv); err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			if err := bucket.Put(kv[0], kv[1]); err != nil {
				return err
			}
		}
		return nil
	})
}

type boltWriteBatch struct {
	tx     *bolt.Tx
	bucket *bolt.Bucket
	closed bool
}

func (bw *boltWriteBatch) Put(key, value []byte) error {
	if bw.closed {
		return errors.New("writebatch closed")
	}
	return bw.bucket.Put(key, value)
}

func (bw *boltWriteBatch) Delete(key []byte) error {
	if bw.closed {
		return errors.New("writebatch closed")
	}
	return bw.bucket.Delete(key)
}

func (bw *boltWriteBatch) Flush() error {
	if bw.closed {
		return errors.New("writebatch closed")
	}
	bw.closed = true
	return bw.tx.Commit()
}

func (bw *boltWriteBatch) Close() {
	if !bw.closed {
		_ = bw.tx.Rollback()
		bw.closed = true
	}
}

func (b *bboltDB) NewWriteBatch() common.WriteBatch {
	tx, err := b.db.Begin(true)
	if err != nil {
		return nil
	}
	return &boltWriteBatch{
		tx:     tx,
		bucket: tx.Bucket(defaultBucket),
	}
}
