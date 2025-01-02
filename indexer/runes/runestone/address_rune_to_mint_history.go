package runestone

import (
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type AddressRuneIdToMintHistory struct {
	Address  Address
	RuneId   *RuneId
	OutPoint *OutPoint
}

func (s *AddressRuneIdToMintHistory) FromString(key string) {
	parts := strings.SplitN(key, "-", 3)
	s.Address = Address(parts[0])
	var err error
	s.RuneId, err = RuneIdFromString(parts[1])
	if err != nil {
		common.Log.Panicf("RuneIdToAddress.FromString-> RuneIdFromString(%s) err:%v", parts[1], err)
	}
	err = s.OutPoint.FromString(parts[2])
	if err != nil {
		common.Log.Panicf("RuneIdToAddress.FromString-> OutPoint.FromString(%s) err:%v", parts[2], err)
	}
}

func (s *AddressRuneIdToMintHistory) ToPb() *pb.AddressRuneIdToMintHistory {
	return &pb.AddressRuneIdToMintHistory{}
}

func (s *AddressRuneIdToMintHistory) String() string {
	return string(s.Address) + s.RuneId.String() + "-" + s.OutPoint.String()
}

type AddressRuneIdToMintHistoryTable struct {
	Table[pb.AddressRuneIdToMintHistory]
}

func NewAddressRuneIdToMintHistoryTable(cache *store.Cache[pb.AddressRuneIdToMintHistory]) *AddressRuneIdToMintHistoryTable {
	return &AddressRuneIdToMintHistoryTable{Table: Table[pb.AddressRuneIdToMintHistory]{cache: cache}}
}

func (s *AddressRuneIdToMintHistoryTable) GetUtxosFromDB(address Address, runeId *RuneId) (ret []Utxo) {
	tblKey := []byte(store.ADDRESS_RUNEID_TO_MINT_HISTORYS + string(address) + "-" + runeId.String() + "-")
	pbVal := s.cache.GetListFromDB(tblKey, false)

	if pbVal != nil {
		ret = make([]Utxo, 0)
		for k := range pbVal {
			v := &RuneIdToMintHistory{}
			v.FromString(k)
			ret = append(ret, v.Utxo)
		}
	}
	return
}

func (s *AddressRuneIdToMintHistoryTable) Insert(key *AddressRuneIdToMintHistory) (ret AddressRuneIdToMintHistory) {
	tblKey := []byte(store.ADDRESS_RUNEID_TO_MINT_HISTORYS + key.String())
	pbVal := s.cache.Set(tblKey, key.ToPb())
	if pbVal != nil {
		ret = *key
	}
	return
}
