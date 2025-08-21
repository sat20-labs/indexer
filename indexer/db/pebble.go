package db

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"os"
	"sort"

	"github.com/cockroachdb/pebble"
	"github.com/sat20-labs/indexer/common"
)

type pebbleDB struct {
	path string
	db   *pebble.DB
}

func openPebbleDB(filepath string, o *pebble.Options) (*pebble.DB, error) {
	if o == nil {
		// 建议：Cache 设为机器内存的 20%~40%
		cache := pebble.NewCache(8 << 30) // 4 GiB，可按需调整
		// 可选：TableCache 默认够用；需要更高并发可单独配置


		o = &pebble.Options{
			Cache:         cache,
			MaxOpenFiles:  10000,      // 多SST场景减少频繁打开
			MemTableSize:  256 << 20,  // 256MB：增大写缓冲，减少 flush 频率
			// 当 memtable 压力大时阻断写入，防止 L0 过度堆积
			MemTableStopWritesThreshold: 4,

			// L0 门限：控制写入背压。NVMe 下可更激进些（更大阈值）
			L0CompactionThreshold:  8,
			L0StopWritesThreshold:  24, // 达到后暂停写入，给 compaction 让路
			LBaseMaxBytes:          2 << 30, // L1 基准容量，放大后减少 L0→L1 频繁抖动

			// 并行压缩（非常关键，避免 compaction 成为瓶颈）
			MaxConcurrentCompactions: func() int { return 4 },

			Levels: func() []pebble.LevelOptions {
				lvls := make([]pebble.LevelOptions, 7)
				for i := range lvls {
					lvls[i].TargetFileSize = 64 << 20  // 64 MiB；可逐层×2
					lvls[i].BlockSize = 16 << 10       // 32 KiB（小 value 可降至 16 KiB 试验）
					// 其余默认即可；表级 Bloom 由 Pebble 管
				}
				// 逐层放大 TargetFileSize（非必须，但对大数据集更友好）
				for i := 1; i < len(lvls); i++ { 
					lvls[i].TargetFileSize = lvls[i-1].TargetFileSize << 1 
				}
				return lvls
			}(),

			// WAL 同步策略：强一致用 Sync；追求吞吐可结合时间门限
			// 注：WALMinSyncInterval 在新版本里可用（按你的 Pebble 版本）
			// WALMinSyncInterval: func() time.Duration {return 5 * time.Millisecond},
		}
		//o.Levels[0].EnsureDefaults()

	
		// o.TableCache = pebble.NewTableCache()
		// 压缩算法（可选）：Zstd 压缩比/速度更佳（取决于 Pebble/Go 版本支持）
		// for i := range opts.Levels {
		// 	opts.Levels[i].Compression = pebble.ZstdCompression
		// }
	}
	return pebble.Open(filepath, o)
}

func NewPebbleDB(path string) common.KVDB {
	db, err := initPebbleDB(path)
	if err != nil {
		common.Log.Errorf("initPebbleDB failed, %v", err)
		return nil
	}
	kvdb := pebbleDB{path: path, db: db}
	return &kvdb
}

func initPebbleDB(path string) (*pebble.DB, error) {
	if path == "" {
		path = "./data/db"
	}
	return openPebbleDB(path, nil)
}

func (p *pebbleDB) get(key []byte) ([]byte, error) {
	val, closer, err := p.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, common.ErrKeyNotFound
		}
		return nil, err
	}
	defer closer.Close()
	return append([]byte{}, val...), nil
}

func (p *pebbleDB) put(key, value []byte) error {
	return p.db.Set(key, value, pebble.Sync)
}

func (p *pebbleDB) remove(key []byte) error {
	return p.db.Delete(key, pebble.Sync)
}

func (p *pebbleDB) close() error {
	return p.db.Close()
}

func (p *pebbleDB) commit() error { 
	return nil 
}

func (p *pebbleDB) Read(key []byte) ([]byte, error) {
	return p.get(key)
}

func (p *pebbleDB) Write(key, value []byte) error {
	if err := p.put(key, value); err != nil {
		return err
	}
	return p.commit()
}

func (p *pebbleDB) Delete(key []byte) error {
	if err := p.remove(key); err != nil {
		return err
	}
	return p.commit()
}

