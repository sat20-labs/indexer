package common

import (
	"fmt"
	"testing"
)

func TestDecimal(t *testing.T) {
	// 测试通过整数创建 Decimal
	d0 := NewDecimal(12345, 3)
	fmt.Printf("Decimal 1: %s\n", d0.String()) // 12.345
	fmt.Printf("Decimal 1: %d\n", d0.Value.Uint64()) // 12345
	fmt.Printf("Decimal 1: %f\n", d0.Float64()) // 12.345
	fmt.Printf("%s\n", d0.GetMaxUint64().String())

	d1 := NewDecimal(12345000000, 6)
	// 测试通过字符串创建 Decimal
	d2, err := NewDecimalFromString("123.456", 6)
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

	// 测试乘法
	d3, _ := NewDecimalFromString("2", 0)
	product := d2.Mul(d3)
	fmt.Printf("Product: %s\n", product.String()) // 246.912

	// 测试除法
	quotient := product.Div(d3)
	fmt.Printf("Quotient: %s\n", quotient.String()) // 123.456

	// 测试平方根
	sqrt := d1.Sqrt()
	fmt.Printf("Square Root of d1: %s\n", sqrt.String())

	// 测试比较
	cmp := d1.Cmp(d2)
	fmt.Printf("Comparison d1 vs d2: %d\n", cmp) // 1 (greater)

	// 测试是否溢出
	isOverflow := d1.IsOverflowUint64()
	fmt.Printf("Is d1 overflow Uint64: %t\n", isOverflow)
}
