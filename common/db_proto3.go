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

func GetRawValuesWithPrefixFromDB(prefix []byte, txn *badger.Txn) (map[string][]byte, error) {
	result := make(map[string][]byte)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()

	for it.Seek(prefix); it.ValidForPrefix([]byte(prefix)); it.Next() {
		item := it.Item()
		value, err := item.ValueCopy(nil)
		if err != nil {
			Log.Warnf("GetValuesWithPrefixFromDBWithProto3 error: %v", err)
			continue
		}
		result[string(item.KeyCopy(nil))] = value
	}
	return result, nil
}

func GetValuesWithPrefixFromDBWithProto3[T protoreflect.ProtoMessage](prefix []byte, txn *badger.Txn) (map[string]*T, error) {
	result := make(map[string]*T)
	itr := txn.NewIterator(badger.DefaultIteratorOptions)
	defer itr.Close()

	for itr.Seek([]byte(prefix)); itr.ValidForPrefix([]byte(prefix)); itr.Next() {
		item := itr.Item()
		var value T
		err := item.Value(func(v []byte) error {
			return proto.Unmarshal(v, value)
		})
		if err != nil {
			Log.Warnf("GetValuesWithPrefixFromDBWithProto3 error: %v", err)
			continue
		}
		result[string(item.KeyCopy(nil))] = &value
	}
	return result, nil
}

func DecodeBytesWithProto3(data []byte, target protoreflect.ProtoMessage) error {
	return proto.Unmarshal(data, target)
}