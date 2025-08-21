package exotic

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"strconv"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

const key_last = "exotic-lastkey"

type BuckStore struct {
	db       common.KVDB
	BuckSize int
	prefix   string
}

func NewBuckStore(db common.KVDB, prefix string) *BuckStore {
	return &BuckStore{
		db:       db,
		BuckSize: 10000,
		prefix:   "exotic-" + prefix + "-",
	}
}

func (bs *BuckStore) Put(key int, value *common.Range) error {
	bucket := bs.getBucket(key)

	dbkey := []byte(bs.prefix + strconv.Itoa(bucket))
	item, err := bs.db.Read(dbkey)
	if err != nil && err != common.ErrKeyNotFound {
		common.Log.Errorf("Get %s: %v", dbkey, err)
		return err
	}

	var storedData map[int]*common.Range
	if err == nil {
		storedData, err = bs.deserialize(item)
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

	err = bs.db.Write(dbkey, serializedData)
	if err != nil {
		common.Log.Errorf("db.Write %s failed: %v", dbkey, err)
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

func (bs *BuckStore) GetLastKey() int {

	dbkey := []byte(key_last)
	val, err := bs.db.Read(dbkey)
	if err != nil {
		common.Log.Errorf("Get %s failed: %v", dbkey, err)
		return -1
	}

	return int(binary.BigEndian.Uint32(val))
}

func (bs *BuckStore) Get(key int) (*common.Range, error) {
	bucket := bs.getBucket(key)

	var value *common.Range
	var ok bool

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

	value, ok = storedData[(key)]
	if !ok {
		common.Log.Errorf("key %d not found in bucket", key)
		return nil, fmt.Errorf("key not found in bucket")
	}

	return value, nil
}

func (bs *BuckStore) BatchPut(valuemap map[int]*common.Range) error {

	lastkey := -1
	buckets := make(map[int]map[int]*common.Range, 0)

	var err error

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

	bs.db.View(func(txn common.ReadBatch) error {
		for bucket, value := range buckets {
			dbkey := []byte(bs.prefix + strconv.Itoa(bucket))
			val, err := txn.Get(dbkey)
			if err == common.ErrKeyNotFound {
				continue
			}
			if err != nil {
				common.Log.Panicf("Get %s failed. %v", dbkey, err)
			}

			storedData, err := bs.deserialize(val)
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

	lastKeyBytes := make([]byte, binary.MaxVarintLen32)
	binary.BigEndian.PutUint32(lastKeyBytes, uint32(lastkey))
	err = db.SetRawDB([]byte(key_last), lastKeyBytes, wb)
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
	bs.db.Delete([]byte(key_last))
}

func (bs *BuckStore) GetAll() map[int]*common.Range {
	result := make(map[int]*common.Range, 0)
	err := bs.db.BatchRead([]byte(bs.prefix), false, func(k, v []byte) error {
		// 设置前缀扫描选项

		bulk, err := bs.deserialize(v)
		if err != nil {
			common.Log.Errorf("Value failed: %v", err)
			return nil
		}
		for k, v := range bulk {
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
