package common

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	badger "github.com/dgraph-io/badger/v4"
)

func GobSetDB1(key []byte, value interface{}, db *badger.DB) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(value); err != nil {
		return err
	}
	err := db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, buf.Bytes())
	})
	return err
}

func SetDB(key []byte, data interface{}, wb *badger.WriteBatch) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(data); err != nil {
		return err
	}
	return wb.Set([]byte(key), []byte(buf.Bytes()))
}

func SetRawDB(key []byte, data []byte, wb *badger.WriteBatch) error {
	return wb.Set(key, data)
}

func SetRawValueToDB(key, value []byte, db *badger.DB) error {
	err := db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
	if err != nil {
		Log.Errorf("Failed to write data: %v\n", err)
	}
	return err
}

func DeleteInDB(key []byte, db *badger.DB) error {
	err := db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
	if err != nil {
		Log.Errorf("Failed to delete key %v: %v\n", key, err)
	}
	return err
}

func GetRawValueFromDB(key []byte, db *badger.DB) ([]byte, error) {
	var ret []byte
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			ret = append([]byte{}, val...)
			return nil
		})
	})

	return ret, err
}

func GetValueFromDB2[T any](key []byte, db *badger.DB) (*T, error) {
	var ret *T
	var err error
	err = db.View(func(txn *badger.Txn) error {
		ret, err = GetValueFromDBWithType[T](key, txn)
		if err != nil {
			return err
		}
		return nil
	})

	return ret, err
}

func GetValueFromDB(key []byte, txn *badger.Txn, target interface{}) error {
	item, err := txn.Get([]byte(key))
	if err != nil {
		return err
	}
	err = item.Value(func(v []byte) error {
		err := DecodeBytes(v, target)
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func GetValueFromDBWithType[T any](key []byte, txn *badger.Txn) (*T, error) {
	item, err := txn.Get([]byte(key))
	if err != nil {
		return nil, err
	}
	var target T
	err = item.Value(func(v []byte) error {
		return DecodeBytes(v, &target)
	})
	return &target, err
}

func IsExistWithPrefixFromDB[T any](prefix []byte, db *badger.DB) (find bool, value T, err error) {
	err = db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		it.Seek([]byte(prefix))
		if it.ValidForPrefix([]byte(prefix)) {
			err = it.Item().Value(func(data []byte) error {
				return DecodeBytes(data, &value)
			})
			find = err == nil
		}
		return err
	})
	return
}

func GetValuesWithPrefixFromDB[T any](prefix []byte, txn *badger.Txn) (map[string]T, error) {
	result := make(map[string]T)
	itr := txn.NewIterator(badger.DefaultIteratorOptions)
	defer itr.Close()

	for itr.Seek([]byte(prefix)); itr.ValidForPrefix([]byte(prefix)); itr.Next() {
		item := itr.Item()
		var value T
		err := item.Value(func(data []byte) error {
			return DecodeBytes(data, &value)
		})
		if err != nil {
			Log.Warnf("GetValuesWithPrefixFromDB error: %v", err)
			continue
		}
		key := item.KeyCopy(nil)
		result[string(key)] = value
	}
	return result, nil
}

func GetValuesWithPrefixFromDB2[T any](prefix []byte, txn *badger.Txn) (map[string]*T, error) {
	result := make(map[string]*T)
	itr := txn.NewIterator(badger.DefaultIteratorOptions)
	defer itr.Close()

	for itr.Seek([]byte(prefix)); itr.ValidForPrefix([]byte(prefix)); itr.Next() {
		item := itr.Item()
		var value T
		err := item.Value(func(data []byte) error {
			return DecodeBytes(data, &value)
		})
		if err != nil {
			Log.Warnf("GetValuesWithPrefixFromDB error: %v", err)
			continue
		}
		key := bytes.Split(item.KeyCopy(nil), prefix)
		result[string(key[0])] = &value
	}
	return result, nil
}

func DecodeBytes(data []byte, target interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(target)
}

func Uint64ToBytes(value uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, value)
	return bytes
}

