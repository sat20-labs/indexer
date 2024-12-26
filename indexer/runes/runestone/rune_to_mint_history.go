package runestone

import (
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type RuneMintHistory struct {
	Address Address
	Rune    Rune
	Utxo    string
}

func (s *RuneMintHistory) ToPb() (ret *pb.RuneMintHistory) {
	ret = &pb.RuneMintHistory{
		Address: string(s.Address),
		Rune:    s.Rune.ToPb(),
		Utxo:    s.Utxo,
	}
	return ret
}

func (s *RuneMintHistory) FromPb(pbVal *pb.RuneMintHistory) {
	s.Address = Address(pbVal.Address)
	s.Rune.FromPb(pbVal.Rune)
	s.Utxo = pbVal.Utxo
}

type RuneMintHistorys []*RuneMintHistory

func (s *RuneMintHistorys) ToPb() (ret *pb.RuneMintHistorys) {
	ret = &pb.RuneMintHistorys{
		MintHistorys: make([]*pb.RuneMintHistory, len(*s)),
	}
	for i, v := range *s {
		ret.MintHistorys[i] = v.ToPb()
	}
	return ret
}

func (s *RuneMintHistorys) FromPb(pbVal *pb.RuneMintHistorys) {
	for _, v := range pbVal.MintHistorys {
		runeMintHistory := &RuneMintHistory{}
		runeMintHistory.FromPb(v)
		*s = append(*s, runeMintHistory)
	}
}

type RuneMintHistorysTable struct {
	Table[pb.RuneMintHistorys]
}

func NewRuneMintHistorysTable(store *store.Cache[pb.RuneMintHistorys]) *RuneMintHistorysTable {
	return &RuneMintHistorysTable{Table: Table[pb.RuneMintHistorys]{cache: store}}
}

func (s *RuneMintHistorysTable) Get(key *Rune) (ret RuneMintHistorys) {
	tblKey := []byte(store.RUNE_TO_MINT_HISTORYS + key.String())
	pbVal := s.cache.Get(tblKey)
	if pbVal != nil {
		ret = RuneMintHistorys{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneMintHistorysTable) GetFromDB(key *Rune) (ret RuneMintHistorys) {
	tblKey := []byte(store.RUNE_TO_MINT_HISTORYS + key.String())
	pbVal, _ := s.cache.GetFromDB(tblKey)
	if pbVal != nil {
		ret = RuneMintHistorys{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneMintHistorysTable) Insert(key *Rune, value RuneMintHistorys) (ret RuneMintHistorys) {
	tblKey := []byte(store.RUNE_TO_MINT_HISTORYS + key.String())
	pbVal := s.cache.Insert(tblKey, value.ToPb())
	if pbVal != nil {
		ret = RuneMintHistorys{}
		ret.FromPb(pbVal)
	}
	return
}
