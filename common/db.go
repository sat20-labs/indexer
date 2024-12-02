package common

import (

	badger "github.com/dgraph-io/badger/v4"
)


func IterateRangeInDB(db *badger.DB, startKey, endKey []byte, processFunc func(key, value []byte) error) error {
	err := db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		it.Seek(startKey)
		for it.Valid() {
			item := it.Item()
			if item.IsDeletedOrExpired() {
				Log.Errorf("IsDeletedOrExpired return true")
				it.Next()
				continue
			}

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
