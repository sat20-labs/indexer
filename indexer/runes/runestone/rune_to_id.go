package runestone

import (
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type RuneToId map[*Rune]*RuneId

type RuneToIdTable struct {
	Table[pb.RuneId]
}

func NewRuneToIdTable(store *store.Store[pb.RuneId]) *RuneToIdTable {
	return &RuneToIdTable{
		Table: Table[pb.RuneId]{store: store},
	}
}

func (s *RuneToIdTable) Insert(key *Rune, value *RuneId) (ret *RuneId) {
	tblKey := []byte(store.RUNE_TO_ID + key.String())
	pbVal := s.store.Insert(tblKey, value.ToPb())
	if pbVal != nil {
		ret = &RuneId{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneToIdTable) InsertNoTransaction(key *Rune, value *RuneId) {
	tblKey := []byte(store.RUNE_TO_ID + key.String())
	s.store.InsertNoTransaction(tblKey, value.ToPb())
}

func (s *RuneToIdTable) Get(key *Rune) (ret *RuneId) {
	tblKey := []byte(store.RUNE_TO_ID + key.String())
	pbVal := s.store.Get(tblKey)
	if pbVal != nil {
		ret = &RuneId{}
		ret.FromPb(pbVal)
	}
	return
}
