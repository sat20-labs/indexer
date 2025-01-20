package runestone

import (
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"lukechampine.com/uint128"
)

type RuneIdOutpointToBalance struct {
	RuneId   *RuneId
	OutPoint *OutPoint
	Balance  Lot
}

func RuneIdOutpointToBalanceFromString(str string) (*RuneIdOutpointToBalance, error) {
	ret := &RuneIdOutpointToBalance{}
	parts := strings.SplitN(str, "-", 3)
	var err error
	ret.RuneId, err = RuneIdFromHex(parts[1])
	if err != nil {
		return nil, err
	}
	ret.OutPoint, err = OutPointFromString(parts[2])
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *RuneIdOutpointToBalance) Key() string {
	return s.RuneId.Hex() + "-" + s.OutPoint.Hex()
}

func (s *RuneIdOutpointToBalance) ToPb() *pb.RuneBalance {
	pbValue := &pb.RuneBalance{
		Balance: &pb.Lot{
			Value: &pb.Uint128{
				Hi: s.Balance.Value.Hi,
				Lo: s.Balance.Value.Lo,
			},
		},
	}
	return pbValue
}

type RuneIdOutpointToBalanceTable struct {
	Table[pb.RuneBalance]
}

func NewRuneIdOutpointToBalancesTable(v *store.Cache[pb.RuneBalance]) *RuneIdOutpointToBalanceTable {
	return &RuneIdOutpointToBalanceTable{Table: Table[pb.RuneBalance]{Cache: v}}
}

func (s *RuneIdOutpointToBalanceTable) Get(v *RuneIdOutpointToBalance) (ret *RuneIdOutpointToBalance) {
	tblKey := []byte(store.RUNEID_OUTPOINT_TO_BALANCE + v.Key())
	pbVal := s.Cache.Get(tblKey)
	if pbVal != nil {
		var err error
		ret, err = RuneIdOutpointToBalanceFromString(string(tblKey))
		if err != nil {
			common.Log.Panicf("RuneIdOutpointToBalanceTable.Get-> GenRuneIdOutpointToBalance(%s) err:%v", string(tblKey), err)
		}
		ret.Balance = Lot{
			Value: uint128.Uint128{
				Hi: pbVal.Balance.Value.Hi,
				Lo: pbVal.Balance.Value.Lo,
			},
		}
	}
	return
}

func (s *RuneIdOutpointToBalanceTable) GetBalances(runeId *RuneId) (ret []*RuneIdOutpointToBalance, err error) {
	tblKey := []byte(store.RUNEID_OUTPOINT_TO_BALANCE + runeId.Hex() + "-")
	pbVal := s.Cache.GetList(tblKey, true)
	if pbVal != nil {
		ret = make([]*RuneIdOutpointToBalance, len(pbVal))
		var i = 0
		for k, v := range pbVal {
			var err error
			balance := &Lot{
				Value: uint128.Uint128{
					Hi: v.Balance.Value.Hi,
					Lo: v.Balance.Value.Lo,
				},
			}
			runeIdOutpointToBalance, err := RuneIdOutpointToBalanceFromString(string(k))
			if err != nil {
				common.Log.Panicf("RuneIdOutpointToBalanceTable.Get-> GenRuneIdOutpointToBalance(%s) err:%v", string(k), err)
			}
			runeIdOutpointToBalance.Balance = *balance
			ret[i] = runeIdOutpointToBalance
			i++
		}
	}
	return
}

func (s *RuneIdOutpointToBalanceTable) Insert(v *RuneIdOutpointToBalance) (ret *RuneIdOutpointToBalance) {
	tblKey := []byte(store.RUNEID_OUTPOINT_TO_BALANCE + v.Key())
	pbVal := s.Cache.Set(tblKey, v.ToPb())
	if pbVal != nil {
		balance := &Lot{
			Value: uint128.Uint128{
				Hi: pbVal.Balance.Value.Hi,
				Lo: pbVal.Balance.Value.Lo,
			},
		}
		var err error
		ret, err = RuneIdOutpointToBalanceFromString(string(tblKey))
		if err != nil {
			common.Log.Panicf("RuneIdOutpointToBalanceTable.Insert-> GenRuneIdOutpointToBalance(%s) err:%v", string(tblKey), err)
		}
		ret.Balance = *balance
	}
	return
}

func (s *RuneIdOutpointToBalanceTable) Remove(key *RuneIdOutpointToBalance) (ret *RuneIdOutpointToBalance) {
	tblKey := []byte(store.RUNEID_OUTPOINT_TO_BALANCE + key.Key())
	pbVal := s.Cache.Delete(tblKey)
	if pbVal != nil {
		balance := &Lot{
			Value: uint128.Uint128{
				Hi: pbVal.Balance.Value.Hi,
				Lo: pbVal.Balance.Value.Lo,
			},
		}
		var err error
		ret, err = RuneIdOutpointToBalanceFromString(string(tblKey))
		if err != nil {
			common.Log.Panicf("RuneIdOutpointToBalanceTable.Insert-> GenRuneIdOutpointToBalance(%s) err:%v", string(tblKey), err)
		}
		ret.Balance = *balance
	}
	return
}
