package runestone

import (
	"strings"

	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type RuneMintHistory struct {
	Address Address
	Rune    Rune
	Utxo    string
}

func (s *RuneMintHistory) FromPb(key string) {
	parts := strings.SplitN(key, ":", 3)
	s.Address = Address(parts[0])
	s.Rune = *NewRune(Uint128FromString(parts[1]))
	s.Utxo = parts[2]
}

func (s *RuneMintHistory) GetKey() string {
	return string(s.Address) + ":" + s.Rune.String() + ":" + s.Utxo
}

type RuneMintHistorys map[string]*RuneMintHistory

func (s RuneMintHistorys) ToPb() (ret *pb.RuneMintHistorys) {
	ret = &pb.RuneMintHistorys{
		MintHistorys: make(map[string]*pb.RuneMintHistory, len(s)),
	}
	for k := range s {
		ret.MintHistorys[k] = &pb.RuneMintHistory{}
	}
	return ret
}

func (s RuneMintHistorys) FromPb(pbVal *pb.RuneMintHistorys) {
	for key := range pbVal.MintHistorys {
		runeMintHistory := &RuneMintHistory{}
		runeMintHistory.FromPb(key)
		s[key] = runeMintHistory
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
	pbVal := s.cache.Set(tblKey, value.ToPb())
	if pbVal != nil {
		ret = RuneMintHistorys{}
		ret.FromPb(pbVal)
	}
	return
}
