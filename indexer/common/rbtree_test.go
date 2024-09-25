package common

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/sat20-labs/ordx/common"
)

func TestIntervalTree(t *testing.T) {
	// 创建一个区间树
	tree := NewSatRBTress()

	// 插入一些区间
	for i := 100; i < 110; i++ {
		tree.Put(int64(i), "name_"+strconv.Itoa(i))
	}

	printRBTree(tree)

	// 查询与给定区间相交的所有区间
	key := common.Range{Start: 99, Size: 4}
	sats := tree.FindSatValuesWithRange(&key)
	for k, v := range sats {
		fmt.Printf("sat: %d %v\n", k, v)
	}

	key = common.Range{Start: 109, Size: 4}
	sats = tree.FindSatValuesWithRange(&key)
	for k, v := range sats {
		fmt.Printf("sat: %d %v\n", k, v)
	}

	//printRBTree(tree)
	tree.RemoveSatsWithRange(&key)
	printRBTree(tree)

	tree.Put(int64(104), "name_"+strconv.Itoa(41))
	tree.Put(int64(104), "name_"+strconv.Itoa(42))
	printRBTree(tree)

	names := tree.FindNode(104)
	fmt.Printf("%v\n", names)

	names = tree.FindNode(100)
	fmt.Printf("%v\n", names)

	names = tree.FindNode(109)
	fmt.Printf("%v\n", names)

	names = tree.FindNode(115)
	fmt.Printf("%v\n", names)

}

func printRBTree(tree *SatRBTree) {
	fmt.Printf("%v", tree.tree)
	fmt.Printf("\n")
}
