package table

import (
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type Utxo string

type RuneIdToMintHistory struct {
	RuneId    *runestone.RuneId
	Utxo      Utxo
	UtxoId    uint64
	Address   string
	AddressId uint64
}

func RuneIdToMintHistoryFromString(key string) (*RuneIdToMintHistory, error) {
	ret := &RuneIdToMintHistory{}
	parts := strings.SplitN(key, "-", 4)
	var err error
	ret.RuneId, err = runestone.RuneIdFromHex(parts[1])
	if err != nil {
		return nil, err
	}
	utxoId, err := strconv.ParseUint(parts[2], 16, 64)
	if err != nil {
		return nil, err
	}
	ret.UtxoId = utxoId
	if !IsLessStorage {
		ret.Utxo = Utxo(parts[3])
	}
	return ret, nil
}

func (s *RuneIdToMintHistory) ToPb() *pb.RuneIdToMintHistory {
	return &pb.RuneIdToMintHistory{
		Address:   s.Address,
		AddressId: s.AddressId,
	}
}

func (s *RuneIdToMintHistory) Key() (ret string) {
	ret = s.RuneId.Hex() + "-" + strconv.FormatUint(s.UtxoId, 16)
	if !IsLessStorage {
		ret += "-" + string(s.Utxo)
	}
	return
}

type RuneToMintHistoryTable struct {
	Table[pb.RuneIdToMintHistory]
}

func NewRuneIdToMintHistoryTable(store *store.Cache[pb.RuneIdToMintHistory]) *RuneToMintHistoryTable {
	return &RuneToMintHistoryTable{Table: Table[pb.RuneIdToMintHistory]{Cache: store}}
}

func (s *RuneToMintHistoryTable) GetList(runeId *runestone.RuneId) (ret []*RuneIdToMintHistory, err error) {
	tblKey := []byte(store.RUNEID_TO_MINT_HISTORYS + runeId.Hex() + "-")
	pbVal := s.Cache.GetList(tblKey, true)

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

func (s *RuneToMintHistoryTable) Insert(v *RuneIdToMintHistory) (ret RuneIdToMintHistory) {
	tblKey := []byte(store.RUNEID_TO_MINT_HISTORYS + v.Key())
	if IsLessStorage {
		v.Utxo = ""
		v.Address = ""
	}
	pbVal := s.Cache.Set(tblKey, v.ToPb())
	if pbVal != nil {
		ret = *v
	}
	return
}
