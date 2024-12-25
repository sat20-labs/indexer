package runestone

import (
	"lukechampine.com/uint128"
)

type Terms struct {
	Amount *uint128.Uint128
	Cap    *uint128.Uint128
	Height [2]*uint64
	Offset [2]*uint64
}

type Etching struct {
	Divisibility *uint8
	Premine      *uint128.Uint128
	Rune         *Rune
	Spacers      *uint32
	Symbol       *rune
	Terms        *Terms
	Turbo        bool
}

const (
	MaxDivisibility = 38
	MaxSpacers      = 0b00000111_11111111_11111111_11111111
)

func (e *Etching) Supply() *uint128.Uint128 {
	//cover panic
	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()
	premine := uint128.Uint128{}
	if e.Premine != nil {
		premine = *e.Premine
	}

	cap := uint128.Uint128{}
	amount := uint128.Uint128{}
	if e.Terms != nil {
		if e.Terms.Cap != nil {
			cap = *e.Terms.Cap
		}
		if e.Terms.Amount != nil {
			amount = *e.Terms.Amount
		}
	}

	supply := cap.Mul(amount)
	supply = supply.Add(premine)

	return &supply
}
