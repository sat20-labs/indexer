package runestone

import (
	"math"

	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"lukechampine.com/uint128"
)

type InscriptionId string

type RuneEntry struct {
	RuneId       RuneId
	Burned       uint128.Uint128
	Divisibility uint8
	Etching      string // Txid
	Parent       *InscriptionId
	Mints        uint128.Uint128
	Number       uint64
	Premine      uint128.Uint128
	SpacedRune   SpacedRune
	Symbol       *rune
	Terms        *Terms
	Timestamp    uint64
	Turbo        bool
}

type MintError int

const (
	MintErrorUnmintable MintError = iota
	MintErrorStart
	MintErrorEnd
	MintErrorCap
)

func (e MintError) Error() string {
	switch e {
	case MintErrorUnmintable:
		return "Unmintable"
	case MintErrorStart:
		return "Start error"
	case MintErrorEnd:
		return "End error"
	case MintErrorCap:
		return "Cap reached"
	default:
		return "Unknown error"
	}
}

func (re *RuneEntry) Mintable(height uint64) (*uint128.Uint128, error) {
	if re.Terms == nil {
		return &uint128.Uint128{}, MintErrorUnmintable
	}

	if start := re.Start(); start != nil && height < *start {
		return &uint128.Uint128{}, MintErrorStart
	}

	if end := re.End(); end != nil && height >= *end {
		return &uint128.Uint128{}, MintErrorEnd
	}

	cap := uint128.Uint128{}
	if re.Terms.Cap != nil {
		cap = *re.Terms.Cap
	}

	if re.Mints.Cmp(cap) >= 0 {
		return &uint128.Uint128{}, MintErrorCap
	}

	amount := uint128.Uint128{}
	if re.Terms.Amount != nil {
		amount = *re.Terms.Amount
	}

	return &amount, nil
}

func (re *RuneEntry) Supply() uint128.Uint128 {
	amount := uint128.Uint128{}
	if re.Terms != nil && re.Terms.Amount != nil {
		amount = *re.Terms.Amount
	}
	return re.Premine.Add(re.Mints.Mul(amount))
}

func (re *RuneEntry) MaxSupply() uint128.Uint128 {
	cap := uint128.Uint128{}
	amount := uint128.Uint128{}
	if re.Terms != nil {
		if re.Terms.Cap != nil {
			cap = *re.Terms.Cap
		}
		if re.Terms.Amount != nil {
			amount = *re.Terms.Amount
		}
	}
	return re.Premine.Add(cap.Mul(amount))
}

func (re *RuneEntry) Pile(amount uint128.Uint128) Pile {
	return Pile{
		Amount:       amount,
		Divisibility: re.Divisibility,
		Symbol:       re.Symbol,
	}
}

func (re *RuneEntry) Start() *uint64 {
	if re.Terms == nil {
		return nil
	}

	var relative, absolute *uint64
	if re.Terms.Offset[0] != nil {
		if re.RuneId.Block <= math.MaxUint64-*re.Terms.Offset[0] {
			relVal := re.RuneId.Block + *re.Terms.Offset[0]
			relative = &relVal
		} else {
			maxVal := uint64(math.MaxUint64)
			relative = &maxVal
		}
	}

	absolute = re.Terms.Height[0]
	if relative != nil && absolute != nil {
		if *relative > *absolute {
			return relative
		}
		return absolute
	} else if relative != nil {
		return relative
	}
	return absolute
}

func (re *RuneEntry) End() *uint64 {
	if re.Terms == nil {
		return nil
	}

	var relative, absolute *uint64
	if re.Terms.Offset[1] != nil {
		if re.RuneId.Block <= math.MaxUint64-*re.Terms.Offset[1] {
			relVal := re.RuneId.Block + *re.Terms.Offset[1]
			relative = &relVal
		} else {
			maxVal := uint64(math.MaxUint64)
			relative = &maxVal
		}
	}

	absolute = re.Terms.Height[1]
	if relative != nil && absolute != nil {
		if *relative < *absolute {
			return relative
		}
		return absolute
	} else if relative != nil {
		return relative
	}
	return absolute
}

