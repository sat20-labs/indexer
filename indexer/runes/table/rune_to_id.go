package table

import (
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type RuneToId map[*runestone.Rune]*runestone.RuneId

type RuneToIdTable struct {
	Table[pb.RuneId]
}

func NewRuneToIdTable(v *store.Cache[pb.RuneId]) *RuneToIdTable {
	return &RuneToIdTable{
		Table: Table[pb.RuneId]{Cache: v},
	}
}

func (s *RuneToIdTable) Insert(key *runestone.Rune, value *runestone.RuneId) (ret *runestone.RuneId) {
	tblKey := []byte(store.RUNE_TO_ID + key.String())
	pbVal := s.Cache.Set(tblKey, value.ToPb())
	if pbVal != nil {
		ret = &runestone.RuneId{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneToIdTable) SetToDB(key *runestone.Rune, value *runestone.RuneId) {
	tblKey := []byte(store.RUNE_TO_ID + key.String())
	s.Cache.SetToDB(tblKey, value.ToPb())
}

func (s *RuneToIdTable) Get(key *runestone.Rune) (ret *runestone.RuneId) {
	tblKey := []byte(store.RUNE_TO_ID + key.String())
	pbVal := s.Cache.Get(tblKey)
	if pbVal != nil {
		ret = &runestone.RuneId{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneToIdTable) GetFromDB(key *runestone.Rune) (ret *runestone.RuneId) {
	tblKey := []byte(store.RUNE_TO_ID + key.String())
	pbVal, _ := s.Cache.GetFromDB(tblKey)
	if pbVal != nil {
		ret = &runestone.RuneId{}
		ret.FromPb(pbVal)
	}
	return
}
