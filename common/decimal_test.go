package common

import (
	"fmt"
	"testing"
)

func TestDecimal(t *testing.T) {
	// 测试通过整数创建 Decimal
	max := int64(21000000)
	precision := int(3)
	d0 := NewDecimal(12345, precision)
	fmt.Printf("Decimal 1 string: %s\n", d0.String()) // 12.345
	fmt.Printf("Decimal 1 Int64: %d\n", d0.IntegerPart()) // 12345
	fmt.Printf("Decimal 1 Float64: %f\n", d0.Float64()) // 12.345
	value := d0.ToInt64WithMax(max)
	fmt.Printf("Decimal 1 ToInt64WithMax: %d\n", value) // 1234500000000
	d00, _ := NewDecimalFromInt64WithMax(value, max, precision)
	fmt.Printf("Decimal 1 NewDecimalFromInt64WithMax: %s\n", d00.String()) // 12.345
	fmt.Printf("%s\n", d0.GetMaxInt64().String())

	d1 := NewDecimal(12345000000, 6)
	// 测试通过字符串创建 Decimal
	d2, err := NewDecimalFromString("-123.456", 6)
	if err != nil {
		t.Fatalf("Failed to create decimal from string: %v", err)
	}
	fmt.Printf("Decimal 2: %s\n", d2.String()) // 123.456

	// 测试加法
	sum := d1.Add(d2)
	fmt.Printf("Sum: %s\n", sum.String()) // 12468.456

	// 测试减法
	diff := d1.Sub(d2)
	fmt.Printf("Difference: %s\n", diff.String()) // 12221.544

	// 测试比较
	cmp := d1.Cmp(d2)
	fmt.Printf("Comparison d1 vs d2: %d\n", cmp) // 1 (greater)

	// 测试是否溢出
	isOverflow := d1.IsOverflowInt64()
	fmt.Printf("Is d1 overflow Uint64: %t\n", isOverflow)
}

func TestDecimalPrecision(t *testing.T) {

	{
		// 测试通过整数创建 runes
		d0, err := NewDecimalFromString("123456789012345678901234567890", 0)
		if err != nil {
			t.Fatalf("Failed to create decimal from string: %v", err)
		}
		fmt.Printf("Decimal 1: %s\n", d0.String()) // 12.345
		fmt.Printf("Decimal 1: %d\n", d0.Value.Uint64()) // 12345
		fmt.Printf("Decimal 1: %f\n", d0.Float64()) // 12.345
	}

	// 测试通过整数创建 ordi
	d0 := NewDecimal(1234567890123456789, 18)
	fmt.Printf("Decimal 1: %s\n", d0.String()) // 12.345
	fmt.Printf("Decimal 1: %d\n", d0.Value.Uint64()) // 12345
	fmt.Printf("Decimal 1: %f\n", d0.Float64()) // 12.345
	fmt.Printf("%s\n", d0.GetMaxInt64().String())

	d1 := NewDecimal(1234567890, 8)
	fmt.Printf("Decimal 1: %s\n", d1.String()) // 12.345
	fmt.Printf("Decimal 1: %d\n", d1.Value.Uint64()) // 12345
	fmt.Printf("Decimal 1: %f\n", d1.Float64()) // 12.345
	fmt.Printf("%s\n", d1.GetMaxInt64().String())


	{
		// 测试通过整数创建 runes
		d0 := NewDecimal(1234567890123456789, 0)
		fmt.Printf("Decimal 1: %s\n", d0.String()) // 12.345
		fmt.Printf("Decimal 1: %d\n", d0.Value.Uint64()) // 12345
		fmt.Printf("Decimal 1: %f\n", d0.Float64()) // 12.345
		fmt.Printf("%s\n", d0.GetMaxInt64().String())
	}
	
}


func TestDecimal_Runes1(t *testing.T) {

	{
		// 测试通过整数创建 runes
		precision := int(10)
		d0, err := NewDecimalFromString("123456789012345678901234567890", int(precision))
		if err != nil {
			t.Fatalf("Failed to create decimal from string: %v", err)
		}
		fmt.Printf("MaxInt64 %s\n", d0.GetMaxInt64().String()) // 9223372036854775807
		fmt.Printf("Decimal 0: %s\n", d0.String())
		shift := 0
		d := NewDecimal(10, 0)
		for d0.IsOverflowInt64() {
			shift++
			d0 = d0.Div(d.Value)
			fmt.Printf("Decimal 0: %s\n", d0.String()) 
		}
		fmt.Printf("shift %d\n", shift)
		fmt.Printf("Decimal 0: %d\n", d0.IntegerPart())
		// 对于max数值非常大，溢出的符文，按照max的shift次数，转码后取出Integer部分就行了

		d1, _ := NewDecimalFromString("12345678901234567890", int(precision))
		fmt.Printf("Decimal 1: %s\n", d1.String())
		d2 := d1.Div(precisionFactor[shift])
		fmt.Printf("Decimal 2: %s\n", d2.String())
		d3 := d2.Mul(precisionFactor[shift])
		fmt.Printf("Decimal 3: %s\n", d3.String())
		d4 := d1.Sub(d3)
		fmt.Printf("Decimal 4: %s\n", d4.String())
	}
}

func TestDecimal_Runes2(t *testing.T) {
	{
		precision := int(10)
		max := int64(10000000000)
		d0, err := NewDecimalFromString("10000000000", int(precision)) // max
		if err != nil {
			t.Fatalf("Failed to create decimal from string: %v", err)
		}
		fmt.Printf("MaxInt64 %s\n", d0.GetMaxInt64().String()) // 9223372036854775807
		fmt.Printf("Decimal 0: %s\n", d0.String())
		shift := 0
		d := NewDecimal(10, 0)
		for d0.IsOverflowInt64() {
			shift++
			d0 = d0.Div(d.Value)
			fmt.Printf("Decimal 0: %s\n", d0.String()) 
		}
		if shift > 0 {
			t.Fatalf("shift must be zero")
		}
		fmt.Printf("shift %d\n", shift)
		fmt.Printf("Decimal 0: %d\n", d0.IntegerPart())
		// 对于max数值非常大，溢出的符文，按照max的shift次数，转码后取出Integer部分就行了

		d1, _ := NewDecimalFromString("1234567890", int(precision))
		if d1.Cmp(d0) > 0 {
			t.Fatalf("limit should be less than max")
		}

		// 对于max没有溢出，并且precision不等于0的数值，按照下面方法折算为int64
		value := d1.ToInt64WithMax(max)
		fmt.Printf("Decimal 1 value: %d\n", value)

		d2, err := NewDecimalFromInt64WithMax(value, max, precision)
		if err != nil {
			t.Fatalf("Failed to create decimal from string: %v", err)
		}
		fmt.Printf("Decimal 2: %s\n", d2.String())
		
		// 不需要再shift
		// bigValue := new(big.Int).Mul(d2.Value, precisionFactor[shift])
		// //quotient, _ := new(big.Int).QuoRem(bigValue, precisionFactor[d.Precition], new(big.Int))
		// d3 := &Decimal{Precition: precision, Value: bigValue}
		// fmt.Printf("Decimal 3: %s\n", d3.String())

		d0, _ = NewDecimalFromString("1234567890", precision)
		d4 := d0.Sub(d2)
		fmt.Printf("Decimal 4: %s\n", d4.String())
	}
	
	// 对符文来说，如果max超出int64，就先移位，最终将资产数量表示为int64
}
