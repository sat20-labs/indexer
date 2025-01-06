package runestone

import (
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"lukechampine.com/uint128"
)

type RuneIdAddressOutpointToBalance struct {
	RuneId   *RuneId
	Address  Address
	OutPoint *OutPoint
	Balance  *Lot
}

func GenRuneIdAddressOutpointToBalance(str string, balance *Lot) (*RuneIdAddressOutpointToBalance, error) {
	ret := &RuneIdAddressOutpointToBalance{}
	parts := strings.SplitN(str, "-", 4)
	var err error
	ret.RuneId, err = RuneIdFromString(parts[1])
	if err != nil {
		return nil, err
	}
	ret.Address = Address(parts[2])

	ret.OutPoint, err = OutPointFromString(parts[3])
	if err != nil {
		return nil, err
	}
	ret.Balance = balance
	return ret, nil
}

func (s *RuneIdAddressOutpointToBalance) String() string {
	return s.RuneId.String() + "-" + string(s.Address) + "-" + s.OutPoint.String()
}

func (s *RuneIdAddressOutpointToBalance) ToPb() *pb.RuneIdToOutpointToBalance {
	pbValue := &pb.RuneIdToOutpointToBalance{
		Balance: &pb.Lot{
			Value: &pb.Uint128{
				Hi: s.Balance.Value.Hi,
				Lo: s.Balance.Value.Lo,
			},
		},
	}
	return pbValue
}

type RuneIdAddressOutpointToBalanceTable struct {
	Table[pb.RuneIdToOutpointToBalance]
}

func NewRuneIdAddressOutpointToBalancesTable(v *store.Cache[pb.RuneIdToOutpointToBalance]) *RuneIdAddressOutpointToBalanceTable {
	return &RuneIdAddressOutpointToBalanceTable{Table: Table[pb.RuneIdToOutpointToBalance]{cache: v}}
}

func (s *RuneIdAddressOutpointToBalanceTable) Get(v *RuneIdAddressOutpointToBalance) (ret *RuneIdAddressOutpointToBalance) {
	tblKey := []byte(store.RUNEID_ADDRESS_OUTPOINT_TO_BALANCE + v.String())
	pbVal := s.cache.Get(tblKey)
	if pbVal != nil {
		var err error
		ret, err = GenRuneIdAddressOutpointToBalance(string(tblKey), v.Balance)
		if err != nil {
			common.Log.Panicf("RuneIdAddressOutpointToBalanceTable.Get-> GenRuneIdAddressOutpointToBalance(%s) err:%v", string(tblKey), err)
		}
	}
	return
}

func (s *RuneIdAddressOutpointToBalanceTable) GetBalances(runeId *RuneId) (ret []*RuneIdAddressOutpointToBalance, err error) {
	tblKey := []byte(store.RUNEID_ADDRESS_OUTPOINT_TO_BALANCE + runeId.String() + "-")
	pbVal := s.cache.GetList(tblKey, true)
	if pbVal != nil {
		ret = make([]*RuneIdAddressOutpointToBalance, len(pbVal))
		var i = 0
		for k, v := range pbVal {
			var err error
			balance := &Lot{
				Value: &uint128.Uint128{
					Hi: v.Balance.Value.Hi,
					Lo: v.Balance.Value.Lo,
				},
			}
			RuneIdAddressOutpointToBalance, err := GenRuneIdAddressOutpointToBalance(string(k), balance)
			if err != nil {
				common.Log.Panicf("RuneIdAddressOutpointToBalanceTable.Get-> GenRuneIdAddressOutpointToBalance(%s) err:%v", string(k), err)
			}
			ret[i] = RuneIdAddressOutpointToBalance
			i++
		}
	}
	return
}

func (s *RuneIdAddressOutpointToBalanceTable) Insert(v *RuneIdAddressOutpointToBalance) (ret *RuneIdAddressOutpointToBalance) {
	tblKey := []byte(store.RUNEID_ADDRESS_OUTPOINT_TO_BALANCE + v.String())
	pbVal := s.cache.Set(tblKey, v.ToPb())
	if pbVal != nil {
		balance := &Lot{
			Value: &uint128.Uint128{
				Hi: pbVal.Balance.Value.Hi,
				Lo: pbVal.Balance.Value.Lo,
			},
		}
		var err error
		ret, err = GenRuneIdAddressOutpointToBalance(string(tblKey), balance)
		if err != nil {
			common.Log.Panicf("RuneToOutpointToBalanceTable.Insert-> GenRuneIdOutpointToBalance(%s) err:%v", string(tblKey), err)
		}
	}
	return
}

func (s *RuneIdAddressOutpointToBalanceTable) Remove(key *RuneIdOutpointToBalance) (ret *RuneIdOutpointToBalance) {
	tblKey := []byte(store.OUTPOINT_TO_BALANCES + key.String())
	pbVal := s.cache.Delete(tblKey)
	if pbVal != nil {
		balance := &Lot{
			Value: &uint128.Uint128{
				Hi: pbVal.Balance.Value.Hi,
				Lo: pbVal.Balance.Value.Lo,
			},
		}
		var err error
		ret, err = GenRuneIdOutpointToBalance(string(tblKey), balance)
		if err != nil {
			common.Log.Panicf("RuneToOutpointToBalanceTable.Insert-> GenRuneIdOutpointToBalance(%s) err:%v", string(tblKey), err)
		}
	}
	return
}
