package common

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"strconv"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

type BuckStore struct {
	db       common.KVDB
	BuckSize int64
	prefix   string
	lastKey string
}

// key = id, value = T
type BuckValue struct {
	Value []byte
}

func NewBuckStore(db common.KVDB, buckSize int64, prefix string) *BuckStore {
	return &BuckStore{
		db:       db,
		BuckSize: buckSize,
		prefix:   prefix,
		lastKey:  prefix+"lk",
	}
}

func (bs *BuckStore) Put(key int64, value *BuckValue) error {
	bucket := bs.getBucket(key)

	dbkey := []byte(bs.prefix + strconv.Itoa(bucket))
	val, err := bs.db.Read(dbkey)
	if err != nil && err != common.ErrKeyNotFound {
		common.Log.Errorf("Get %s: %v", dbkey, err)
		return err
	}

	var storedData map[int64]*BuckValue
	if err == nil {
		storedData, err = bs.deserialize(val)
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

	err = bs.db.Write(dbkey, serializedData)
	if err != nil {
		common.Log.Errorf("db.Write %s failed: %v", dbkey, err)
		return err
	}

	lastKeyBytes := make([]byte, binary.MaxVarintLen64)
	binary.BigEndian.PutUint64(lastKeyBytes, uint64(key))
	err = bs.db.Write([]byte(bs.lastKey), lastKeyBytes)
	if err != nil {
		common.Log.Panicf("failed to update DB: %v", err)
	}

	return nil
}

func (bs *BuckStore) GetLastKey() int64 {
	dbkey := []byte(bs.lastKey)
	val, err := bs.db.Read(dbkey)
	if err != nil {
		common.Log.Errorf("Get %s failed: %v", dbkey, err)
		return -1
	}

	return int64(binary.BigEndian.Uint64(val))
}

func (bs *BuckStore) Get(key int64) (*BuckValue, error) {
	bucket := bs.getBucket(key)

	var value *BuckValue

	dbkey := []byte(bs.prefix + strconv.Itoa(bucket))
	val, err := bs.db.Read(dbkey)
	if err != nil {
		common.Log.Errorf("Get %s failed: %v", dbkey, err)
		return nil, err
	}

	storedData, err := bs.deserialize(val)
	if err != nil {
		common.Log.Errorf("Value %s failed: %v", dbkey, err)
		return nil, err
	}

	var ok bool
	value, ok = storedData[key]
	if !ok {
		common.Log.Errorf("key %d not found in bucket", key)
		return nil, fmt.Errorf("key not found in bucket")
	}

	return value, nil
}

func (bs *BuckStore) getBucketData(bucket int) map[int64]*BuckValue {
	result := make(map[int64]*BuckValue)

	dbkey := []byte(bs.prefix + strconv.Itoa(bucket))
	val, err := bs.db.Read(dbkey)
	if err != nil {
		common.Log.Errorf("Get %s failed: %v", dbkey, err)
		return nil
	}

	result, err = bs.deserialize(val)

	if err != nil {
		common.Log.Errorf("Value %s failed: %v", dbkey, err)
		return nil
	}

	return result
}

func (bs *BuckStore) BatchGet(start, end int64) map[int64]*BuckValue {
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

func (bs *BuckStore) BatchPut(valuemap map[int64]*BuckValue) error {
	lastkey := int64(-1)
	buckets := make(map[int]map[int64]*BuckValue, 0)

	var err error

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

	bs.db.View(func(txn common.ReadBatch) error {
		for bucket, value := range buckets {
			dbkey := []byte(bs.prefix + strconv.Itoa(bucket))
			item, err := txn.Get(dbkey)
			if err == common.ErrKeyNotFound {
				continue
			}
			if err != nil {
				common.Log.Panicf("Get %s failed. %v", dbkey, err)
			}

			storedData, err := bs.deserialize(item)
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
	defer wb.Close()
	for bucket, value := range buckets {
		dbkey := []byte(bs.prefix + strconv.Itoa(bucket))
		err = db.SetDB(dbkey, value, wb)
		if err != nil {
			common.Log.Panicf("SetDB %s failed. %v", dbkey, err)
		}
	}

	if lastkey != -1 {
		lastKeyBytes := make([]byte, binary.MaxVarintLen64)
		binary.BigEndian.PutUint64(lastKeyBytes, uint64(lastkey))
		err = db.SetRawDB([]byte(bs.lastKey), lastKeyBytes, wb)
		if err != nil {
			common.Log.Panicf("SetRawDB %s failed. %v", bs.lastKey, err)
		}
	}

	err = wb.Flush()
	if err != nil {
		common.Log.Panicf("Indexer.updateBasicDB-> Error satwb flushing writes to db %v", err)
	}

	return nil
}

func (bs *BuckStore) Reset() {
	bs.db.DropPrefix([]byte(bs.prefix))
}

func (bs *BuckStore) GetAll() map[int64]*BuckValue {
	result := make(map[int64]*BuckValue, 0)
	err := bs.db.BatchRead([]byte(bs.prefix), false, func(k, v []byte) error {

		if string(k) == bs.lastKey {
			return nil
		}

		storedData, err := bs.deserialize(v)
		if err != nil {
			common.Log.Errorf("Value %s failed: %v", string(k), err)
			return nil
		}
		for k, v := range storedData {
			result[k] = v
		}

		return nil
	})

	if err != nil {
		common.Log.Errorf("GetAll failed: %v", err)
		return nil
	}

	return result
}

func (bs *BuckStore) getBucket(key int64) int {
	bucket := key / bs.BuckSize
	return int(bucket)
}

func (bs *BuckStore) serialize(data map[int64]*BuckValue) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		common.Log.Errorf("Encode failed : %v", err)
		return nil, err
	}
	return buf.Bytes(), nil
}

func (bs *BuckStore) deserialize(serializedData []byte) (map[int64]*BuckValue, error) {
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
