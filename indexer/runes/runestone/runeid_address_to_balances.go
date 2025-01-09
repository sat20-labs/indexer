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

type RuneIdAddressToBalance struct {
	RuneId    *RuneId
	AddressId uint64
	Address   Address
	Balance   *Lot
}

func RuneIdAddressToBalanceFromString(str string) (*RuneIdAddressToBalance, error) {
	ret := &RuneIdAddressToBalance{}
	parts := strings.SplitN(str, "-", 3)

	var err error
	ret.RuneId, err = RuneIdFromHex(parts[1])
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
		Address: string(s.Address),
	}
	return pbValue
}

type RuneIdAddressToBalanceTable struct {
	Table[pb.RuneIdAddressToBalance]
}

func NewRuneIdAddressToBalanceTable(v *store.Cache[pb.RuneIdAddressToBalance]) *RuneIdAddressToBalanceTable {
	return &RuneIdAddressToBalanceTable{Table: Table[pb.RuneIdAddressToBalance]{cache: v}}
}

func (s *RuneIdAddressToBalanceTable) Get(v *RuneIdAddressToBalance) (ret *RuneIdAddressToBalance) {
	tblKey := []byte(store.RUNEID_ADDRESS_TO_BALANCE + v.Key())
	pbVal := s.cache.Get(tblKey)
	if pbVal != nil {
		var err error
		ret, err = RuneIdAddressToBalanceFromString(string(tblKey))
		if err != nil {
			common.Log.Panicf("RuneIdAddressToBalanceTable.Get-> RuneIdAddressToBalanceFromString(%s) err:%v", string(tblKey), err)
			return nil
		}
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

func (s *RuneIdAddressToBalanceTable) GetBalances(runeId *RuneId) (ret []*RuneIdAddressToBalance, err error) {
	tblKey := []byte(store.RUNEID_ADDRESS_TO_BALANCE + runeId.Hex() + "-")
	pbVal := s.cache.GetList(tblKey, true)
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
			ret[i].Address = Address(v.Address)
			ret[i].Balance = &Lot{
				Value: &uint128.Uint128{Hi: v.Balance.Value.Hi, Lo: v.Balance.Value.Lo}}
			i++
		}
	}
	return
}

func (s *RuneIdAddressToBalanceTable) Insert(v *RuneIdAddressToBalance, runeentry1 *RuneEntry) {
	tblKey := []byte(store.RUNEID_ADDRESS_TO_BALANCE + v.Key())
	if v.Address == "tb1pc5j5j5nsk00rxhvytthzu26f2aqjzyaxunfjnv73h0hhsg4q48jqk6d4ph" && v.RuneId.Block == 30562 && v.RuneId.Tx == 50 {
		if runeentry1 != nil {
			pile := runeentry1.Pile(*v.Balance.Value)
			pilestr := pile.String()
			common.Log.Infof("RuneIdAddressToBalanceTable.Insert-> runeId:%s, address:%s, pile:%s ",
				v.RuneId.String(), v.Address, pilestr)
		}

	}
	s.cache.Set(tblKey, v.ToPb())
}

func (s *RuneIdAddressToBalanceTable) Remove(v *RuneIdAddressToBalance) {
	tblKey := []byte(store.RUNEID_ADDRESS_TO_BALANCE + v.Key())
	if v.RuneId == nil {
		common.Log.Infof("RuneIdAddressToBalanceTable.Insert-> runeId is empty, runeId:%s, addressId:%d", v.RuneId.Hex(), v.AddressId)
	}
	if v.Address == "tb1pc5j5j5nsk00rxhvytthzu26f2aqjzyaxunfjnv73h0hhsg4q48jqk6d4ph" && v.RuneId.Block == 30562 && v.RuneId.Tx == 50 {
		common.Log.Infof("RuneIdAddressToBalanceTable.Insert-> address is empty, runeId:%s, addressId:%d", v.RuneId.Hex(), v.AddressId)
	}
	s.cache.Delete(tblKey)
}
