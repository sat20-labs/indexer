package ft

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/sat20-labs/indexer/common"
	indexer "github.com/sat20-labs/indexer/indexer/common"
	"github.com/sat20-labs/indexer/indexer/exotic"
)

func TestIntervalTree(t *testing.T) {
	// 创建一个区间树
	tree := indexer.NewRBTress()

	// 插入一些区间
	tree.Put(&common.Range{Start: 1, Size: 5}, "UTXO(1)")
	tree.Put(&common.Range{Start: 1, Size: 5}, "UTXO(1.1)")
	tree.Put(&common.Range{Start: 1, Size: 4}, "UTXO(1.2)")
	tree.Put(&common.Range{Start: 1, Size: 6}, "UTXO(1.3)")
	tree.Put(&common.Range{Start: 7, Size: 4}, "UTXO(2)")
	tree.Put(&common.Range{Start: 13, Size: 7}, "UTXO(3)")
	tree.Put(&common.Range{Start: 26, Size: 10}, "UTXO(4)")
	tree.Put(&common.Range{Start: 38, Size: 12}, "UTXO(5)")
	printRBTree(tree)

	// 查询与给定区间相交的所有区间
	key := common.Range{Start: 4, Size: 26}
	intersections := tree.FindIntersections(&key)
	for _, v := range intersections {
		fmt.Printf("Intersections: %s %d-%d\n", v.Value.(string), v.Rng.Start, v.Rng.Size)
	}

	printRBTree(tree)
	tree.RemoveRange(&key)

	tree.Put(&key, "UTXO(6)")
	printRBTree(tree)
}

func printRBTree(tree *indexer.RangeRBTree) {
	fmt.Println(tree)
	fmt.Printf("\n")
}


func TestSplitRange(t *testing.T) {

	{
		tree := indexer.NewRBTress()

		// 测试数据
		rangeA := common.Range{Start: 5, Size: 2}
		rangeB := common.Range{Start: 1, Size: 10}

		tree.AddMintInfo(&rangeA, "utxo_A")
		printRBTree(tree)
		tree.AddMintInfo(&rangeB, "utxo_B")
		printRBTree(tree)
	}

	{
		tree := indexer.NewRBTress()

		// 测试数据
		rangeA := common.Range{Start: 1, Size: 10}
		rangeB := common.Range{Start: 5, Size: 2}

		tree.AddMintInfo(&rangeA, "utxo_A")
		printRBTree(tree)
		tree.AddMintInfo(&rangeB, "utxo_B")
		printRBTree(tree)
	}

	{
		tree := indexer.NewRBTress()

		// 测试数据
		rangeA := common.Range{Start: 1, Size: 5}
		rangeB := common.Range{Start: 4, Size: 6}

		tree.AddMintInfo(&rangeA, "utxo_A")
		printRBTree(tree)
		tree.AddMintInfo(&rangeB, "utxo_B")
		printRBTree(tree)
	}

	{
		tree := indexer.NewRBTress()

		// 测试数据
		rangeA := common.Range{Start: 4, Size: 6}
		rangeB := common.Range{Start: 1, Size: 5}

		tree.AddMintInfo(&rangeA, "utxo_A")
		printRBTree(tree)
		tree.AddMintInfo(&rangeB, "utxo_B")
		printRBTree(tree)
	}

}

func TestPizzaRange(t *testing.T) {

	tree := indexer.NewRBTress()

	// 测试数据
	for i, rng := range exotic.PizzaRanges {
		tree.AddMintInfo(rng, strconv.Itoa(i))
	}

	if len(exotic.PizzaRanges) != tree.Size() {
		t.Fatalf("")
	}

	printRBTree(tree)

}
