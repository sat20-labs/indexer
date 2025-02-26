package table

import (
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type AddressRuneIdToMintHistory struct {
	AddressId uint64
	Address   runestone.Address
	RuneId    *runestone.RuneId
	OutPoint  *runestone.OutPoint
}

func AddressRuneIdToMintHistoryFromString(str string) (*AddressRuneIdToMintHistory, error) {
	ret := &AddressRuneIdToMintHistory{}
	parts := strings.SplitN(str, "-", 5)
	var err error
	ret.AddressId, err = strconv.ParseUint(parts[1], 16, 64)
	if err != nil {
		return nil, err
	}
	ret.RuneId, err = runestone.RuneIdFromHex(parts[2])
	if err != nil {
		return nil, err
	}
	ret.OutPoint, err = runestone.OutPointFromString(parts[3])
	if err != nil {
		return nil, err
	}
	if !IsLessStorage {
		ret.Address = runestone.Address(parts[4])
	}
	return ret, nil
}

func (s *AddressRuneIdToMintHistory) ToPb() *pb.AddressRuneIdToMintHistory {
	return &pb.AddressRuneIdToMintHistory{}
}

func (s *AddressRuneIdToMintHistory) Key() (ret string) {
	ret = strconv.FormatUint(s.AddressId, 16) + "-" + s.RuneId.Hex() + "-" + s.OutPoint.Hex()
	if !IsLessStorage {
		ret += "-" + string(s.Address)
	}
	return
}

type AddressRuneIdToMintHistoryTable struct {
	Table[pb.AddressRuneIdToMintHistory]
}

func NewAddressRuneIdToMintHistoryTable(cache *store.Cache[pb.AddressRuneIdToMintHistory]) *AddressRuneIdToMintHistoryTable {
	return &AddressRuneIdToMintHistoryTable{Table: Table[pb.AddressRuneIdToMintHistory]{Cache: cache}}
}

func (s *AddressRuneIdToMintHistoryTable) GetList(addressId uint64, runeId *runestone.RuneId) (ret []*AddressRuneIdToMintHistory, err error) {
	tblKey := []byte(store.ADDRESS_RUNEID_TO_MINT_HISTORYS + strconv.FormatUint(addressId, 16) + "-" + runeId.Hex() + "-")
	pbVal := s.Cache.GetList(tblKey, false)

	if pbVal != nil {
		ret = make([]*AddressRuneIdToMintHistory, len(pbVal))
		var i = 0
		for k := range pbVal {
			v, err := AddressRuneIdToMintHistoryFromString(k)
			if err != nil {
				return nil, err
			}
			ret[i] = v
			i++
		}
	}
	return
}

func (s *AddressRuneIdToMintHistoryTable) Insert(value *AddressRuneIdToMintHistory) (ret AddressRuneIdToMintHistory) {
	tblKey := []byte(store.ADDRESS_RUNEID_TO_MINT_HISTORYS + value.Key())
	pbVal := s.Cache.Set(tblKey, value.ToPb())
	if pbVal != nil {
		ret = *value
	}
	return
}

func (s *AddressRuneIdToMintHistoryTable) Remove(v *AddressRuneIdToMintHistory) {
	tblKey := []byte(store.ADDRESS_RUNEID_TO_MINT_HISTORYS + v.Key())
	s.Cache.Delete(tblKey)
}