func BytesToUint64(bytes []byte) uint64 {
	return binary.BigEndian.Uint64(bytes)
}

func GetUTXODBKey(utxo string) []byte {
	// 2d0a64a14faa9dc707dc84647a4e0dd1d4f31753e8a85574128bc8110e312e10 (testnet)
	// 输出有10万个
	parts := strings.Split(utxo, ":")
	data, err := hex.DecodeString(parts[0])
	if err != nil {
		Log.Panicf("wrong utxo format %s", utxo)
	}

	ret := append([]byte(DB_KEY_UTXO), data...)
	return append(ret, []byte(parts[1])...)
}

func GetAddressDBKey(address string) []byte {
	return []byte(DB_KEY_ADDRESS + address)
}

func GetAddressValueDBKey(addressid uint64, utxoid uint64, typ, i int) []byte {
	if i == 0 {
		return []byte(fmt.Sprintf(DB_KEY_ADDRESSVALUE+"%x-%x-%x", addressid, utxoid, typ))
	} else {
		return []byte(fmt.Sprintf(DB_KEY_ADDRESSVALUE+"%x-%x-%x-%x", addressid, utxoid, typ, i))
	}
}

func GetUtxoIdKey(id uint64) []byte {
	return []byte(fmt.Sprintf(DB_KEY_UTXOID+"%x", id))
}

func GetBlockDBKey(height int) []byte {
	return []byte(fmt.Sprintf(DB_KEY_BLOCK+"%x", height))
}

func BindUtxoDBKeyToId(utxoDBKey []byte, id uint64, wb *badger.WriteBatch) error {
	return wb.Set((GetUtxoIdKey(id)), (utxoDBKey))
}

func UnBindUtxoId(id uint64, wb *badger.WriteBatch) error {
	return wb.Delete((GetUtxoIdKey(id)))
}

func GetUtxoByID(db *badger.DB, id uint64) (string, error) {
	var key []byte
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(GetUtxoIdKey(id))
		if err != nil {
			//Log.Errorf("GetUtxoByID %x error: %v", id, err)
			return err
		}
		return item.Value(func(val []byte) error {
			key = append([]byte{}, val...)
			return nil
		})
	})

	if err != nil {
		return "", err
	}

	return GetUtxoByDBKey(key)
}

func GetUtxoByDBKey(key []byte) (string, error) {
	// 至少35字节，前两位 u-，中间32位是utxo，最后是vout
	klen := len(key)
	plen := len(DB_KEY_UTXO)
	utxoBytes := key[plen : 32+plen]
	utxo := hex.EncodeToString(utxoBytes)
	vout := string(key[32+plen : klen])

	return utxo + ":" + vout, nil
}

func GetAddressIdKey(id uint64) []byte {
	return []byte(fmt.Sprintf(DB_KEY_ADDRESSID+"%d", id))
}

func BindAddressDBKeyToId(address string, id uint64, wb *badger.WriteBatch) error {
	err := wb.Set((GetAddressIdKey(id)), []byte(address))
	if err != nil {
		return err
	}

	return wb.Set(GetAddressDBKey(address), Uint64ToBytes(id))
}

func UnBindAddressId(address string, id uint64, wb *badger.WriteBatch) error {
	wb.Delete((GetAddressIdKey(id)))
	wb.Delete(GetAddressDBKey(address))
	return nil
}

func GetAddressByID(db *badger.DB, id uint64) (string, error) {
	var key []byte
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(GetAddressIdKey(id))
		if err != nil {
			Log.Errorf("GetAddressByID %x error: %v", id, err)
			return err
		}
		return item.Value(func(val []byte) error {
			key = append([]byte{}, val...)
			return nil
		})
	})
	return strings.TrimPrefix(string(key), DB_KEY_ADDRESS), err
}

