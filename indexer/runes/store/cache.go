package store

import (
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
	"google.golang.org/protobuf/proto"
)

type Cache[T any] struct {
	pval map[string][]byte
	db   *badger.DB
	wb   *badger.WriteBatch
}

func NewCache[T any](db *badger.DB) *Cache[T] {
	return &Cache[T]{
		db:   db,
		pval: make(map[string][]byte),
	}
}

func (s *Cache[T]) SetWb(wb *badger.WriteBatch) {
	s.wb = wb
}

func (s *Cache[T]) Get(key []byte) (ret *T) {
	var val T
	if s.pval[string(key)] != nil {
		msg := any(&val).(proto.Message)
		proto.Unmarshal(s.pval[string(key)], msg)
		ret = &val
		return
	}

	ret, bytes := s.GetFromDB(key)
	if len(bytes) > 0 {
		s.pval[string(key)] = bytes
	}
	return
}

func (s *Cache[T]) Remove(key []byte) (ret *T) {
	ret = s.Get(key)
	if ret == nil {
		return
	}
	err := s.wb.Delete(key)
	if err != nil {
		common.Log.Panicf("Store.Remove-> err: %v", err)
	}
	return
}

func (s *Cache[T]) Insert(key []byte, msg proto.Message) (ret *T) {
	ret = s.Remove(key)
	val, err := proto.Marshal(msg)
	if err != nil {
		common.Log.Panicf("Store.Insert-> key: %s, proto.Marshal err: %v", string(key), err.Error())
	}
	s.pval[string(key)] = val
	return
}

func (s *Cache[T]) Flush() {
	for key, val := range s.pval {
		err := s.wb.Set([]byte(key), val)
		if err != nil {
			common.Log.Panicf("Store.Flush-> key: %s, wb.Set err: %v", key, err.Error())
		}
	}
	s.pval = make(map[string][]byte, 0)
}

func (s *Cache[T]) SetToDB(key []byte, val proto.Message) {
	err := s.db.Update(func(txn *badger.Txn) error {
		val, err := proto.Marshal(val)
		if err != nil {
			return err
		}
		return txn.Set(key, val)
	})
	if err != nil {
		common.Log.Panicf("Store.SetToDB-> err: %v", err.Error())
	}
}

func (s *Cache[T]) GetFromDB(key []byte) (ret *T, raw []byte) {
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil && err != badger.ErrKeyNotFound {
			return err
		}
		if item == nil {
			return nil
		}
		var val T
		err = item.Value(func(v []byte) error {
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
		common.Log.Panicf("Store.GetFromDB-> err: %v", err)
	}

	return
}
