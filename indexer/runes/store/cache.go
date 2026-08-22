package store

import (
	"container/list"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/sat20-labs/indexer/common"
	"google.golang.org/protobuf/proto"
)

type ActionType int

const (
	PUT ActionType = 1
	DEL ActionType = 2
)

const defaultRunesReadCacheMB = 64

var counter int64

type DbLog struct {
	Val       []byte
	Type      ActionType
	ExistInDb bool
	TimeStamp int64
}

type readCacheValue struct {
	key   string
	value []byte
	size  int64
}

type boundedReadCache struct {
	mu       sync.Mutex
	maxBytes int64
	used     int64
	items    map[string]*list.Element
	order    *list.List
}

func configuredReadCacheBytes() int64 {
	cacheMB := defaultRunesReadCacheMB
	if raw := os.Getenv("INDEXER_RUNES_READ_CACHE_MB"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			common.Log.Warnf("invalid INDEXER_RUNES_READ_CACHE_MB=%q, use %dMB", raw, cacheMB)
		} else {
			cacheMB = value
		}
	}
	return int64(cacheMB) << 20
}

func newBoundedReadCache(maxBytes int64) *boundedReadCache {
	return &boundedReadCache{
		maxBytes: maxBytes,
		items:    make(map[string]*list.Element),
		order:    list.New(),
	}
}

func (c *boundedReadCache) Get(key string) ([]byte, bool) {
	if c == nil || c.maxBytes <= 0 {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	element := c.items[key]
	if element == nil {
		return nil, false
	}
	c.order.MoveToFront(element)
	return element.Value.(*readCacheValue).value, true
}

func (c *boundedReadCache) Set(key string, value []byte) {
	if c == nil || c.maxBytes <= 0 || len(value) == 0 {
		return
	}
	owned := append([]byte(nil), value...)
	size := int64(len(key) + len(owned))
	if size > c.maxBytes {
		c.Delete(key)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if element := c.items[key]; element != nil {
		old := element.Value.(*readCacheValue)
		c.used -= old.size
		old.value = owned
		old.size = size
		c.used += size
		c.order.MoveToFront(element)
	} else {
		entry := &readCacheValue{key: key, value: owned, size: size}
		c.items[key] = c.order.PushFront(entry)
		c.used += size
	}
	for c.used > c.maxBytes {
		element := c.order.Back()
		if element == nil {
			break
		}
		entry := element.Value.(*readCacheValue)
		delete(c.items, entry.key)
		c.used -= entry.size
		c.order.Remove(element)
	}
}

func (c *boundedReadCache) Delete(key string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if element := c.items[key]; element != nil {
		entry := element.Value.(*readCacheValue)
		delete(c.items, key)
		c.used -= entry.size
		c.order.Remove(element)
	}
}

func (c *boundedReadCache) Len() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.items)
}

func (c *boundedReadCache) UsedBytes() int64 {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.used
}

type DbWrite struct {
	Db             common.KVDB
	pending        *cmap.ConcurrentMap[string, *DbLog]
	readCache      *boundedReadCache
	cloneTimeStamp int64
}

func NewDbWrite(database common.KVDB) *DbWrite {
	pending := cmap.New[*DbLog]()
	return &DbWrite{
		Db:        database,
		pending:   &pending,
		readCache: newBoundedReadCache(configuredReadCacheBytes()),
	}
}

func (s *DbWrite) ClearLogs() {
	s.pending.Clear()
}

func cloneLog(log *DbLog) *DbLog {
	if log == nil {
		return nil
	}
	return &DbLog{
		Val:       append([]byte(nil), log.Val...),
		Type:      log.Type,
		ExistInDb: log.ExistInDb,
		TimeStamp: log.TimeStamp,
	}
}

