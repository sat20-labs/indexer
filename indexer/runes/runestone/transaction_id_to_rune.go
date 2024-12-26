package runestone

import (
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type TransactionIdToRuneTable struct {
	Table[pb.Rune]
}

func NewTransactionIdToRuneTable(store *store.Cache[pb.Rune]) *TransactionIdToRuneTable {
	return &TransactionIdToRuneTable{Table: Table[pb.Rune]{cache: store}}
}

func (s *TransactionIdToRuneTable) Get(key string) (ret *Rune) {
	tblKey := []byte(store.RUNE_TO_ID + key)
	pbVal := s.cache.Get(tblKey)
	if pbVal != nil {
		ret = &Rune{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *TransactionIdToRuneTable) GetFromDB(key string) (ret *Rune) {
	tblKey := []byte(store.RUNE_TO_ID + key)
	pbVal, _ := s.cache.GetFromDB(tblKey)
	if pbVal != nil {
		ret = &Rune{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *TransactionIdToRuneTable) Insert(key string, value *Rune) (ret *Rune) {
	tblKey := []byte(store.RUNE_TO_ID + key)
	pbVal := s.cache.Insert(tblKey, value.ToPb())
	if pbVal != nil {
		ret = &Rune{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *TransactionIdToRuneTable) SetToDB(key string, value *Rune) {
	tblKey := []byte(store.RUNE_TO_ID + key)
	s.cache.SetToDB(tblKey, value.ToPb())
}

func (s *TransactionIdToRuneTable) Flush() {
	s.cache.Flush()
}
