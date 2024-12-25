package store

import (
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
	"google.golang.org/protobuf/proto"
)

type Store[T any] struct {
	pval map[string][]byte
	db   *badger.DB
	txn  *badger.Txn
}

func NewStore[T any](db *badger.DB) *Store[T] {
	return &Store[T]{
		db:   db,
		pval: make(map[string][]byte),
	}
}

func (s *Store[T]) SetTxn(txn *badger.Txn) {
	s.txn = txn
}

func (s *Store[T]) Get(key []byte) (ret *T) {
	var val T
	if s.pval[string(key)] != nil {
		msg := any(&val).(proto.Message)
		proto.Unmarshal(s.pval[string(key)], msg)
		ret = &val
		return
	}
	item, err := s.txn.Get(key)
	if err != nil && err != badger.ErrKeyNotFound {
		common.Log.Panicf("Store.Get-> err: %v", err)
	}
	if item == nil {
		return nil
	}

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
		s.pval[string(key)] = v
		return nil
	})

	if err != nil {
		common.Log.Panicf("Store.Get-> err: %v", err)
	}
	return
}

func (s *Store[T]) Remove(key []byte) (ret *T) {
	ret = s.Get(key)
	if ret == nil {
		return
	}
	err := s.txn.Delete(key)
	if err != nil {
		common.Log.Panicf("Store.Remove-> err: %v", err)
	}
	return
}

func (s *Store[T]) Insert(key []byte, msg proto.Message) (ret *T) {
	ret = s.Remove(key)
	val, err := proto.Marshal(msg)
	if err != nil {
		common.Log.Panicf("Store.Insert-> key: %s, proto.Marshal err: %v", string(key), err.Error())
	}
	s.pval[string(key)] = val
	return
}

func (s *Store[T]) Flush() {
	for key, val := range s.pval {
		err := s.txn.Set([]byte(key), val)
		if err != nil {
			common.Log.Panicf("Store.Flush-> key: %s, txn.Set err: %v", key, err.Error())
		}
	}
	s.pval = make(map[string][]byte, 0)
}

func (s *Store[T]) InsertNoTransaction(key []byte, val proto.Message) {
	err := s.db.Update(func(txn *badger.Txn) error {
		val, err := proto.Marshal(val)
		if err != nil {
			return err
		}
		return txn.Set(key, val)
	})
	if err != nil {
		common.Log.Panicf("Store.InsertNoTransaction-> err: %v", err.Error())
	}
}

func (s *Store[T]) GetNoTransaction(key []byte) (ret *T) {
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil && err != badger.ErrKeyNotFound {
			common.Log.Panicf("Store.GetNoTransaction-> err: %v", err)
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
			return nil
		})
		return err
	})

	if err != nil {
		common.Log.Panicf("Store.GetNoTransaction-> err: %v", err)
	}

	return
}
