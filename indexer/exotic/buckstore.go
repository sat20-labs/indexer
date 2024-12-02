package exotic

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"strconv"

	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
)

const key_last = "exotic-lastkey"

type BuckStore struct {
	db       *badger.DB
	BuckSize int
	prefix   string
}

func NewBuckStore(db *badger.DB, prefix string) *BuckStore {
	return &BuckStore{
		db:       db,
		BuckSize: 10000,
		prefix:   "exotic-" + prefix + "-",
	}
}

func (bs *BuckStore) Put(key int, value *common.Range) error {
	bucket := bs.getBucket(key)

	err := bs.db.Update(func(txn *badger.Txn) error {
		dbkey := []byte(bs.prefix + strconv.Itoa(bucket))
		item, err := txn.Get(dbkey)
		if err != nil && err != badger.ErrKeyNotFound {
			common.Log.Errorf("Get %s: %v", dbkey, err)
			return err
		}

		var storedData map[int]*common.Range

		if err == nil {
			err = item.Value(func(val []byte) error {
				storedData, err = bs.deserialize(val)
				return err
			})
			if err != nil {
				common.Log.Errorf("deserialize: %v", err)
				return err
			}
		} else {
			storedData = make(map[int]*common.Range)
		}

		storedData[(key)] = value

		serializedData, err := bs.serialize(storedData)
		if err != nil {
			return err
		}

		err = txn.SetEntry(&badger.Entry{
			Key:   dbkey,
			Value: serializedData,
		})
		if err != nil {
			common.Log.Errorf("SetEntry %s failed: %v", dbkey, err)
			return err
		}

		lastKeyBytes := make([]byte, binary.MaxVarintLen32)
		binary.BigEndian.PutUint32(lastKeyBytes, uint32(key))
		err = txn.SetEntry(&badger.Entry{
			Key:   []byte(key_last),
			Value: lastKeyBytes,
		})
		if err != nil {
			common.Log.Errorf("SetEntry %s failed: %v", key_last, err)
			return err
		}

		return nil
	})

	if err != nil {
		common.Log.Panicf("failed to update Badger DB: %v", err)
	}

	return nil
}

func (bs *BuckStore) GetLastKey() int {
	key := -1
	bs.db.View(func(txn *badger.Txn) error {
		dbkey := []byte(key_last)
		item, err := txn.Get(dbkey)
		if err != nil {
			common.Log.Errorf("Get %s failed: %v", dbkey, err)
			return err
		}

		item.Value(func(val []byte) error {
			key = int(binary.BigEndian.Uint32(val))
			return nil
		})

		return nil
	})

	return key
}

func (bs *BuckStore) Get(key int) (*common.Range, error) {
	bucket := bs.getBucket(key)

	var value *common.Range
	var ok bool
	err := bs.db.View(func(txn *badger.Txn) error {
		dbkey := []byte(bs.prefix + strconv.Itoa(bucket))
		item, err := txn.Get(dbkey)
		if err != nil {
			common.Log.Errorf("Get %s failed: %v", dbkey, err)
			return err
		}

		var storedData map[int]*common.Range
		err = item.Value(func(val []byte) error {
			storedData, err = bs.deserialize(val)
			return err
		})
		if err != nil {
			common.Log.Errorf("Value %s failed: %v", dbkey, err)
			return err
		}

		value, ok = storedData[(key)]
		if !ok {
			common.Log.Errorf("key %d not found in bucket", key)
			return fmt.Errorf("key not found in bucket")
		}

		return nil
	})

	if err != nil {
		common.Log.Errorf("failed to read %d from Badger DB %v", key, err)
		return nil, err
	}

	return value, nil
}

