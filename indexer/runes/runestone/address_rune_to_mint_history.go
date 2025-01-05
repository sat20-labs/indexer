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
	ret.Address = Address(parts[1])
	var err error
	ret.RuneId, err = RuneIdFromString(parts[2])
	if err != nil {
		return nil, err
	}
	ret.OutPoint, err = OutPointFromString(parts[3])
	if err != nil {
		return nil, err
	}
	addressId, err := strconv.ParseUint(parts[4], 16, 64)
	if err != nil {
		return nil, err
	}
	ret.AddressId = addressId
	return ret, nil
}

func (s *AddressRuneIdToMintHistory) ToPb() *pb.AddressRuneIdToMintHistory {
	return &pb.AddressRuneIdToMintHistory{}
}

func (s *AddressRuneIdToMintHistory) String() string {
	return string(s.Address) + "-" + s.RuneId.String() + "-" + s.OutPoint.String() + "-" + strconv.FormatUint(s.AddressId, 16)
}

type AddressRuneIdToMintHistoryTable struct {
	Table[pb.AddressRuneIdToMintHistory]
}

func NewAddressRuneIdToMintHistoryTable(cache *store.Cache[pb.AddressRuneIdToMintHistory]) *AddressRuneIdToMintHistoryTable {
	return &AddressRuneIdToMintHistoryTable{Table: Table[pb.AddressRuneIdToMintHistory]{cache: cache}}
}

func (s *AddressRuneIdToMintHistoryTable) GetUtxos(address Address, runeId *RuneId) (ret []Utxo, err error) {
	tblKey := []byte(store.ADDRESS_RUNEID_TO_MINT_HISTORYS + string(address) + "-" + runeId.String() + "-")
	pbVal := s.cache.GetList(tblKey, false)

	if pbVal != nil {
		ret = make([]Utxo, len(pbVal))
		var i = 0
		for k := range pbVal {
			v, err := AddressRuneIdToMintHistoryFromString(k)
			if err != nil {
				return nil, err
			}
			ret[i] = Utxo(v.OutPoint.String())
			i++
		}
	}
	return
}

func (s *AddressRuneIdToMintHistoryTable) Insert(value *AddressRuneIdToMintHistory) (ret AddressRuneIdToMintHistory) {
	tblKey := []byte(store.ADDRESS_RUNEID_TO_MINT_HISTORYS + value.String())
	pbVal := s.cache.Set(tblKey, value.ToPb())
	if pbVal != nil {
		ret = *value
	}
	return
}
