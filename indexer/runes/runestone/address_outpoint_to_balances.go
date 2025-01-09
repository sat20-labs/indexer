package runestone

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"lukechampine.com/uint128"
)

type AddressOutpointToBalance struct {
	AddressId uint64
	OutPoint  *OutPoint
	Address   Address
	RuneId    *RuneId
	Balance   *Lot
}

func AddressOutpointToBalanceFromString(str string) (*AddressOutpointToBalance, error) {
	ret := &AddressOutpointToBalance{}
	parts := strings.SplitN(str, "-", 3)
	var err error
	ret.AddressId, err = strconv.ParseUint(parts[1], 16, 64)
	if err != nil {
		return nil, err
	}

	ret.OutPoint, err = OutPointFromHex(parts[2])
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *AddressOutpointToBalance) Key() string {
	return fmt.Sprintf("%x", s.AddressId) + "-" + s.OutPoint.Hex()
}

func (s *AddressOutpointToBalance) ToPb() *pb.AddressOutpointToBalance {
	if s.Address == "" {
		common.Log.Info("test")
	}
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
	return &AddressOutpointToBalancesTable{Table: Table[pb.AddressOutpointToBalance]{cache: v}}
}

func (s *AddressOutpointToBalancesTable) Get(v *AddressOutpointToBalance) (ret *AddressOutpointToBalance) {
	tblKey := []byte(store.ADDRESS_OUTPOINT_TO_BALANCE + v.Key())
	pbVal := s.cache.Get(tblKey)
	if pbVal != nil {
		var err error
		ret, err = AddressOutpointToBalanceFromString(string(tblKey))
		if err != nil {
			common.Log.Panicf("AddressOutpointToBalanceTable.Get-> AddressOutpointToBalanceFromString(%s) err:%v", string(tblKey), err)
		}
		ret.RuneId = &RuneId{Block: pbVal.RuneId.Block, Tx: pbVal.RuneId.Tx}
		ret.Address = Address(pbVal.Address)
		ret.Balance = &Lot{
			Value: &uint128.Uint128{
				Hi: pbVal.Balance.Value.Hi,
				Lo: pbVal.Balance.Value.Lo,
			},
		}
	}
	return
}

func (s *AddressOutpointToBalancesTable) GetBalances(addressId uint64) (ret []*AddressOutpointToBalance, err error) {
	tblKey := []byte(store.ADDRESS_OUTPOINT_TO_BALANCE + fmt.Sprintf("%x", addressId) + "-")
	pbVal := s.cache.GetList(tblKey, true)
	if pbVal != nil {
		ret = make([]*AddressOutpointToBalance, len(pbVal))
		var i = 0
		for k, v := range pbVal {
			var err error
			lot := &Lot{
				Value: &uint128.Uint128{Hi: v.Balance.Value.Hi, Lo: v.Balance.Value.Lo},
			}
			addressOutpointToBalance, err := AddressOutpointToBalanceFromString(k)
			if err != nil {
				return nil, err
			}
			addressOutpointToBalance.Address = Address(v.Address)
			addressOutpointToBalance.RuneId = &RuneId{Block: v.RuneId.Block, Tx: v.RuneId.Tx}
			addressOutpointToBalance.Balance = lot
			ret[i] = addressOutpointToBalance
			i++
		}
	}
	return
}

func (s *AddressOutpointToBalancesTable) Insert(v *AddressOutpointToBalance) {
	if v.Address == "tb1pc5j5j5nsk00rxhvytthzu26f2aqjzyaxunfjnv73h0hhsg4q48jqk6d4ph" && v.RuneId.Block == 30562 && v.RuneId.Tx == 50 {
		common.Log.Debugf("RuneIdAddressToBalanceTable.Insert-> address is empty, runeId:%s, addressId:%d", v.RuneId.Hex(), v.AddressId)
	}
	tblKey := []byte(store.ADDRESS_OUTPOINT_TO_BALANCE + v.Key())
	s.cache.Set(tblKey, v.ToPb())
}

func (s *AddressOutpointToBalancesTable) Remove(v *AddressOutpointToBalance) {
	if v.Address == "tb1pc5j5j5nsk00rxhvytthzu26f2aqjzyaxunfjnv73h0hhsg4q48jqk6d4ph" && v.RuneId.Block == 30562 && v.RuneId.Tx == 50 {
		common.Log.Debugf("RuneIdAddressToBalanceTable.Insert-> address is empty, runeId:%s, addressId:%d", v.RuneId.Hex(), v.AddressId)
	}
	tblKey := []byte(store.ADDRESS_OUTPOINT_TO_BALANCE + v.Key())
	s.cache.Delete(tblKey)
}
