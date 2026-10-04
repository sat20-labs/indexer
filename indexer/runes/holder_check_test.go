package runes

import (
	"bytes"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/indexer/common"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"github.com/sat20-labs/indexer/indexer/runes/table"
	"lukechampine.com/uint128"
)

type holderScanDB struct {
	common.KVDB
	holderScans int
}

func (d *holderScanDB) Scan(options common.ScanOptions, visit func(k, v []byte) error) error {
	if bytes.Equal(options.Prefix, []byte(store.RUNEID_ADDRESS_TO_BALANCE)) {
		d.holderScans++
	}
	return d.KVDB.Scan(options, visit)
}

func newHolderCheckIndexer(t *testing.T) (*Indexer, *holderScanDB) {
	t.Helper()
	kv := indexdb.NewKVDBWithCache(t.TempDir(), 1)
	if kv == nil {
		t.Fatal("open holder test DB")
	}
	t.Cleanup(func() {
		if err := kv.Close(); err != nil {
			t.Error(err)
		}
	})
	db := &holderScanDB{KVDB: kv}
	return NewIndexer(db, &chaincfg.MainNetParams, false), db
}

func holderRow(rune *runestone.RuneId, address, amount uint64) *table.RuneIdAddressToBalance {
	return &table.RuneIdAddressToBalance{RuneId: rune, AddressId: address, Balance: runestone.Lot{Value: uint128.From64(amount)}}
}

func TestCollectHolderTotalsMergedState(t *testing.T) {
	idx, db := newHolderCheckIndexer(t)
	a := &runestone.RuneId{Block: 840000, Tx: 1}
	b := &runestone.RuneId{Block: 840000, Tx: 2}
	c := &runestone.RuneId{Block: 840000, Tx: 3}
	for _, row := range []*table.RuneIdAddressToBalance{
		holderRow(a, 1, 10), holderRow(a, 2, 20), holderRow(a, 3, 30), holderRow(b, 1, 5),
	} {
		idx.runeIdAddressToBalanceTbl.Insert(row)
	}
	idx.dbWrite.FlushToDB()
	idx.runeIdAddressToBalanceTbl.Insert(holderRow(a, 1, 25))
	idx.runeIdAddressToBalanceTbl.Remove(holderRow(a, 2, 20))
	idx.runeIdAddressToBalanceTbl.Insert(holderRow(a, 4, 40))
	idx.runeIdAddressToBalanceTbl.Insert(holderRow(c, 5, 99))
	idx.runeIdAddressToBalanceTbl.Remove(holderRow(c, 5, 99))
	store.NewCache[pb.RuneId](idx.dbWrite).Set([]byte("b-unrelated"), &pb.RuneId{Block: 1})
	totals, holders, err := idx.collectHolderTotals()
	if err != nil {
		t.Fatal(err)
	}
	if db.holderScans != 1 {
		t.Fatalf("holder scans=%d, want one traversal for all runes", db.holderScans)
	}
	if len(totals) != 2 || totals[a.String()].amount != uint128.From64(95) || totals[a.String()].count != 3 ||
		totals[b.String()].amount != uint128.From64(5) || totals[b.String()].count != 1 {
		t.Fatalf("merged holder totals: %#v", totals)
	}
	if len(holders) != 3 || !holders[1] || !holders[3] || !holders[4] {
		t.Fatalf("merged holders: %v", holders)
	}
	if got := idx.runeIdAddressToBalanceTbl.Get(holderRow(a, 1, 25)); got == nil || got.Balance.Value != uint128.From64(25) {
		t.Fatal("holder check changed the pending replacement")
	}
	if got := idx.runeIdAddressToBalanceTbl.Get(holderRow(a, 2, 20)); got != nil {
		t.Fatal("holder check lost the pending deletion")
	}
	durable := idx.runeIdAddressToBalanceTbl.Cache.GetListFromDB([]byte(store.RUNEID_ADDRESS_TO_BALANCE+a.Hex()+"-"), true)
	if len(durable) != 3 || durable[store.RUNEID_ADDRESS_TO_BALANCE+holderRow(a, 1, 10).Key()].Balance.Value.Lo != 10 {
		t.Fatal("holder check changed durable balances")
	}
}

func TestCollectHolderTotalsEmpty(t *testing.T) {
	idx, db := newHolderCheckIndexer(t)
	totals, holders, err := idx.collectHolderTotals()
	if err != nil || len(totals) != 0 || len(holders) != 0 || db.holderScans != 1 {
		t.Fatalf("empty totals=%v holders=%v scans=%d err=%v", totals, holders, db.holderScans, err)
	}
}

func TestCollectHolderTotalsRejectsOverflow(t *testing.T) {
	idx, _ := newHolderCheckIndexer(t)
	rune := &runestone.RuneId{Block: 840000, Tx: 1}
	row := holderRow(rune, 1, 0)
	row.Balance.Value = uint128.Max
	idx.runeIdAddressToBalanceTbl.Insert(row)
	idx.runeIdAddressToBalanceTbl.Insert(holderRow(rune, 2, 1))
	if _, _, err := idx.collectHolderTotals(); err == nil {
		t.Fatal("overflow must fail instead of wrapping the aggregate balance")
	}
}
