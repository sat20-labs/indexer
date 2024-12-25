package runestone

import (
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type TransactionIdToRuneTable struct {
	Table[pb.Rune]
}

func NewTransactionIdToRuneTable(store *store.Store[pb.Rune]) *TransactionIdToRuneTable {
	return &TransactionIdToRuneTable{Table: Table[pb.Rune]{store: store}}
}

func (s *TransactionIdToRuneTable) Get(key string) (ret *Rune) {
	tblKey := []byte(store.RUNE_TO_ID + key)
	pbVal := s.store.Get(tblKey)
	if pbVal != nil {
		ret = &Rune{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *TransactionIdToRuneTable) GetNoTransaction(key string) (ret *Rune) {
	tblKey := []byte(store.RUNE_TO_ID + key)
	pbVal := s.store.GetNoTransaction(tblKey)
	if pbVal != nil {
		ret = &Rune{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *TransactionIdToRuneTable) Insert(key string, value *Rune) (ret *Rune) {
	tblKey := []byte(store.RUNE_TO_ID + key)
	pbVal := s.store.Insert(tblKey, value.ToPb())
	if pbVal != nil {
		ret = &Rune{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *TransactionIdToRuneTable) InsertNoTransaction(key string, value *Rune) {
	tblKey := []byte(store.RUNE_TO_ID + key)
	s.store.InsertNoTransaction(tblKey, value.ToPb())
}

func (s *TransactionIdToRuneTable) Flush() {
	s.store.Flush()
}
