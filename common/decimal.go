package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"lukechampine.com/uint128"
)

const MAX_PRECISION = 64

var precisionFactor [64]*big.Int = [64]*big.Int{
	new(big.Int).Exp(big.NewInt(10), big.NewInt(0), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(1), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(2), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(3), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(4), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(5), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(7), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(8), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(10), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(11), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(12), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(13), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(14), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(16), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(19), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(21), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(22), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(23), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(25), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(26), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(27), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(28), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(29), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(31), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(32), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(33), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(34), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(35), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(37), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(38), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(39), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(40), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(41), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(42), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(43), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(44), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(45), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(46), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(47), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(48), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(49), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(50), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(51), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(52), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(53), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(54), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(55), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(56), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(57), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(58), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(59), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(60), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(61), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(62), nil),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(63), nil),
}

// Decimal represents a fixed-point decimal number with 18 decimal places
type Decimal struct {
	Precision int
	Value     *big.Int
}

func NewDefaultDecimal(v int64) *Decimal {
	return &Decimal{Precision: 0, Value: new(big.Int).SetInt64(v)}
}

// v 是乘10的p次方后的值，也就是需要往前点p个小数点才是真正的值
func NewDecimalWithScale(v int64, p int) *Decimal {
	if p > MAX_PRECISION {
		p = MAX_PRECISION
	}
	return &Decimal{Precision: p, Value: new(big.Int).SetInt64(v)}
}

// v是整数部分的值，小数点不动 （跟NewDecimalFromString类似）
func NewDecimal(v int64, p int) *Decimal {
	if p > MAX_PRECISION {
		Log.Panic("too big precision")
	}
	value := big.NewInt(v)
	if p != 0 {
		value = new(big.Int).Mul(value, precisionFactor[p])
	}

	return &Decimal{Precision: p, Value: value}
}

// NewDecimalFromString creates a Decimal instance from a string
func NewDecimalFromString(s string, maxPrecision int) (*Decimal, error) {
	if s == "" {
		return nil, errors.New("empty string")
	}

	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid decimal format: %s", s)
	}

	integerPartStr := parts[0]
	if integerPartStr == "" || integerPartStr == "+" {
		return nil, errors.New("empty integer")
	}
	var neg bool
	if integerPartStr[0] == '-' {
		neg = true
	}

	// SetString("-0", 10) 结果跟 SetString("0", 10) 一样，需要特殊处理
	integerPart, ok := new(big.Int).SetString(parts[0], 10)
	if !ok {
		return nil, fmt.Errorf("invalid integer format: %s", parts[0])
	}

	currPrecision := 0
	decimalPart := big.NewInt(0)
	if len(parts) == 2 {
		decimalPartStr := parts[1]
		if decimalPartStr == "" || decimalPartStr[0] == '-' || decimalPartStr[0] == '+' {
			return nil, errors.New("empty decimal")
		}

		currPrecision = len(decimalPartStr)
		if currPrecision > maxPrecision {
			return nil, fmt.Errorf("decimal exceeds maximum precition: %s", s)
		}
		n := maxPrecision - currPrecision
		for i := 0; i < n; i++ {
			decimalPartStr += "0"
		}
		decimalPart, ok = new(big.Int).SetString(decimalPartStr, 10)
		if !ok || decimalPart.Sign() < 0 {
			return nil, fmt.Errorf("invalid decimal format: %s", parts[0])
		}
	}

	value := new(big.Int).Mul(integerPart, precisionFactor[maxPrecision])
	if value.Sign() < 0 {
		value = value.Sub(value, decimalPart)
	} else {
		value = value.Add(value, decimalPart)
	}

	if value.Sign() > 0 && neg {
		value.Neg(value)
	}

	return &Decimal{Precision: int(maxPrecision), Value: value}, nil
}

func (d *Decimal) Clone() *Decimal {
	if d == nil {
		return nil
	}
	return &Decimal{Precision: d.Precision, Value: new(big.Int).Set(d.Value)}
}

