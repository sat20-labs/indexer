package store

import (
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"
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
	logs            map[string]*CacheLog
)

type Cache[T any] struct{}

func NewCache[T any]() *Cache[T] { return &Cache[T]{} }

func SetDB(v *badger.DB) {
	storeDb = v
}

func SetWriteBatch(v *badger.WriteBatch) {
	storeWriteBatch = v
}

func SetCacheLogs(v map[string]*CacheLog) {
	logs = v
}

func FlushToDB() {
	if len(logs) == 0 {
		return
	}

	var totalBytes int
	count := len(logs)
	for key, action := range logs {
		totalBytes += len(key)
		totalBytes += len(action.Val)
		totalBytes += 4
		totalBytes += len(key)
	}
	common.Log.Infof("Cache::FlushToDB-> logs count: %d, total bytes: %d", count, totalBytes)

	for key, action := range logs {
		if action.Type == PUT {
			storeWriteBatch.Set([]byte(key), action.Val)
		}
	}
	err := storeWriteBatch.Flush()
	if err != nil {
		common.Log.Panicf("Cache::FlushToDB-> err: %v", err.Error())
	}
	storeDb.Update(func(txn *badger.Txn) error {
		for key, action := range logs {
			if action.Type == DEL && action.ExistInDb {
				err := txn.Delete([]byte(key))
				if err != nil {
					common.Log.Panicf("Cache::FlushToDB-> err: %v", err.Error())
				}
			}
		}
		return nil
	})
}

func (s *Cache[T]) Get(key []byte) (ret *T) {
	keyStr := string(key)
	log := logs[keyStr]
	if log != nil {
		if log.Type == DEL {
			return
		}
		var out T
		msg := any(&out).(proto.Message)
		proto.Unmarshal(log.Val, msg)
		ret = &out
		return
	}

	var raw []byte
	ret, raw = s.GetFromDB(key)
	if len(raw) > 0 {
		logs[keyStr] = &CacheLog{
			Val:       raw,
			Type:      INIT,
			ExistInDb: true,
			TimeStamp: 0,
		}
	}
	return
}

func (s *Cache[T]) Delete(key []byte) (ret *T) {
	ret = s.Get(key)
	if ret == nil {
		return
	}
	log := logs[string(key)]
	log.Type = DEL
	log.TimeStamp = time.Now().UnixNano()
	return
}

func (s *Cache[T]) Set(key []byte, msg proto.Message) (ret *T) {
	ret = s.Get(key)
	val, err := proto.Marshal(msg)
	if err != nil {
		common.Log.Panicf("Cache.Set-> key: %s, proto.Marshal err: %v", string(key), err.Error())
	}
	log := logs[string(key)]
	if log == nil {
		logs[string(key)] = &CacheLog{}
		log = logs[string(key)]
	}
	log.Val = val
	log.Type = PUT
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
