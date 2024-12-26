package runestone

import (
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"lukechampine.com/uint128"
)

type RuneHolder struct {
	Address Address
	Balance uint128.Uint128
}

func (s *RuneHolder) ToPb() (ret *pb.RuneHolder) {
	ret = &pb.RuneHolder{
		Address: string(s.Address),
		Balance: &pb.Uint128{Lo: s.Balance.Lo, Hi: s.Balance.Hi},
	}
	return ret
}

func (s *RuneHolder) FromPb(pbValue *pb.RuneHolder) {
	s.Address = Address(pbValue.Address)
	s.Balance = uint128.Uint128{Lo: pbValue.Balance.Lo, Hi: pbValue.Balance.Hi}
}

type RuneHolders []*RuneHolder

func (s *RuneHolders) ToPb() (ret *pb.RuneHolders) {
	ret = &pb.RuneHolders{
		Holders: make([]*pb.RuneHolder, len(*s)),
	}
	for i, runeHolder := range *s {
		ret.Holders[i] = runeHolder.ToPb()
	}
	return ret
}

func (s *RuneHolders) FromPb(pbValues *pb.RuneHolders) {
	for _, holder := range pbValues.Holders {
		runeHolder := &RuneHolder{}
		runeHolder.FromPb(holder)
		*s = append(*s, runeHolder)
	}
}

type RuneHoldersTable struct {
	Table[pb.RuneHolders]
}

func NewRuneHoldersTable(store *store.Cache[pb.RuneHolders]) *RuneHoldersTable {
	return &RuneHoldersTable{Table: Table[pb.RuneHolders]{cache: store}}
}

func (s *RuneHoldersTable) Get(key *Rune) (ret RuneHolders) {
	tblKey := []byte(store.RUNE_TO_ADDRESS_HOLDER + key.String())
	pbVal := s.cache.Get(tblKey)
	if pbVal != nil {
		ret = RuneHolders{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneHoldersTable) GetFromDB(key *Rune) (ret RuneHolders) {
	tblKey := []byte(store.RUNE_TO_ADDRESS_HOLDER + key.String())
	pbVal, _ := s.cache.GetFromDB(tblKey)
	if pbVal != nil {
		ret = RuneHolders{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneHoldersTable) Insert(key *Rune, value RuneHolders) (ret RuneHolders) {
	tblKey := []byte(store.RUNE_TO_ADDRESS_HOLDER + key.String())
	pbVal := s.cache.Insert(tblKey, value.ToPb())
	if pbVal != nil {
		ret = RuneHolders{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneHoldersTable) Flush() {
	s.cache.Flush()
}
