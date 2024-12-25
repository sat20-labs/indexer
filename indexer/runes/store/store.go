package store

import (
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
	"google.golang.org/protobuf/proto"
)

type Store[T any] struct {
	db  *badger.DB
	txn *badger.Txn
}

func NewStore[T any](db *badger.DB) *Store[T] {
	return &Store[T]{
		db: db,
	}
}

func (s *Store[T]) SetTxn(txn *badger.Txn) {
	s.txn = txn
}

func (s *Store[T]) Get(key []byte) (ret *T) {
	item, err := s.txn.Get(key)
	if err != nil && err != badger.ErrKeyNotFound {
		common.Log.Panicf("Store.Get-> err: %v", err)
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

	if err != nil {
		common.Log.Panicf("Store.Get-> err: %v", err)
	}
	return
}

func (s *Store[T]) Remove(key []byte) (ret *T) {
	ret = s.Get(key)
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
		common.Log.Panicf("Store.Insert-> err: %v", err.Error())
	}
	s.txn.Set(key, val)
	return
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