// String returns the string representation of a Decimal instance
func (d *Decimal) String() string {
	if d == nil {
		return "0"
	}
	value := new(big.Int).Abs(d.Value)
	quotient, remainder := new(big.Int).QuoRem(value, precisionFactor[d.Precision], new(big.Int))
	sign := ""
	if d.Value.Sign() < 0 {
		sign = "-"
	}
	if remainder.Sign() == 0 {
		return fmt.Sprintf("%s%s", sign, quotient.String())
	}
	decimalPart := fmt.Sprintf("%0*d", d.Precision, remainder)
	decimalPart = strings.TrimRight(decimalPart, "0")
	return fmt.Sprintf("%s%s.%s", sign, quotient.String(), decimalPart)
}

func NewDecimalFromFormatString(s string) (*Decimal, error) {
	parts := strings.Split(s, ":")
	switch len(parts) {
	case 1:
		return NewDecimalFromString(s, 0)
	case 2:
		precision, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}
		return NewDecimalFromString(parts[0], precision)
	default:
		return nil, fmt.Errorf("invalid format")
	}
}

func (d *Decimal) ToFormatString() string {
	if d == nil {
		return "0:0"
	}
	return fmt.Sprintf("%s:%d", d.String(), d.Precision)
}

// 实现自定义 JSON 序列化逻辑
func (d *Decimal) MarshalJSON() ([]byte, error) {
    return json.Marshal(map[string]interface{}{
        "Precision": d.Precision,
        "Value":     d.Value.String(),
    })
}

// 实现自定义反序列化逻辑
func (d *Decimal) UnmarshalJSON(data []byte) error {
    var tmp struct {
        Precision int `json:"Precision"`
        Value     string `json:"Value"`
    }

    if err := json.Unmarshal(data, &tmp); err != nil {
        return err
    }

	n := new(big.Int)
    n, ok := n.SetString(tmp.Value, 10)
    if !ok {
        return fmt.Errorf("invalid value %s", tmp.Value)
    }

    d.Precision = tmp.Precision
    d.Value = n
    return nil
}

// alignPrecision 将两个 Decimal 对齐到同一精度，返回新值和目标精度
func alignPrecision(a, b *Decimal) (aVal, bVal *big.Int, precision int) {
	// 对齐到更高的精度
	if a.Precision > b.Precision {
		factor := precisionFactor[a.Precision-b.Precision]
		bVal := new(big.Int).Mul(b.Value, factor)
		return new(big.Int).Set(a.Value), bVal, a.Precision
	} else {
		factor := precisionFactor[b.Precision-a.Precision]
		aVal := new(big.Int).Mul(a.Value, factor)
		return aVal, new(big.Int).Set(b.Value), b.Precision
	}
}

func (d *Decimal) NewPrecision(precision int) *Decimal {
	if d == nil {
		return nil
	}
	if d.Precision == precision {
		return d.Clone()
	}
	val := new(big.Int).Set(d.Value)
	if d.Precision > precision {
        factor := precisionFactor[d.Precision-precision]
        val = val.Div(val, factor)
    } else if d.Precision < precision {
        factor := precisionFactor[precision-d.Precision]
        val = val.Mul(val, factor)
    }
    return &Decimal{Precision: precision, Value: val}
}

func (d *Decimal) SetPrecision(precision int) *Decimal {
	if d == nil {
		return nil
	}
	if d.Precision == precision {
		return d
	}
	val := new(big.Int).Set(d.Value)
	if d.Precision > precision {
        factor := precisionFactor[d.Precision-precision]
        val = val.Div(val, factor)
    } else if d.Precision < precision {
        factor := precisionFactor[precision-d.Precision]
        val = val.Mul(val, factor)
    }
	d.Precision = precision
	d.Value = val
    return d
}

