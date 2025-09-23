package db

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"os"
	"sync"

	"github.com/bmatsuo/lmdb-go/lmdb"
	"github.com/sat20-labs/indexer/common"
)

var (
	_ common.KVDB       = (*lmdbDB)(nil)
	_ common.WriteBatch = (*lmdbWriteBatch)(nil)
	_ common.ReadBatch  = (*lmdbReadBatch)(nil)
)

// =============== 工具 ===============

func nextLmdbPrefix(prefix []byte) []byte {
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
	return nil
}

// =============== DB 对象与构造 ===============


// 单机嵌入：创建/打开 LMDB 环境与默认库
func NewLMDB(path string) *lmdbDB {
	env, err := lmdb.NewEnv()
	if err != nil {
		common.Log.Panicf("lmdb.NewEnv: %v", err)
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		common.Log.Panicf("mkdir %s: %v", path, err)
	}
	// 足够大的映射，避免运行期扩容失败
	mapsize := int64(512 << 30) // 512 GiB，按需调大/调小
	if err := env.SetMapSize(mapsize); err != nil {
		common.Log.Panicf("env.SetMapSize: %v", err)
	}
	if err := env.SetMaxDBs(16); err != nil {
		common.Log.Panicf("env.SetMaxDBs: %v", err)
	}
	if err := env.SetMaxReaders(1024); err != nil {
		common.Log.Panicf("env.SetMaxReaders: %v", err)
	}

	if err := env.Open(path, lmdb.MapAsync | lmdb.WriteMap, 0o644); err != nil {
		common.Log.Panicf("env.Open: %v", err)
	}

	var dbi lmdb.DBI
	if err := env.Update(func(txn *lmdb.Txn) error {
		var e error
		dbi, e = txn.OpenDBI("main", lmdb.Create)
		return e
	}); err != nil {
		common.Log.Panicf("open main dbi: %v", err)
	}

	return &lmdbDB{env: env, dbi: dbi}
}

func (l *lmdbDB) Close() error {
	// env.Close 会等待未结束的 txn；此处无长活 txn
	return l.env.Close()
}

// =============== 单条操作（每次一个事务） ===============

func (l *lmdbDB) Read(key []byte) ([]byte, error) {
	var val []byte
	err := l.env.View(func(txn *lmdb.Txn) error {
		v, err := txn.Get(l.dbi, key)
		if lmdb.IsNotFound(err) {
			return common.ErrKeyNotFound
		}
		if err != nil {
			return err
		}
		val = append(val[:0], v...) // copy out
		return nil
	})
	return val, err
}

func (l *lmdbDB) Write(key, value []byte) error {
	return l.env.Update(func(txn *lmdb.Txn) error {
		// 直接 Put；小写入这就够快了
		return txn.Put(l.dbi, key, value, 0)
	})
}

func (l *lmdbDB) Delete(key []byte) error {
	return l.env.Update(func(txn *lmdb.Txn) error {
		err := txn.Del(l.dbi, key, nil)
		if lmdb.IsNotFound(err) {
			// 与 Pebble 对齐：删除不存在不报错
			return nil
		}
		return err
	})
}


// =============== 批量只读 ReadBatch ===============

type lmdbReadBatch struct {
	txn *lmdb.Txn
	dbi lmdb.DBI
}

func (l *lmdbDB) View(fn func(common.ReadBatch) error) error {
	return l.env.View(func(txn *lmdb.Txn) error {
		return fn(&lmdbReadBatch{txn: txn, dbi: l.dbi})
	})
}

func (rb *lmdbReadBatch) Get(key []byte) ([]byte, error) {
	v, err := rb.txn.Get(rb.dbi, key)
	if lmdb.IsNotFound(err) {
		return nil, common.ErrKeyNotFound
	}
	if err != nil {
		return nil, err
	}
	return append([]byte{}, v...), nil
}

func (rb *lmdbReadBatch) GetRef(key []byte) ([]byte, error) {
	v, err := rb.txn.Get(rb.dbi, key)
	if lmdb.IsNotFound(err) {
		return nil, common.ErrKeyNotFound
	}
	return v, err
}

// =============== 遍历读（含前缀/起始/反向） ===============

