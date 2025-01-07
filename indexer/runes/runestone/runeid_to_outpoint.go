package runestone

import (
	"strings"

	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type RuneIdToOutpoint struct {
	RuneId   *RuneId
	Outpoint *OutPoint
}

func RuneIdToOutpointFromString(str string) (*RuneIdToOutpoint, error) {
	runeIdToOutpoint := &RuneIdToOutpoint{}
	parts := strings.SplitN(str, "-", 3)
	var err error
	runeIdToOutpoint.RuneId, err = RuneIdFromHex(parts[1])
	if err != nil {
		return nil, err
	}
	runeIdToOutpoint.Outpoint, err = OutPointFromHex(parts[2])
	if err != nil {
		return nil, err
	}
	return runeIdToOutpoint, nil
}

func (s *RuneIdToOutpoint) ToPb() *pb.RuneIdToOutpoint {
	return &pb.RuneIdToOutpoint{}
}

func (s *RuneIdToOutpoint) String() string {
	return s.RuneId.Hex() + "-" + s.Outpoint.Hex()
}

type RuneIdToOutpointTable struct {
	Table[pb.RuneIdToOutpoint]
}

func NewRuneIdToUtxoTable(store *store.Cache[pb.RuneIdToOutpoint]) *RuneIdToOutpointTable {
	return &RuneIdToOutpointTable{Table: Table[pb.RuneIdToOutpoint]{cache: store}}
}

func (s *RuneIdToOutpointTable) GetOutpoints(runeId *RuneId) (ret []*OutPoint, err error) {
	tblKey := []byte(store.RUNEID_TO_UTXO + runeId.Hex() + "-")
	pbVal := s.cache.GetList(tblKey, false)
	if pbVal != nil {
		ret = make([]*OutPoint, len(pbVal))
		var i = 0
		for k := range pbVal {
			v, err := RuneIdToOutpointFromString(k)
			if err != nil {
				return nil, err
			}
			ret[i] = v.Outpoint
			i++
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