// 精度调整，同时四舍五入
func (d *Decimal) SetPrecisionWithRound(targetPrecision int) Decimal {
    if d.Precision == targetPrecision {
        return Decimal{
            Precision: targetPrecision,
            Value:     new(big.Int).Set(d.Value),
        }
    }

    if d.Precision < targetPrecision {
        // 精度提升：直接补 0
        factor := new(big.Int).Exp(
            big.NewInt(10),
            big.NewInt(int64(targetPrecision-d.Precision)),
            nil,
        )

        return Decimal{
            Precision: targetPrecision,
            Value:     new(big.Int).Mul(d.Value, factor),
        }
    }

    // from.Precision > targetPrecision → 需要 round
    diff := d.Precision - targetPrecision

    factor := new(big.Int).Exp(
        big.NewInt(10),
        big.NewInt(int64(diff)),
        nil,
    )

    q := new(big.Int)
    r := new(big.Int)

    q.QuoRem(d.Value, factor, r)

    // |r| * 2 >= factor → round half up
    absR := new(big.Int).Abs(r)
    absR.Mul(absR, big.NewInt(2))

    if absR.Cmp(factor) >= 0 {
        if d.Value.Sign() >= 0 {
            q.Add(q, big.NewInt(1))
        } else {
            q.Sub(q, big.NewInt(1))
        }
    }

    return Decimal{
        Precision: targetPrecision,
        Value:     q,
    }
}


func (d *Decimal) Add(other *Decimal) *Decimal {
	if d == nil && other == nil {
		return nil
	}
	if d == nil {
		// 缩放 other 到 d 的精度（d==nil时直接返回other的拷贝）
		return other.Clone()
	}
	if other == nil {
		return d.Clone()
	}
	// 对齐 other 到 d 的精度
	aVal := new(big.Int).Set(d.Value)
	bVal := new(big.Int).Set(other.Value)
	if d.Precision > other.Precision {
		factor := precisionFactor[d.Precision-other.Precision]
		bVal = bVal.Mul(bVal, factor)
	} else if d.Precision < other.Precision {
		factor := precisionFactor[other.Precision-d.Precision]
		bVal = bVal.Div(bVal, factor)
	}
	value := new(big.Int).Add(aVal, bVal)
	return &Decimal{Precision: d.Precision, Value: value}
}

// 提高效率
func (d *Decimal) AddInPlace(other *Decimal) {
    if other == nil || other.Value == nil {
        return
    }

    if d.Value == nil {
        // 不共享 other.Value，显式拷贝
        d.Value = new(big.Int).Set(other.Value)
        d.Precision = other.Precision
        return
    }

    if d.Precision != other.Precision {
        panic("decimal precision mismatch")
    }

    // big.Int.Add 在 z 已存在时不会分配
    d.Value.Add(d.Value, other.Value)
}



// Add adds two Decimal instances and returns a new Decimal instance
func DecimalAdd(a, b *Decimal) *Decimal {
	n := a.Clone()
	return n.Add(b)
}

func (d *Decimal) Sub(other *Decimal) *Decimal {
	if d == nil && other == nil {
		return nil
	}
	if d == nil {
		neg := other.Clone()
		neg.Value.Neg(neg.Value)
		return neg
	}
	if other == nil {
		return d.Clone()
	}
	// 对齐 other 到 d 的精度
	aVal := new(big.Int).Set(d.Value)
	bVal := new(big.Int).Set(other.Value)
	if d.Precision > other.Precision {
		factor := precisionFactor[d.Precision-other.Precision]
		bVal = bVal.Mul(bVal, factor)
	} else if d.Precision < other.Precision {
		factor := precisionFactor[other.Precision-d.Precision]
		bVal = bVal.Div(bVal, factor)
	}
	value := new(big.Int).Sub(aVal, bVal)
	return &Decimal{Precision: d.Precision, Value: value}
}

// Sub subtracts two Decimal instances and returns a new Decimal instance
func DecimalSub(a, b *Decimal) *Decimal {
	n := a.Clone()
	return n.Sub(b)
}

// 精度跟a对齐
func (d *Decimal) Mul(other *Decimal) *Decimal {
	if d == nil || other == nil {
		return nil
	}
	// 先相乘
	value := new(big.Int).Mul(d.Value, other.Value)
	// 缩放回a的精度
	if other.Precision > 0 {
		value = value.Div(value, precisionFactor[other.Precision])
	}
	return &Decimal{Precision: d.Precision, Value: value}
}

