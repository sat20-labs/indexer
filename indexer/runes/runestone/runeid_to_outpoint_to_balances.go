package runestone

import (
	"strings"

	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type RuneIdToOutpointToBalance struct {
	RuneId   *RuneId
	OutPoint *OutPoint
	Balance  *Lot
}

func RuneIdToOutpointToBalanceFromString(str string) (*RuneIdToOutpointToBalance, error) {
	ret := &RuneIdToOutpointToBalance{}
	parts := strings.SplitN(str, "-", 4)
	var err error
	ret.RuneId, err = RuneIdFromString(parts[1])
	if err != nil {
		return nil, err
	}
	ret.OutPoint, err = OutPointFromString(parts[2])
	if err != nil {
		return nil, err
	}
	ret.Balance, err = LotFromString(parts[3])
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *RuneIdToOutpointToBalance) String() string {
	return s.RuneId.String() + "-" + s.OutPoint.String() + "-" + s.Balance.String()
}

func (s *RuneIdToOutpointToBalance) ToPb() *pb.RuneIdToOutpointToBalance {
	pbValue := &pb.RuneIdToOutpointToBalance{}

	return pbValue
}

func (s *RuneIdToOutpointToBalance) FromPb(pbValue *pb.RuneIdToOutpointToBalance) {

}

type RuneIdToOutpointToBalanceTable struct {
	Table[pb.RuneIdToOutpointToBalance]
}

func NewRuneIdToOutpointToBalancesTable(v *store.Cache[pb.RuneIdToOutpointToBalance]) *RuneIdToOutpointToBalanceTable {
	return &RuneIdToOutpointToBalanceTable{Table: Table[pb.RuneIdToOutpointToBalance]{cache: v}}
}

func (s *RuneIdToOutpointToBalanceTable) Get(key *RuneIdToOutpointToBalance) (ret RuneIdToOutpointToBalance) {
	tblKey := []byte(store.RUNEID_TO_OUTPOINT_TO_BALANCE + key.String())
	pbVal := s.cache.Get(tblKey)
	if pbVal != nil {
		ret = RuneIdToOutpointToBalance{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneIdToOutpointToBalanceTable) Insert(key *RuneIdToOutpointToBalance) (ret *RuneIdToOutpointToBalance) {
	tblKey := []byte(store.RUNEID_TO_OUTPOINT_TO_BALANCE + key.String())
	pbVal := s.cache.Set(tblKey, key.ToPb())
	if pbVal != nil {
		ret = &RuneIdToOutpointToBalance{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneIdToOutpointToBalanceTable) Remove(key *RuneIdToOutpointToBalance) (ret *RuneIdToOutpointToBalance) {
	tblKey := []byte(store.OUTPOINT_TO_BALANCES + key.String())
	pbVal := s.cache.Delete(tblKey)
	if pbVal != nil {
		ret = &RuneIdToOutpointToBalance{}
		ret.FromPb(pbVal)
	}
	return
}
