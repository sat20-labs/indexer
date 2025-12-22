package db

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"os"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/bloom"
	"github.com/sat20-labs/indexer/common"
)

const (
	maxBatchSize = 1280 << 20 // 1280MB，安全
	maxItemSize  = 64 << 20  // 单条数据兜底
)


type pebbleDB struct {
	path string
	db   *pebble.DB
}

/*
用 小 SST + 单线程 compaction 编译，
再用 大 cache + 大 block 服务，
这是 工业级索引器（包括 CockroachDB 内部）常用的策略。
*/

// 数据编译期的参数
// 编译期性能提升的正确路径是：
// 更多 cache（不是更大 SST）
// 更大的 memtable
// 更精准的 bloom
// 8KB block size
func buildOptions() *pebble.Options {
    cache := pebble.NewCache(32 << 30) // 32GB：显著提升随机读

    return &pebble.Options{
        Cache:        cache,
        MaxOpenFiles: 50000,

        // —— MemTable：极其关键 ——
        MemTableSize:                64 << 20, // 64MB（提高热 key 命中）
        MemTableStopWritesThreshold: 3,        // 最多 ~192MB memtable

        // —— L0 控制 ——
        L0CompactionThreshold:  6,
        L0StopWritesThreshold:  12,

        // —— 防止层级过大 ——
        LBaseMaxBytes: 1 << 30, // 1GB

        // —— compaction 仍然单线程，防 OOM ——
        MaxConcurrentCompactions: func() int { return 1 },

        Levels: func() []pebble.LevelOptions {
            lvls := make([]pebble.LevelOptions, 7)
            for i := range lvls {
                lvls[i] = pebble.LevelOptions{
                    TargetFileSize: 128 << 20, // 128MB：安全上限
                    BlockSize:      8 << 10,   // 8KB：point lookup 最优区间
                    FilterPolicy:   bloom.FilterPolicy(12), // ↑ Bloom 精度
                    FilterType:     pebble.TableFilter,
                }
            }
            return lvls
        }(),
    }
}


// 索引完成后，重启进程，用下面的参数打开同一个 DB：
// 索引完成后，重启进程，用下面的参数打开同一个 DB：
func serveOptions() *pebble.Options {
    cache := pebble.NewCache(32 << 30) // 32GB

    return &pebble.Options{
        Cache:        cache,
        MaxOpenFiles: 50000,

        MemTableSize:                64 << 20,
        MemTableStopWritesThreshold: 4,

        // compaction 几乎不会发生
        MaxConcurrentCompactions: func() int { return 2 },

        Levels: func() []pebble.LevelOptions {
            lvls := make([]pebble.LevelOptions, 7)
            for i := range lvls {
                lvls[i].TargetFileSize = 256 << 20 // 256MB
                lvls[i].BlockSize = 16 << 10       // 提高点查效率
                lvls[i].FilterPolicy = bloom.FilterPolicy(10)
                lvls[i].FilterType = pebble.TableFilter
            }
            return lvls
        }(),
    }
}


func openPebbleDB(filepath string, o *pebble.Options) (*pebble.DB, error) {
	if o == nil {
		o = buildOptions()
		
		// // 建议：Cache 设为机器内存的 20%~40%
		// cache := pebble.NewCache(16 << 30) // 16 GiB，可按需调整
		// // 可选：TableCache 默认够用；需要更高并发可单独配置


		// o = &pebble.Options{
		// 	Cache:         cache,
		// 	MaxOpenFiles:  50000,      // 多SST场景减少频繁打开
		// 	MemTableSize:  64 << 20,  // 64MB：增大写缓冲，减少 flush 频率
		// 	// 当 memtable 压力大时阻断写入，防止 L0 过度堆积
		// 	MemTableStopWritesThreshold: 4,

		// 	// L0 门限：控制写入背压。NVMe 下可更激进些（更大阈值）
		// 	L0CompactionThreshold:  8,
		// 	L0StopWritesThreshold:  24, // 达到后暂停写入，给 compaction 让路
		// 	LBaseMaxBytes:          2 << 30, // L1 基准容量，放大后减少 L0→L1 频繁抖动

		// 	// 并行压缩（非常关键，避免 compaction 成为瓶颈）
		// 	MaxConcurrentCompactions: func() int { return 1 },

		// 	Levels: func() []pebble.LevelOptions {
		// 		lvls := make([]pebble.LevelOptions, 7)
		// 		for i := range lvls {
		// 			lvls[i].TargetFileSize = 64 << 20  // 64 MiB；可逐层×2
		// 			lvls[i].BlockSize = 8 << 10       // 8 KiB（小 value， 适合点查）
		// 			lvls[i].FilterPolicy = bloom.FilterPolicy(10) // 10 bits/entry
		// 			lvls[i].FilterType = pebble.TableFilter

		// 			// if i > 0 {
		// 			// 	lvls[i].Compression = pebble.ZstdCompression // 压缩更大 TODO 后面测试后开启
		// 			// }
		// 		}
		// 		// 逐层放大 TargetFileSize（非必须，但对大数据集更友好）
		// 		for i := 1; i < len(lvls); i++ { 
		// 			lvls[i].TargetFileSize = lvls[i-1].TargetFileSize << 1 
		// 		}
		// 		return lvls
		// 	}(),

		// 	// WAL 同步策略：强一致用 Sync；追求吞吐可结合时间门限
		// 	// 注：WALMinSyncInterval 在新版本里可用（按你的 Pebble 版本）
		// 	// WALMinSyncInterval: func() time.Duration {return 5 * time.Millisecond},
		// }
		// //o.Levels[0].EnsureDefaults()

	
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

var PrintLog = false
func (p *pebbleWriteBatch) Put(key, value []byte) error {
	if p.closed {
		return errors.New("writebatch closed")
	}

	itemSize := len(key) + len(value)
	if PrintLog {
		common.Log.Infof("%s size %d", string(key), itemSize)
	}

	// 单条记录过大，单独提交
	if itemSize >= maxItemSize {
		b := p.db.NewBatch()
		defer b.Close()
		if err := b.Set(key, value, nil); err != nil {
			return err
		}
		return b.Commit(pebble.Sync)
	}

	if err := p.ensureCapacity(itemSize); err != nil {
		return err
	}
	return p.batch.Set(key, value, nil)
}

func (p *pebbleWriteBatch) Delete(key []byte) error {
	if p.closed {
		return errors.New("writebatch closed")
	}

	if err := p.ensureCapacity(len(key)); err != nil {
		return err
	}
	return p.batch.Delete(key, nil)
}

func (p *pebbleWriteBatch) Flush() error {
	if p.closed {
		return errors.New("writebatch closed")
	}
	err := p.batch.Commit(pebble.Sync)
	return err
}

func (p *pebbleWriteBatch) Close() {
	if p.closed {
		return
	}
	p.closed = true
	_ = p.batch.Close()
}

func (p *pebbleWriteBatch) ensureCapacity(extra int) error {
	if p.closed {
		return errors.New("writebatch closed")
	}

	// 如果再写入 extra 后会超过阈值，先提交
	batchSize := p.batch.Len()
	if batchSize+extra >= maxBatchSize {
		common.Log.Infof("ensureCapacity commit data...")
		if err := p.batch.Commit(pebble.Sync); err != nil {
			return err
		}
		p.batch.Close()
		p.batch = p.db.NewBatch()
	}
	return nil
}


func (p *pebbleDB) NewWriteBatch() common.WriteBatch {
	return &pebbleWriteBatch{db: p.db, batch: p.db.NewBatch()}
}
