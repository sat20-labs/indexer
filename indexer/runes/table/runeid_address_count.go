package table

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type RuneIdAddressToCount struct {
	RuneId    *runestone.RuneId
	AddressId uint64
	Address   runestone.Address
	Count     uint64
}

func RuneIdAddressToCountFromString(str string) (*RuneIdAddressToCount, error) {
	ret := &RuneIdAddressToCount{}
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
	if !IsLessStorage {
		ret.Address = runestone.Address(parts[3])
	}
	return ret, nil
}

func (s *RuneIdAddressToCount) Key() string {
	ret := s.RuneId.Hex() + "-" + fmt.Sprintf("%x", s.AddressId)
	if !IsLessStorage {
		ret += "-" + string(s.Address)
	}
	return ret
}

func (s *RuneIdAddressToCount) ToPb() *pb.RuneIdAddressToCount {
	pbValue := &pb.RuneIdAddressToCount{
		Count: s.Count,
	}
	return pbValue
}

func (s *RuneIdAddressToCount) FromPb(pbValue *pb.RuneIdAddressToCount) {
	s.Count = pbValue.Count
}

type RuneIdAddressToCountTable struct {
	Table[pb.RuneIdAddressToCount]
}

func NewRuneIdAddressToCountTable(v *store.Cache[pb.RuneIdAddressToCount]) *RuneIdAddressToCountTable {
	return &RuneIdAddressToCountTable{Table: Table[pb.RuneIdAddressToCount]{Cache: v}}
}

func (s *RuneIdAddressToCountTable) Get(v *RuneIdAddressToCount) (ret *RuneIdAddressToCount) {
	tblKey := []byte(store.RUNEID_ADDRESS_TO_COUNT + v.Key())
	pbVal := s.Cache.Get(tblKey)
	if pbVal != nil {
		var err error
		ret, err = RuneIdAddressToCountFromString(string(tblKey))
		if err != nil {
			common.Log.Panicf("RuneIdAddressToCountTable.Get-> RuneIdAddressToCountFromString(%s) err:%v", string(tblKey), err)
			return nil
		}
		ret.Count = pbVal.Count

	}
	return
}

func (s *RuneIdAddressToCountTable) Insert(v *RuneIdAddressToCount) {
	tblKey := []byte(store.RUNEID_ADDRESS_TO_COUNT + v.Key())
	if IsLessStorage {
		v.Address = ""
	}
	s.Cache.Set(tblKey, v.ToPb())
}

func (s *RuneIdAddressToCountTable) Remove(v *RuneIdAddressToCount) (ret *RuneIdAddressToCount) {
	tblKey := []byte(store.RUNEID_ADDRESS_TO_COUNT + v.Key())
	pbVal := s.Cache.Delete(tblKey)
	if pbVal != nil {
		ret = &RuneIdAddressToCount{}
		ret.FromPb(pbVal)
		ret.RuneId = v.RuneId
		ret.AddressId = v.AddressId
		ret.Address = v.Address
	}
	return
}
