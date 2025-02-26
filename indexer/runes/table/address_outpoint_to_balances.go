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

type AddressOutpointToBalance struct {
	AddressId uint64
	Address   runestone.Address
	OutPoint  *runestone.OutPoint
	RuneId    *runestone.RuneId
	Balance   runestone.Lot
}

func AddressOutpointToBalanceFromString(str string) (*AddressOutpointToBalance, error) {
	ret := &AddressOutpointToBalance{}
	parts := strings.SplitN(str, "-", 3)
	var err error
	ret.AddressId, err = strconv.ParseUint(parts[1], 16, 64)
	if err != nil {
		return nil, err
	}
	ret.OutPoint, err = runestone.OutPointFromString(parts[2])
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *AddressOutpointToBalance) Key() string {
	return fmt.Sprintf("%x", s.AddressId) + "-" + s.OutPoint.Hex()
}

func (s *AddressOutpointToBalance) ToPb() *pb.AddressOutpointToBalance {
	pbValue := &pb.AddressOutpointToBalance{
		Balance: &pb.Lot{
			Value: &pb.Uint128{
				Hi: s.Balance.Value.Hi,
				Lo: s.Balance.Value.Lo,
			},
		},
		Address: string(s.Address),
		RuneId:  &pb.RuneId{Block: s.RuneId.Block, Tx: s.RuneId.Tx},
	}
	return pbValue
}

type AddressOutpointToBalancesTable struct {
	Table[pb.AddressOutpointToBalance]
}

func NewAddressOutpointToBalancesTable(v *store.Cache[pb.AddressOutpointToBalance]) *AddressOutpointToBalancesTable {
	return &AddressOutpointToBalancesTable{Table: Table[pb.AddressOutpointToBalance]{Cache: v}}
}

func (s *AddressOutpointToBalancesTable) Get(v *AddressOutpointToBalance) (ret *AddressOutpointToBalance) {
	tblKey := []byte(store.ADDRESS_OUTPOINT_TO_BALANCE + v.Key())
	pbVal := s.Cache.Get(tblKey)
	if pbVal != nil {
		var err error
		ret, err = AddressOutpointToBalanceFromString(string(tblKey))
		if err != nil {
			common.Log.Panicf("AddressOutpointToBalanceTable.Get-> AddressOutpointToBalanceFromString(%s) err:%v", string(tblKey), err)
		}
		if pbVal.RuneId != nil {
			ret.RuneId = &runestone.RuneId{Block: pbVal.RuneId.Block, Tx: pbVal.RuneId.Tx}
		}
		ret.Address = runestone.Address(pbVal.Address)
		ret.Balance = runestone.Lot{
			Value: uint128.Uint128{
				Hi: pbVal.Balance.Value.Hi,
				Lo: pbVal.Balance.Value.Lo,
			},
		}
	}
	return
}

func (s *AddressOutpointToBalancesTable) GetBalances(addressId uint64) (ret []*AddressOutpointToBalance, err error) {
	tblKey := []byte(store.ADDRESS_OUTPOINT_TO_BALANCE + fmt.Sprintf("%x", addressId) + "-")
	pbVal := s.Cache.GetList(tblKey, true)
	if pbVal != nil {
		ret = make([]*AddressOutpointToBalance, len(pbVal))
		var i = 0
		for k, v := range pbVal {
			var err error
			lot := &runestone.Lot{
				Value: uint128.Uint128{Hi: v.Balance.Value.Hi, Lo: v.Balance.Value.Lo},
			}
			addressOutpointToBalance, err := AddressOutpointToBalanceFromString(k)
			if err != nil {
				return nil, err
			}
			addressOutpointToBalance.Address = runestone.Address(v.Address)
			addressOutpointToBalance.RuneId = &runestone.RuneId{Block: v.RuneId.Block, Tx: v.RuneId.Tx}
			addressOutpointToBalance.Balance = *lot
			ret[i] = addressOutpointToBalance
			i++
		}
	}
	return
}

func (s *AddressOutpointToBalancesTable) Insert(v *AddressOutpointToBalance) {
	tblKey := []byte(store.ADDRESS_OUTPOINT_TO_BALANCE + v.Key())
	if IsLessStorage {
		v.Address = ""
	}
	s.Cache.Set(tblKey, v.ToPb())
}

func (s *AddressOutpointToBalancesTable) Remove(v *AddressOutpointToBalance) (ret *AddressOutpointToBalance) {
	tblKey := []byte(store.ADDRESS_OUTPOINT_TO_BALANCE + v.Key())
	pbVal := s.Cache.Delete(tblKey)
	if pbVal != nil {
		ret = &AddressOutpointToBalance{}
		var err error
		ret, err = AddressOutpointToBalanceFromString(string(tblKey))
		if err != nil {
			common.Log.Panicf("AddressOutpointToBalancesTable.Remove-> AddressOutpointToBalanceFromString(%s) err:%v", string(tblKey), err)
		}
	}
	return
}

func (s *AddressOutpointToBalancesTable) IsExistOnlyOne(addressId uint64) (ret bool) {
	tblKey := []byte(store.ADDRESS_OUTPOINT_TO_BALANCE + fmt.Sprintf("%x", addressId) + "-")
	count := 0
	s.Cache.IsExist(tblKey, func(k []byte, v *pb.AddressOutpointToBalance) bool {
		count++
		return count > 1
	})
	ret = count == 1
	return
}

func (s *AddressOutpointToBalancesTable) IsExist(addressId uint64, runeId *runestone.RuneId) (ret bool) {
	tblKey := []byte(store.ADDRESS_OUTPOINT_TO_BALANCE + fmt.Sprintf("%x", addressId) + "-")
	ret = s.Cache.IsExist(tblKey, func(k []byte, v *pb.AddressOutpointToBalance) bool {
		if v.RuneId.Block == runeId.Block && v.RuneId.Tx == runeId.Tx {
			return true
		}
		return false
	})
	return
}
