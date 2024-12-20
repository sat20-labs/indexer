package runestone

import (
	"errors"
	"fmt"

	"lukechampine.com/uint128"
)

type Pile struct {
	Amount       uint128.Uint128
	Divisibility uint8
	Symbol       *rune
}

var ErrDivisibilityTooLarge = errors.New("divisibility is too large, causing overflow")

func (p Pile) String() string {
	cutoff, err := calculateCutoff(p.Divisibility)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	whole := p.Amount.Div(cutoff)
	fractional := p.Amount.Mod(cutoff)

	var result string
	if fractional.IsZero() {
		result = whole.String()
	} else {
		fractionalStr := fmt.Sprintf("%0*s", p.Divisibility, fractional.String())
		width := int(p.Divisibility)
		for fractional.Mod(uint128.From64(10)).IsZero() {
			fractional = fractional.Div(uint128.From64(10))
			width--
		}
		fractionalStr = fractionalStr[:width]
		result = fmt.Sprintf("%s.%s", whole.String(), fractionalStr)
	}

	symbol := '¤'
	if p.Symbol != nil {
		symbol = *p.Symbol
	}

	return fmt.Sprintf("%s\u00A0%c", result, symbol)
}

func calculateCutoff(divisibility uint8) (uint128.Uint128, error) {
	if divisibility >= 39 {
		return uint128.Uint128{}, ErrDivisibilityTooLarge
	}

	cutoff := uint128.From64(1)
	ten := uint128.From64(10)
	for i := uint8(0); i < divisibility; i++ {
		cutoff = cutoff.Mul(ten)
		if cutoff.IsZero() {
			return uint128.Uint128{}, ErrDivisibilityTooLarge
		}
	}
	return cutoff, nil
}
