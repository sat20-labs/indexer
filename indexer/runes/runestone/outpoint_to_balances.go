package runestone

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"lukechampine.com/uint128"
)

type OutPoint struct {
	Txid   string
	Vout   uint32
	UtxoId uint64
}

func (s *OutPoint) Hex() string {
	return fmt.Sprintf("%s:%x:%x", s.Txid, s.Vout, s.UtxoId)
}

func (s *OutPoint) String() string {
	return fmt.Sprintf("%s:%d:%d", s.Txid, s.Vout, s.UtxoId)
}

func (s *OutPoint) Utxo() string {
	return fmt.Sprintf("%s:%d", s.Txid, s.Vout)
}

func (s *OutPoint) Key() string {
	return fmt.Sprintf("%x", s.UtxoId)
}

func OutPointFromUtxoId(utxoId uint64) *OutPoint {
	return &OutPoint{UtxoId: utxoId}
}

func OutPointFromUtxo(utxo string, utxoId uint64) (*OutPoint, error) {
	outpoint := &OutPoint{}
	parts := strings.Split(utxo, ":")
	outpoint.Txid = parts[0]
	vout, err := strconv.ParseUint(parts[1], 16, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid vout: %v", err)
	}
	outpoint.Vout = uint32(vout)
	outpoint.UtxoId = utxoId
	return outpoint, nil
}

func OutPointFromHex(str string) (*OutPoint, error) {
	outpoint := &OutPoint{}
	parts := strings.Split(str, ":")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid outpoint format")
	}
	outpoint.Txid = parts[0]
	vout, err := strconv.ParseUint(parts[1], 16, 32)
	if err != nil {
		return nil, err
	}
	outpoint.Vout = uint32(vout)
	utxoId, err := strconv.ParseUint(parts[2], 16, 64)
	if err != nil {
		return nil, err
	}
	outpoint.UtxoId = utxoId
	return outpoint, nil
}

func OutPointFromString(str string) (*OutPoint, error) {
	outpoint := &OutPoint{}
	parts := strings.Split(str, ":")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid outpoint format")
	}
	outpoint.Txid = parts[0]
	vout, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return nil, err
	}
	outpoint.Vout = uint32(vout)
	utxoId, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		return nil, err
	}
	outpoint.UtxoId = utxoId
	return outpoint, nil
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
			(slice[i].RuneId.Block == slice[j].RuneId.Block && slice[i].RuneId.Tx == slice[j].RuneId.Tx && slice[i].Lot.Cmp(slice[j].Lot.Value) < 0)
	})
	return slice
}

type OutpointToBalancesValue struct {
	Utxo       string
	RuneIdLots []*RuneIdLot
}

// type OutpointToBalancesValue []RuneIdLot

func (s *OutpointToBalancesValue) ToPb() *pb.OutpointToBalances {
	pbValue := &pb.OutpointToBalances{
		Value: &pb.OutpointToBalancesValue{
			Utxo:       s.Utxo,
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
	s.Utxo = pbValue.Value.Utxo
	for _, pbRuneIdLot := range pbValue.Value.RuneIdLots {
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
	return &OutpointToBalancesTable{Table: Table[pb.OutpointToBalances]{cache: s}}
}

func (s *OutpointToBalancesTable) Get(key *OutPoint) (ret OutpointToBalancesValue) {
	tblKey := []byte(store.OUTPOINT_TO_BALANCES + key.Key())
	pbVal := s.cache.Get(tblKey)
	if pbVal != nil {
		ret = OutpointToBalancesValue{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *OutpointToBalancesTable) Insert(key *OutPoint, value *OutpointToBalancesValue) (ret *OutpointToBalancesValue) {
	if key.Txid == "27d547acffd10a4a47f18f5f97aaa71e276e826e7fb076003a14404d060295c0" {
		common.Log.Debugf("RuneIndexer.InsertOutpointToBalances-> key.Txid is empty")
	}
	tblKey := []byte(store.OUTPOINT_TO_BALANCES + key.Key())
	pbVal := s.cache.Set(tblKey, value.ToPb())
	if pbVal != nil {
		ret = &OutpointToBalancesValue{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *OutpointToBalancesTable) Remove(key *OutPoint) (ret *OutpointToBalancesValue) {
	if key.Txid == "27d547acffd10a4a47f18f5f97aaa71e276e826e7fb076003a14404d060295c0" {
		common.Log.Debugf("RuneIndexer.InsertOutpointToBalances-> key.Txid is empty")
	}
	tblKey := []byte(store.OUTPOINT_TO_BALANCES + key.Key())
	pbVal := s.cache.Delete(tblKey)
	if pbVal != nil {
		ret = &OutpointToBalancesValue{}
		ret.FromPb(pbVal)
	}
	return
}
