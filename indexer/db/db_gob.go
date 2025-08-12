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

func GobSetDB(key []byte, value interface{}, db KVDB) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(value); err != nil {
		return err
	}
	return db.Write(key, buf.Bytes())
}

func GobGetDB(key []byte, value interface{}, db KVDB) error {
	buf, err := db.Read(key)
	if err != nil {
		return err
	}
	return DecodeBytes(buf, value)
}

func SetDB(key []byte, data interface{}, wb WriteBatch) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(data); err != nil {
		return err
	}
	return wb.Put(key, buf.Bytes())
}

func SetRawDB(key []byte, data []byte, wb WriteBatch) error {
	return wb.Put(key, data)
}

func SetRawValueToDB(key, value []byte, db KVDB) error {
	return db.Write(key, value)
}

func DeleteInDB(key []byte, db KVDB) error {
	return db.Delete(key)
}

func GetRawValueFromDB(key []byte, db KVDB) ([]byte, error) {
	return db.Read(key)
}


func GetRawValueFromTxn(key []byte, db ReadBatch) ([]byte, error) {
	return db.Get(key)
}

func GetValueFromDB(key []byte, v interface{}, db KVDB) (error) {
	buf, err := db.Read(key)
	if err != nil {
		return err
	}
	if err := DecodeBytes(buf, v); err != nil {
		return err
	}
	return nil
}

func GetValueFromTxn(key []byte, v interface{}, db ReadBatch) (error) {
	buf, err := db.Get(key)
	if err != nil {
		return err
	}
	if err := DecodeBytes(buf, v); err != nil {
		return err
	}
	return nil
}

func GetValueFromDB2[T any](key []byte, db KVDB) (*T, error) {
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

func GetAddressValueDBKey(addressid uint64, utxoid uint64, typ, i int) []byte {
	if i == 0 {
		return []byte(fmt.Sprintf(common.DB_KEY_ADDRESSVALUE+"%x-%x-%x", addressid, utxoid, typ))
	}
	return []byte(fmt.Sprintf(common.DB_KEY_ADDRESSVALUE+"%x-%x-%x-%x", addressid, utxoid, typ, i))
}

func GetUtxoIdKey(id uint64) []byte {
	return []byte(fmt.Sprintf(common.DB_KEY_UTXOID+"%x", id))
}

func GetBlockDBKey(height int) []byte {
	return []byte(fmt.Sprintf(common.DB_KEY_BLOCK+"%x", height))
}

func BindUtxoDBKeyToId(utxoDBKey []byte, id uint64, wb WriteBatch) error {
	return wb.Put(GetUtxoIdKey(id), utxoDBKey)
}

func UnBindUtxoId(id uint64, wb WriteBatch) error {
	return wb.Delete(GetUtxoIdKey(id))
}

func GetUtxoByID(db KVDB, id uint64) (string, error) {
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

func BindAddressDBKeyToId(address string, id uint64, wb WriteBatch) error {
	if err := wb.Put(GetAddressIdKey(id), []byte(address)); err != nil {
		return err
	}
	return wb.Put(GetAddressDBKey(address), common.Uint64ToBytes(id))
}

func UnBindAddressId(address string, id uint64, wb WriteBatch) error {
	wb.Delete(GetAddressIdKey(id))
	wb.Delete(GetAddressDBKey(address))
	return nil
}

func GetAddressByIDFromDB(ldb KVDB, id uint64) (string, error) {
	key, err := ldb.Read(GetAddressIdKey(id))
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(string(key), common.DB_KEY_ADDRESS), nil
}

func GetAddressByID(txn ReadBatch, id uint64) (string, error) {
	key, err := txn.Get(GetAddressIdKey(id))
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(string(key), common.DB_KEY_ADDRESS), nil
}

func GetAddressIdFromDB(db KVDB, address string) (uint64, error) {
	key, err := db.Read(GetAddressDBKey(address))
	if err != nil {
		return common.INVALID_ID, err
	}
	return common.BytesToUint64(key), nil
}


func GetAddressIdFromTxn(db ReadBatch, address string) (uint64, error) {
	key, err := db.Get(GetAddressDBKey(address))
	if err != nil {
		return common.INVALID_ID, err
	}
	return common.BytesToUint64(key), nil
}

func CheckKeyExists(db KVDB, key []byte) bool {
	_, err := db.Read(key)
	return err == nil
}

func CheckKeyExistsFromTxn(db ReadBatch, key []byte) bool {
	_, err := db.Get(key)
	return err == nil
}

func BackupDB(fname string, db KVDB) error {
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

func RestoreDB(backupFile string, db KVDB) error {
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