func (p *pebbleDB) DropPrefix(prefix []byte) error {
	wb := p.NewWriteBatch()
	defer wb.Close()

	err := p.BatchRead(prefix, false, func(k, v []byte) error {
		wb.Delete(k)
		return nil
	})
	if err != nil {
		return err
	}
	return wb.Flush()
}

func (p *pebbleDB) DropAll() error {
	wb := p.NewWriteBatch()
	defer wb.Close()

	err := p.BatchRead(nil, false, func(k, v []byte) error {
		wb.Delete(k)
		return nil
	})
	if err != nil {
		return err
	}
	return wb.Flush()
}

func (p *pebbleDB) Close() error {
	return p.close()
}

// nextPrefix 返回“字典序上紧邻 prefix 的下界”，可作为 UpperBound（开区间）。
// 若 prefix 全为 0xFF，返回 nil（表示无上界）；此时要多一道 HasPrefix 检查。
func nextPrefix(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	out := append([]byte{}, prefix...)
	for i := len(out) - 1; i >= 0; i-- {
		if out[i] != 0xFF {
			out[i]++
			return out[:i+1]
		}
	}
	// 全 0xFF，没有更大前缀；返回 nil 表示不设上界
	return nil
}

// 统一的迭代器入口：支持前缀、起始键、正/反向
func (p *pebbleDB) iter(prefix, start []byte, reverse bool, r func(k, v []byte) error) error {
	var lower, upper []byte
	if len(prefix) > 0 {
		lower = prefix
		upper = nextPrefix(prefix) // 作为开区间上界
	}

	it, err := p.db.NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	if err != nil {
		return err
	}
	defer it.Close()

	if reverse {
		// 反向起点选择：优先 <start 的最大键；否则落到边界内的最后一个键
		var ok bool
		if len(start) > 0 {
			// clamp 到边界
			if len(lower) > 0 && bytes.Compare(start, lower) < 0 {
				start = lower
			}
			if upper != nil && bytes.Compare(start, upper) >= 0 {
				start = upper
			}
			ok = it.SeekLT(start)
			if !ok {
				ok = it.Last()
			}
		} else {
			ok = it.Last()
		}

		for ; ok; ok = it.Prev() {
			k := it.Key()
			// 当 upper==nil（prefix 全 0xFF）时，需要手动判断 HasPrefix
			if len(prefix) > 0 && upper == nil && !bytes.HasPrefix(k, prefix) {
				break
			}
			if err := r(append([]byte{}, k...), append([]byte{}, it.Value()...)); err != nil {
				return err
			}
		}
		return it.Error()
	}

	// 正向
	var ok bool
	if len(start) > 0 {
		ok = it.SeekGE(start)
	} else if len(prefix) > 0 {
		ok = it.SeekGE(prefix)
	} else {
		ok = it.First()
	}
	for ; ok; ok = it.Next() {
		k := it.Key()
		if len(prefix) > 0 && upper == nil && !bytes.HasPrefix(k, prefix) {
			break
		}
		if err := r(append([]byte{}, k...), append([]byte{}, it.Value()...)); err != nil {
			return err
		}
	}
	return it.Error()
}

func (p *pebbleDB) BatchRead(prefix []byte, reverse bool, r func(k, v []byte) error) error {
	return p.iter(prefix, nil, reverse, r)
}

func (p *pebbleDB) BatchReadV2(prefix, seekKey []byte, reverse bool, r func(k, v []byte) error) error {
	// seekKey 可在 prefix 内/外；iter() 会自动 clamp 到边界范围
	return p.iter(prefix, seekKey, reverse, r)
}

type pebbleReadBatch struct {
	snap *pebble.Snapshot
	it *pebble.Iterator
}

func (p *pebbleReadBatch) Get(key []byte) ([]byte, error) {
	// val, closer, err := p.snap.Get(key)
	// if err != nil {
	// 	if errors.Is(err, pebble.ErrNotFound) {
	// 		return nil, common.ErrKeyNotFound
	// 	}
	// 	return nil, err
	// }
	// defer closer.Close()
	// return append([]byte{}, val...), nil
	if p.it.SeekGE(key) && bytes.Equal(p.it.Key(), key) {
		return append([]byte{}, p.it.Value()...), nil
	} 
	return nil, common.ErrKeyNotFound
}


