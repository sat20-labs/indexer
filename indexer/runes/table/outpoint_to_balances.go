package table

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"lukechampine.com/uint128"
)

type OutPoint struct {
	UtxoId uint64
}

func (s *OutPoint) Hex() string {
	return fmt.Sprintf("%x", s.UtxoId)
}

func (s *OutPoint) Key() string {
	return fmt.Sprintf("%x", s.UtxoId)
}

func OutPointFromUtxoId(utxoId uint64) *OutPoint {
	return &OutPoint{UtxoId: utxoId}
}

func OutPointFromString(str string) (*OutPoint, error) {
	outpoint := &OutPoint{}
	utxoId, err := strconv.ParseUint(str, 16, 64)
	if err != nil {
		return nil, err
	}
	outpoint.UtxoId = utxoId
	return outpoint, nil
}

type RuneIdLot struct {
	RuneId runestone.RuneId
	Lot    runestone.Lot
}

type RuneIdLotMap map[runestone.RuneId]*runestone.Lot

func (s RuneIdLotMap) Get(id *runestone.RuneId) *runestone.Lot {
	return s[*id]
}

func (s RuneIdLotMap) GetOrDefault(id *runestone.RuneId) *runestone.Lot {
	key := *id
	if s[key] == nil {
		s[key] = &runestone.Lot{Value: uint128.Uint128{}}
	}
	return s[key]
}

func (s RuneIdLotMap) GetSortArray() (ret []*RuneIdLot) {
	if len(s) == 0 {
		return
	}

	slice := make([]*RuneIdLot, len(s))
	var i = 0
	for k, v := range s {
		slice[i] = &RuneIdLot{RuneId: k, Lot: *v}
		i++
	}
	sort.Slice(slice, func(i, j int) bool {
		return slice[i].RuneId.Block < slice[j].RuneId.Block ||
			(slice[i].RuneId.Block == slice[j].RuneId.Block && slice[i].RuneId.Tx < slice[j].RuneId.Tx) ||
			(slice[i].RuneId.Block == slice[j].RuneId.Block && slice[i].RuneId.Tx == slice[j].RuneId.Tx && slice[i].Lot.Cmp(&slice[j].Lot.Value) < 0)
	})
	return slice
}

type OutpointToBalancesValue struct {
	UtxoId     uint64
	AddressId  uint64
	RuneIdLots []*RuneIdLot
}

func (s *OutpointToBalancesValue) ToPb() *pb.OutpointToBalances {
	pbValue := &pb.OutpointToBalances{
		Value: &pb.OutpointToBalancesValue{
			UtxoId:       s.UtxoId,
			AddressId:  s.AddressId,
			RuneIdLots: make([]*pb.RuneIdLot, len(s.RuneIdLots)),
		},
	}
	for i, runeIdLot := range s.RuneIdLots {
		runeId := &pb.RuneId{
			Block: runeIdLot.RuneId.Block,
			Tx:    runeIdLot.RuneId.Tx,
		}
		lot := &pb.Lot{
			Value: &pb.Uint128{
				Hi: runeIdLot.Lot.Value.Hi,
				Lo: runeIdLot.Lot.Value.Lo,
			},
		}
		pbValue.Value.RuneIdLots[i] = &pb.RuneIdLot{
			RuneId: runeId,
			Lot:    lot,
		}
	}
	return pbValue
}

func (s *OutpointToBalancesValue) FromPb(pbValue *pb.OutpointToBalances) {
	s.UtxoId = pbValue.Value.UtxoId
	s.AddressId = pbValue.Value.AddressId
	for _, pbRuneIdLot := range pbValue.Value.RuneIdLots {
		runeId := runestone.RuneId{
			Block: pbRuneIdLot.RuneId.Block,
			Tx:    pbRuneIdLot.RuneId.Tx,
		}
		lot := runestone.Lot{
			Value: uint128.Uint128{
				Hi: pbRuneIdLot.Lot.Value.Hi,
				Lo: pbRuneIdLot.Lot.Value.Lo,
			},
		}
		s.RuneIdLots = append(s.RuneIdLots, &RuneIdLot{
			RuneId: runeId,
			Lot:    lot,
		})
	}
}

type OutpointToBalancesTable struct {
	Table[pb.OutpointToBalances]
}

func NewOutpointToBalancesTable(s *store.Cache[pb.OutpointToBalances]) *OutpointToBalancesTable {
	return &OutpointToBalancesTable{Table: Table[pb.OutpointToBalances]{Cache: s}}
}

func (s *OutpointToBalancesTable) Get(key *OutPoint) (ret OutpointToBalancesValue) {
	tblKey := []byte(store.OUTPOINT_TO_BALANCES + key.Key())
	pbVal := s.Cache.Get(tblKey)
	if pbVal != nil {
		ret = OutpointToBalancesValue{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *OutpointToBalancesTable) Insert(key *OutPoint, value *OutpointToBalancesValue) (ret *OutpointToBalancesValue) {
	tblKey := []byte(store.OUTPOINT_TO_BALANCES + key.Key())
	pbVal := s.Cache.Set(tblKey, value.ToPb())
	if pbVal != nil {
		ret = &OutpointToBalancesValue{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *OutpointToBalancesTable) Remove(key *OutPoint) (ret *OutpointToBalancesValue) {
	tblKey := []byte(store.OUTPOINT_TO_BALANCES + key.Key())
	pbVal := s.Cache.Delete(tblKey)
	if pbVal != nil {
		ret = &OutpointToBalancesValue{}
		ret.FromPb(pbVal)
	}
	return
}
