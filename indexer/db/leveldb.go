package db

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"os"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type kvDB struct {
	path string
	db   *leveldb.DB
}

func openDB(filepath string, o *opt.Options) (*leveldb.DB, error) {
	if o == nil {
		o = &opt.Options{}
	}
	db, err := leveldb.OpenFile(filepath, o)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func NewKVDB(path string) KVDB {
	db, err := initDB(path)
	if err != nil {
		return nil
	}
	kvdb := kvDB{path: path, db: db}
	return &kvdb
}

func initDB(path string) (*leveldb.DB, error) {
	if path == "" {
		path = "./data/db"
	}
	opts := &opt.Options{}
	return openDB(path, opts)
}

func (p *kvDB) get(key []byte) ([]byte, error) {
	val, err := p.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, ErrKeyNotFound
		}
		return nil, err
	}
	return append([]byte{}, val...), nil
}

func (p *kvDB) put(key, value []byte) error {
	return p.db.Put(key, value, nil)
}

func (p *kvDB) remove(key []byte) error {
	return p.db.Delete(key, nil)
}

func (p *kvDB) close() error {
	return p.db.Close()
}

func (p *kvDB) commit() error { return nil }

func (p *kvDB) Read(key []byte) ([]byte, error) {
	return p.get(key)
}

func (p *kvDB) Write(key, value []byte) error {
	if err := p.put(key, value); err != nil {
		return err
	}
	return p.commit()
}

func (p *kvDB) Delete(key []byte) error {
	if err := p.remove(key); err != nil {
		return err
	}
	return p.commit()
}

func (p *kvDB) DropPrefix(prefix []byte) error {
	deletingKeyMap := make(map[string]bool)
	err := p.iterForwardWithPrefix(prefix, nil, func(k []byte, v []byte) error {
		deletingKeyMap[string(k)] = true
		return nil
	})
	if err != nil {
		return err
	}
	wb := p.NewWriteBatch()
	defer wb.Close()

	for k := range deletingKeyMap {
		wb.Delete([]byte(k))
	}
	return wb.Flush()
}

func (p *kvDB) Close() error {
	return p.close()
}

func (p *kvDB) iterForwardWithPrefix(prefix []byte, start []byte, r func(k, v []byte) error) error {
	var itUtil *util.Range
	if len(prefix) > 0 {
		itUtil = util.BytesPrefix(prefix)
	}
	it := p.db.NewIterator(itUtil, nil)
	defer it.Release()

	if len(start) > 0 {
		it.Seek(start)
	} else if len(prefix) > 0 {
		it.Seek(prefix)
	} else {
		it.First()
	}

	for ; it.Valid(); it.Next() {
		k := it.Key()
		if len(prefix) > 0 && !bytes.HasPrefix(k, prefix) {
			break
		}
		if err := r(append([]byte{}, k...), append([]byte{}, it.Value()...)); err != nil {
			return err
		}
	}
	return it.Error()
}

func (p *kvDB) BatchRead(prefix []byte, reverse bool, r func(k, v []byte) error) error {
	if !reverse {
		return p.iterForwardWithPrefix(prefix, nil, r)
	}
	var kvs [][2][]byte
	if err := p.iterForwardWithPrefix(prefix, nil, func(k, v []byte) error {
		kvs = append(kvs, [2][]byte{append([]byte{}, k...), append([]byte{}, v...)})
		return nil
	}); err != nil {
		return err
	}
	for i := len(kvs) - 1; i >= 0; i-- {
		if err := r(kvs[i][0], kvs[i][1]); err != nil {
			return err
		}
	}
	return nil
}

func (p *kvDB) BatchReadV2(prefix, seekKey []byte, reverse bool, r func(k, v []byte) error) error {
	if !reverse {
		start := seekKey
		if len(start) == 0 && len(prefix) > 0 {
			start = prefix
		}
		return p.iterForwardWithPrefix(prefix, start, r)
	}
	var kvs [][2][]byte
	start := seekKey
	if len(start) == 0 && len(prefix) > 0 {
		start = prefix
	}
	if err := p.iterForwardWithPrefix(prefix, start, func(k, v []byte) error {
		kvs = append(kvs, [2][]byte{append([]byte{}, k...), append([]byte{}, v...)})
		return nil
	}); err != nil {
		return err
	}
	if len(seekKey) > 0 {
		idx := -1
		for i := 0; i < len(kvs); i++ {
			if bytes.Compare(kvs[i][0], seekKey) <= 0 {
				idx = i
			} else {
				break
			}
		}
		for i := idx; i >= 0; i-- {
			if err := r(kvs[i][0], kvs[i][1]); err != nil {
				return err
			}
		}
		return nil
	}
	for i := len(kvs) - 1; i >= 0; i-- {
		if err := r(kvs[i][0], kvs[i][1]); err != nil {
			return err
		}
	}
	return nil
}

func (p *kvDB) BackupToFile(fname string) error {
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	return p.BatchRead(nil, false, func(k, v []byte) error {
		return enc.Encode([2][]byte{k, v})
	})
}

func (p *kvDB) RestoreFromFile(backupFile string) error {
	f, err := os.Open(backupFile)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	for {
		var kv [2][]byte
		if err := dec.Decode(&kv); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if err := p.db.Put(kv[0], kv[1], nil); err != nil {
			return err
		}
	}
	return nil
}

type kvWriteBatch struct {
	db     *leveldb.DB
	batch  *leveldb.Batch
	closed bool
}

func (p *kvWriteBatch) Put(key, value []byte) error {
	if p.closed {
		return errors.New("writebatch closed")
	}
	p.batch.Put(key, value)
	return nil
}

func (p *kvWriteBatch) Delete(key []byte) error {
	if p.closed {
		return errors.New("writebatch closed")
	}
	p.batch.Delete(key)
	return nil
}

func (p *kvWriteBatch) Flush() error {
	if p.closed {
		return errors.New("writebatch closed")
	}
	return p.db.Write(p.batch, nil)
}

func (p *kvWriteBatch) Close() {
	p.closed = true
	p.batch = nil
}

func (p *kvDB) NewWriteBatch() WriteBatch {
	return &kvWriteBatch{db: p.db, batch: &leveldb.Batch{}}
}
