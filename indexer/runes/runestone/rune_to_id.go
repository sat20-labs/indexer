package runestone

import (
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type RuneToId map[*Rune]*RuneId

type RuneToIdTable struct {
	Table[pb.RuneId]
}

func NewRuneToIdTable(v *store.Cache[pb.RuneId]) *RuneToIdTable {
	return &RuneToIdTable{
		Table: Table[pb.RuneId]{cache: v},
	}
}

func (s *RuneToIdTable) Insert(key *Rune, value *RuneId) (ret *RuneId) {
	tblKey := []byte(store.RUNE_TO_ID + key.String())
	pbVal := s.cache.Insert(tblKey, value.ToPb())
	if pbVal != nil {
		ret = &RuneId{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneToIdTable) SetToDB(key *Rune, value *RuneId) {
	tblKey := []byte(store.RUNE_TO_ID + key.String())
	s.cache.SetToDB(tblKey, value.ToPb())
}

func (s *RuneToIdTable) Get(key *Rune) (ret *RuneId) {
	tblKey := []byte(store.RUNE_TO_ID + key.String())
	pbVal := s.cache.Get(tblKey)
	if pbVal != nil {
		ret = &RuneId{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneToIdTable) GetFromDB(key *Rune) (ret *RuneId) {
	tblKey := []byte(store.RUNE_TO_ID + key.String())
	pbVal, _ := s.cache.GetFromDB(tblKey)
	if pbVal != nil {
		ret = &RuneId{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneToIdTable) Flush() {
	s.cache.Flush()
}
