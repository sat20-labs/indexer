package store

import (
	"testing"

	"github.com/sat20-labs/indexer/common"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"google.golang.org/protobuf/proto"
)

func openRunesStoreTestDB(t *testing.T) common.KVDB {
	t.Helper()
	kv := indexdb.NewKVDBWithCache(t.TempDir(), 1)
	if kv == nil {
		t.Fatal("open test database")
	}
	t.Cleanup(func() {
		if err := kv.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})
	return kv
}

func writeRuneID(t *testing.T, kv common.KVDB, key string, block, tx uint64) {
	t.Helper()
	raw, err := proto.Marshal(&pb.RuneId{Block: block, Tx: uint32(tx)})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := kv.Write([]byte(key), raw); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestBoundedReadCacheEvictsByBytes(t *testing.T) {
	cache := newBoundedReadCache(10)
	cache.Set("a", []byte("1234")) // 5 bytes including key
	cache.Set("b", []byte("5678")) // 10 bytes total
	if cache.Len() != 2 {
		t.Fatalf("cache len=%d, want 2", cache.Len())
	}
	cache.Set("c", []byte("9012"))
	if cache.Len() != 2 || cache.UsedBytes() > 10 {
		t.Fatalf("cache len=%d bytes=%d", cache.Len(), cache.UsedBytes())
	}
	if _, ok := cache.Get("a"); ok {
		t.Fatal("least-recently-used entry was not evicted")
	}
}

func TestDBReadCacheIsNotClonedIntoPendingWrites(t *testing.T) {
	kv := openRunesStoreTestDB(t)
	writeRuneID(t, kv, "rune-1", 1, 2)

	owner := NewDbWrite(kv)
	cache := NewCache[pb.RuneId](owner)
	got := cache.Get([]byte("rune-1"))
	if got == nil || got.Block != 1 || got.Tx != 2 {
		t.Fatalf("read value=%v", got)
	}
	if owner.readCache.Len() != 1 || owner.pending.Count() != 0 {
		t.Fatalf("read cache=%d pending=%d", owner.readCache.Len(), owner.pending.Count())
	}

	clone := NewDbWrite(kv)
	owner.Clone(clone)
	if clone.pending.Count() != 0 || clone.readCache.Len() != 0 {
		t.Fatalf("clone copied read state: pending=%d cache=%d", clone.pending.Count(), clone.readCache.Len())
	}
}

func TestPendingWriteCloneFlushAndSubtract(t *testing.T) {
	kv := openRunesStoreTestDB(t)
	live := NewDbWrite(kv)
	cache := NewCache[pb.RuneId](live)
	cache.Set([]byte("rune-1"), &pb.RuneId{Block: 1, Tx: 2})
	if live.pending.Count() != 1 {
		t.Fatalf("pending=%d, want 1", live.pending.Count())
	}

	snapshot := NewDbWrite(kv)
	live.Clone(snapshot)
	cache.Set([]byte("rune-1"), &pb.RuneId{Block: 3, Tx: 4})
	// Production removes the captured snapshot from live state before the
	// snapshot is flushed. This preserves any newer write and marks it as an
	// update of the soon-to-exist DB key.
	snapshot.Subtract(live)
	snapshot.FlushToDB()
	if live.pending.Count() != 1 {
		t.Fatalf("newer pending write was removed")
	}
	pending, ok := live.pending.Get("rune-1")
	if !ok || !pending.ExistInDb {
		t.Fatalf("newer pending write did not learn that the snapshot exists in DB: %#v", pending)
	}

	live.FlushToDB()
	if live.pending.Count() != 0 {
		t.Fatalf("pending not cleared after flush")
	}
	got := cache.Get([]byte("rune-1"))
	if got == nil || got.Block != 3 || got.Tx != 4 {
		t.Fatalf("final value=%v", got)
	}
}

func TestDeleteOfNewPendingWriteIsNetNoop(t *testing.T) {
	kv := openRunesStoreTestDB(t)
	owner := NewDbWrite(kv)
	cache := NewCache[pb.RuneId](owner)
	cache.Set([]byte("new"), &pb.RuneId{Block: 1, Tx: 1})
	if previous := cache.Delete([]byte("new")); previous == nil {
		t.Fatal("delete did not return pending value")
	}
	if owner.pending.Count() != 0 {
		t.Fatalf("new put followed by delete left pending state")
	}
	if got := cache.Get([]byte("new")); got != nil {
		t.Fatalf("deleted pending value still visible: %v", got)
	}
}

func TestGetListOverlaysPendingPutAndDelete(t *testing.T) {
	kv := openRunesStoreTestDB(t)
	writeRuneID(t, kv, "p-1", 1, 1)
	writeRuneID(t, kv, "p-2", 2, 2)
	owner := NewDbWrite(kv)
	cache := NewCache[pb.RuneId](owner)

	cache.Delete([]byte("p-1"))
	cache.Set([]byte("p-3"), &pb.RuneId{Block: 3, Tx: 3})
	items := cache.GetList([]byte("p-"), true)
	if len(items) != 2 || items["p-1"] != nil || items["p-2"] == nil || items["p-3"] == nil {
		t.Fatalf("overlay result=%v", items)
	}
	if items["p-3"].Block != 3 {
		t.Fatalf("pending put not decoded: %v", items["p-3"])
	}
}