func GetAddressByIDFromDBTxn(txn *badger.Txn, id uint64) (string, error) {
	var key []byte

	item, err := txn.Get(GetAddressIdKey(id))
	if err != nil {
		Log.Errorf("GetAddressByIDFromDBTxn %x error: %v", id, err)
		return "", err
	}
	err = item.Value(func(val []byte) error {
		key = append([]byte{}, val...)
		return nil
	})

	return strings.TrimPrefix(string(key), DB_KEY_ADDRESS), err
}

func GetAddressIdFromDB(db *badger.DB, address string) (uint64, error) {
	var key []byte
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(GetAddressDBKey(address))
		if err != nil {
			Log.Errorf("GetAddressIdFromDB %s error: %v", address, err)
			return err
		}
		return item.Value(func(val []byte) error {
			key = append([]byte{}, val...)
			return nil
		})
	})
	if err != nil {
		return INVALID_ID, err
	}
	return BytesToUint64(key), err
}

func GetAddressIdFromDBTxn(txn *badger.Txn, address string) (uint64, error) {
	var key []byte

	item, err := txn.Get(GetAddressDBKey(address))
	if err != nil {
		//Log.Errorf("GetAddressIdFromDBTxn %s error: %v", address, err)
		return INVALID_ID, err
	}
	err = item.Value(func(val []byte) error {
		key = append([]byte{}, val...)
		return nil
	})

	return BytesToUint64(key), err
}

func CheckKeyExists(db *badger.DB, key []byte) bool {
	var exists bool

	err := db.View(func(txn *badger.Txn) error {
		_, err := txn.Get((key))
		if err == nil {
			exists = true
		} else if err == badger.ErrKeyNotFound {
			exists = false
		} else {
			return err
		}

		return nil
	})

	if err != nil {
		Log.Errorf("failed to check key existence: %v", err)
		return false
	}

	return exists
}

// 不能与其他DB读写混用，要确保这一点
func RunBadgerGC(db *badger.DB) {
	// 只有在跑数据后，打开，压缩数据，同时做严格的数据检查
	if db.IsClosed() {
		return
	}

	for {
		err := db.RunValueLogGC(0.5)
		Log.Infof("RunValueLogGC return %v", err)
		if err == badger.ErrNoRewrite {
			break
		} else if err != nil {
			break
		}
	}
	db.Sync()
	Log.Info("badgerGc: RunValueLogGC is done")

	//Log.Infof("levels: %v", db.LevelsToString())
}

func BackupDB(fname string, db *badger.DB) error {
	// 创建备份文件
	backupFile, err := os.Create(fname)
	if err != nil {
		Log.Errorf("create file %s failed. %v", fname, err)
		return err
	}
	defer backupFile.Close()

	// 执行备份
	since := uint64(0) // 从最早的事务开始备份
	latestVersion, err := db.Backup(backupFile, since)
	if err != nil {
		Log.Errorf("Backup failed. %v", err)
		return err
	}

	Log.Infof("Backup completed, new version: %d\n", latestVersion)
	return nil
}

func RestoreDB(backupFile string, targetDir string) error {
	// 打开备份文件
	backup, err := os.Open(backupFile)
	if err != nil {
		Log.Errorf("Open file %s failed. %v", backupFile, err)
		return err
	}
	defer backup.Close()

	// 创建目标数据库目录
	err = os.MkdirAll(targetDir, os.ModePerm)
	if err != nil {
		Log.Errorf("MkdirAll %s failed. %v", targetDir, err)
		return err
	}

	// 创建 Badger 数据库
	opts := badger.DefaultOptions(targetDir).WithInMemory(false)
	db, err := badger.Open(opts)
	if err != nil {
		Log.Errorf("Open DB failed. %v", err)
		return err
	}
	defer db.Close()

	// 执行恢复
	err = db.Load(backup, 0)
	if err != nil {
		Log.Errorf("Load DB failed. %v", err)
		return err
	}

	Log.Info("DB restored")
	return nil
}
