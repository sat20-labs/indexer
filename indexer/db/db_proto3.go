package db

import (
	"github.com/sat20-labs/indexer/common"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func SetDBWithProto3(key []byte, data protoreflect.ProtoMessage, wb common.WriteBatch) error {
    dataBytes, err := proto.Marshal(data)
    if err != nil {
        return err
    }
    return wb.Put(key, dataBytes)
}

func GetValueFromDB2WithProto3(key []byte, target protoreflect.ProtoMessage, db common.KVDB) error {
    data, err := db.Read(key)
    if err != nil {
        return err
    }
    return proto.Unmarshal(data, target)
}

func GetValueFromDBWithProto3(key []byte, ldb common.KVDB, target protoreflect.ProtoMessage) error {
    data, err := ldb.Read(key)
    if err != nil {
        return err
    }
    return proto.Unmarshal(data, target)
}

func GetValueFromTxnWithProto3(key []byte, txn common.ReadBatch, target protoreflect.ProtoMessage) error {
    data, err := txn.Get(key)
    if err != nil {
        return err
    }
    return proto.Unmarshal(data, target)
}

func GetValueFromDBWithTypeWithProto3[T protoreflect.ProtoMessage](key []byte, db common.KVDB) (T, error) {
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


func GetAddressDataFromDBV2(db common.KVDB, address string) (*common.AddressValueInDBV2, error) {
	var result common.AddressValueInDBV2

	v, err := db.Read(GetAddressDBKeyV2(address))
	if err != nil {
		//common.Log.Errorf("GetAddressIdFromDBTxn %s error: %v", address, err)
		return nil, err
	}
	
	err = DecodeBytesWithProto3(v, &result)
	return &result, err
}


func GetAddressDataFromDBTxnV2(txn common.ReadBatch, address string) (*common.AddressValueInDBV2, error) {
	var result common.AddressValueInDBV2
	err := GetValueFromTxnWithProto3(GetAddressDBKeyV2(address), txn, &result)
	if err != nil {
		// 即使没有数据，也可以尝试读取已经保存的id
		addrId, err := GetAddressIdFromTxn(txn, address)
		if err != nil {
			return nil, err
		}
		result.AddressId = addrId
		result.AddressType = int32(common.GetAddressTypeFromAddress(address))
	}
	return &result, nil
}
