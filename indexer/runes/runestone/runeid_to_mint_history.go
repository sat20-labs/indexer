package runestone

import (
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type Utxo string

type RuneIdToMintHistory struct {
	RuneId *RuneId
	Utxo   Utxo
}

func (s *RuneIdToMintHistory) FromString(key string) {
	parts := strings.SplitN(key, "-", 3)
	var err error
	s.RuneId, err = RuneIdFromString(parts[1])
	if err != nil {
		common.Log.Panicf("RuneIdToAddress.FromString-> RuneIdFromString(%s) err:%v", parts[1], err)
	}
	s.Utxo = Utxo(parts[2])
}

func (s *RuneIdToMintHistory) ToPb() *pb.RuneIdToMintHistory {
	return &pb.RuneIdToMintHistory{}
}

func (s *RuneIdToMintHistory) String() string {
	return s.RuneId.String() + "-" + string(s.Utxo)
}

type RuneToMintHistoryTable struct {
	Table[pb.RuneIdToMintHistory]
}

func NewRuneIdToMintHistoryTable(store *store.Cache[pb.RuneIdToMintHistory]) *RuneToMintHistoryTable {
	return &RuneToMintHistoryTable{Table: Table[pb.RuneIdToMintHistory]{cache: store}}
}

func (s *RuneToMintHistoryTable) GetUtxosFromDB(runeId *RuneId) (ret []Utxo) {
	tblKey := []byte(store.RUNEID_TO_MINT_HISTORYS + runeId.String() + "-")
	pbVal := s.cache.GetListFromDB(tblKey, false)

	if pbVal != nil {
		ret = make([]Utxo, len(pbVal))
		var i = 0
		for k := range pbVal {
			v := &RuneIdToMintHistory{}
			v.FromString(k)
			ret[i] = v.Utxo
			i++
		}
	}
	return
}

func (s *RuneToMintHistoryTable) Insert(key *RuneIdToMintHistory) (ret RuneIdToMintHistory) {
	tblKey := []byte(store.RUNEID_TO_MINT_HISTORYS + key.String())
	pbVal := s.cache.Set(tblKey, key.ToPb())
	if pbVal != nil {
		ret = *key
	}
	return
}
