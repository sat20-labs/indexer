package db

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sat20-labs/indexer/common"
)

func GobSetDB(key []byte, value interface{}, db common.KVDB) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(value); err != nil {
		return err
	}
	return db.Write(key, buf.Bytes())
}

func GobGetDB(key []byte, value interface{}, db common.KVDB) error {
	buf, err := db.Read(key)
	if err != nil {
		return err
	}
	return DecodeBytes(buf, value)
}

func SetDB(key []byte, data interface{}, wb common.WriteBatch) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(data); err != nil {
		return err
	}
	return wb.Put(key, buf.Bytes())
}

func SetRawDB(key []byte, data []byte, wb common.WriteBatch) error {
	return wb.Put(key, data)
}

func SetRawValueToDB(key, value []byte, db common.KVDB) error {
	return db.Write(key, value)
}

func DeleteInDB(key []byte, db common.KVDB) error {
	return db.Delete(key)
}

func GetRawValueFromDB(key []byte, db common.KVDB) ([]byte, error) {
	return db.Read(key)
}


func GetRawValueFromTxn(key []byte, db common.ReadBatch) ([]byte, error) {
	return db.Get(key)
}

func GetValueFromDB(key []byte, v interface{}, db common.KVDB) (error) {
	buf, err := db.Read(key)
	if err != nil {
		return err
	}
	if err := DecodeBytes(buf, v); err != nil {
		return err
	}
	return nil
}

func GetValueFromTxn(key []byte, v interface{}, db common.ReadBatch) (error) {
	buf, err := db.Get(key)
	if err != nil {
		return err
	}
	if err := DecodeBytes(buf, v); err != nil {
		return err
	}
	return nil
}

func GetValueFromDB2[T any](key []byte, db common.KVDB) (*T, error) {
	var ret T
	buf, err := db.Read(key)
	if err != nil {
		return nil, err
	}
	if err := DecodeBytes(buf, &ret); err != nil {
		return nil, err
	}
	return &ret, nil
}

func DecodeBytes(data []byte, target interface{}) error {
	return gob.NewDecoder(bytes.NewBuffer(data)).Decode(target)
}

func GetUTXODBKey(utxo string) []byte {
	parts := strings.Split(utxo, ":")
	data, err := hex.DecodeString(parts[0])
	if err != nil {
		common.Log.Panicf("wrong utxo format %s", utxo)
	}
	return append(append([]byte(common.DB_KEY_UTXO), data...), []byte(parts[1])...)
}

func GetAddressDBKey(address string) []byte {
	return []byte(common.DB_KEY_ADDRESS + address)
}

func GetAddressDBKeyV2(address string) []byte {
	return []byte(common.DB_KEY_ADDRESSV2 + address)
}

func GetAddressValueDBKey(addressid uint64, utxoid uint64) []byte {
	return []byte(fmt.Sprintf(common.DB_KEY_ADDRESSVALUE+"%x-%x", addressid, utxoid))
}

func GetUtxoIdKey(id uint64) []byte {
	return []byte(fmt.Sprintf(common.DB_KEY_UTXOID+"%x", id))
}

func GetBlockDBKey(height int) []byte {
	return []byte(fmt.Sprintf(common.DB_KEY_BLOCK+"%x", height))
}

func BindUtxoDBKeyToId(utxoDBKey []byte, id uint64, wb common.WriteBatch) error {
	return wb.Put(GetUtxoIdKey(id), utxoDBKey)
}

func UnBindUtxoId(id uint64, wb common.WriteBatch) error {
	return wb.Delete(GetUtxoIdKey(id))
}

func GetUtxoByID(db common.KVDB, id uint64) (string, error) {
	key, err := db.Read(GetUtxoIdKey(id))
	if err != nil {
		return "", err
	}
	return GetUtxoByDBKey(key)
}

func GetUtxoByDBKey(key []byte) (string, error) {
	plen := len(common.DB_KEY_UTXO)
	return hex.EncodeToString(key[plen:plen+32]) + ":" + string(key[plen+32:]), nil
}

