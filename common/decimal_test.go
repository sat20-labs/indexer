package common

import (
	"fmt"
	"testing"

	"lukechampine.com/uint128"
)

func TestDecimal(t *testing.T) {
	// 测试通过整数创建 Decimal
	{
		// d1, _ := NewDecimalFromString("1999997", 2)
		// d2, _ := NewDecimalFromString("0.00015", 10)
		// d3 := DecimalMul(d1, d2)
		d3,_ := NewDecimalFromString("0.49", 10)
		fmt.Printf("d3  string: %s\n", d3.String())  
		fmt.Printf("d3  float64: %v\n", d3.Float64())  
		fmt.Printf("d3  Int64: %v\n", d3.Int64())  
		fmt.Printf("d3  Ceil: %v\n", d3.Ceil())  
		fmt.Printf("d3  Floor: %v\n", d3.Floor())
		fmt.Printf("d3  Round: %v\n", d3.Round())
	}
	

	precision := int(3)
	d0 := NewDecimal(12345, precision)
	fmt.Printf("Decimal 1 string: %s\n", d0.String())     // 12.345
	fmt.Printf("Decimal 1 Int64: %d\n", d0.IntegerPart()) // 12
	fmt.Printf("Decimal 1 Float64: %f\n", d0.Float64())   // 12.345
	
	fmt.Printf("%s\n", d0.GetMaxInt64().String())
	d01 := *d0
	d02 := d01
	d01.Value.SetInt64(2)
	fmt.Printf("Decimal d0 string: %s\n", d0.String())     
	fmt.Printf("Decimal d01 string: %s\n", d01.String())     
	fmt.Printf("Decimal d02 string: %s\n", d02.String())     


	d1 := NewDecimal(12345000000, 6)
	fmt.Printf("Decimal 1: %s\n", d1.String()) // 123456
	// 测试通过字符串创建 Decimal
	d2, err := NewDecimalFromString("123.456", 6)
	if err != nil {
		t.Fatalf("Failed to create decimal from string: %v", err)
	}
	fmt.Printf("Decimal 2: %s\n", d2.String()) // 123.456

	// 测试加法
	sum := DecimalAdd(d1, d2)
	fmt.Printf("Sum: %s\n", sum.String()) // 12468.456

	d3 := NewDecimal(123, 2)
	fmt.Printf("Decimal 3: %s\n", d3.String()) // 123

	mul := DecimalMul(d3, d2)
	fmt.Printf("mul: %s\n", mul.String()) // 12468.456

	div := DecimalDiv(d3, d2)
	fmt.Printf("div: %s\n", div.String()) // 12468.456

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
		fmt.Printf("Decimal 1: %s\n", d0.String())       // 12.345
		fmt.Printf("Decimal 1: %d\n", d0.Value.Uint64()) // 12345
		fmt.Printf("Decimal 1: %f\n", d0.Float64())      // 12.345
	}

	// 测试通过整数创建 ordi
	d0 := NewDecimal(1234567890123456789, 18)
	fmt.Printf("Decimal 1: %s\n", d0.String())       // 12.345
	fmt.Printf("Decimal 1: %d\n", d0.Value.Uint64()) // 12345
	fmt.Printf("Decimal 1: %f\n", d0.Float64())      // 12.345
	fmt.Printf("%s\n", d0.GetMaxInt64().String())

	d1 := NewDecimal(1234567890, 8)
	fmt.Printf("Decimal 1: %s\n", d1.String())       // 12.345
	fmt.Printf("Decimal 1: %d\n", d1.Value.Uint64()) // 12345
	fmt.Printf("Decimal 1: %f\n", d1.Float64())      // 12.345
	fmt.Printf("%s\n", d1.GetMaxInt64().String())

	{
		// 测试通过整数创建 runes
		d0 := NewDecimal(1234567890123456789, 0)
		fmt.Printf("Decimal 1: %s\n", d0.String())       // 12.345
		fmt.Printf("Decimal 1: %d\n", d0.Value.Uint64()) // 12345
		fmt.Printf("Decimal 1: %f\n", d0.Float64())      // 12.345
		fmt.Printf("%s\n", d0.GetMaxInt64().String())
	}

}

