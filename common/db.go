package common

import (
	"fmt"

	badger "github.com/dgraph-io/badger/v4"
)


func SeekItemInDB(searchKey []byte, db *badger.DB) []byte {
	// 查找第一个大于或等于给定键的项
	var result []byte
	err := db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		it.Seek(searchKey)

		if it.Valid() {
			item := it.Item()
			result = item.KeyCopy(nil)
			return nil
			//fmt.Printf("Key: %s, Value: %s\n", key, value)
		} else {
			return fmt.Errorf("no item found")
		}
	})
	if err != nil {
		Log.Errorf("can't find key %v", searchKey)
	}

	return result
}

func IterateRangeInDB(db *badger.DB, startKey, endKey []byte, processFunc func(key, value []byte) error) error {
	err := db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		it.Seek(startKey)
		for it.Valid() {
			item := it.Item()
			key := item.KeyCopy(nil)
			if compareKeys(key, endKey) > 0 {
				break
			}

			value, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			err = processFunc(key, value)
			if err != nil {
				return err
			}

			it.Next()
		}

		return nil
	})

	return err
}

func compareKeys(key1, key2 []byte) int {
	if len(key1) < len(key2) {
		return -1
	} else if len(key1) > len(key2) {
		return 1
	}
	return bytesCompare(key1, key2)
}

func bytesCompare(a, b []byte) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] < b[i] {
			return -1
		} else if a[i] > b[i] {
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	} else if len(a) > len(b) {
		return 1
	}
	return 0
}
