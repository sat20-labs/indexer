package ns

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"strconv"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

const key_last = DB_PREFIX_BUCK + "lk"

type NSBuckStore struct {
	db       common.KVDB
	BuckSize int
	prefix   string
}

type BuckValue struct {
	Name string
	Sat  int64
}

func NewBuckStore(db common.KVDB) *NSBuckStore {
	return &NSBuckStore{
		db:       db,
		BuckSize: 1000,
		prefix:   DB_PREFIX_BUCK,
	}
}

func (bs *NSBuckStore) Put(key int, value *BuckValue) error {
	bucket := bs.getBucket(key)

	dbkey := []byte(bs.prefix + strconv.Itoa(bucket))
	item, err := bs.db.Read(dbkey)
	if err != nil && err != common.ErrKeyNotFound {
		common.Log.Errorf("Get %s: %v", dbkey, err)
		return err
	}

	var storedData map[int]*BuckValue
	if err == nil {
		storedData, err = bs.deserialize(item)
		if err != nil {
			common.Log.Errorf("deserialize: %v", err)
			return err
		}
	} else {
		storedData = make(map[int]*BuckValue, 0)
	}

	storedData[key] = value
	serializedData, err := bs.serialize(storedData)
	if err != nil {
		return err
	}

	err = bs.db.Write(dbkey, serializedData)
	if err != nil {
		common.Log.Errorf("SetEntry %s failed: %v", dbkey, err)
		return err
	}

	lastKeyBytes := make([]byte, binary.MaxVarintLen32)
	binary.BigEndian.PutUint32(lastKeyBytes, uint32(key))
	err = bs.db.Write([]byte(key_last), lastKeyBytes)

	if err != nil {
		common.Log.Panicf("failed to update DB: %v", err)
	}

	return nil
}

func (bs *NSBuckStore) GetLastKey() int {

	dbkey := []byte(key_last)
	item, err := bs.db.Read(dbkey)
	if err != nil {
		common.Log.Errorf("Get %s failed: %v", dbkey, err)
		return -1
	}

	return int(binary.BigEndian.Uint32(item))

}

func (bs *NSBuckStore) Get(key int) (*BuckValue, error) {
	bucket := bs.getBucket(key)

	var value *BuckValue

	dbkey := []byte(bs.prefix + strconv.Itoa(bucket))
	item, err := bs.db.Read(dbkey)
	if err != nil {
		common.Log.Errorf("Get %s failed: %v", dbkey, err)
		return nil, err
	}

	storedData, err := bs.deserialize(item)
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

func (bs *NSBuckStore) getBucketData(bucket int) map[int]*BuckValue {

	dbkey := []byte(bs.prefix + strconv.Itoa(bucket))
	item, err := bs.db.Read(dbkey)
	if err != nil {
		common.Log.Errorf("Get %s failed: %v", dbkey, err)
		return nil
	}

	result, err := bs.deserialize(item)
	if err != nil {
		common.Log.Errorf("Value %s failed: %v", dbkey, err)
		return nil
	}

	return result
}

func (bs *NSBuckStore) BatchGet(start, end int) map[int]*BuckValue {
	result := make(map[int]*BuckValue)

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

func (bs *NSBuckStore) BatchPut(valuemap map[int]*BuckValue) error {
	lastkey := -1
	buckets := make(map[int]map[int]*BuckValue, 0)

	var err error

	for key, value := range valuemap {
		bucket := bs.getBucket(key)
		rngmap, ok := buckets[bucket]
		if ok {
			rngmap[key] = value
		} else {
			rngmap = make(map[int]*BuckValue)
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
		lastKeyBytes := make([]byte, binary.MaxVarintLen32)
		binary.BigEndian.PutUint32(lastKeyBytes, uint32(lastkey))
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

func (bs *NSBuckStore) Reset() {
	bs.db.DropPrefix([]byte(bs.prefix))
}

func (bs *NSBuckStore) GetAll() map[int]*BuckValue {
	result := make(map[int]*BuckValue, 0)
	err := bs.db.BatchRead([]byte(bs.prefix), false, func(k, v []byte) error {

		if string(k) == key_last {
			return nil
		}

		storedData, err := bs.deserialize(v)
		if err != nil {
			// last_key
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

func (bs *NSBuckStore) getBucket(key int) int {
	bucket := key / bs.BuckSize
	return bucket
}

func (bs *NSBuckStore) serialize(data map[int]*BuckValue) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		common.Log.Errorf("Encode failed : %v", err)
		return nil, err
	}
	return buf.Bytes(), nil
}

func (bs *NSBuckStore) deserialize(serializedData []byte) (map[int]*BuckValue, error) {
	var data map[int]*BuckValue
	buf := bytes.NewBuffer(serializedData)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&data)
	if err != nil {
		common.Log.Errorf("Decode failed : %v", err)
		return nil, err
	}
	return data, nil
}