func (d *Decimal) MulBigInt(other *big.Int) *Decimal {
	if d == nil || other == nil {
		return nil
	}
	value := new(big.Int).Mul(d.Value, other)
	return &Decimal{Precision: d.Precision, Value: value}
	//d.Value = value
	//return d
}

// 精度为a
func DecimalMul(a, b *Decimal) *Decimal {
	n := a.Clone()
	return n.Mul(b)
}

// 精度为a+b
func DecimalMulV2(a, b *Decimal) *Decimal {
	n := a.Clone()
	return n.MulV2(b)
}

// 精度为a+b
func (d *Decimal) MulV2(other *Decimal) *Decimal {
	if d == nil || other == nil {
		return nil
	}
	value := new(big.Int).Mul(d.Value, other.Value)
	precision := d.Precision + other.Precision
	return &Decimal{Precision: precision, Value: value}
}

// 除法，精度为a
func (d *Decimal) Div(other *Decimal) *Decimal {
	if d == nil || other == nil || other.Sign() == 0 {
		return nil
	}
	// 先将a的Value放大other.Precision倍，避免精度丢失
	scaledA := new(big.Int).Mul(d.Value, precisionFactor[other.Precision])
	value := new(big.Int).Div(scaledA, other.Value)
	return &Decimal{Precision: d.Precision, Value: value}
}

func (d *Decimal) DivBigInt(other *big.Int) *Decimal {
	if d == nil || other == nil || other.Sign() == 0 {
		return nil
	}
	value := new(big.Int).Div(d.Value, other)
	return &Decimal{Precision: d.Precision, Value: value}
	//d.Value = value
	//return d
}

// 精度为a
func DecimalDiv(a, b *Decimal) *Decimal {
	n := a.Clone()
	return n.Div(b)
}

// Sqrt 计算定点数的平方根，保持 Precision 精度
func (d *Decimal) Sqrt() *Decimal {
	if d == nil {
		return nil
	}
	// scale = 10^Precision
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(d.Precision)), nil)

	// 放大：Value * (10^Precision)，保证平方根后仍有足够的小数精度
	scaledVal := new(big.Int).Mul(d.Value, scale)

	// 开平方
	sqrtVal := new(big.Int).Sqrt(scaledVal)

	return &Decimal{
		Precision: d.Precision,
		Value:     sqrtVal,
	}
}

func DecimalSqrt(a *Decimal) *Decimal {
	n := a.Clone()
	return n.Sqrt()
}

func (d *Decimal) Cmp(other *Decimal) int {
	if d == nil && other == nil {
		return 0
	}
	if other == nil {
		return d.Value.Sign()
	}
	if d == nil {
		return -other.Value.Sign()
	}
	if d.Precision == other.Precision {
		return d.Value.Cmp(other.Value)
	}
	// 精度不一致，调整到相同精度再比较
	if d.Precision > other.Precision {
		factor := precisionFactor[d.Precision-other.Precision]
		otherVal := new(big.Int).Mul(other.Value, factor)
		return d.Value.Cmp(otherVal)
	} else {
		factor := precisionFactor[other.Precision-d.Precision]
		dVal := new(big.Int).Mul(d.Value, factor)
		return dVal.Cmp(other.Value)
	}
}

func (d *Decimal) Sign() int {
	if d == nil {
		return 0
	}
	return d.Value.Sign()
}

func (d *Decimal) Abs() *Decimal {
	if d == nil {
		return nil
	}
	r := d.Clone()
	r.Value.Abs(r.Value)
	return r
}

func (d *Decimal) IsOverflowInt64() bool {
	if d == nil {
		return false
	}

	integerPart := new(big.Int).SetInt64(math.MaxInt64)
	value := new(big.Int).Mul(integerPart, precisionFactor[d.Precision])
	return d.Value.Cmp(value) > 0
}

func (d *Decimal) IsOverflowUint64() bool {
	if d == nil {
		return false
	}

	integerPart := new(big.Int).SetUint64(math.MaxInt64)
	value := new(big.Int).Mul(integerPart, precisionFactor[d.Precision])
	return d.Value.Cmp(value) > 0
}

