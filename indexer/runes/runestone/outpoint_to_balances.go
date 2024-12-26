package runestone

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"lukechampine.com/uint128"
)

type OutPoint struct {
	Txid string
	Vout uint32
}

func (s *OutPoint) String() string {
	return fmt.Sprintf("%s:%d", s.Txid, s.Vout)
}

func (s *OutPoint) From(str string) error {
	parts := strings.Split(str, ":")
	if len(parts) != 2 {
		return errors.New("invalid format: expected 'txid:vout'")
	}
	s.Txid = parts[0]
	vout, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return fmt.Errorf("invalid vout: %v", err)
	}
	s.Vout = uint32(vout)
	return nil
}

type RuneIdLot struct {
	RuneId RuneId
	Lot    Lot
}

type RuneIdLotMap map[RuneId]*Lot

func (s RuneIdLotMap) Get(id *RuneId) *Lot {
	return s[*id]
}

func (s RuneIdLotMap) GetOrDefault(id *RuneId) *Lot {
	key := *id
	if s[key] == nil {
		s[key] = &Lot{Value: &uint128.Uint128{}}
	}
	return s[key]
}

func (s RuneIdLotMap) GetSortArray() OutpointToRuneBalances {
	slice := make(OutpointToRuneBalances, len(s))
	var i = 0
	for k, v := range s {
		slice[i] = RuneIdLot{RuneId: k, Lot: *v}
		i++
	}
	sort.Slice(slice, func(i, j int) bool {
		return slice[i].RuneId.Block < slice[j].RuneId.Block ||
			(slice[i].RuneId.Block == slice[j].RuneId.Block && slice[i].RuneId.Tx < slice[j].RuneId.Tx) ||
			(slice[i].RuneId.Block == slice[j].RuneId.Block && slice[i].RuneId.Tx == slice[j].RuneId.Tx && slice[i].Lot.Cmp(slice[j].Lot.Value) < 0)
	})
	return slice
}

type OutpointToRuneBalances []RuneIdLot

func (s *OutpointToRuneBalances) ToPb() *pb.OutpointToRuneBalances {
	pbValue := &pb.OutpointToRuneBalances{
		RuneIdLots: make([]*pb.RuneIdLot, len(*s)),
	}
	for i, runeIdLot := range *s {
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
		pbValue.RuneIdLots[i] = &pb.RuneIdLot{
			RuneId: runeId,
			Lot:    lot,
		}
	}
	return pbValue
}

func (s *OutpointToRuneBalances) FromPb(pbValue *pb.OutpointToRuneBalances) {
	for _, pbRuneIdLot := range pbValue.RuneIdLots {
		runeId := RuneId{
			Block: pbRuneIdLot.RuneId.Block,
			Tx:    pbRuneIdLot.RuneId.Tx,
		}
		lot := Lot{
			Value: &uint128.Uint128{
				Hi: pbRuneIdLot.Lot.Value.Hi,
				Lo: pbRuneIdLot.Lot.Value.Lo,
			},
		}
		*s = append(*s, RuneIdLot{
			RuneId: runeId,
			Lot:    lot,
		})
	}
}

type OutpointToRuneBalancesTable struct {
	Table[pb.OutpointToRuneBalances]
}

func NewOutpointToRuneBalancesTable(s *store.Cache[pb.OutpointToRuneBalances]) *OutpointToRuneBalancesTable {
	return &OutpointToRuneBalancesTable{Table: Table[pb.OutpointToRuneBalances]{cache: s}}
}

func (s *OutpointToRuneBalancesTable) Get(key *OutPoint) (ret *OutpointToRuneBalances) {
	tblKey := []byte(store.OUTPOINT_TO_BALANCES + key.String())
	pbVal := s.cache.Get(tblKey)
	if pbVal != nil {
		ret = &OutpointToRuneBalances{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *OutpointToRuneBalancesTable) GetFromDB(key *OutPoint) (ret *OutpointToRuneBalances) {
	tblKey := []byte(store.OUTPOINT_TO_BALANCES + key.String())
	pbVal, _ := s.cache.GetFromDB(tblKey)
	if pbVal != nil {
		ret = &OutpointToRuneBalances{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *OutpointToRuneBalancesTable) Insert(key *OutPoint, value OutpointToRuneBalances) (ret *OutpointToRuneBalances) {
	tblKey := []byte(store.OUTPOINT_TO_BALANCES + key.String())
	pbVal := s.cache.Insert(tblKey, value.ToPb())
	if pbVal != nil {
		ret = &OutpointToRuneBalances{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *OutpointToRuneBalancesTable) Remove(key *OutPoint) (ret *OutpointToRuneBalances) {
	tblKey := []byte(store.OUTPOINT_TO_BALANCES + key.String())
	pbVal := s.cache.Remove(tblKey)
	if pbVal != nil {
		ret = &OutpointToRuneBalances{}
		ret.FromPb(pbVal)
	}
	return
}
