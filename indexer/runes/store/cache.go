package store

import (
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
	"google.golang.org/protobuf/proto"
)

type Action struct {
	Key []byte
	Val []byte
}

var (
	// db
	storeDb           *badger.DB
	storeDbWriteBatch *badger.WriteBatch
	// operation
	actions []Action
	kvs     map[string][]byte
)

type Cache[T any] struct{}

func NewCache[T any]() *Cache[T] { return &Cache[T]{} }

func SetDB(v *badger.DB) {
	storeDb = v
}

func SetWriteBatch(v *badger.WriteBatch) {
	storeDbWriteBatch = v
}

func FlushToDB() {
	if len(actions) == 0 {
		return
	}
	for _, action := range actions {
		storeDbWriteBatch.Set(action.Key, action.Val)
	}
	err := storeDbWriteBatch.Flush()
	if err != nil {
		common.Log.Panicf("Cache::FlushToDb-> err: %v", err.Error())
	}
}

func ResetCache() {
	kvs = make(map[string][]byte)
	actions = make([]Action, 0)
}

func (s *Cache[T]) Get(key []byte) (ret *T) {
	var out T
	item := kvs[string(key)]
	if len(item) > 0 {
		msg := any(&out).(proto.Message)
		proto.Unmarshal(item, msg)
		ret = &out
		return
	}

	ret, raw := s.GetFromDB(key)
	if len(raw) > 0 {
		kvs[string(key)] = raw
	}
	return
}

func (s *Cache[T]) Remove(key []byte) (ret *T) {
	ret = s.Get(key)
	if ret == nil {
		return
	}
	delete(kvs, string(key))
	actions = append(actions, Action{Key: key, Val: []byte{}})
	return
}

func (s *Cache[T]) Insert(key []byte, msg proto.Message) (ret *T) {
	ret = s.Remove(key)
	val, err := proto.Marshal(msg)
	if err != nil {
		common.Log.Panicf("Cache.Insert-> key: %s, proto.Marshal err: %v", string(key), err.Error())
	}
	kvs[string(key)] = val
	actions = append(actions, Action{Key: key, Val: val})
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
		var val T
		err = item.Value(func(v []byte) error {
			if len(v) == 0 {
				ret = nil
				raw = nil
				return nil
			}
			msg, ok := any(&val).(proto.Message)
			if !ok {
				return fmt.Errorf("type %T does not implement proto.Message", val)
			}
			err = proto.Unmarshal(v, msg)
			if err != nil {
				return err
			}
			ret = &val
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