func GetAddressIdKey(id uint64) []byte {
	return []byte(fmt.Sprintf(common.DB_KEY_ADDRESSID+"%d", id))
}

func BindAddressDBKeyToId(address string, id uint64, wb common.WriteBatch) error {
	if err := wb.Put(GetAddressIdKey(id), []byte(address)); err != nil {
		return err
	}
	return wb.Put(GetAddressDBKey(address), common.Uint64ToBytes(id))
}

func UnBindAddressId(address string, id uint64, wb common.WriteBatch) error {
	wb.Delete(GetAddressIdKey(id))
	wb.Delete(GetAddressDBKey(address))
	return nil
}

func GetAddressByIDFromDB(ldb common.KVDB, id uint64) (string, error) {
	key, err := ldb.Read(GetAddressIdKey(id))
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(string(key), common.DB_KEY_ADDRESS), nil
}

func GetAddressByIDFromTxn(txn common.ReadBatch, id uint64) (string, error) {
	key, err := txn.Get(GetAddressIdKey(id))
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(string(key), common.DB_KEY_ADDRESS), nil
}

func GetAddressByID(txn common.ReadBatch, id uint64) (string, error) {
	key, err := txn.Get(GetAddressIdKey(id))
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(string(key), common.DB_KEY_ADDRESS), nil
}

func GetAddressIdFromDB(db common.KVDB, address string) (uint64, error) {
	key, err := db.Read(GetAddressDBKey(address))
	if err != nil {
		return common.INVALID_ID, err
	}
	return common.BytesToUint64(key), nil
}


func GetAddressIdFromTxn(db common.ReadBatch, address string) (uint64, error) {
	key, err := db.Get(GetAddressDBKey(address))
	if err != nil {
		return common.INVALID_ID, err
	}
	return common.BytesToUint64(key), nil
}


func GetAddressDataFromDBV2(db common.KVDB, address string) (*common.AddressValueInDBV2, error) {
	var result common.AddressValueInDBV2

	v, err := db.Read(GetAddressDBKeyV2(address))
	if err != nil {
		//common.Log.Errorf("GetAddressIdFromDBTxn %s error: %v", address, err)
		return nil, err
	}
	
	err = DecodeBytes(v, &result)
	return &result, err
}

func GetAddressDataFromDBTxnV2(txn common.ReadBatch, address string) (*common.AddressValueInDBV2, error) {
	var result common.AddressValueInDBV2
	err := GetValueFromTxnWithProto3(GetAddressDBKeyV2(address), txn, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func CheckKeyExists(db common.KVDB, key []byte) bool {
	_, err := db.Read(key)
	return err == nil
}

func CheckKeyExistsFromTxn(db common.ReadBatch, key []byte) bool {
	_, err := db.Get(key)
	return err == nil
}

func BackupDB(fname string, db common.KVDB) error {
	if bdb, ok := db.(interface{ BackupToFile(string) error }); ok {
		return bdb.BackupToFile(fname)
	}
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	return db.BatchRead(nil, false, func(k, v []byte) error {
		return enc.Encode([2][]byte{k, v})
	})
}

func RestoreDB(backupFile string, db common.KVDB) error {
	if rdb, ok := db.(interface{ RestoreFromFile(string) error }); ok {
		return rdb.RestoreFromFile(backupFile)
	}
	f, err := os.Open(backupFile)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	wb := db.NewWriteBatch()
	defer wb.Close()
	for {
		var kv [2][]byte
		if err := dec.Decode(&kv); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if err := wb.Put(kv[0], kv[1]); err != nil {
			return err
		}
	}
	return wb.Flush()
}


func IterateRangeInDB(db common.KVDB, prefix, startKey, endKey []byte, 
	processFunc func(key, value []byte) error) error {
    return db.BatchReadV2(prefix, startKey, false, func(k, v []byte) error {
        // 检查是否超过结束键
        if len(endKey) > 0 && bytes.Compare(k, endKey) > 0 {
            return fmt.Errorf("reach the endkey") // 作为特殊信号来终止迭代
        }
        return processFunc(k, v)
    })
}

