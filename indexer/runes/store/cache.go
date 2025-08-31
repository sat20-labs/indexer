package store

import (
	"fmt"
	"strings"
	"sync/atomic"
	"unsafe"

	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/sat20-labs/indexer/common"
	"google.golang.org/protobuf/proto"
)

type ActionType int

const (
	INIT ActionType = 0
	PUT  ActionType = 1
	DEL  ActionType = 2
)

var counter int64

type DbLog struct {
	Val       []byte
	Type      ActionType
	ExistInDb bool
	TimeStamp int64
}

type DbWrite struct {
	Db             common.KVDB
	logs           *cmap.ConcurrentMap[string, *DbLog]
	cloneTimeStamp int64
}

func NewDbWrite(db common.KVDB, logs *cmap.ConcurrentMap[string, *DbLog]) *DbWrite {
	return &DbWrite{
		Db:   db,
		logs: logs,
	}
}

func (s *DbWrite) clearLogs() {
	s.logs.Clear()
}

func (s *DbWrite) FlushToDB() {
	var totalBytes int64
	count := s.logs.Count()
	if count != 0 {
		wb := s.Db.NewWriteBatch()
		defer wb.Close()
		for log := range s.logs.IterBuffered() {
			totalBytes += int64(len(log.Key))
			totalBytes += int64(unsafe.Sizeof(log.Val))
			totalBytes += int64(len(log.Val.Val))
			if log.Val.Type == PUT {
				wb.Put([]byte(log.Key), log.Val.Val)
			}
		}

		isFinishUpdate := false
		for {
			if isFinishUpdate {
				break
			}

			for log := range s.logs.IterBuffered() {
				if log.Val.Type == DEL && log.Val.ExistInDb {
					err := wb.Delete([]byte(log.Key))
					if err != nil {
						common.Log.Panicf("DbWrite.FlushToDB-> storeDb.Update err:%s", err.Error())
					}
					log.Val.ExistInDb = false
				}
			}
			isFinishUpdate = true
		}

		err := wb.Flush()
		if err != nil {
			common.Log.Panicf("DbWrite.FlushToDB-> WriteBatch.Flush err:%s", err.Error())
		}
	}
	s.clearLogs()

	updateCount := 0
	remmoveCount := 0
	for v := range s.logs.IterBuffered() {
		if v.Val.Type == DEL {
			remmoveCount++
		} else if v.Val.Type == PUT {
			updateCount++
		}
	}
	common.Log.Infof("DbWrite.FlushToDB-> logs count:%d, update count:%d, remove count:%d, total bytes:%d",
		count, updateCount, remmoveCount, totalBytes)
}

func (s *DbWrite) Clone(clone *DbWrite) *DbWrite {
	for log := range s.logs.IterBuffered() {
		newLog := &DbLog{
			Type:      log.Val.Type,
			ExistInDb: log.Val.ExistInDb,
			TimeStamp: log.Val.TimeStamp,
		}
		if log.Val.Val != nil {
			newLog.Val = make([]byte, len(log.Val.Val))
			copy(newLog.Val, log.Val.Val)
		}
		clone.logs.Set(log.Key, newLog)
	}

	clone.cloneTimeStamp = atomic.AddInt64(&counter, 1)
	return clone
}

func (s *DbWrite) Subtract(dbWrite *DbWrite) {
	for log := range s.logs.IterBuffered() {
		if log.Val.TimeStamp <= s.cloneTimeStamp {
			dbWrite.logs.Remove(log.Key)
		}
	}
}

type Cache[T any] struct {
	dbWrite *DbWrite
}

func NewCache[T any](dbWrite *DbWrite) *Cache[T] {
	return &Cache[T]{
		dbWrite: dbWrite,
	}
}

func (s *Cache[T]) Get(key []byte) (ret *T) {
	keyStr := string(key)
	logs := s.dbWrite.logs
	count := logs.Count()
	if count != 0 {
		log, ok := logs.Get(keyStr)
		if ok {
			if log.Type == DEL {
				return
			}
			var out T
			msg := any(&out).(proto.Message)
			err := proto.Unmarshal(log.Val, msg)
			if err != nil {
				common.Log.Panicf("Cache.Get-> key: %s, proto.Unmarshal err: %v", keyStr, err.Error())
			}
			ret = &out
			return
		}
	}

	var raw []byte
	ret, raw = s.GetFromDB(key)
	if len(raw) > 0 {
		s.dbWrite.logs.Set(keyStr, &DbLog{
			Val:       raw,
			Type:      INIT,
			ExistInDb: true,
			TimeStamp: 0,
		})
	}
	return
}

func (s *Cache[T]) Delete(key []byte) (ret *T) {
	ret = s.Get(key)
	if ret == nil {
		return
	}
	log, ok := s.dbWrite.logs.Get(string(key))
	if ok {
		log.Type = DEL
		log.TimeStamp = atomic.AddInt64(&counter, 1)
	} else {
		// must be in cache
		common.Log.Panicf("Cache.Delete-> key: %s, not found in logs", string(key))
	}
	return
}

