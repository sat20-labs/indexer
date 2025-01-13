package store

import (
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/dgraph-io/badger/v4"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/sat20-labs/indexer/common"
	"google.golang.org/protobuf/proto"
)

type ActionType int

const (
	INIT ActionType = 0
	PUT  ActionType = 1
	DEL  ActionType = 2
)

type CacheLog struct {
	Val       []byte
	Type      ActionType
	ExistInDb bool
	TimeStamp int64
}

var (
	storeDb         *badger.DB
	storeWriteBatch *badger.WriteBatch
	logs            *cmap.ConcurrentMap[string, *CacheLog]
)

type Cache[T any] struct{}

func NewCache[T any]() *Cache[T] { return &Cache[T]{} }

func SetDB(v *badger.DB) {
	storeDb = v
}

func SetWriteBatch(v *badger.WriteBatch) {
	storeWriteBatch = v
}

func SetCacheLogs(v *cmap.ConcurrentMap[string, *CacheLog]) {
	logs = v
}

func FlushToDB() {
	var totalBytes int64
	count := logs.Count()
	if count != 0 {
		for log := range logs.IterBuffered() {
			totalBytes += int64(len(log.Key))
			totalBytes += int64(unsafe.Sizeof(log.Val))
			totalBytes += int64(len(log.Val.Val))
			if log.Val.Type == PUT {
				storeWriteBatch.Set([]byte(log.Key), log.Val.Val)
			}
		}

		err := storeWriteBatch.Flush()
		if err != nil {
			common.Log.Panicf("Cache::FlushToDB-> storeWriteBatch.Flush err:%s", err.Error())
		}
		storeDb.Update(func(txn *badger.Txn) error {
			for log := range logs.IterBuffered() {
				if log.Val.Type == DEL && log.Val.ExistInDb {
					err := txn.Delete([]byte(log.Key))
					if err != nil {
						common.Log.Panicf("Cache::FlushToDB-> storeDb.Update err:%s", err.Error())
					}
				}
			}
			return nil
		})
	}
	common.Log.Debugf("Cache::FlushToDB-> logs count:%d, total bytes:%d", count, totalBytes)
}

func (s *Cache[T]) Get(key []byte) (ret *T) {
	keyStr := string(key)
	if logs != nil {
		log, ok := logs.Get(keyStr)
		if ok {
			if log.Type == DEL {
				return
			}
			var out T
			msg := any(&out).(proto.Message)
			proto.Unmarshal(log.Val, msg)
			ret = &out
			return
		}
	}

	var raw []byte
	ret, raw = s.GetFromDB(key)
	if logs != nil && len(raw) > 0 {
		logs.Set(keyStr, &CacheLog{
			Val:       raw,
			Type:      INIT,
			ExistInDb: true,
			TimeStamp: 0,
		})
	}
	return
}

func (s *Cache[T]) Delete(key []byte) (ret *T) {
	ret = s.Get(key)
	if ret == nil {
		return
	}
	log, ok := logs.Get(string(key))
	if ok {
		log.Type = DEL
		log.TimeStamp = time.Now().UnixNano()
	} else {
		// must be in cache
		common.Log.Panicf("Cache.Delete-> key: %s, not found in logs", string(key))
	}
	return
}

func (s *Cache[T]) Set(key []byte, msg proto.Message) (ret *T) {
	ret = s.Get(key)
	val, err := proto.Marshal(msg)
	if err != nil {
		common.Log.Panicf("Cache.Set-> key: %s, proto.Marshal err: %v", string(key), err.Error())
	}
	log, ok := logs.Get(string(key))
	if !ok {
		log = &CacheLog{}
		logs.Set(string(key), log)
	}
	log.Type = PUT
	log.Val = val
	log.TimeStamp = time.Now().UnixNano()
	return
}

func (s *Cache[T]) SetToDB(key []byte, val proto.Message) {
	err := storeDb.Update(func(txn *badger.Txn) error {
		val, err := proto.Marshal(val)
		if err != nil {
			return err
		}
		return txn.Set(key, val)
	})
	if err != nil {
		common.Log.Panicf("Cache.SetToDB-> err: %v", err.Error())
	}
}

func (s *Cache[T]) GetList(keyPrefix []byte, isNeedValue bool) (ret map[string]*T) {
	ret = s.GetListFromDB(keyPrefix, isNeedValue)
	if len(ret) == 0 {
		ret = make(map[string]*T)
	}
	if logs == nil {
		return
	}
	keyPrefixStr := string(keyPrefix)
	for log := range logs.IterBuffered() {
		if strings.HasPrefix(log.Key, keyPrefixStr) {
			if log.Val.Type == DEL {
				delete(ret, log.Key)
			} else if log.Val.Type == PUT {
				var out T
				if isNeedValue {
					msg := any(&out).(proto.Message)
					proto.Unmarshal(log.Val.Val, msg)
				}
				if ret == nil {
					ret = make(map[string]*T)
				}
				ret[log.Key] = &out
			}
		}
	}
	for log := range logs.IterBuffered() {
		if strings.HasPrefix(log.Key, keyPrefixStr) {
			if log.Val.Type == DEL {
				delete(ret, log.Key)
			} else if log.Val.Type == PUT {
				var out T
				if isNeedValue {
					msg := any(&out).(proto.Message)
					proto.Unmarshal(log.Val.Val, msg)
				}
				if ret == nil {
					ret = make(map[string]*T)
				}
				ret[log.Key] = &out
			}
		}
	}
	return
}

func (s *Cache[T]) GetListFromDB(keyPrefix []byte, isNeedValue bool) (ret map[string]*T) {
	err := storeDb.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(keyPrefix); it.ValidForPrefix(keyPrefix); it.Next() {
			item := it.Item()
			if item.IsDeletedOrExpired() {
				continue
			}

			key := item.KeyCopy(nil)
			var out T
			if isNeedValue {
				v, err := item.ValueCopy(nil)
				if err != nil {
					return err
				}
				msg, ok := any(&out).(proto.Message)
				if !ok {
					return fmt.Errorf("type %T does not implement proto.Message", out)
				}
				err = proto.Unmarshal(v, msg)
				if err != nil {
					return err
				}
			}

			if ret == nil {
				ret = make(map[string]*T)
			}

			ret[string(key)] = &out
		}
		return nil
	})

	if err != nil {
		common.Log.Panicf("Cache.GetListFromDB-> err:%s", err.Error())
	}
	return
}

func (s *Cache[T]) GetFromDB(key []byte) (ret *T, raw []byte) {
	err := storeDb.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil && err != badger.ErrKeyNotFound {
			return err
		}
		if item == nil {
			return nil
		}
		if item.IsDeletedOrExpired() {
			return nil
		}
		var out T
		err = item.Value(func(v []byte) error {
			if len(v) == 0 {
				ret = nil
				raw = nil
				return nil
			}
			msg, ok := any(&out).(proto.Message)
			if !ok {
				return fmt.Errorf("type %T does not implement proto.Message", out)
			}
			err = proto.Unmarshal(v, msg)
			if err != nil {
				return err
			}
			ret = &out
			raw = v
			return nil
		})
		return err
	})

	if err != nil {
		common.Log.Panicf("Cache.GetFromDB-> err: %v", err)
	}

	return
}
