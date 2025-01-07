package runestone

import (
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type AddressRuneIdToMintHistory struct {
	Address   Address
	AddressId uint64
	RuneId    *RuneId
	OutPoint  *OutPoint
}

func AddressRuneIdToMintHistoryFromString(str string) (*AddressRuneIdToMintHistory, error) {
	ret := &AddressRuneIdToMintHistory{}
	parts := strings.SplitN(str, "-", 5)
	var err error
	ret.AddressId, err = strconv.ParseUint(parts[1], 16, 64)
	if err != nil {
		return nil, err
	}
	ret.RuneId, err = RuneIdFromHex(parts[2])
	if err != nil {
		return nil, err
	}
	ret.OutPoint, err = OutPointFromHex(parts[3])
	if err != nil {
		return nil, err
	}
	ret.Address = Address(parts[1])

	return ret, nil
}

func (s *AddressRuneIdToMintHistory) ToPb() *pb.AddressRuneIdToMintHistory {
	return &pb.AddressRuneIdToMintHistory{}
}

func (s *AddressRuneIdToMintHistory) Key() string {
	return strconv.FormatUint(s.AddressId, 16) + "-" + s.RuneId.Hex() + "-" + s.OutPoint.Hex() + "-" + string(s.Address)
}

type AddressRuneIdToMintHistoryTable struct {
	Table[pb.AddressRuneIdToMintHistory]
}

func NewAddressRuneIdToMintHistoryTable(cache *store.Cache[pb.AddressRuneIdToMintHistory]) *AddressRuneIdToMintHistoryTable {
	return &AddressRuneIdToMintHistoryTable{Table: Table[pb.AddressRuneIdToMintHistory]{cache: cache}}
}

func (s *AddressRuneIdToMintHistoryTable) GetList(addressId uint64, runeId *RuneId) (ret []*AddressRuneIdToMintHistory, err error) {
	tblKey := []byte(store.ADDRESS_RUNEID_TO_MINT_HISTORYS + strconv.FormatUint(addressId, 16) + "-" + runeId.Hex() + "-")
	pbVal := s.cache.GetList(tblKey, false)

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
	pbVal := s.cache.Set(tblKey, value.ToPb())
	if pbVal != nil {
		ret = *value
	}
	return
}