func (s *RuneEntry) ToPb() *pb.RuneEntry {
	pbValue := &pb.RuneEntry{
		RuneId: &pb.RuneId{
			Block: s.RuneId.Block,
			Tx:    s.RuneId.Tx,
		},
		Burned:       &pb.Uint128{Hi: s.Burned.Hi, Lo: s.Burned.Lo},
		Divisibility: &pb.Uint8{Value: uint32(s.Divisibility)},
		Etching:      s.Etching,
		Mints:        &pb.Uint128{Hi: s.Mints.Hi, Lo: s.Mints.Lo},
		Number:       s.Number,
		Premine:      &pb.Uint128{Hi: s.Premine.Hi, Lo: s.Premine.Lo},
		SpacedRune: &pb.SpacedRune{
			Rune:    &pb.Rune{Value: &pb.Uint128{Hi: s.SpacedRune.Rune.Value.Hi, Lo: s.SpacedRune.Rune.Value.Lo}},
			Spacers: s.SpacedRune.Spacers,
		},
		Timestamp: s.Timestamp,
		Turbo:     s.Turbo,
	}

	if s.Parent == nil {
		pbValue.Parent = nil
	} else {
		pbValue.Parent = &pb.InscriptionId{Value: string(*s.Parent)}
	}

	// set symbol
	if s.Symbol != nil {
		pbValue.Symbol = &pb.Symbol{Value: int32(*s.Symbol)}
	}

	// set terms
	if s.Terms != nil {
		pbValue.Terms = &pb.Terms{}
		if s.Terms.Cap != nil {
			pbValue.Terms.Cap = &pb.Uint128{Hi: s.Terms.Cap.Hi, Lo: s.Terms.Cap.Lo}
		}
		if s.Terms.Amount != nil {
			pbValue.Terms.Amount = &pb.Uint128{Hi: s.Terms.Amount.Hi, Lo: s.Terms.Amount.Lo}
		}
		if s.Terms.Height[0] != nil {
			pbValue.Terms.StartHeight = *s.Terms.Height[0]
		}
		if s.Terms.Height[1] != nil {
			pbValue.Terms.EndHeight = *s.Terms.Height[1]
		}
		if s.Terms.Offset[0] != nil {
			pbValue.Terms.StartOffset = *s.Terms.Offset[0]
		}
		if s.Terms.Offset[1] != nil {
			pbValue.Terms.EndOffset = *s.Terms.Offset[1]
		}
	}
	return pbValue
}

func (s *RuneEntry) FromPb(pbValue *pb.RuneEntry) {
	s.RuneId.Block = pbValue.RuneId.Block
	s.RuneId.Tx = pbValue.RuneId.Tx
	s.Burned = uint128.Uint128{Hi: pbValue.Burned.Hi, Lo: pbValue.Burned.Lo}
	s.Divisibility = uint8(pbValue.Divisibility.Value)
	s.Etching = pbValue.Etching
	parent := InscriptionId(pbValue.Parent.Value)
	s.Parent = &parent
	s.Mints = uint128.Uint128{Hi: pbValue.Mints.Hi, Lo: pbValue.Mints.Lo}
	s.Number = pbValue.Number
	s.Premine = uint128.Uint128{Hi: pbValue.Premine.Hi, Lo: pbValue.Premine.Lo}
	s.SpacedRune.Rune.Value = uint128.Uint128{
		Lo: pbValue.SpacedRune.Rune.Value.Lo,
		Hi: pbValue.SpacedRune.Rune.Value.Hi,
	}
	s.SpacedRune.Spacers = pbValue.SpacedRune.Spacers
	s.Timestamp = pbValue.Timestamp
	s.Turbo = pbValue.Turbo

	// set Symbol
	if pbValue.Symbol != nil {
		s.Symbol = &pbValue.Symbol.Value
	} else {
		s.Symbol = nil
	}

	// set Terms
	if pbValue.Terms != nil {
		if s.Terms == nil {
			s.Terms = &Terms{}
		}
		if pbValue.Terms.Cap != nil {
			s.Terms.Cap = &uint128.Uint128{Hi: pbValue.Terms.Cap.Hi, Lo: pbValue.Terms.Cap.Lo}
		}
		if pbValue.Terms.Amount != nil {
			s.Terms.Amount = &uint128.Uint128{Hi: pbValue.Terms.Amount.Hi, Lo: pbValue.Terms.Amount.Lo}
		}
		if pbValue.Terms.StartHeight != 0 {
			startHeight := pbValue.Terms.StartHeight
			s.Terms.Height[0] = &startHeight
		}
		if pbValue.Terms.EndHeight != 0 {
			endHeight := pbValue.Terms.EndHeight
			s.Terms.Height[1] = &endHeight
		}
		if pbValue.Terms.StartOffset != 0 {
			startOffset := pbValue.Terms.StartOffset
			s.Terms.Offset[0] = &startOffset
		}
		if pbValue.Terms.EndOffset != 0 {
			endOffset := pbValue.Terms.EndOffset
			s.Terms.Offset[1] = &endOffset
		}
	} else {
		s.Terms = nil
	}
}
