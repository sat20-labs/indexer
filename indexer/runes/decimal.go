package runes

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"lukechampine.com/uint128"
)

type Decimal struct {
	Value *uint128.Uint128
	Scale uint8
}

func NewDecimal(value *uint128.Uint128, scale uint8) *Decimal {
	return &Decimal{
		Value: value,
		Scale: scale,
	}
}

func (d *Decimal) ToInteger(divisibility uint8) (uint64, error) {
	if divisibility < d.Scale {
		return 0, errors.New("excessive precision")
	}

	difference := divisibility - d.Scale
	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(difference)), nil)
	result := new(big.Int).Mul(d.Value.Big(), multiplier)
	if !result.IsUint64() {
		return 0, errors.New("amount out of range")
	}

	return result.Uint64(), nil
}

func (d Decimal) String() string {
	if d.Value.IsZero() {
		return "0"
	}

	// Convert Uint128 to big.Int for easier manipulation
	valueBig := new(big.Int).SetBytes(d.Value.Big().Bytes())

	magnitude := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(d.Scale)), nil)
	integer := new(big.Int).Div(valueBig, magnitude)
	fraction := new(big.Int).Mod(valueBig, magnitude)

	result := integer.String()

	if fraction.Sign() > 0 {
		fractionStr := fmt.Sprintf("%0*s", d.Scale, fraction.String())
		fractionStr = strings.TrimRight(fractionStr, "0")
		if fractionStr != "" {
			result += "." + fractionStr
		}
	}

	return result
}

// func (d Decimal) String() string {
// 	magnitude := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(d.Scale)), nil)
// 	integer := new(big.Int).Div(d.Value.Big(), magnitude)
// 	fraction := new(big.Int).Mod(d.Value.Big(), magnitude)

// 	var result string
// 	result += integer.String()

// 	if fraction.Cmp(big.NewInt(0)) > 0 {
// 		fractionStr := fraction.String()
// 		width := int(d.Scale)
// 		for fraction.Mod(fraction, big.NewInt(10)).Cmp(big.NewInt(0)) == 0 {
// 			fraction.Div(fraction, big.NewInt(10))
// 			width--
// 		}
// 		result += "." + fmt.Sprintf("%0*s", width, fractionStr)
// 	}

// 	return result
// }

func NewDecimalFromString(s string) (*Decimal, error) {
	var ret Decimal
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return &ret, errors.New("invalid decimal format")
	}

	integerPart := parts[0]
	decimalPart := ""
	if len(parts) == 2 {
		decimalPart = parts[1]
	}

	if integerPart == "" && decimalPart == "" {
		return &ret, errors.New("empty decimal")
	}

	if integerPart != "" {
		value, err := uint128.FromString(integerPart)
		if err != nil {
			return &ret, err
		}
		ret.Value = &value
	}

	if decimalPart != "" {
		trailingZeros := 0
		for i := len(decimalPart) - 1; i >= 0 && decimalPart[i] == '0'; i-- {
			trailingZeros++
		}
		significantDigits := len(decimalPart) - trailingZeros
		decimalValue, err := strconv.ParseUint(decimalPart, 10, 64)
		if err != nil {
			return &ret, err
		}
		decimalValue /= uint64(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(trailingZeros)), nil).Uint64())
		value := uint128.From64(uint64(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(significantDigits)), nil).Uint64()) + decimalValue)
		ret.Value = &value
		ret.Scale = uint8(significantDigits)
	} else {
		ret.Scale = 0
	}

	return &ret, nil
}
