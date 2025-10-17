package table

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"lukechampine.com/uint128"
)

type RuneIdAddressToBalance struct {
	RuneId    *runestone.RuneId
	AddressId uint64
	Balance   runestone.Lot
}

func RuneIdAddressToBalanceFromString(str string) (*RuneIdAddressToBalance, error) {
	ret := &RuneIdAddressToBalance{}
	parts := strings.SplitN(str, "-", 3)

	var err error
	ret.RuneId, err = runestone.RuneIdFromHex(parts[1])
	if err != nil {
		return nil, err
	}
	ret.AddressId, err = strconv.ParseUint(parts[2], 16, 64)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *RuneIdAddressToBalance) Key() string {
	return s.RuneId.Hex() + "-" + fmt.Sprintf("%x", s.AddressId)
}

func (s *RuneIdAddressToBalance) ToPb() *pb.RuneIdAddressToBalance {
	pbValue := &pb.RuneIdAddressToBalance{
		Balance: &pb.Lot{
			Value: &pb.Uint128{
				Hi: s.Balance.Value.Hi,
				Lo: s.Balance.Value.Lo,
			},
		},
		AddressId: s.AddressId,
	}
	return pbValue
}

type RuneIdAddressToBalanceTable struct {
	Table[pb.RuneIdAddressToBalance]
}

func NewRuneIdAddressToBalanceTable(v *store.Cache[pb.RuneIdAddressToBalance]) *RuneIdAddressToBalanceTable {
	return &RuneIdAddressToBalanceTable{Table: Table[pb.RuneIdAddressToBalance]{Cache: v}}
}

func (s *RuneIdAddressToBalanceTable) Get(v *RuneIdAddressToBalance) (ret *RuneIdAddressToBalance) {
	tblKey := []byte(store.RUNEID_ADDRESS_TO_BALANCE + v.Key())
	pbVal := s.Cache.Get(tblKey)
	if pbVal != nil {
		var err error
		ret, err = RuneIdAddressToBalanceFromString(string(tblKey))
		if err != nil {
			common.Log.Panicf("RuneIdAddressToBalanceTable.Get-> RuneIdAddressToBalanceFromString(%s) err:%v", string(tblKey), err)
			return nil
		}
		ret.AddressId = (pbVal.AddressId)
		ret.Balance = runestone.Lot{
			Value: uint128.Uint128{
				Hi: pbVal.Balance.Value.Hi,
				Lo: pbVal.Balance.Value.Lo,
			},
		}
	}
	return
}

func (s *RuneIdAddressToBalanceTable) GetBalances(runeId *runestone.RuneId) (ret []*RuneIdAddressToBalance, err error) {
	tblKey := []byte(store.RUNEID_ADDRESS_TO_BALANCE + runeId.Hex() + "-")
	pbVal := s.Cache.GetList(tblKey, true)
	if pbVal != nil {
		ret = make([]*RuneIdAddressToBalance, len(pbVal))
		var i = 0
		for k, v := range pbVal {
			var err error
			runeIdAddressOutpointToBalance, err := RuneIdAddressToBalanceFromString(string(k))
			if err != nil {
				return nil, err
			}
			ret[i] = runeIdAddressOutpointToBalance
			ret[i].AddressId = (v.AddressId)
			ret[i].Balance = runestone.Lot{
				Value: uint128.Uint128{Hi: v.Balance.Value.Hi, Lo: v.Balance.Value.Lo}}
			i++
		}
	}
	return
}

func (s *RuneIdAddressToBalanceTable) Insert(v *RuneIdAddressToBalance) {
	tblKey := []byte(store.RUNEID_ADDRESS_TO_BALANCE + v.Key())
	s.Cache.Set(tblKey, v.ToPb())
}

func (s *RuneIdAddressToBalanceTable) Remove(v *RuneIdAddressToBalance) {
	tblKey := []byte(store.RUNEID_ADDRESS_TO_BALANCE + v.Key())
	s.Cache.Delete(tblKey)
}