func (s *DbWrite) FlushToDB() {
	count := s.pending.Count()
	updates := 0
	removes := 0
	var totalBytes int64
	if count == 0 {
		common.Log.Infof("DbWrite.FlushToDB-> pending count:0, update count:0, remove count:0, total bytes:0")
		return
	}

	wb := s.Db.NewWriteBatch()
	defer wb.Close()
	flushed := make([]struct {
		key string
		log *DbLog
	}, 0, count)
	for item := range s.pending.IterBuffered() {
		log := cloneLog(item.Val)
		totalBytes += int64(len(item.Key) + len(log.Val))
		switch log.Type {
		case PUT:
			if err := wb.Put([]byte(item.Key), log.Val); err != nil {
				common.Log.Panicf("DbWrite.FlushToDB put %s: %v", item.Key, err)
			}
			updates++
		case DEL:
			if log.ExistInDb {
				if err := wb.Delete([]byte(item.Key)); err != nil {
					common.Log.Panicf("DbWrite.FlushToDB delete %s: %v", item.Key, err)
				}
				removes++
			}
		}
		flushed = append(flushed, struct {
			key string
			log *DbLog
		}{item.Key, log})
	}
	if err := wb.Flush(); err != nil {
		common.Log.Panicf("DbWrite.FlushToDB flush: %v", err)
	}

	for _, item := range flushed {
		if item.log.Type == PUT {
			s.readCache.Set(item.key, item.log.Val)
		} else {
			s.readCache.Delete(item.key)
		}
	}
	s.ClearLogs()
	common.Log.Infof(
		"DbWrite.FlushToDB-> pending count:%d, update count:%d, remove count:%d, total bytes:%d, read cache entries:%d, read cache bytes:%d",
		count, updates, removes, totalBytes, s.readCache.Len(), s.readCache.UsedBytes(),
	)
}

func (s *DbWrite) Clone(clone *DbWrite) *DbWrite {
	for item := range s.pending.IterBuffered() {
		clone.pending.Set(item.Key, cloneLog(item.Val))
	}
	clone.cloneTimeStamp = atomic.AddInt64(&counter, 1)
	return clone
}

// Subtract removes entries captured by this snapshot from the live DbWrite.
// A newer timestamp remains pending; after the snapshot flushes, its next write
// should be treated as updating an existing database key.
func (s *DbWrite) Subtract(live *DbWrite) {
	for item := range s.pending.IterBuffered() {
		current, ok := live.pending.Get(item.Key)
		if !ok {
			common.Log.Panicf("pending log %s not found", item.Key)
		}
		if current.TimeStamp == item.Val.TimeStamp {
			live.pending.Remove(item.Key)
			continue
		}
		updated := cloneLog(current)
		updated.ExistInDb = true
		live.pending.Set(item.Key, updated)
	}
}

type Cache[T any] struct {
	dbWrite *DbWrite
}

func NewCache[T any](dbWrite *DbWrite) *Cache[T] {
	return &Cache[T]{dbWrite: dbWrite}
}

func decodeProto[T any](raw []byte) *T {
	if len(raw) == 0 {
		return nil
	}
	var out T
	msg, ok := any(&out).(proto.Message)
	if !ok {
		common.Log.Panicf("type %T does not implement proto.Message", out)
	}
	if err := proto.Unmarshal(raw, msg); err != nil {
		common.Log.Panicf("proto unmarshal %T: %v", out, err)
	}
	return &out
}

func (s *Cache[T]) Get(key []byte) *T {
	keyStr := string(key)
	if log, ok := s.dbWrite.pending.Get(keyStr); ok {
		if log.Type == DEL {
			return nil
		}
		return decodeProto[T](log.Val)
	}
	if raw, ok := s.dbWrite.readCache.Get(keyStr); ok {
		return decodeProto[T](raw)
	}
	value, err := s.dbWrite.Db.Read(key)
	if err != nil {
		return nil
	}
	s.dbWrite.readCache.Set(keyStr, value)
	return decodeProto[T](value)
}

func (s *Cache[T]) Delete(key []byte) *T {
	keyStr := string(key)
	previous := s.Get(key)
	if previous == nil {
		return nil
	}

	if existing, ok := s.dbWrite.pending.Get(keyStr); ok {
		if existing.Type == PUT && !existing.ExistInDb {
			s.dbWrite.pending.Remove(keyStr)
			s.dbWrite.readCache.Delete(keyStr)
			return previous
		}
		s.dbWrite.pending.Set(keyStr, &DbLog{
			Type:      DEL,
			ExistInDb: existing.ExistInDb,
			TimeStamp: atomic.AddInt64(&counter, 1),
		})
	} else {
		s.dbWrite.pending.Set(keyStr, &DbLog{
			Type:      DEL,
			ExistInDb: true,
			TimeStamp: atomic.AddInt64(&counter, 1),
		})
	}
	s.dbWrite.readCache.Delete(keyStr)
	return previous
}