func (s *Cache[T]) Set(key []byte, msg proto.Message) (ret *T) {
	ret = s.Get(key)
	val, err := proto.Marshal(msg)
	if err != nil {
		common.Log.Panicf("Cache.Set-> key: %s, proto.Marshal err: %v", string(key), err.Error())
	}
	log, ok := s.dbWrite.logs.Get(string(key))
	if !ok {
		log = &DbLog{}
		s.dbWrite.logs.Set(string(key), log)
	}
	log.Type = PUT
	log.Val = val
	log.TimeStamp = atomic.AddInt64(&counter, 1)
	return
}

func (s *Cache[T]) SetToDB(key []byte, val proto.Message) {

	v, err := proto.Marshal(val)
	if err != nil {
		return
	}
	err = s.dbWrite.Db.Write(key, v)
	if err != nil {
		common.Log.Panicf("Cache.SetToDB-> err: %v", err.Error())
	}
}

func (s *Cache[T]) IsExist(keyPrefix []byte, cb func(key []byte, value *T) bool) (ret bool) {
	if s.dbWrite.logs.Count() != 0 {
		keyPrefixStr := string(keyPrefix)
		for log := range s.dbWrite.logs.IterBuffered() {
			if strings.HasPrefix(log.Key, keyPrefixStr) {
				if log.Val.Type != DEL {
					var out T
					msg := any(&out).(proto.Message)
					err := proto.Unmarshal(log.Val.Val, msg)
					if err != nil {
						common.Log.Panicf("Cache.GetList-> key: %s, proto.Unmarshal err: %v", log.Key, err.Error())
					}
					ret = cb([]byte(log.Key), &out)
					if ret {
						return
					}
				}
			}
		}
	}
	ret = s.IsExistFromDB(keyPrefix, cb)
	return
}

func (s *Cache[T]) GetList(keyPrefix []byte, isNeedValue bool) (ret map[string]*T) {
	ret = s.GetListFromDB(keyPrefix, isNeedValue)
	if len(ret) == 0 {
		ret = make(map[string]*T)
	}
	keyPrefixStr := string(keyPrefix)
	for log := range s.dbWrite.logs.IterBuffered() {
		if strings.HasPrefix(log.Key, keyPrefixStr) {
			if log.Val.Type == DEL {
				delete(ret, log.Key)
			} else if log.Val.Type == PUT {
				var out T
				if isNeedValue {
					msg := any(&out).(proto.Message)
					err := proto.Unmarshal(log.Val.Val, msg)
					if err != nil {
						common.Log.Panicf("Cache.GetList-> key: %s, proto.Unmarshal err: %v", log.Key, err.Error())
					}
				}
				ret[log.Key] = &out
			}
		}
	}
	return
}

func (s *Cache[T]) GetListFromDB(keyPrefix []byte, isNeedValue bool) (ret map[string]*T) {
	err := s.dbWrite.Db.BatchRead(keyPrefix, false, func(k, v []byte) error {

		var out T
		if isNeedValue {
			msg, ok := any(&out).(proto.Message)
			if !ok {
				common.Log.Errorf("type %T does not implement proto.Message", out)
				return nil
			}
			err := proto.Unmarshal(v, msg)
			if err != nil {
				common.Log.Errorf("type %T Unmarshal failed, %v", out, err)
				return nil
			}
		}

		if ret == nil {
			ret = make(map[string]*T)
		}

		ret[string(k)] = &out

		return nil
	})

	if err != nil {
		common.Log.Errorf("Cache.GetListFromDB-> err:%s", err.Error())
	}
	return
}

func (s *Cache[T]) IsExistFromDB(keyPrefix []byte, cb func(key []byte, value *T) bool) (ret bool) {
	err := s.dbWrite.Db.BatchRead(keyPrefix, false, func(k, v []byte) error {

		var out T
		msg, ok := any(&out).(proto.Message)
		if !ok {
			return fmt.Errorf("type %T does not implement proto.Message", out)
		}
		err := proto.Unmarshal(v, msg)
		if err != nil {
			return err
		}
		ret = cb(k, &out)
		if ret {
			return nil
		}

		return nil
	})

	if err != nil {
		common.Log.Errorf("Cache.GetListFromDB-> err:%s", err.Error())
		ret = false
	}
	return
}

func (s *Cache[T]) GetFromDB(key []byte) (ret *T, raw []byte) {

	v, err := s.dbWrite.Db.Read(key)
	if err != nil && err != common.ErrKeyNotFound {
		return
	}

	var out T
	if len(v) == 0 {
		return
	}
	msg, ok := any(&out).(proto.Message)
	if !ok {
		common.Log.Errorf("type %T does not implement proto.Message", out)
		return
	}
	err = proto.Unmarshal(v, msg)
	if err != nil {
		common.Log.Errorf("Unmarshal failed, %v", err)
		return
	}
	ret = &out
	raw = v

	return
}
