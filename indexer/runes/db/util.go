package db

import (
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func Get[T any](key []byte) (*T, error) {
	var ret T
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		err = item.Value(func(v []byte) error {
			msg, ok := any(&ret).(proto.Message)
			if !ok {
				return fmt.Errorf("type %T does not implement proto.Message", ret)
			}
			return proto.Unmarshal(v, msg)
		})
		return err
	})
	return &ret, err
}

func Set(key []byte, value protoreflect.ProtoMessage) error {
	val, err := proto.Marshal(value)
	if err != nil {
		return err
	}
	err = db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, val)
	})
	return err
}

func Delete(key []byte) error {
	return db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

func Remove[T any](key []byte) (*T, error) {
	ret, err := Get[T](key)
	if err != nil {
		return nil, err
	}
	err = Delete(key)
	return ret, err
}
