package runestone

import (
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type Address string

type RuneIdToAddress struct {
	RuneId  *RuneId
	Address Address
}

func (s *RuneIdToAddress) FromString(key string) {
	parts := strings.SplitN(key, "-", 2)
	var err error
	s.RuneId, err = RuneIdFromString(parts[0])
	if err != nil {
		common.Log.Panicf("RuneIdToAddress.FromString-> RuneIdFromString(%s) err:%v", parts[0], err)
	}
	s.Address = Address(parts[1])
}

func (s *RuneIdToAddress) ToPb() *pb.RuneIdToAddress {
	return &pb.RuneIdToAddress{}
}

func (s *RuneIdToAddress) String() string {
	return s.RuneId.String() + "-" + string(s.Address)
}

type RuneToAddressTable struct {
	Table[pb.RuneIdToAddress]
}

func NewRuneIdToAddressTable(cache *store.Cache[pb.RuneIdToAddress]) *RuneToAddressTable {
	return &RuneToAddressTable{Table: Table[pb.RuneIdToAddress]{cache: cache}}
}

func (s *RuneToAddressTable) GetAddressesFromDB(runeId *RuneId) (ret []Address) {
	tblKey := []byte(store.RUNEID_TO_ADDRESS + runeId.String() + "-")
	pbVal := s.cache.GetListFromDB(tblKey, false)

	if pbVal != nil {
		ret = make([]Address, 0)
		for k := range pbVal {
			v := &RuneIdToAddress{}
			v.FromString(k)
			ret = append(ret, v.Address)
		}
	}
	return
}

func (s *RuneToAddressTable) Insert(key *RuneIdToAddress) (ret RuneIdToAddress) {
	tblKey := []byte(store.RUNEID_TO_ADDRESS + key.String())
	pbVal := s.cache.Set(tblKey, key.ToPb())
	if pbVal != nil {
		ret = *key
	}
	return
}
