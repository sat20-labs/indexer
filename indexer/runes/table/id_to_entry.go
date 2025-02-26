package table

import (
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type RuneIdToEntry map[*runestone.RuneId]*runestone.RuneEntry

type RuneIdToEntryTable struct {
	Table[pb.RuneEntry]
}

func NewRuneIdToEntryTable(store *store.Cache[pb.RuneEntry]) *RuneIdToEntryTable {
	return &RuneIdToEntryTable{Table: Table[pb.RuneEntry]{Cache: store}}
}

func (s *RuneIdToEntryTable) Get(key *runestone.RuneId) (ret *runestone.RuneEntry) {
	tblKey := []byte(store.ID_TO_ENTRY + key.Hex())
	pbVal := s.Cache.Get(tblKey)
	if pbVal != nil {
		ret = &runestone.RuneEntry{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneIdToEntryTable) GetFromDB(key *runestone.RuneId) (ret *runestone.RuneEntry) {
	tblKey := []byte(store.ID_TO_ENTRY + key.Hex())
	pbVal, _ := s.Cache.GetFromDB(tblKey)
	if pbVal != nil {
		ret = &runestone.RuneEntry{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneIdToEntryTable) GetList() (ret map[string]*runestone.RuneEntry) {
	prefixKey := []byte(store.ID_TO_ENTRY)
	list := s.Cache.GetList(prefixKey, true)
	if len(list) == 0 {
		return
	}
	ret = make(map[string]*runestone.RuneEntry)
	for k, v := range list {
		key := k[len(prefixKey):]
		ret[key] = &runestone.RuneEntry{}
		ret[key].FromPb(v)
	}
	return ret
}

func (s *RuneIdToEntryTable) Insert(key *runestone.RuneId, value *runestone.RuneEntry) (ret *runestone.RuneEntry) {
	tblKey := []byte(store.ID_TO_ENTRY + key.Hex())
	pbVal := s.Cache.Set(tblKey, value.ToPb())
	if pbVal != nil {
		ret = &runestone.RuneEntry{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneIdToEntryTable) Remove(key *runestone.RuneId) (ret *runestone.RuneEntry) {
	tblKey := []byte(store.ID_TO_ENTRY + key.Hex())
	pbVal := s.Cache.Delete(tblKey)
	if pbVal != nil {
		ret = &runestone.RuneEntry{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneIdToEntryTable) SetToDB(key *runestone.RuneId, value *runestone.RuneEntry) {
	tblKey := []byte(store.ID_TO_ENTRY + key.Hex())
	s.Cache.SetToDB(tblKey, value.ToPb())
}
