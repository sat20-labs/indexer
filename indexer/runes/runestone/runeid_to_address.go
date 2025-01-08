package runestone

import (
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type Address string

type RuneIdToAddress struct {
	RuneId    *RuneId
	AddressId uint64
	Address   Address
}

func RuneIdToAddressFromString(str string) (*RuneIdToAddress, error) {
	ret := &RuneIdToAddress{}
	parts := strings.SplitN(str, "-", 4)
	var err error
	ret.RuneId, err = RuneIdFromHex(parts[1])
	if err != nil {
		return nil, err
	}

	addressId, err := strconv.ParseUint(parts[2], 16, 64)
	if err != nil {
		return nil, err
	}
	ret.AddressId = addressId
	ret.Address = Address(parts[3])
	return ret, nil
}

func (s *RuneIdToAddress) ToPb() *pb.RuneIdToAddress {
	return &pb.RuneIdToAddress{}
}

func (s *RuneIdToAddress) Key() string {
	return s.RuneId.Hex() + "-" + strconv.FormatUint(s.AddressId, 16) + "-" + string(s.Address)
}

type RuneToAddressTable struct {
	Table[pb.RuneIdToAddress]
}

func NewRuneIdToAddressTable(cache *store.Cache[pb.RuneIdToAddress]) *RuneToAddressTable {
	return &RuneToAddressTable{Table: Table[pb.RuneIdToAddress]{cache: cache}}
}

func (s *RuneToAddressTable) GetList(runeId *RuneId) (ret []*RuneIdToAddress, err error) {
	tblKey := []byte(store.RUNEID_TO_ADDRESS + runeId.Hex() + "-")
	pbVal := s.cache.GetList(tblKey, false)

	if pbVal != nil {
		ret = make([]*RuneIdToAddress, len(pbVal))
		var i = 0
		for k := range pbVal {
			var err error
			v, err := RuneIdToAddressFromString(k)
			if err != nil {
				return nil, err
			}

			ret[i] = v
			i++
		}
	}
	return
}

func (s *RuneToAddressTable) Insert(v *RuneIdToAddress) (ret RuneIdToAddress) {
	tblKey := []byte(store.RUNEID_TO_ADDRESS + v.Key())
	pbVal := s.cache.Set(tblKey, v.ToPb())
	if pbVal != nil {
		ret = *v
	}
	return
}