func (l *lmdbDB) iter(prefix, start []byte, reverse bool, r func(k, v []byte) error) error {
	lower := prefix
	upper := nextLmdbPrefix(prefix)

	return l.env.View(func(txn *lmdb.Txn) error {
		cur, err := txn.OpenCursor(l.dbi)
		if err != nil {
			return err
		}
		defer cur.Close()

		step := lmdb.Next
		if reverse {
			step = lmdb.Prev
		}

		// 计算起点
		var k, v []byte
		if reverse {
			// 反向：定位到上界之前的最大键
			if len(upper) > 0 {
				k, v, err = cur.Get(upper, nil, lmdb.SetRange)
				if lmdb.IsNotFound(err) {
					// 没有 >= upper 的键，则从最后一个开始
					k, v, err = cur.Get(nil, nil, lmdb.Last)
				} else if err == nil {
					// 回到 < upper 的上一个
					k, v, err = cur.Get(nil, nil, lmdb.Prev)
				}
			} else {
				// 无上界，直接最后一个
				k, v, err = cur.Get(nil, nil, lmdb.Last)
			}
			if err != nil && !lmdb.IsNotFound(err) {
				return err
			}
			// 若指定了 start，以 SeekRange(start) 再回退一个
			if len(start) > 0 {
				k, v, err = cur.Get(start, nil, lmdb.SetRange)
				if lmdb.IsNotFound(err) {
					k, v, err = cur.Get(nil, nil, lmdb.Last)
				} else if err == nil {
					k, v, err = cur.Get(nil, nil, lmdb.Prev)
				}
				if err != nil && !lmdb.IsNotFound(err) {
					return err
				}
			}
		} else {
			// 正向
			if len(start) > 0 {
				k, v, err = cur.Get(start, nil, lmdb.SetRange)
			} else if len(lower) > 0 {
				k, v, err = cur.Get(lower, nil, lmdb.SetRange)
			} else {
				k, v, err = cur.Get(nil, nil, lmdb.First)
			}
			if err != nil && !lmdb.IsNotFound(err) {
				return err
			}
		}

		advance := func() (kk, vv []byte, e error) { return cur.Get(nil, nil, uint(step)) }

		// 遍历
		for {
			if lmdb.IsNotFound(err) {
				break
			}
			// 边界检查
			if len(prefix) > 0 {
				if !bytes.HasPrefix(k, prefix) {
					// 正向超过上界，或反向低于下界，结束
					if !reverse && upper != nil && bytes.Compare(k, upper) >= 0 {
						break
					}
					if reverse && bytes.Compare(k, lower) < 0 {
						break
					}
					// 无 upper 情况（prefix 全 0xFF），也要按 HasPrefix 退出
					if upper == nil && !bytes.HasPrefix(k, prefix) {
						break
					}
				}
			}

			if err := r(append([]byte{}, k...), append([]byte{}, v...)); err != nil {
				return err
			}

			k, v, err = advance()
		}
		return nil
	})
}

func (l *lmdbDB) BatchRead(prefix []byte, reverse bool, r func(k, v []byte) error) error {
	return l.iter(prefix, nil, reverse, r)
}

func (l *lmdbDB) BatchReadV2(prefix, seekKey []byte, reverse bool, r func(k, v []byte) error) error {
	return l.iter(prefix, seekKey, reverse, r)
}

// =============== Drop 接口（高效版） ===============

func (l *lmdbDB) DropPrefix(prefix []byte) error {
	if len(prefix) == 0 {
		return l.DropAll()
	}
	lower := prefix
	upper := nextLmdbPrefix(prefix)
	return l.env.Update(func(txn *lmdb.Txn) error {
		cur, err := txn.OpenCursor(l.dbi)
		if err != nil {
			return err
		}
		defer cur.Close()

		k, _, err := cur.Get(lower, nil, lmdb.SetRange)
		for {
			if lmdb.IsNotFound(err) {
				break
			}
			if err != nil {
				return err
			}
			if upper != nil && bytes.Compare(k, upper) >= 0 {
				break
			}
			if !bytes.HasPrefix(k, prefix) { // 无 upper 的保护
				break
			}
			// 删除当前 key（无 dup）
			if e := cur.Del(0); e != nil && !lmdb.IsNotFound(e) {
				return e
			}
			// 删除后，游标在下一个条目之前位置，继续 Next
			k, _, err = cur.Get(nil, nil, lmdb.Next)
		}
		return nil
	})
}

func (l *lmdbDB) DropAll() error {
	// 用 lmdb.Drop 清空并保留 DBI（更快）
	return l.env.Update(func(txn *lmdb.Txn) error {
		return txn.Drop(l.dbi, false)
	})
}

// =============== 备份 / 恢复（与 Pebble 版保持一致行为） ===============

func (l *lmdbDB) BackupToFile(fname string) error {
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	total := 0
	err = l.BatchRead(nil, false, func(k, v []byte) error {
		total++
		return enc.Encode([2][]byte{k, v})
	})
	if err != nil {
		common.Log.Errorf("BackupToFile %s failed: %v", fname, err)
		return err
	}
	common.Log.Infof("BackupToFile %s succeed, total %d", fname, total)
	return nil
}

