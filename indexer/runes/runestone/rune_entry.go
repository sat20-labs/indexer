package runestone

import (
	"math"

	"lukechampine.com/uint128"
)

type InscriptionId string

type RuneEntry struct {
	RuneId       RuneId
	Burned       uint128.Uint128
	Divisibility uint8
	Etching      [32]byte // Txid
	Parent       InscriptionId
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

func (re *RuneEntry) Mintable(height uint64) (uint128.Uint128, error) {
	if re.Terms == nil {
		return uint128.Zero, MintErrorUnmintable
	}

	if start := re.Start(); start != nil && height < *start {
		return uint128.Zero, MintErrorStart
	}

	if end := re.End(); end != nil && height >= *end {
		return uint128.Zero, MintErrorEnd
	}

	cap := uint128.Zero
	if re.Terms.Cap != nil {
		cap = *re.Terms.Cap
	}

	if re.Mints.Cmp(cap) >= 0 {
		return uint128.Zero, MintErrorCap
	}

	amount := uint128.Zero
	if re.Terms.Amount != nil {
		amount = *re.Terms.Amount
	}

	return amount, nil
}

func (re *RuneEntry) Supply() uint128.Uint128 {
	amount := uint128.Zero
	if re.Terms != nil && re.Terms.Amount != nil {
		amount = *re.Terms.Amount
	}
	return re.Premine.Add(re.Mints.Mul(amount))
}

func (re *RuneEntry) MaxSupply() uint128.Uint128 {
	cap := uint128.Zero
	amount := uint128.Zero
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
		if re.RuneId.Block <= math.MaxUint64-*re.Terms.Offset[0] {
			relVal := re.RuneId.Block + *re.Terms.Offset[0]
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
