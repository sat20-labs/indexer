package common

import (
	"fmt"
	"testing"
)

func TestDecimal(t *testing.T) {
	// 测试通过整数创建 Decimal
	max := int64(21000000)
	precision := uint(3)
	d0 := NewDecimal(12345, precision)
	fmt.Printf("Decimal 1: %s\n", d0.String()) // 12.345
	fmt.Printf("Decimal 1: %d\n", d0.Value.Uint64()) // 12345
	fmt.Printf("Decimal 1: %f\n", d0.Float64()) // 12.345
	value := d0.ToInt64WithMax(max)
	fmt.Printf("Decimal 1: %d\n", value) // 1234500000000
	d00, _ := NewDecimalFromInt64WithMax(value, max, precision)
	fmt.Printf("Decimal 1: %s\n", d00.String()) // 12.345
	fmt.Printf("%s\n", d0.GetMaxUint64().String())

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
	// 测试通过整数创建 ordi
	d0 := NewDecimal(12345678901234567890, 18)
	fmt.Printf("Decimal 1: %s\n", d0.String()) // 12.345
	fmt.Printf("Decimal 1: %d\n", d0.Value.Uint64()) // 12345
	fmt.Printf("Decimal 1: %f\n", d0.Float64()) // 12.345
	fmt.Printf("%s\n", d0.GetMaxUint64().String())

	d1 := NewDecimal(1234567890, 8)
	fmt.Printf("Decimal 1: %s\n", d1.String()) // 12.345
	fmt.Printf("Decimal 1: %d\n", d1.Value.Uint64()) // 12345
	fmt.Printf("Decimal 1: %f\n", d1.Float64()) // 12.345
	fmt.Printf("%s\n", d1.GetMaxUint64().String())
}