func (s *Cache[T]) Set(key []byte, msg proto.Message) *T {
	keyStr := string(key)
	previous := s.Get(key)
	value, err := proto.Marshal(msg)
	if err != nil {
		common.Log.Panicf("Cache.Set marshal %s: %v", keyStr, err)
	}
	existsInDB := previous != nil
	if existing, ok := s.dbWrite.pending.Get(keyStr); ok {
		existsInDB = existing.ExistInDb
	}
	s.dbWrite.pending.Set(keyStr, &DbLog{
		Val:       value,
		Type:      PUT,
		ExistInDb: existsInDB,
		TimeStamp: atomic.AddInt64(&counter, 1),
	})
	return previous
}

func (s *Cache[T]) SetToDB(key []byte, value proto.Message) {
	raw, err := proto.Marshal(value)
	if err != nil {
		common.Log.Panicf("Cache.SetToDB marshal %s: %v", key, err)
	}
	if err := s.dbWrite.Db.Write(key, raw); err != nil {
		common.Log.Panicf("Cache.SetToDB write %s: %v", key, err)
	}
	s.dbWrite.readCache.Set(string(key), raw)
}

func (s *Cache[T]) pendingForPrefix(prefix []byte) (puts map[string]*DbLog, deletes map[string]bool) {
	puts = make(map[string]*DbLog)
	deletes = make(map[string]bool)
	prefixStr := string(prefix)
	for item := range s.dbWrite.pending.IterBuffered() {
		if !strings.HasPrefix(item.Key, prefixStr) {
			continue
		}
		if item.Val.Type == DEL {
			deletes[item.Key] = true
		} else {
			puts[item.Key] = cloneLog(item.Val)
		}
	}
	return puts, deletes
}

func (s *Cache[T]) IsExist(keyPrefix []byte, cb func(key []byte, value *T) bool) bool {
	puts, deletes := s.pendingForPrefix(keyPrefix)
	for key, log := range puts {
		if cb([]byte(key), decodeProto[T](log.Val)) {
			return true
		}
	}
	found := false
	if err := s.dbWrite.Db.Scan(common.ScanOptions{Prefix: keyPrefix}, func(key, value []byte) error {
		keyStr := string(key)
		if deletes[keyStr] || puts[keyStr] != nil {
			return nil
		}
		if cb(key, decodeProto[T](value)) {
			found = true
			return common.ErrStopScan
		}
		return nil
	}); err != nil {
		common.Log.Panicf("Cache.IsExist scan %s: %v", keyPrefix, err)
	}
	return found
}

func (s *Cache[T]) GetList(keyPrefix []byte, needValue bool) map[string]*T {
	result := make(map[string]*T)
	if err := s.dbWrite.Db.Scan(common.ScanOptions{Prefix: keyPrefix, KeysOnly: !needValue}, func(key, value []byte) error {
		var decoded *T
		if needValue {
			decoded = decodeProto[T](value)
		} else {
			decoded = new(T)
		}
		result[string(key)] = decoded
		return nil
	}); err != nil {
		common.Log.Panicf("Cache.GetList scan %s: %v", keyPrefix, err)
	}
	puts, deletes := s.pendingForPrefix(keyPrefix)
	for key := range deletes {
		delete(result, key)
	}
	for key, log := range puts {
		if needValue {
			result[key] = decodeProto[T](log.Val)
		} else {
			result[key] = new(T)
		}
	}
	return result
}

func (s *Cache[T]) GetListFromDB(keyPrefix []byte, needValue bool) map[string]*T {
	result := make(map[string]*T)
	if err := s.dbWrite.Db.Scan(common.ScanOptions{Prefix: keyPrefix, KeysOnly: !needValue}, func(key, value []byte) error {
		if needValue {
			result[string(key)] = decodeProto[T](value)
		} else {
			result[string(key)] = new(T)
		}
		return nil
	}); err != nil {
		common.Log.Errorf("Cache.GetListFromDB scan %s: %v", keyPrefix, err)
	}
	return result
}

func (s *Cache[T]) IsExistFromDB(keyPrefix []byte, cb func(key []byte, value *T) bool) bool {
	found := false
	if err := s.dbWrite.Db.Scan(common.ScanOptions{Prefix: keyPrefix}, func(key, value []byte) error {
		if cb(key, decodeProto[T](value)) {
			found = true
			return common.ErrStopScan
		}
		return nil
	}); err != nil && !errors.Is(err, common.ErrStopScan) {
		common.Log.Errorf("Cache.IsExistFromDB scan %s: %v", keyPrefix, err)
	}
	return found
}

func (s *Cache[T]) GetFromDB(key []byte) (ret *T, raw []byte) {
	value, err := s.dbWrite.Db.Read(key)
	if err != nil {
		return nil, nil
	}
	return decodeProto[T](value), value
}
