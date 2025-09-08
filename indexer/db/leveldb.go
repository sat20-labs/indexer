package db

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/sat20-labs/indexer/common"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type levelDB struct {
	path string
	db   *leveldb.DB
}

func openLevelDB(filepath string, o *opt.Options) (*leveldb.DB, error) {
	if o == nil {
		o = &opt.Options{
			// ------------------------
			// 压缩 & SSTable 参数
			// ------------------------
			Compression: opt.SnappyCompression, // 压缩比 + 读取速度权衡
			BlockSize:   32 * 1024,             // 每个 block 32KB，增大随机读吞吐量
			WriteBuffer: 256 * opt.MiB,          // MemTable 大小，单线程写入可以更大
			CompactionTableSize: 64 * opt.MiB,  // 单个 SST 大小，减少文件数，提升 scan
			Filter: filter.NewBloomFilter(16),    // Bloom Filter（默认 10bit/key）
	
			// ------------------------
			// 并发与缓存
			// ------------------------
			BlockCacheCapacity: 4096 * opt.MiB,  // 数据块缓存，提升随机读
			OpenFilesCacheCapacity: 10000,        // 文件句柄缓存，避免频繁 open/close
	
			// ------------------------
			// 删除/过期数据
			// ------------------------
			DisableSeeksCompaction: true, // 大量删除时减少 seek-trigger compaction
		}
	}
	db, err := leveldb.OpenFile(filepath, o)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func NewLevelDB(path string) common.KVDB {
	db, err := initLevelDB(path)
	if err != nil {
		return nil
	}
	levelDB := levelDB{path: path, db: db}
	return &levelDB
}

func initLevelDB(path string) (*leveldb.DB, error) {
	if path == "" {
		path = "./data/db"
	}
	opts := &opt.Options{}
	return openLevelDB(path, opts)
}

func (p *levelDB) get(key []byte) ([]byte, error) {
	val, err := p.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, common.ErrKeyNotFound
		}
		return nil, err
	}
	return append([]byte{}, val...), nil
}

func (p *levelDB) put(key, value []byte) error {
	return p.db.Put(key, value, nil)
}

func (p *levelDB) remove(key []byte) error {
	return p.db.Delete(key, nil)
}

func (p *levelDB) close() error {
	return p.db.Close()
}

func (p *levelDB) commit() error { return nil }

func (p *levelDB) Read(key []byte) ([]byte, error) {
	return p.get(key)
}

func (p *levelDB) Write(key, value []byte) error {
	if err := p.put(key, value); err != nil {
		return err
	}
	return p.commit()
}

func (p *levelDB) Delete(key []byte) error {
	if err := p.remove(key); err != nil {
		return err
	}
	return p.commit()
}

