package runestone

import (
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type Address string

type RuneIdToAddress struct {
	RuneId    *RuneId
	Address   Address
	AddressId uint64
}

func (s *RuneIdToAddress) FromString(key string) {
	parts := strings.SplitN(key, "-", 4)
	var err error
	s.RuneId, err = RuneIdFromString(parts[1])
	if err != nil {
		common.Log.Panicf("RuneIdToAddress.FromString-> RuneIdFromString(%s) err:%v", parts[1], err)
	}
	s.Address = Address(parts[2])
	addressId, err := strconv.ParseUint(parts[3], 16, 64)
	if err != nil {
		common.Log.Panicf("RuneIdToAddress.FromString-> strconv.ParseUint(%s) err:%v", parts[3], err)
	}
	s.AddressId = addressId

}

func (s *RuneIdToAddress) ToPb() *pb.RuneIdToAddress {
	return &pb.RuneIdToAddress{}
}

func (s *RuneIdToAddress) String() string {
	adressId := strconv.FormatUint(s.AddressId, 16)
	return s.RuneId.String() + "-" + string(s.Address) + "-" + adressId
}

type RuneToAddressTable struct {
	Table[pb.RuneIdToAddress]
}

func NewRuneIdToAddressTable(cache *store.Cache[pb.RuneIdToAddress]) *RuneToAddressTable {
	return &RuneToAddressTable{Table: Table[pb.RuneIdToAddress]{cache: cache}}
}

func (s *RuneToAddressTable) GetAddresses(runeId *RuneId) (ret []Address) {
	tblKey := []byte(store.RUNEID_TO_ADDRESS + runeId.String() + "-")
	pbVal := s.cache.GetList(tblKey, false)

	if pbVal != nil {
		ret = make([]Address, len(pbVal))
		var i = 0
		for k := range pbVal {
			v := &RuneIdToAddress{}
			v.FromString(k)
			ret[i] = v.Address
			i++
		}
	}
	return
}

func (s *RuneToAddressTable) Insert(v *RuneIdToAddress) (ret RuneIdToAddress) {
	tblKey := []byte(store.RUNEID_TO_ADDRESS + v.String())
	pbVal := s.cache.Set(tblKey, v.ToPb())
	if pbVal != nil {
		ret = *v
	}
	return
}
