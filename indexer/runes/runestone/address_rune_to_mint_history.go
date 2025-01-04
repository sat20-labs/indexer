package runestone

import (
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type AddressRuneIdToMintHistory struct {
	Address   Address
	AddressId uint64
	RuneId    *RuneId
	OutPoint  *OutPoint
}

func (s *AddressRuneIdToMintHistory) FromString(key string) {
	parts := strings.SplitN(key, "-", 5)
	s.Address = Address(parts[1])
	var err error
	if s.RuneId == nil {
		s.RuneId = &RuneId{}
	}
	s.RuneId, err = RuneIdFromString(parts[2])
	if err != nil {
		common.Log.Panicf("RuneIdToAddress.FromString-> RuneIdFromString(%s) err:%v", parts[1], err)
	}
	if s.OutPoint == nil {
		s.OutPoint = &OutPoint{}
	}
	err = s.OutPoint.FromString(parts[3])
	if err != nil {
		common.Log.Panicf("RuneIdToAddress.FromString-> OutPoint.FromString(%s) err:%v", parts[2], err)
	}

	addressId, err := strconv.ParseUint(parts[4], 16, 64)
	if err != nil {
		common.Log.Panicf("RuneIdToAddress.FromString-> strconv.ParseUint(%s) err:%v", parts[4], err)
	}
	s.AddressId = addressId
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

func (s *AddressRuneIdToMintHistoryTable) GetUtxos(address Address, runeId *RuneId) (ret []Utxo) {
	tblKey := []byte(store.ADDRESS_RUNEID_TO_MINT_HISTORYS + string(address) + "-" + runeId.String() + "-")
	pbVal := s.cache.GetList(tblKey, false)

	if pbVal != nil {
		ret = make([]Utxo, len(pbVal))
		var i = 0
		for k := range pbVal {
			v := &AddressRuneIdToMintHistory{}
			v.FromString(k)
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
