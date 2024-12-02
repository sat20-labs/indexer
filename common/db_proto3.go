package common

import (
	badger "github.com/dgraph-io/badger/v4"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func SetDBWithProto3(key []byte, data protoreflect.ProtoMessage, wb *badger.WriteBatch) error {
	dataBytes, err := proto.Marshal(data)
	if err != nil {
		return err
	}
	return wb.Set([]byte(key), dataBytes)
}

func GetValueFromDB2WithProto3(key []byte, target protoreflect.ProtoMessage, db *badger.DB) (error) {
	var err error
	err = db.View(func(txn *badger.Txn) error {
		err = GetValueFromDBWithProto3(key, txn, target)
		if err != nil {
			return err
		}
		return nil
	})

	return err
}

func GetValueFromDBWithProto3(key []byte, txn *badger.Txn, target protoreflect.ProtoMessage) error {
	item, err := txn.Get([]byte(key))
	if err != nil {
		return err
	}
	return item.Value(func(v []byte) error {
		return proto.Unmarshal(v, target)
	})
}

func GetValueFromDBWithTypeWithProto3[T protoreflect.ProtoMessage](key []byte, txn *badger.Txn) (T, error) {
	var ret T
	item, err := txn.Get([]byte(key))
	if err != nil {
		return ret, err
	}
	err = item.Value(func(v []byte) error {
		return proto.Unmarshal(v, ret)
	})
	return ret, err
}


func DecodeBytesWithProto3(data []byte, target protoreflect.ProtoMessage) error {
	return proto.Unmarshal(data, target)
}