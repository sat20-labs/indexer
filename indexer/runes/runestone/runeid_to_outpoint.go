package runestone

import (
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type RuneIdToOutpoint struct {
	RuneId   *RuneId
	Outpoint *OutPoint
}

func (s *RuneIdToOutpoint) FromString(key string) {
	parts := strings.SplitN(key, "-", 2)
	var err error
	s.RuneId, err = RuneIdFromString(parts[0])
	if err != nil {
		common.Log.Panicf("RuneIdToAddress.FromString-> RuneIdFromString(%s) err:%v", parts[0], err)
	}
	err = s.Outpoint.FromString(parts[1])
	if err != nil {
		common.Log.Panicf("RuneIdToAddress.FromString-> OutPoint.FromString(%s) err:%v", parts[1], err)
	}
}

func (s *RuneIdToOutpoint) ToPb() *pb.RuneIdToOutpoint {
	return &pb.RuneIdToOutpoint{}
}

func (s *RuneIdToOutpoint) String() string {
	return s.RuneId.String() + "-" + s.Outpoint.String()
}

type RuneIdToOutpointTable struct {
	Table[pb.RuneIdToOutpoint]
}

func NewRuneIdToUtxoTable(store *store.Cache[pb.RuneIdToOutpoint]) *RuneIdToOutpointTable {
	return &RuneIdToOutpointTable{Table: Table[pb.RuneIdToOutpoint]{cache: store}}
}

func (s *RuneIdToOutpointTable) GetOutpointsFromDB(runeId *RuneId) (ret []*OutPoint) {
	tblKey := []byte(store.RUNEID_TO_UTXO + runeId.String() + "-")
	pbVal := s.cache.GetListFromDB(tblKey, false)
	if pbVal != nil {
		ret = make([]*OutPoint, 0)
		for k := range pbVal {
			v := &RuneIdToOutpoint{}
			v.FromString(k)
			ret = append(ret, v.Outpoint)
		}
	}
	return
}

func (s *RuneIdToOutpointTable) Insert(key *RuneIdToOutpoint) (ret RuneIdToOutpoint) {
	tblKey := []byte(store.RUNEID_TO_UTXO + key.String())
	pbVal := s.cache.Set(tblKey, key.ToPb())
	if pbVal != nil {
		ret = *key
	}
	return
}