func TestDecimal_Runes1(t *testing.T) {

	{
		// 测试通过整数创建 runes
		precision := int(1)
		d0, err := NewDecimalFromString("100000000000000100000000000000", int(precision))
		if err != nil {
			t.Fatalf("Failed to create decimal from string: %v", err)
		}
		fmt.Printf("Decimal 0: %s\n", d0.String())
		fmt.Printf("Decimal 0: %d\n", d0.IntegerPart())
	}

	{
		// 测试通过整数创建 runes
		precision := int(1)
		d0, err := NewDecimalFromString("20000000", int(precision))
		if err != nil {
			t.Fatalf("Failed to create decimal from string: %v", err)
		}
		fmt.Printf("Decimal 0: %s\n", d0.String())
		fmt.Printf("Decimal 0: %d\n", d0.IntegerPart())
		
	
	}

	{
		// 测试通过整数创建 runes
		precision := int(0)
		d0, err := NewDecimalFromString("21000000000000000", int(precision))
		if err != nil {
			t.Fatalf("Failed to create decimal from string: %v", err)
		}
		fmt.Printf("Decimal 0: %s\n", d0.String())
		fmt.Printf("Decimal 0: %d\n", d0.IntegerPart())

	}

	{
		precision := int(1)
		d0, err := NewDecimalFromString("100000000000000100000000000000", int(precision))
		if err != nil {
			t.Fatalf("Failed to create decimal from string: %v", err)
		}
		fmt.Printf("Decimal 0: %s\n", d0.String())
		fmt.Printf("Decimal 0: %d\n", d0.IntegerPart())
		
		
		shift := 0
		d := NewDecimal(10, 0)
		for d0.IsOverflowInt64() {
			shift++
			d0 = d0.Div(d)
			fmt.Printf("Decimal 0: %s\n", d0.String())
		}
		fmt.Printf("shift %d\n", shift)
		fmt.Printf("Decimal 0: %d\n", d0.IntegerPart())
		// 对于max数值非常大，溢出的符文，按照max的shift次数，转码后取出Integer部分就行了

		d1, _ := NewDecimalFromString("12345678901234567890", int(precision))
		fmt.Printf("Decimal 1: %s\n", d1.String())
	}
}

func TestDecimal_Runes2(t *testing.T) {
	{
		precision := int(10)
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
			d0 = d0.Div(d)
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

		// 不需要再shift
		// bigValue := new(big.Int).Mul(d2.Value, precisionFactor[shift])
		// //quotient, _ := new(big.Int).QuoRem(bigValue, precisionFactor[d.Precition], new(big.Int))
		// d3 := &Decimal{Precition: precision, Value: bigValue}
		// fmt.Printf("Decimal 3: %s\n", d3.String())

		d0, _ = NewDecimalFromString("1234567890", precision)
	}

	// 对符文来说，如果max超出int64，就先移位，最终将资产数量表示为int64
}

func convertTest(t *testing.T, supply, amt uint128.Uint128) {
	amtInt64 := Uint128ToInt64(supply, amt)
	fmt.Printf("amtInt64 %d\n", amtInt64)
	amt2 := Int64ToUint128(supply, amtInt64)
	fmt.Printf("amt2 %s\n", amt2.String())
	if amt.Cmp(amt2) != 0 {
		t.Errorf("amt different %s", amt.Sub(amt2).String())
	}
}

func TestDecimal_Runes3(t *testing.T) {

	supply, _ := uint128.FromString("10000000")
	amt, _ := uint128.FromString("60")
	convertTest(t, supply, amt)
	decimal := NewDecimalFromUint128(amt, 1)
	fmt.Printf("amt %s\n", decimal.String())

	amt2, _ := NewDecimalFromString("60", 1)
	fmt.Printf("amt2 %s\n", amt2.String())
	fmt.Printf("amt2 integer %d\n", amt2.IntegerPart())
	

	supply, _ = uint128.FromString("2000000")
	amt, _ = uint128.FromString("11")
	convertTest(t, supply, amt)

	supply, _ = uint128.FromString("21000000")
	amt, _ = uint128.FromString("1000")
	convertTest(t, supply, amt)
}
