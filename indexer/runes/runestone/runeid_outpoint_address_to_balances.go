package runestone

import (
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"lukechampine.com/uint128"
)

type RuneIdOutpointAddressToBalance struct {
	RuneId   *RuneId
	Address  Address
	OutPoint *OutPoint
	Balance  *Lot
}

func GenRuneIdOutpointAddressToBalance(str string, address string, balance *Lot) (*RuneIdOutpointAddressToBalance, error) {
	ret := &RuneIdOutpointAddressToBalance{}
	parts := strings.SplitN(str, "-", 3)
	var err error
	ret.RuneId, err = RuneIdFromHex(parts[1])
	if err != nil {
		return nil, err
	}

	ret.OutPoint, err = OutPointFromHex(parts[2])
	if err != nil {
		return nil, err
	}
	ret.Address = Address(address)
	ret.Balance = balance
	return ret, nil
}

func (s *RuneIdOutpointAddressToBalance) String() string {
	return s.RuneId.Hex() + "-" + s.OutPoint.Hex()
}

func (s *RuneIdOutpointAddressToBalance) ToPb() *pb.RuneAddressBalance {
	pbValue := &pb.RuneAddressBalance{
		Balance: &pb.Lot{
			Value: &pb.Uint128{
				Hi: s.Balance.Value.Hi,
				Lo: s.Balance.Value.Lo,
			},
		},
		Address: string(s.Address),
	}
	return pbValue
}

type RuneIdAddressOutpointToBalanceTable struct {
	Table[pb.RuneAddressBalance]
}

func NewRuneIdAddressOutpointToBalancesTable(v *store.Cache[pb.RuneAddressBalance]) *RuneIdAddressOutpointToBalanceTable {
	return &RuneIdAddressOutpointToBalanceTable{Table: Table[pb.RuneAddressBalance]{cache: v}}
}

func (s *RuneIdAddressOutpointToBalanceTable) Get(v *RuneIdOutpointAddressToBalance) (ret *RuneIdOutpointAddressToBalance) {
	tblKey := []byte(store.RUNEID_OUTPOINT_TO_ADDRESS_BALANCE + v.String())
	pbVal := s.cache.Get(tblKey)
	if pbVal != nil {
		var err error
		ret, err = GenRuneIdOutpointAddressToBalance(string(tblKey), string(v.Address), v.Balance)
		if err != nil {
			common.Log.Panicf("RuneIdAddressOutpointToBalanceTable.Get-> GenRuneIdAddressOutpointToBalance(%s) err:%v", string(tblKey), err)
		}
	}
	return
}

func (s *RuneIdAddressOutpointToBalanceTable) GetBalances(runeId *RuneId) (ret []*RuneIdOutpointAddressToBalance, err error) {
	tblKey := []byte(store.RUNEID_OUTPOINT_TO_ADDRESS_BALANCE + runeId.Hex() + "-")
	pbVal := s.cache.GetList(tblKey, true)
	if pbVal != nil {
		ret = make([]*RuneIdOutpointAddressToBalance, len(pbVal))
		var i = 0
		for k, v := range pbVal {
			var err error
			balance := &Lot{
				Value: &uint128.Uint128{
					Hi: v.Balance.Value.Hi,
					Lo: v.Balance.Value.Lo,
				},
			}
			RuneIdAddressOutpointToBalance, err := GenRuneIdOutpointAddressToBalance(string(k), v.Address, balance)
			if err != nil {
				common.Log.Panicf("RuneIdAddressOutpointToBalanceTable.Get-> GenRuneIdAddressOutpointToBalance(%s) err:%v", string(k), err)
			}
			ret[i] = RuneIdAddressOutpointToBalance
			i++
		}
	}
	return
}

func (s *RuneIdAddressOutpointToBalanceTable) Insert(v *RuneIdOutpointAddressToBalance) {
	tblKey := []byte(store.RUNEID_OUTPOINT_TO_ADDRESS_BALANCE + v.String())
	s.cache.Set(tblKey, v.ToPb())
}

func (s *RuneIdAddressOutpointToBalanceTable) Remove(key *RuneIdOutpointAddressToBalance) {
	tblKey := []byte(store.RUNEID_OUTPOINT_TO_ADDRESS_BALANCE + key.String())
	s.cache.Delete(tblKey)
}
