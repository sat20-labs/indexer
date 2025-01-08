package runestone

import (
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"lukechampine.com/uint128"
)

type RuneIdOutpointAddressToBalance struct {
	RuneId    *RuneId
	OutPoint  *OutPoint
	AddressId uint64
	Address   Address
	Balance   *Lot
}

func RuneIdOutpointAddressToBalanceFromString(str string) (*RuneIdOutpointAddressToBalance, error) {
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
	return ret, nil
}

func (s *RuneIdOutpointAddressToBalance) Key() string {
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
		Address:   string(s.Address),
		AddressId: s.AddressId,
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
	tblKey := []byte(store.RUNEID_OUTPOINT_TO_ADDRESS_BALANCE + v.Key())
	pbVal := s.cache.Get(tblKey)
	if pbVal != nil {
		var err error
		ret, err = RuneIdOutpointAddressToBalanceFromString(string(tblKey))
		if err != nil {
			common.Log.Panicf("RuneIdAddressOutpointToBalanceTable.Get-> GenRuneIdAddressOutpointToBalance(%s) err:%v", string(tblKey), err)
		}
		ret.Address = Address(pbVal.Address)
		ret.Balance = &Lot{
			Value: &uint128.Uint128{
				Hi: pbVal.Balance.Value.Hi,
				Lo: pbVal.Balance.Value.Lo,
			},
		}
		ret.AddressId = pbVal.AddressId
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
			lot := &Lot{
				Value: &uint128.Uint128{Hi: v.Balance.Value.Hi, Lo: v.Balance.Value.Lo},
			}
			runeIdAddressOutpointToBalance, err := RuneIdOutpointAddressToBalanceFromString(k)
			if err != nil {
				return nil, err
			}
			runeIdAddressOutpointToBalance.Address = Address(v.Address)
			runeIdAddressOutpointToBalance.Balance = lot
			ret[i] = runeIdAddressOutpointToBalance
			i++
		}
	}
	return
}

func (s *RuneIdAddressOutpointToBalanceTable) Insert(v *RuneIdOutpointAddressToBalance) {
	tblKey := []byte(store.RUNEID_OUTPOINT_TO_ADDRESS_BALANCE + v.Key())
	s.cache.Set(tblKey, v.ToPb())
}

func (s *RuneIdAddressOutpointToBalanceTable) Remove(v *RuneIdOutpointAddressToBalance) {
	tblKey := []byte(store.RUNEID_OUTPOINT_TO_ADDRESS_BALANCE + v.Key())
	s.cache.Delete(tblKey)
}
