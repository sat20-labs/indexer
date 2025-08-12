package db

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func SetDBWithProto3(key []byte, data protoreflect.ProtoMessage, wb WriteBatch) error {
    dataBytes, err := proto.Marshal(data)
    if err != nil {
        return err
    }
    return wb.Put(key, dataBytes)
}

func GetValueFromDB2WithProto3(key []byte, target protoreflect.ProtoMessage, db KVDB) error {
    data, err := db.Read(key)
    if err != nil {
        return err
    }
    return proto.Unmarshal(data, target)
}

func GetValueFromDBWithProto3(key []byte, ldb KVDB, target protoreflect.ProtoMessage) error {
    return ldb.View(func(txn ReadBatch) error {
		return GetValueFromTxnWithProto3(key, txn, target)
	})
}

func GetValueFromTxnWithProto3(key []byte, txn ReadBatch, target protoreflect.ProtoMessage) error {
    data, err := txn.Get(key)
    if err != nil {
        return err
    }
    return proto.Unmarshal(data, target)
}

func GetValueFromDBWithTypeWithProto3[T protoreflect.ProtoMessage](key []byte, db KVDB) (T, error) {
    var ret T
    data, err := db.Read(key)
    if err != nil {
        return ret, err
    }
    err = proto.Unmarshal(data, ret)
    return ret, err
}

func DecodeBytesWithProto3(data []byte, target protoreflect.ProtoMessage) error {
    return proto.Unmarshal(data, target)
}