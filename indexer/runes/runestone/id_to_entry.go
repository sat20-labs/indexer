package runestone

import (
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type RuneIdToEntry map[*RuneId]*RuneEntry

type RuneIdToEntryTable struct {
	Table[pb.RuneEntry]
}

func NewRuneIdToEntryTable(store *store.Store[pb.RuneEntry]) *RuneIdToEntryTable {
	return &RuneIdToEntryTable{Table: Table[pb.RuneEntry]{store: store}}
}

func (s *RuneIdToEntryTable) Get(key *RuneId) (ret *RuneEntry) {
	tblKey := []byte(store.ID_TO_ENTRY + key.String())
	pbVal := s.store.Get(tblKey)
	if pbVal != nil {
		ret = &RuneEntry{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneIdToEntryTable) GetNoTransaction(key *RuneId) (ret *RuneEntry) {
	tblKey := []byte(store.ID_TO_ENTRY + key.String())
	pbVal := s.store.GetNoTransaction(tblKey)
	if pbVal != nil {
		ret = &RuneEntry{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneIdToEntryTable) Insert(key *RuneId, value *RuneEntry) (ret *RuneEntry) {
	tblKey := []byte(store.ID_TO_ENTRY + key.String())
	pbVal := s.store.Insert(tblKey, value.ToPb())
	if pbVal != nil {
		ret = &RuneEntry{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneIdToEntryTable) InsertNoTransaction(key *RuneId, value *RuneEntry) {
	tblKey := []byte(store.ID_TO_ENTRY + key.String())
	s.store.InsertNoTransaction(tblKey, value.ToPb())
}

func (s *RuneIdToEntryTable) Flush() {
	s.store.Flush()
}
