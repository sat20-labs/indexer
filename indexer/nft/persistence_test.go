package nft

import (
	"reflect"
	"testing"

	"github.com/sat20-labs/indexer/common"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
)

func openNftPersistenceTestDB(t *testing.T) common.KVDB {
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

func TestSatOffsetsForPersistenceFiltersZeroAndSorts(t *testing.T) {
	got := satOffsetsForPersistence(99, map[int64]int64{
		30: 3,
		0:  9,
		10: 1,
		20: 2,
	})
	want := []*SatOffset{
		{Sat: 10, Offset: 1},
		{Sat: 20, Offset: 2},
		{Sat: 30, Offset: 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sat offsets = %#v, want %#v", got, want)
	}
	for _, item := range got {
		if item == nil || item.Sat == 0 {
			t.Fatalf("invalid persisted item: %#v", item)
		}
	}
}

func seedNftBase(t *testing.T, kv common.KVDB, id, sat int64) {
	t.Helper()
	wb := kv.NewWriteBatch()
	defer wb.Close()
	value := &common.InscribeBaseContent{
		Id:            id,
		Sat:           sat,
		InscriptionId: "inscription",
		ContentType:   []byte("not-an-id"),
	}
	if err := indexdb.SetDBWithProto3([]byte(GetNftKey(id)), value, wb); err != nil {
		t.Fatalf("write NFT %d: %v", id, err)
	}
	if err := wb.Flush(); err != nil {
		t.Fatalf("flush NFT %d: %v", id, err)
	}
}

func TestGetNftWithIDUsesPrimaryRecordWithoutBuckStore(t *testing.T) {
	kv := openNftPersistenceTestDB(t)
	seedNftBase(t, kv, 1, 123)

	wb := kv.NewWriteBatch()
	location := &common.NftsInSat{
		Sat:            123,
		OwnerAddressId: 7,
		UtxoId:         99,
		Offset:         5,
		Nfts:           []int64{1},
	}
	if err := indexdb.SetDBWithProto3([]byte(GetSatKey(123)), location, wb); err != nil {
		wb.Close()
		t.Fatalf("write NFT location: %v", err)
	}
	if err := wb.Flush(); err != nil {
		wb.Close()
		t.Fatalf("flush NFT location: %v", err)
	}
	wb.Close()

	indexer := NewNftIndexer(kv)
	indexer.nftIdToinscriptionMap = make(map[int64]*common.Nft)
	indexer.contentTypeMap = make(map[int]string)
	got := indexer.getNftWithId(1)
	if got == nil || got.Base == nil {
		t.Fatal("NFT was not loaded from the primary table")
	}
	if got.Base.Sat != 123 || got.OwnerAddressId != 7 || got.UtxoId != 99 || got.Offset != 5 {
		t.Fatalf("NFT = %#v", got)
	}
	if _, err := kv.Read([]byte(DB_PREFIX_BUCK + "lk")); err != common.ErrKeyNotFound {
		t.Fatalf("BuckStore was unexpectedly required, read error=%v", err)
	}
}

func TestGetNftsScansPrimaryKeysAndOverlaysPending(t *testing.T) {
	kv := openNftPersistenceTestDB(t)
	seedNftBase(t, kv, 0, 100)
	seedNftBase(t, kv, 2, 102)
	seedNftBase(t, kv, 3, 103)

	indexer := NewNftIndexer(kv)
	indexer.status = &common.NftStatus{Count: 4}
	indexer.nftAdded = []*common.Nft{{Base: &common.InscribeBaseContent{Id: 1, Sat: 101}}}

	got, total := indexer.GetNfts(0, 4)
	if total != 4 {
		t.Fatalf("total=%d, want 4", total)
	}
	if want := []int64{0, 1, 2, 3}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ids=%v, want %v", got, want)
	}

	got, total = indexer.GetNfts(2, 2)
	if total != 4 || !reflect.DeepEqual(got, []int64{2, 3}) {
		t.Fatalf("page ids=%v total=%d", got, total)
	}
}

func TestParseNftKeyRoundTrip(t *testing.T) {
	for _, id := range []int64{0, 1, 123456, -1, -99} {
		got, err := ParseNftKey(GetNftKey(id))
		if err != nil || got != id {
			t.Fatalf("id=%d got=%d err=%v", id, got, err)
		}
	}
}

func TestGetNftsIncludesCursedIDs(t *testing.T) {
	kv := openNftPersistenceTestDB(t)
	for _, id := range []int64{-4, -3, -1, 0, 1, 2} {
		seedNftBase(t, kv, id, 100+id)
	}
	indexer := NewNftIndexer(kv)
	indexer.status = &common.NftStatus{Count: 3, CurseCount: 4}
	indexer.nftAdded = []*common.Nft{{Base: &common.InscribeBaseContent{Id: -2, Sat: 98}}}
	for _, tc := range []struct {
		name         string
		start, limit int
		want         []int64
	}{
		{"negative", -3, 3, []int64{-3, -2, -1}},
		{"cross-zero", -2, 4, []int64{-2, -1, 0, 1}},
		{"negative-page", -4, 2, []int64{-4, -3}},
		{"below-cursed-range", -6, 4, []int64{-4, -3}},
		{"before-cursed-range", -6, 2, nil},
		{"positive", 1, 4, []int64{1, 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, total := indexer.GetNfts(tc.start, tc.limit)
			if total != 3 || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ids=%v total=%d, want ids=%v total=3", got, total, tc.want)
			}
		})
	}
	indexer.status.Count = 0
	if got, total := indexer.GetNfts(-3, 3); total != 0 || !reflect.DeepEqual(got, []int64{-3, -2, -1}) {
		t.Fatalf("cursed-only ids=%v total=%d", got, total)
	}
}
