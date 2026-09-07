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

// L2 DB 兼容约束：SatoshiNet indexer 使用此接口写入同步状态，禁止修改 Gob 编码或增加格式封装。
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

// L2 DB 兼容约束：SatoshiNet indexer 的区块、UTXO、STP、ticker 和同步状态等记录使用此接口。
// 禁止修改 Gob 编码或增加格式封装；L1 如需新格式，应另定义接口。
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

// L2 DB 兼容约束：SatoshiNet indexer 使用此接口读取已有 Gob 记录，禁止修改记录格式。
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

// L2 DB 兼容约束：SatoshiNet indexer 使用此接口读取已有 Gob 记录，禁止修改记录格式。
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

// L2 DB 兼容约束：SatoshiNet indexer 使用此接口解码已有 Gob 记录，禁止修改编码或格式封装。
func DecodeBytes(data []byte, target interface{}) error {
	return gob.NewDecoder(bytes.NewBuffer(data)).Decode(target)
}

// L2 DB 兼容约束：SatoshiNet indexer 的 u- 表使用此键，禁止修改前缀、txid 字节和 vout 文本编码。
func GetUTXODBKey(utxo string) []byte {
	parts := strings.Split(utxo, ":")
	data, err := hex.DecodeString(parts[0])
	if err != nil {
		common.Log.Panicf("wrong utxo format %s", utxo)
	}
	return append(append([]byte(common.DB_KEY_UTXO), data...), []byte(parts[1])...)
}

// L2 DB 兼容约束：SatoshiNet indexer 的地址 ID 映射使用此键，禁止修改 a- 加地址的格式。
func GetAddressDBKey(address string) []byte {
	return []byte(common.DB_KEY_ADDRESS + address)
}

// L2 DB 兼容约束：SatoshiNet indexer 的地址表使用此键，禁止修改 a2- 加地址的格式。
func GetAddressDBKeyV2(address string) []byte {
	return []byte(common.DB_KEY_ADDRESSV2 + address)
}

func GetAddressValueDBKey(addressid uint64, utxoid uint64) []byte {
	return []byte(fmt.Sprintf(common.DB_KEY_ADDRESSVALUE+"%x-%x", addressid, utxoid))
}

// L2 DB 兼容约束：SatoshiNet indexer 使用此键，禁止修改 ui- 加十六进制 UTXO ID 的格式。
func GetUtxoIdKey(id uint64) []byte {
	return []byte(fmt.Sprintf(common.DB_KEY_UTXOID+"%x", id))
}

// L2 DB 兼容约束：SatoshiNet indexer 使用此键，禁止修改 b- 加十六进制高度的格式。
func GetBlockDBKey(height int) []byte {
	return []byte(fmt.Sprintf(common.DB_KEY_BLOCK+"%x", height))
}

// L2 DB 兼容约束：SatoshiNet indexer 使用此映射，禁止修改 ui- 的键或所存 u- 键原始字节。
func BindUtxoDBKeyToId(utxoDBKey []byte, id uint64, wb common.WriteBatch) error {
	return wb.Put(GetUtxoIdKey(id), utxoDBKey)
}

func UnBindUtxoId(id uint64, wb common.WriteBatch) error {
	return wb.Delete(GetUtxoIdKey(id))
}

// L2 DB 兼容约束：SatoshiNet indexer 使用此接口读取 ui- 映射，禁止修改映射格式和解码含义。
func GetUtxoByID(db common.KVDB, id uint64) (string, error) {
	key, err := db.Read(GetUtxoIdKey(id))
	if err != nil {
		return "", err
	}
	return GetUtxoByDBKey(key)
}

// L2 DB 兼容约束：SatoshiNet indexer 使用此接口解析已落盘的 u- 键，禁止修改解码格式。
func GetUtxoByDBKey(key []byte) (string, error) {
	plen := len(common.DB_KEY_UTXO)
	return hex.EncodeToString(key[plen:plen+32]) + ":" + string(key[plen+32:]), nil
}

// L2 DB 兼容约束：SatoshiNet indexer 使用此键，禁止修改 ai- 加十进制地址 ID 的格式。
func GetAddressIdKey(id uint64) []byte {
	return []byte(fmt.Sprintf(common.DB_KEY_ADDRESSID+"%d", id))
}

// L2 DB 兼容约束：SatoshiNet indexer 使用此双向映射，禁止修改 ai- 的地址文本或 a- 的 8 字节大端 ID。
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

// L2 DB 兼容约束：SatoshiNet indexer 使用此接口读取 ai- 映射，禁止修改映射格式和解码含义。
func GetAddressByIDFromDB(ldb common.KVDB, id uint64) (string, error) {
	key, err := ldb.Read(GetAddressIdKey(id))
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(string(key), common.DB_KEY_ADDRESS), nil
}

// L2 DB 兼容约束：SatoshiNet indexer 使用此接口读取 ai- 映射，禁止修改映射格式和解码含义。
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

// L2 DB 兼容约束：SatoshiNet indexer 使用此接口读取 a- 映射，禁止修改 8 字节大端 ID 格式。
func GetAddressIdFromDB(db common.KVDB, address string) (uint64, error) {
	key, err := db.Read(GetAddressDBKey(address))
	if err != nil {
		return common.INVALID_ID, err
	}
	return common.BytesToUint64(key), nil
}


// L2 DB 兼容约束：SatoshiNet indexer 的地址表读取路径使用此接口，禁止修改 8 字节大端 ID 格式。
func GetAddressIdFromTxn(db common.ReadBatch, address string) (uint64, error) {
	key, err := db.Get(GetAddressDBKey(address))
	if err != nil {
		return common.INVALID_ID, err
	}
	return common.BytesToUint64(key), nil
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


