package runestone

import (
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type Utxo string

type RuneIdToMintHistory struct {
	RuneId    *RuneId
	Utxo      Utxo
	UtxoId    uint64
	Address   string
	AddressId uint64
}

func RuneIdToMintHistoryFromString(key string) (*RuneIdToMintHistory, error) {
	ret := &RuneIdToMintHistory{}
	parts := strings.SplitN(key, "-", 4)
	var err error
	ret.RuneId, err = RuneIdFromHex(parts[1])
	if err != nil {
		return nil, err
	}
	ret.Utxo = Utxo(parts[2])
	utxoId, err := strconv.ParseUint(parts[3], 16, 64)
	if err != nil {
		return nil, err
	}
	ret.UtxoId = utxoId
	return ret, nil
}

func (s *RuneIdToMintHistory) ToPb() *pb.RuneIdToMintHistory {
	return &pb.RuneIdToMintHistory{
		Address:   s.Address,
		AddressId: s.AddressId,
	}
}

func (s *RuneIdToMintHistory) String() string {
	return s.RuneId.Hex() + "-" + string(s.Utxo) + "-" + strconv.FormatUint(s.UtxoId, 16)
}

type RuneToMintHistoryTable struct {
	Table[pb.RuneIdToMintHistory]
}

func NewRuneIdToMintHistoryTable(store *store.Cache[pb.RuneIdToMintHistory]) *RuneToMintHistoryTable {
	return &RuneToMintHistoryTable{Table: Table[pb.RuneIdToMintHistory]{cache: store}}
}

func (s *RuneToMintHistoryTable) GetList(runeId *RuneId) (ret []*RuneIdToMintHistory, err error) {
	tblKey := []byte(store.RUNEID_TO_MINT_HISTORYS + runeId.Hex() + "-")
	pbVal := s.cache.GetList(tblKey, false)

	if pbVal != nil {
		ret = make([]*RuneIdToMintHistory, len(pbVal))
		var i = 0
		for k, v := range pbVal {
			runeIdToMintHistory, err := RuneIdToMintHistoryFromString(k)
			if err != nil {
				return nil, err
			}
			runeIdToMintHistory.Address = v.Address
			runeIdToMintHistory.AddressId = v.AddressId
			ret[i] = runeIdToMintHistory
			i++
		}
	}
	return
}

func (s *RuneToMintHistoryTable) Insert(key *RuneIdToMintHistory) (ret RuneIdToMintHistory) {
	tblKey := []byte(store.RUNEID_TO_MINT_HISTORYS + key.String())
	pbVal := s.cache.Set(tblKey, key.ToPb())
	if pbVal != nil {
		ret = *key
	}
	return
}
