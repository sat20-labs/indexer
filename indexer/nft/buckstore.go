package nft

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"strconv"

	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
)

const key_last = DB_PREFIX_BUCK + "lk"

type NftBuckStore struct {
	db       *badger.DB
	BuckSize int64
	prefix   string
}

// key == nft Id
type BuckValue struct {
	Sat int64
}

func NewBuckStore(db *badger.DB) *NftBuckStore {
	return &NftBuckStore{
		db:       db,
		BuckSize: 10000,
		prefix:   DB_PREFIX_BUCK,
	}
}

func (bs *NftBuckStore) Put(key int64, value *BuckValue) error {
	bucket := bs.getBucket(key)

	err := bs.db.Update(func(txn *badger.Txn) error {
		dbkey := []byte(bs.prefix + strconv.Itoa(bucket))
		item, err := txn.Get(dbkey)
		if err != nil && err != badger.ErrKeyNotFound {
			common.Log.Errorf("Get %s: %v", dbkey, err)
			return err
		}

		var storedData map[int64]*BuckValue

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
			storedData = make(map[int64]*BuckValue, 0)
		}

		storedData[key] = value

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

		lastKeyBytes := make([]byte, binary.MaxVarintLen64)
		binary.BigEndian.PutUint64(lastKeyBytes, uint64(key))
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

func (bs *NftBuckStore) GetLastKey() int64 {
	key := int64(-1)
	bs.db.View(func(txn *badger.Txn) error {
		dbkey := []byte(key_last)
		item, err := txn.Get(dbkey)
		if err != nil {
			common.Log.Errorf("Get %s failed: %v", dbkey, err)
			return err
		}

		item.Value(func(val []byte) error {
			key = int64(binary.BigEndian.Uint64(val))
			return nil
		})

		return nil
	})

	return key
}

func (bs *NftBuckStore) Get(key int64) (*BuckValue, error) {
	bucket := bs.getBucket(key)

	var value *BuckValue
	err := bs.db.View(func(txn *badger.Txn) error {
		dbkey := []byte(bs.prefix + strconv.Itoa(bucket))
		item, err := txn.Get(dbkey)
		if err != nil {
			common.Log.Errorf("Get %s failed: %v", dbkey, err)
			return err
		}

		var storedData map[int64]*BuckValue
		err = item.Value(func(val []byte) error {
			storedData, err = bs.deserialize(val)
			return err
		})
		if err != nil {
			common.Log.Errorf("Value %s failed: %v", dbkey, err)
			return err
		}

		var ok bool
		value, ok = storedData[key]
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

func (bs *NftBuckStore) getBucketData(bucket int) map[int64]*BuckValue {
	result := make(map[int64]*BuckValue)
	bs.db.View(func(txn *badger.Txn) error {
		dbkey := []byte(bs.prefix + strconv.Itoa(bucket))
		item, err := txn.Get(dbkey)
		if err != nil {
			common.Log.Errorf("Get %s failed: %v", dbkey, err)
			return err
		}

		err = item.Value(func(val []byte) error {
			result, err = bs.deserialize(val)
			return err
		})
		if err != nil {
			common.Log.Errorf("Value %s failed: %v", dbkey, err)
			return err
		}

		return nil
	})

	return result
}

func (bs *NftBuckStore) BatchGet(start, end int64) map[int64]*BuckValue {
	result := make(map[int64]*BuckValue)

	lastKey := bs.GetLastKey()
	if lastKey < start {
		return result
	}

	bucket1 := bs.getBucket(start)
	bucket2 := bs.getBucket(end)

	for i := bucket1; i <= bucket2; i++ {
		bmap := bs.getBucketData(i)
		for k, v := range bmap {
			if k >= start && k <= end {
				result[k] = v
			}
		}
	}

	return result
}

func (bs *NftBuckStore) BatchPut(valuemap map[int64]*BuckValue) error {
	lastkey := int64(-1)
	buckets := make(map[int]map[int64]*BuckValue, 0)

	var err error
	bs.db.View(func(txn *badger.Txn) error {

		for key, value := range valuemap {
			bucket := bs.getBucket(key)
			rngmap, ok := buckets[bucket]
			if ok {
				rngmap[key] = value
			} else {
				rngmap = make(map[int64]*BuckValue)
				rngmap[key] = value
				buckets[bucket] = rngmap
			}
			if key > lastkey {
				lastkey = key
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

			var storedData map[int64]*BuckValue
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

	if lastkey != -1 {
		lastKeyBytes := make([]byte, binary.MaxVarintLen64)
		binary.BigEndian.PutUint64(lastKeyBytes, uint64(lastkey))
		err = common.SetRawDB([]byte(key_last), lastKeyBytes, wb)
		if err != nil {
			common.Log.Panicf("SetRawDB %s failed. %v", key_last, err)
		}
	}

	err = wb.Flush()
	if err != nil {
		common.Log.Panicf("Indexer.updateBasicDB-> Error satwb flushing writes to db %v", err)
	}

	return nil
}

func (bs *NftBuckStore) Reset() {
	bs.db.DropPrefix([]byte(bs.prefix))
}

// id -> sat
func (bs *NftBuckStore) GetAll() map[int64]*BuckValue {
	result := make(map[int64]*BuckValue, 0)
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

			if string(item.Key()) == key_last {
				continue
			}

			var storedData map[int64]*BuckValue
			err = item.Value(func(val []byte) error {
				storedData, err = bs.deserialize(val)
				return err
			})
			if err != nil {
				common.Log.Errorf("Value %s failed: %v", item.Key(), err)
				continue
			}
			for k, v := range storedData {
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

func (bs *NftBuckStore) getBucket(key int64) int {
	bucket := key / bs.BuckSize
	return int(bucket)
}

func (bs *NftBuckStore) serialize(data map[int64]*BuckValue) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		common.Log.Errorf("Encode failed : %v", err)
		return nil, err
	}
	return buf.Bytes(), nil
}

func (bs *NftBuckStore) deserialize(serializedData []byte) (map[int64]*BuckValue, error) {
	var data map[int64]*BuckValue
	buf := bytes.NewBuffer(serializedData)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&data)
	if err != nil {
		common.Log.Errorf("Decode failed : %v", err)
		return nil, err
	}
	return data, nil
}