func (p *pebbleReadBatch) GetRef(key []byte) ([]byte, error) {
	// val, closer, err := p.snap.Get(key)
	// if err != nil {
	// 	if errors.Is(err, pebble.ErrNotFound) {
	// 		return nil, common.ErrKeyNotFound
	// 	}
	// 	return nil, err
	// }
	// defer closer.Close()
	// return val, nil
	if p.it.SeekGE(key) && bytes.Equal(p.it.Key(), key) {
		return p.it.Value(), nil
	} 
	return nil, common.ErrKeyNotFound
}

func (p *pebbleReadBatch) MultiGet(keys [][]byte) ([][]byte, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	results := make([][]byte, len(keys))
	for i, k := range keys {
		if ok := p.it.SeekGE(k); ok && bytes.Equal(p.it.Key(), k) {
			// 必须拷贝，避免底层 buffer 复用
			val := append([]byte{}, p.it.Value()...)
			results[i] = val
		} else {
			results[i] = nil
		}
	}
	return results, nil
}

func (p *pebbleReadBatch) MultiGetSorted(keys [][]byte) (map[string][]byte, error) {
	
	sortedKeys := make([][]byte, len(keys))
	copy(sortedKeys, keys)
	sort.Slice(sortedKeys, func(i, j int) bool {
		return bytes.Compare(sortedKeys[i], sortedKeys[j]) < 0
	})

	result := make(map[string][]byte, len(keys))
	i := 0
	for p.it.Next() && i < len(sortedKeys) {
		key := sortedKeys[i]
		for p.it.Valid() && bytes.Compare(p.it.Key(), key) >= 0 {
			if bytes.Equal(p.it.Key(), key) {
				// 命中 key
				valCopy := append([]byte(nil), p.it.Value()...) 
				result[string(key)] = valCopy
				i++
				if i >= len(sortedKeys) {
					break
				}
				key = sortedKeys[i]
			} else if bytes.Compare(p.it.Key(), key) > 0 {
				// 数据库 key 比当前目标 key 大，说明该 key 不存在
				i++
				if i < len(sortedKeys) {
					key = sortedKeys[i]
				}
			}
		}
	}

	if err := p.it.Error(); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *pebbleDB) View(fn func(txn common.ReadBatch) error) error {
	snap := p.db.NewSnapshot()
	defer snap.Close()

	it, err := snap.NewIter(nil)
	if err != nil {
		return err
	}
	defer it.Close()

	rb := pebbleReadBatch{
		snap: snap,
		it: it,
	}
	return fn(&rb)
	// rb := &pebbleReadBatchDirect{db: p.db}
    // return fn(rb)
}

// type pebbleReadBatchDirect struct{ db *pebble.DB }
// func (r *pebbleReadBatchDirect) Get(key []byte) ([]byte, error) {
//     val, closer, err := r.db.Get(key)
// 	if errors.Is(err, pebble.ErrNotFound) {
// 		return nil, common.ErrKeyNotFound
// 	}
//     defer closer.Close()
//     buf := make([]byte, len(val))
// 	copy(buf, val)
//     return buf, nil
// }

func (p *pebbleDB) Update(fn func(any) error) error {
	batch := p.db.NewBatch()
	if err := fn(batch); err != nil {
		return err
	}
	return batch.Commit(pebble.Sync)
}

func (p *pebbleDB) BackupToFile(fname string) error {
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
	return nil
}

func (p *pebbleDB) RestoreFromFile(backupFile string) error {
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
		if err := p.db.Set(kv[0], kv[1], pebble.Sync); err != nil {
			return err
		}
	}
	return nil
}

type pebbleWriteBatch struct {
	db     *pebble.DB
	batch  *pebble.Batch
	closed bool
}

func (p *pebbleWriteBatch) Put(key, value []byte) error {
	if p.closed {
		return errors.New("writebatch closed")
	}
	return p.batch.Set(key, value, nil)
}

func (p *pebbleWriteBatch) Delete(key []byte) error {
	if p.closed {
		return errors.New("writebatch closed")
	}
	return p.batch.Delete(key, nil)
}

func (p *pebbleWriteBatch) Flush() error {
	if p.closed {
		return errors.New("writebatch closed")
	}
	return p.batch.Commit(pebble.Sync)
}

func (p *pebbleWriteBatch) Close() {
	p.closed = true
	_ = p.batch.Close()
}

func (p *pebbleDB) NewWriteBatch() common.WriteBatch {
	return &pebbleWriteBatch{db: p.db, batch: p.db.NewBatch()}
}
