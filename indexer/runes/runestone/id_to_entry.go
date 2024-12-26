package runestone

import (
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type RuneIdToEntry map[*RuneId]*RuneEntry

type RuneIdToEntryTable struct {
	Table[pb.RuneEntry]
}

func NewRuneIdToEntryTable(store *store.Cache[pb.RuneEntry]) *RuneIdToEntryTable {
	return &RuneIdToEntryTable{Table: Table[pb.RuneEntry]{cache: store}}
}

func (s *RuneIdToEntryTable) Get(key *RuneId) (ret *RuneEntry) {
	tblKey := []byte(store.ID_TO_ENTRY + key.String())
	pbVal := s.cache.Get(tblKey)
	if pbVal != nil {
		ret = &RuneEntry{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneIdToEntryTable) GetFromDB(key *RuneId) (ret *RuneEntry) {
	tblKey := []byte(store.ID_TO_ENTRY + key.String())
	pbVal, _ := s.cache.GetFromDB(tblKey)
	if pbVal != nil {
		ret = &RuneEntry{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneIdToEntryTable) Insert(key *RuneId, value *RuneEntry) (ret *RuneEntry) {
	tblKey := []byte(store.ID_TO_ENTRY + key.String())
	pbVal := s.cache.Insert(tblKey, value.ToPb())
	if pbVal != nil {
		ret = &RuneEntry{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneIdToEntryTable) SetToDB(key *RuneId, value *RuneEntry) {
	tblKey := []byte(store.ID_TO_ENTRY + key.String())
	s.cache.SetToDB(tblKey, value.ToPb())
}

func (s *RuneIdToEntryTable) Flush() {
	s.cache.Flush()
}
