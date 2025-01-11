package runestone

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sat20-labs/indexer/common"
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

func (p Pile) Decimal() (*common.Decimal, error) {
	cutoff, err := calculateCutoff(p.Divisibility)
	if err != nil {
		return nil, fmt.Errorf("Error: %v", err)
	}

	whole := p.Amount.Div(cutoff)
	fractional := p.Amount.Mod(cutoff)

	var resultStr string
	if fractional.IsZero() {
		resultStr = whole.String()
	} else {
		fractionalStr := fmt.Sprintf("%0*s", p.Divisibility, fractional.String())
		width := int(p.Divisibility)
		for fractional.Mod(uint128.From64(10)).IsZero() {
			fractional = fractional.Div(uint128.From64(10))
			width--
		}
		fractionalStr = fractionalStr[:width]
		resultStr = fmt.Sprintf("%s.%s", whole.String(), fractionalStr)
	}
	precision := getPrecision(resultStr)
	ret := common.NewDecimalFromUint128(p.Amount, precision)
	return ret, err
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

func getPrecision(numberStr string) int {
	dotIndex := strings.Index(numberStr, ".")
	if dotIndex == -1 {
		return 0
	}
	return len(numberStr) - dotIndex - 1
}
