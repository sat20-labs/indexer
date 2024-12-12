package runestone

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/indexer/runes/db"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"lukechampine.com/uint128"
)

// type OutpointToBalances map[*OutpointToBalanceKey]OutpointToBalanceValue

type OutpointToBalanceKey struct {
	Txid string
	Vout uint32
}

func (s *OutpointToBalanceKey) String() string {
	return fmt.Sprintf("%s:%d", s.Txid, s.Vout)
}

func (s *OutpointToBalanceKey) From(str string) error {
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

type RuneIdLotMap map[RuneIdKey]*Lot

func (s RuneIdLotMap) GetOrDefault(id *RuneId) *Lot {
	key := id.ToByte()
	if s[key] == nil {
		s[key] = &Lot{Value: uint128.Zero}
	}
	return s[key]
}

type RuneIdLogMapVec map[uint32]RuneIdLotMap

type OutpointToBalanceValue []RuneIdLot

func (s OutpointToBalanceValue) ToPb() *pb.OutpointToBalanceValue {
	pbValue := &pb.OutpointToBalanceValue{
		RuneIdLots: make([]*pb.RuneIdLot, len(s)),
	}
	for i, runeIdLot := range s {
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

func (s OutpointToBalanceValue) FromPb(pbValue *pb.OutpointToBalanceValue) {
	s = make(OutpointToBalanceValue, len(pbValue.RuneIdLots))
	for i, pbRuneIdLot := range pbValue.RuneIdLots {
		runeId := RuneId{
			Block: pbRuneIdLot.RuneId.Block,
			Tx:    pbRuneIdLot.RuneId.Tx,
		}
		lot := Lot{
			Value: uint128.Uint128{
				Hi: pbRuneIdLot.Lot.Value.Hi,
				Lo: pbRuneIdLot.Lot.Value.Lo,
			},
		}
		s[i] = RuneIdLot{
			RuneId: runeId,
			Lot:    lot,
		}
	}
}

type OutpointToBalancesTable struct {
}

func (s OutpointToBalancesTable) Insert(key *OutpointToBalanceKey, value OutpointToBalanceValue) (oldValue *OutpointToBalanceValue, err error) {
	tableKey := []byte(db.OUTPOINT_TO_BALANCES_KEY + key.String())
	oldPbValue, err := db.Get[pb.OutpointToBalanceValue](tableKey)
	if err != nil {
		return nil, err
	}
	oldValue.FromPb(oldPbValue)
	pbValue := value.ToPb()
	err = db.Set(tableKey, pbValue)
	return
}

func (s OutpointToBalancesTable) Remove(key *OutpointToBalanceKey) (oldValue OutpointToBalanceValue, err error) {
	tableKey := []byte(db.OUTPOINT_TO_BALANCES_KEY + key.String())
	pbOldValue, err := db.Remove[pb.OutpointToBalanceValue](tableKey)
	if err != nil {
		return nil, err
	}
	oldValue.FromPb(pbOldValue)
	return
}