func (bs *BuckStore) BatchPut(valuemap map[int]*common.Range) error {

	lastkey := -1
	buckets := make(map[int]map[int]*common.Range, 0)

	var err error
	bs.db.View(func(txn *badger.Txn) error {

		for height, value := range valuemap {
			bucket := bs.getBucket(height)
			rngmap, ok := buckets[bucket]
			if ok {
				rngmap[height] = value
			} else {
				rngmap = make(map[int]*common.Range)
				rngmap[height] = value
				buckets[bucket] = rngmap
			}
			if height > lastkey {
				lastkey = height
			}
		}

		for bucket, value := range buckets {
			dbkey := []byte(bs.prefix + strconv.Itoa(bucket))
			item, err := txn.Get(dbkey)
			if err == badger.ErrKeyNotFound {
				continue
			}
			if err != nil {
				common.Log.Panicf("Get %s failed. %v", dbkey, err)
			}

			var storedData map[int]*common.Range
			err = item.Value(func(val []byte) error {
				storedData, err = bs.deserialize(val)
				return err
			})
			if err != nil {
				common.Log.Panicf("Value %s failed. %v", dbkey, err)
			}
			for height, rng := range storedData {
				value[height] = rng
			}
		}
		return nil
	})

	wb := bs.db.NewWriteBatch()
	defer wb.Cancel()
	for bucket, value := range buckets {
		dbkey := []byte(bs.prefix + strconv.Itoa(bucket))
		err = common.SetDB(dbkey, value, wb)
		if err != nil {
			common.Log.Panicf("SetDB %s failed. %v", dbkey, err)
		}
	}

	lastKeyBytes := make([]byte, binary.MaxVarintLen32)
	binary.BigEndian.PutUint32(lastKeyBytes, uint32(lastkey))
	err = common.SetRawDB([]byte(key_last), lastKeyBytes, wb)
	if err != nil {
		common.Log.Panicf("SetRawDB %s failed. %v", key_last, err)
	}

	err = wb.Flush()
	if err != nil {
		common.Log.Panicf("Indexer.updateBasicDB-> Error satwb flushing writes to db %v", err)
	}

	return nil
}

func (bs *BuckStore) Reset() {
	lastkey := bs.GetLastKey()
	if lastkey < 0 {
		return
	}
	bs.db.DropPrefix([]byte(bs.prefix))
	bs.db.Update(func(txn *badger.Txn) error {
		txn.Delete([]byte(key_last))
		return nil
	})
}

func (bs *BuckStore) GetAll() map[int]*common.Range {
	result := make(map[int]*common.Range, 0)
	err := bs.db.View(func(txn *badger.Txn) error {
		// 设置前缀扫描选项
		prefixBytes := []byte(bs.prefix)
		prefixOptions := badger.DefaultIteratorOptions
		prefixOptions.Prefix = prefixBytes

		// 使用前缀扫描选项创建迭代器
		it := txn.NewIterator(prefixOptions)
		defer it.Close()

		var err error
		// 遍历匹配前缀的key
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			if item.IsDeletedOrExpired() {
				continue
			}

			var bulk map[int]*common.Range
			err = item.Value(func(val []byte) error {
				bulk, err = bs.deserialize(val)
				return err
			})
			if err != nil {
				common.Log.Errorf("Value failed: %v", err)
				continue
			}
			for k, v := range bulk {
				result[k] = v
			}
		}
		return nil
	})

	if err != nil {
		common.Log.Errorf("GetAll failed: %v", err)
		return nil
	}

	return result
}

func (bs *BuckStore) getBucket(key int) int {
	bucket := key / bs.BuckSize
	return bucket
}

func (bs *BuckStore) serialize(data map[int]*common.Range) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		common.Log.Errorf("Encode failed : %v", err)
		return nil, err
	}
	return buf.Bytes(), nil
}

func (bs *BuckStore) deserialize(serializedData []byte) (map[int]*common.Range, error) {
	var data map[int]*common.Range
	buf := bytes.NewBuffer(serializedData)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&data)
	if err != nil {
		common.Log.Errorf("Decode failed : %v", err)
		return nil, err
	}
	return data, nil
}
