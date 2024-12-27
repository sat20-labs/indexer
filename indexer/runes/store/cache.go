package store

import (
	"fmt"

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
	common.Log.Infof("Cache::FlushToDB-> actions count: %d, total bytes: %d", count, totalBytes)

	for key, action := range logs {
		if action.Type == PUT {
			storeWriteBatch.Set([]byte(key), action.Val)
		}
	}
	for key, action := range logs {
		if action.Type == DEL && action.ExistInDb {
			storeDb.Update(func(txn *badger.Txn) error {
				err := txn.Delete([]byte(key))
				if err != nil {
					common.Log.Panicf("Cache::FlushToDB-> err: %v", err.Error())
				}
				return nil
			})
		}
	}
	err := storeWriteBatch.Flush()
	if err != nil {
		common.Log.Panicf("Cache::FlushToDB-> err: %v", err.Error())
	}
}

func ResetCache() {
	logs = make(map[string]*CacheLog)
}

func (s *Cache[T]) Get(key []byte) (ret *T) {
	keyStr := string(key)
	action := logs[keyStr]
	if action != nil {
		if action.Type == DEL {
			return
		}
		var out T
		msg := any(&out).(proto.Message)
		proto.Unmarshal(action.Val, msg)
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
	return
}

func (s *Cache[T]) Set(key []byte, msg proto.Message) (ret *T) {
	ret = s.Get(key)
	val, err := proto.Marshal(msg)
	if err != nil {
		common.Log.Panicf("Cache.Set-> key: %s, proto.Marshal err: %v", string(key), err.Error())
	}
	logs[string(key)] = &CacheLog{
		Val:  val,
		Type: PUT,
	}
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

func (s *Cache[T]) GetFromDB(key []byte) (ret *T, raw []byte) {
	err := storeDb.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil && err != badger.ErrKeyNotFound {
			return err
		}
		if item == nil {
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
