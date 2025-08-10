package nft

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"strconv"

	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

const key_last = DB_PREFIX_BUCK + "lk"

type NftBuckStore struct {
	db       db.KVDB
	BuckSize int64
	prefix   string
}

// key == nft Id
type BuckValue struct {
	Sat int64
}

func NewBuckStore(db db.KVDB) *NftBuckStore {
	return &NftBuckStore{
		db:       db,
		BuckSize: 10000,
		prefix:   DB_PREFIX_BUCK,
	}
}

func (bs *NftBuckStore) Put(key int64, value *BuckValue) error {
	bucket := bs.getBucket(key)


		dbkey := []byte(bs.prefix + strconv.Itoa(bucket))
		val, err := bs.db.Read(dbkey)
		if err != nil && err != badger.ErrKeyNotFound {
			common.Log.Errorf("Get %s: %v", dbkey, err)
			return err
		}

		storedData, err := bs.deserialize(val)
		if err != nil {
			common.Log.Errorf("deserialize: %v", err)
			return err
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
		err = bs.db.Write([]byte(key_last), lastKeyBytes)
		if err != nil {
			common.Log.Errorf("db.Write %s failed: %v", key_last, err)
			return err
		}


	if err != nil {
		common.Log.Panicf("failed to update Badger DB: %v", err)
	}

	return nil
}

func (bs *NftBuckStore) GetLastKey() int64 {
		dbkey := []byte(key_last)
		val, err := bs.db.Read(dbkey)
		if err != nil {
			common.Log.Errorf("Get %s failed: %v", dbkey, err)
			return -1
		}

		return int64(binary.BigEndian.Uint64(val))
}

func (bs *NftBuckStore) Get(key int64) (*BuckValue, error) {
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

func (bs *NftBuckStore) getBucketData(bucket int) map[int64]*BuckValue {
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
		item, err := bs.db.Read(dbkey)
		if err == badger.ErrKeyNotFound {
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
		err = db.SetRawDB([]byte(key_last), lastKeyBytes, wb)
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
	err := bs.db.BatchRead([]byte(bs.prefix), false, func(k, v []byte) error {

		if string(k) == key_last {
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