func (l *lmdbDB) RestoreFromFile(fname string) error {
	f, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	// 批量导入：一个写事务内执行，效率更高
	return l.env.Update(func(txn *lmdb.Txn) error {
		for {
			var kv [2][]byte
			if err := dec.Decode(&kv); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return err
			}
			// PutReserve 尝试
			if b, e := txn.PutReserve(l.dbi, kv[0], len(kv[1]), 0); e == nil {
				copy(b, kv[1])
			} else if e := txn.Put(l.dbi, kv[0], kv[1], 0); e != nil {
				return e
			}
		}
		return nil
	})
}


const (
	// 每个事务最多写多少条记录，避免事务太大
	maxTxnBatch = 100_000
	// LMDB 初始大小 (1TB)，用稀疏文件，不会占用实际磁盘
)

type lmdbDB struct {
	env       *lmdb.Env
	dbi       lmdb.DBI
	writeLock sync.Mutex
}

type writeOp struct {
	key   []byte
	value []byte
	del   bool
}

// --- KVDB 接口部分 (关键的 WriteBatch 入口) ---


type lmdbWriteBatch struct {
    env       *lmdb.Env
    dbi       lmdb.DBI
    ops       []writeOp
    parent    *lmdbDB
    flushed   bool
    unlockOnce sync.Once
}

// NewWriteBatch: 保持锁定，WriteBatch 负责释放锁（一次）
func (l *lmdbDB) NewWriteBatch() common.WriteBatch {
    l.writeLock.Lock()
    return &lmdbWriteBatch{
        env:    l.env,
        dbi:    l.dbi,
        parent: l,
        ops:    make([]writeOp, 0, 1024),
    }
}

// Put/Delete unchanged (keep copying to avoid external buffer reuse)
func (wb *lmdbWriteBatch) Put(key, value []byte) error {
    wb.ops = append(wb.ops, writeOp{key: append([]byte{}, key...), value: append([]byte{}, value...)})
    return nil
}
func (wb *lmdbWriteBatch) Delete(key []byte) error {
    wb.ops = append(wb.ops, writeOp{key: append([]byte{}, key...), del: true})
    return nil
}

// helper to release lock once
func (wb *lmdbWriteBatch) releaseLock() {
    wb.unlockOnce.Do(func() {
        wb.parent.writeLock.Unlock()
    })
}

// Flush: 分块提交，每块遇到 MapFull 自动扩容并重试；优先用 PutReserve 来减少拷贝
func (wb *lmdbWriteBatch) Flush() error {
    if wb.flushed {
        wb.releaseLock() // 如果已经 flushed，也要确保释放锁（一次）
        return nil
    }

    total := len(wb.ops)
    for i := 0; i < total; i += maxTxnBatch {
        end := i + maxTxnBatch
        if end > total {
            end = total
        }
        chunk := wb.ops[i:end]

        // retry loop for this chunk (handle MapFull)
        for {
            err := wb.env.Update(func(txn *lmdb.Txn) error {
                for _, op := range chunk {
                    if op.del {
                        if err := txn.Del(wb.dbi, op.key, nil); err != nil {
                            if lmdb.IsNotFound(err) {
                                continue
                            }
                            return err
                        }
                    } else {
                        // 优先尝试 PutReserve 减少额外拷贝
                        if b, e := txn.PutReserve(wb.dbi, op.key, len(op.value), 0); e == nil {
                            copy(b, op.value)
                        } else {
                            if e := txn.Put(wb.dbi, op.key, op.value, 0); e != nil {
                                return e
                            }
                        }
                    }
                }
                return nil
            })

            if err == nil {
                break // chunk done
            }

            // 如果是 MapFull，扩容并重试
            if err == lmdb.MapFull {
                // 获取当前 map size 并扩容（乘2）
                info, infoErr := wb.env.Info()
                if infoErr != nil {
                    wb.releaseLock()
                    return infoErr
                }
                newSize := int64(info.MapSize) * 2
                if setErr := wb.env.SetMapSize(newSize); setErr != nil {
                    wb.releaseLock()
                    return setErr
                }
                // loop will retry
                continue
            }

            // 其他错误直接返回（并释放锁）
            wb.releaseLock()
            return err
        }
    }

    wb.flushed = true
    wb.releaseLock()
    return nil
}

// Close: 确保释放锁一次，不重复 unlock
func (wb *lmdbWriteBatch) Close() {
    // Close 时若还没 Flush，就尝试 Flush（你也可以改为放弃ops）
    if !wb.flushed {
        _ = wb.Flush() // Flush 会释放锁
        // 如果 Flush 出错，这里忽略（Flush 已返回错误到调用方）
    } else {
        wb.releaseLock()
    }
    // free ops
    wb.ops = nil
}