func (d *Decimal) IsZero() bool {
	return d.Sign() == 0
}

func (d *Decimal) SetValue(value int64) {
	if d == nil {
		return
	}
	d.Value = new(big.Int).SetInt64(value)
}

func (d *Decimal) GetMaxInt64() *Decimal {
	if d == nil {
		return nil
	}
	integerPart := new(big.Int).SetInt64(math.MaxInt64)
	value := new(big.Int).Mul(integerPart, precisionFactor[d.Precision])
	return &Decimal{Precision: d.Precision, Value: value}
}

func (d *Decimal) GetMaxUint64() *Decimal {
	if d == nil {
		return nil
	}
	integerPart := new(big.Int).SetUint64(math.MaxUint64)
	value := new(big.Int).Mul(integerPart, precisionFactor[d.Precision])
	return &Decimal{Precision: d.Precision, Value: value}
}

func (d *Decimal) Float64() float64 {
	if d == nil {
		return 0
	}
	fVal := new(big.Float).SetInt(d.Value)
	scale := new(big.Float).SetInt(precisionFactor[d.Precision])
	result, _ := new(big.Float).Quo(fVal, scale).Float64()
	return result
}

// 等同于Floor
func (d *Decimal) Int64() int64 {
	if d == nil {
		return 0
	}
	if d.Precision == 0 {
		return d.Value.Int64()
	}

	return d.IntegerPart()
}

// 向上取整
func (d *Decimal) Ceil() int64 {
	if d == nil {
		return 0
	}
	if d.Precision == 0 {
		return d.Value.Int64()
	}

	return int64(math.Ceil(d.Float64()))
}

// 向下取整
func (d *Decimal) Floor() int64 {
	if d == nil {
		return 0
	}
	if d.Precision == 0 {
		return d.Value.Int64()
	}

	return int64(math.Floor(d.Float64()))
}

// 4舍5入
func (d *Decimal) Round() int64 {
	if d == nil {
		return 0
	}
	if d.Precision == 0 {
		return d.Value.Int64()
	}

	return int64(math.Round(d.Float64()))
}

func (d *Decimal) IntegerPart() int64 {
	if d == nil {
		return 0
	}
	value := new(big.Int).Set(d.Value)
	quotient, _ := new(big.Int).QuoRem(value, precisionFactor[d.Precision], new(big.Int))
	return quotient.Int64()
}

func NewDecimalFromUint128(n uint128.Uint128, precision int) *Decimal {
	hi := new(big.Int).SetUint64(n.Hi)
	hi.Lsh(hi, 64)
	value := new(big.Int).SetUint64(n.Lo)
	value.Add(value, hi)
	return &Decimal{Precision: precision, Value: value}
}

func (d *Decimal) ToUint128() uint128.Uint128 {
	if d == nil {
		return uint128.Uint128{}
	}
	lo := d.Value.Uint64()
	hi := new(big.Int).Rsh(d.Value, 64).Uint64()
	return uint128.Uint128{Lo: lo, Hi: hi}
}

func decimalDigits(n uint64) int {
	return int(math.Floor(math.Log10(float64(n))) + 1)
}

func Uint128ToInt64(supply, amt uint128.Uint128) int64 {
	if supply.Hi == 0 {
		return amt.Big().Int64()
	}

	q, _ := supply.QuoRem64(math.MaxInt64)
	scaleIndex := decimalDigits(q.Lo)

	return int64(amt.Div64(precisionFactor[scaleIndex].Uint64()).Lo)
}

func Int64ToUint128(supply uint128.Uint128, amt int64) uint128.Uint128 {
	if supply.Hi == 0 {
		return uint128.From64(uint64(amt))
	}

	q, _ := supply.QuoRem64(math.MaxInt64)
	scaleIndex := decimalDigits(q.Lo)
	result := uint128.From64(uint64(amt))
	return result.Mul64(precisionFactor[scaleIndex].Uint64())
}
