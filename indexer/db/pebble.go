package db

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"os"

	"github.com/cockroachdb/pebble"
	"github.com/sat20-labs/indexer/common"
)

type pebbleDB struct {
	path string
	db   *pebble.DB
}

func openDB(filepath string, o *pebble.Options) (*pebble.DB, error) {
	if o == nil {
		// 建议：Cache 设为机器内存的 20%~40%
		cache := pebble.NewCache(4 << 30) // 4 GiB，可按需调整
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
			LBaseMaxBytes:          512 << 20, // L1 基准容量，放大后减少 L0→L1 频繁抖动

			// 并行压缩（非常关键，避免 compaction 成为瓶颈）
			MaxConcurrentCompactions: func() int { return 4 },

			// 表级别选项
			// Levels: []pebble.LevelOptions{
			// 	// L0 使用全局默认
			// 	{}, // L0
			// 	// 从 L1 起启用 BloomFilter & 压缩（按需开 Zstd）
			// 	// {BloomFilter: pebble.BloomFilter(10)}, // L1
			// 	// {BloomFilter: pebble.BloomFilter(10)}, // L2
			// 	// {BloomFilter: pebble.BloomFilter(10)}, // L3
			// 	// {BloomFilter: pebble.BloomFilter(10)}, // L4
			// 	// {BloomFilter: pebble.BloomFilter(10)}, // L5
			// 	// {BloomFilter: pebble.BloomFilter(10)}, // L6
			// },

			// WAL 同步策略：强一致用 Sync；追求吞吐可结合时间门限
			// 注：WALMinSyncInterval 在新版本里可用（按你的 Pebble 版本）
			// WALMinSyncInterval: 5 * time.Millisecond,
		}
		// 压缩算法（可选）：Zstd 压缩比/速度更佳（取决于 Pebble/Go 版本支持）
		// for i := range opts.Levels {
		// 	opts.Levels[i].Compression = pebble.ZstdCompression
		// }
	}
	return pebble.Open(filepath, o)
}

func NewKVDB(path string) KVDB {
	db, err := initDB(path)
	if err != nil {
		return nil
	}
	kvdb := pebbleDB{path: path, db: db}
	return &kvdb
}

func initDB(path string) (*pebble.DB, error) {
	if path == "" {
		path = "./data/db"
	}
	return openDB(path, nil)
}

func (p *pebbleDB) get(key []byte) ([]byte, error) {
	val, closer, err := p.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrKeyNotFound
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

type kvReadBatch struct {
	snap *pebble.Snapshot
}

func (p *kvReadBatch) Get(key []byte) ([]byte, error) {
	val, closer, err := p.snap.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrKeyNotFound
		}
		return nil, err
	}
	defer closer.Close()
	return append([]byte{}, val...), nil
}

func (p *pebbleDB) View(fn func(txn ReadBatch) error) error {
	snap := p.db.NewSnapshot()
	defer snap.Close()
	rb := kvReadBatch{snap: snap}
	return fn(&rb)
}

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

type kvWriteBatch struct {
	db     *pebble.DB
	batch  *pebble.Batch
	closed bool
}

func (p *kvWriteBatch) Put(key, value []byte) error {
	if p.closed {
		return errors.New("writebatch closed")
	}
	return p.batch.Set(key, value, nil)
}

func (p *kvWriteBatch) Delete(key []byte) error {
	if p.closed {
		return errors.New("writebatch closed")
	}
	return p.batch.Delete(key, nil)
}

func (p *kvWriteBatch) Flush() error {
	if p.closed {
		return errors.New("writebatch closed")
	}
	return p.batch.Commit(pebble.Sync)
}

func (p *kvWriteBatch) Close() {
	p.closed = true
	_ = p.batch.Close()
}

func (p *pebbleDB) NewWriteBatch() WriteBatch {
	return &kvWriteBatch{db: p.db, batch: p.db.NewBatch()}
}