func (p *levelDB) DropPrefix(prefix []byte) error {
	deletingKeyMap := make(map[string]bool)
	err := p.BatchRead(prefix, false, func(k, v []byte) error {
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

func (p *levelDB) DropAll() error {
	deletingKeyMap := make(map[string]bool)
	err := p.BatchRead(nil, false, func(k, v []byte) error {
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

func (p *levelDB) Close() error {
	return p.close()
}

func (p *levelDB) iterForwardWithPrefix(prefix []byte, start []byte, r func(k, v []byte) error) error {
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

func (p *levelDB) iterReverseWithPrefix(prefix []byte, start []byte, r func(k, v []byte) error) error {
    it := p.db.NewIterator(util.BytesPrefix(prefix), nil)
    defer it.Release()

    if len(start) > 0 {
        it.Seek(start)
        if !it.Valid() {
            it.Last()
        }
    } else {
        it.Last()
    }

    for ; it.Valid(); it.Prev() {
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


func (p *levelDB) BatchRead(prefix []byte, reverse bool, r func(k, v []byte) error) error {
	if reverse {
		return p.iterReverseWithPrefix(prefix, nil, r)
	}
	
	return p.iterForwardWithPrefix(prefix, nil, r)
}

func (p *levelDB) BatchReadV2(prefix, seekKey []byte, reverse bool, r func(k, v []byte) error) error {
	start := seekKey
	if len(start) == 0 && len(prefix) > 0 {
		start = prefix
	}
	if reverse {
		return p.iterReverseWithPrefix(prefix, start, r)
	}
	return p.iterForwardWithPrefix(prefix, start, r)
}


type levelDBReadBatch struct {
	db     *leveldb.DB
	snap *leveldb.Snapshot
	it iterator.Iterator
}

func (p *levelDBReadBatch) Get(key []byte) ([]byte, error) {
	// r, err := p.snap.Get(key, nil)
	// if err != nil {
	// 	if err == leveldb.ErrNotFound {
	// 		return nil, common.ErrKeyNotFound
	// 	}
	// 	return nil, err
	// }
	// return r, nil
	if p.it.Seek(key) && bytes.Equal(p.it.Key(), key) {
		return append([]byte{}, p.it.Value()...), nil
	} 
	return nil, common.ErrKeyNotFound
}


func (p *levelDBReadBatch) GetRef(key []byte) ([]byte, error) {
	// r, err := p.snap.Get(key, nil)
	// if err != nil {
	// 	if err == leveldb.ErrNotFound {
	// 		return nil, common.ErrKeyNotFound
	// 	}
	// 	return nil, err
	// }
	// return r, nil
	if p.it.Seek(key) && bytes.Equal(p.it.Key(), key) {
		return p.it.Value(), nil
	} 
	return nil, common.ErrKeyNotFound
}

// View 在一致性快照中执行只读操作
func (p *levelDB) View(fn func(txn common.ReadBatch) error) error {
	snap, err := p.db.GetSnapshot()
	if err != nil {
		return err
	}
	defer snap.Release()

	it := snap.NewIterator(nil, nil)
    defer it.Release()

	rb := levelDBReadBatch{
		db: p.db,
		snap: snap,
		it: it,
	}

	return fn(&rb)
}

// Update 批量写操作（原子性提交）
// 用法类似 badger 的 db.Update
func (p *levelDB) Update(fn func(any) error) error {
	batch := new(leveldb.Batch)

	// 让用户在闭包里构造批量写
	if err := fn(batch); err != nil {
		return err
	}

	// 原子性提交 batch
	return p.db.Write(batch, nil)
}

func (p *levelDB) BackupToFile(fname string) error {
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	total := 0
	err = p.BatchRead(nil, false, func(k, v []byte) error {
		total++
		return enc.Encode([2][]byte{k, v})
	})

	if err != nil {
		common.Log.Errorf("BackupToFile %s failed, %v", fname, err)
		return err
	}

	common.Log.Infof("BackupToFile %s succeed, total %d", fname, total)

	return err
}

func (p *levelDB) RestoreFromFile(backupFile string) error {
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


func (p *levelDB) CompareWithBackupFile(backupFile string) error {
	f, err := os.Open(backupFile)
	if err != nil {
		return err
	}
	defer f.Close()

	itemsInFile := make(map[string][]byte)
	dec := gob.NewDecoder(f)
	for {
		var kv [2][]byte
		if err := dec.Decode(&kv); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		itemsInFile[string(kv[0])] = append([]byte{}, kv[1]...)
	}

	itemsInDB := make(map[string][]byte)
	p.BatchRead(nil, false, func(k, v []byte) error {
		itemsInDB[string(k)] = append([]byte{}, v...)
		return nil
	})

	if len(itemsInFile) != len(itemsInDB) {
		common.Log.Errorf("count different %d %d", len(itemsInFile), len(itemsInDB))
		return fmt.Errorf("count different %d %d", len(itemsInFile), len(itemsInDB))
	}

	succ := true
	for k, v := range itemsInFile {
		v2, ok := itemsInDB[k]
		if !ok {
			common.Log.Errorf("can't find key %s in db", k)
		} else if !bytes.Equal(v, v2) {
			common.Log.Errorf("key %s value different", k)
			succ = false
		}
	}

	for k, v := range itemsInDB {
		v2, ok := itemsInFile[k]
		if !ok {
			common.Log.Errorf("can't find key %s in file", k)
		} else if !bytes.Equal(v, v2) {
			common.Log.Errorf("key %s value different", k)
			succ = false
		}
	}

	if succ {
		common.Log.Infof("db file check succeed")
	} else {
		common.Log.Infof("db file check failed")
	}
	

	return nil
}

type levelDBWriteBatch struct {
	db     *leveldb.DB
	batch  *leveldb.Batch
	closed bool
}

func (p *levelDBWriteBatch) Put(key, value []byte) error {
	if p.closed {
		return errors.New("writebatch closed")
	}
	p.batch.Put(key, value)
	return nil
}

func (p *levelDBWriteBatch) Delete(key []byte) error {
	if p.closed {
		return errors.New("writebatch closed")
	}
	p.batch.Delete(key)
	return nil
}

func (p *levelDBWriteBatch) Flush() error {
	if p.closed {
		return errors.New("writebatch closed")
	}
	return p.db.Write(p.batch, nil)
}

func (p *levelDBWriteBatch) Close() {
	p.closed = true
	p.batch = nil
}

func (p *levelDB) NewWriteBatch() common.WriteBatch {
	return &levelDBWriteBatch{db: p.db, batch: &leveldb.Batch{}}
}
