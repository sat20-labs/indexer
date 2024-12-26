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
	fmt.Printf("Decimal 1: %s\n", d0.String()) // 12.345
	fmt.Printf("Decimal 1: %d\n", d0.Value.Uint64()) // 12345
	fmt.Printf("Decimal 1: %f\n", d0.Float64()) // 12.345
	value := d0.ToInt64WithMax(max)
	fmt.Printf("Decimal 1: %d\n", value) // 1234500000000
	d00, _ := NewDecimalFromInt64WithMax(value, max, precision)
	fmt.Printf("Decimal 1: %s\n", d00.String()) // 12.345
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
	isOverflow := d1.IsOverflowUint64()
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


func TestDecimal128(t *testing.T) {

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
			d0 = d0.Div(d)
			fmt.Printf("Decimal 0: %s\n", d0.String()) 
		}
		fmt.Printf("shift %d\n", shift)
		fmt.Printf("Decimal 0: %d\n", d0.Int64())

		d1 := NewDecimal(d0.Int64(), precision)
		for shift > 0 {
			d1 = d1.Mul(d)
			shift--
			fmt.Printf("Decimal 1: %s\n", d1.String()) 
		}
		fmt.Printf("Decimal 1: %s\n", d1.String())

		d0, _ = NewDecimalFromString("123456789012345678901234567890", precision)
		d2 := d0.Sub(d1)
		fmt.Printf("Decimal 2: %s\n", d2.String())
	}
	
	// 对符文来说，如果max超出int64，就先移位，最终将资产数量表示为int64
}
