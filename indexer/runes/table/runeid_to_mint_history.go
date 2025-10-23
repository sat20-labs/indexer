package table

import (
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"lukechampine.com/uint128"
)


type RuneIdToMintHistory struct {
	RuneId    *runestone.RuneId
	UtxoId    uint64
	AddressId uint64
	Amount    runestone.Lot
}

func RuneIdToMintHistoryFromString(key string) (*RuneIdToMintHistory, error) {
	ret := &RuneIdToMintHistory{}
	parts := strings.SplitN(key, "-", 4)
	var err error
	ret.RuneId, err = runestone.RuneIdFromHex(parts[1])
	if err != nil {
		return nil, err
	}
	addressId, err := strconv.ParseUint(parts[2], 16, 64)
	if err != nil {
		return nil, err
	}
	ret.AddressId = addressId
	return ret, nil
}

func (s *RuneIdToMintHistory) ToPb() *pb.RuneIdToMintHistory {
	return &pb.RuneIdToMintHistory{
		UtxoId: s.UtxoId,
		Amount: &pb.Lot{
			Value: &pb.Uint128{
				Hi: s.Amount.Value.Hi,
				Lo: s.Amount.Value.Lo,
			},
		},
	}
}

func (s *RuneIdToMintHistory) Key() (ret string) {
	ret = s.RuneId.Hex() + "-" + strconv.FormatUint(s.AddressId, 16)
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
			runeIdToMintHistory.UtxoId = v.UtxoId
			runeIdToMintHistory.Amount = runestone.Lot{
				Value: uint128.Uint128{
						Hi: v.Amount.Value.Hi,
						Lo: v.Amount.Value.Lo,
					},
				}
			ret[i] = runeIdToMintHistory
			i++
		}
	}
	return
}

func (s *RuneToMintHistoryTable) Insert(v *RuneIdToMintHistory) (ret RuneIdToMintHistory) {
	tblKey := []byte(store.RUNEID_TO_MINT_HISTORYS + v.Key())
	pbVal := s.Cache.Set(tblKey, v.ToPb())
	if pbVal != nil {
		ret = *v
	}
	return
}
