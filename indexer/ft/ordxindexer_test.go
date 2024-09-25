package ft

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/ordx/common"
	indexer "github.com/sat20-labs/ordx/indexer/common"
	"github.com/sat20-labs/ordx/indexer/exotic"
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

func TestPerformanceInitFromDB(t *testing.T) {
	opts := badger.DefaultOptions("../../ordx-db").
		WithLoggingLevel(badger.WARNING).
		WithBlockCacheSize(2000 << 20)

	db, err := badger.Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	disksize, uncompresssize := db.EstimateSize([]byte("u-"))
	fmt.Printf("disksize size %d, uncompresssize %d\n", disksize/(1024*1024), uncompresssize/(1024*1024))

	lsm, vlog := db.Size()
	fmt.Printf("lsm size %d, vlog %d\n", lsm/(1024*1024), vlog/(1024*1024))

	// lvs := db.Levels()
	// fmt.Println(lvs)

	// tbs := db.Tables()
	// fmt.Println(tbs)

	count := 0
	startTime := time.Now()
	err = db.View(func(txn *badger.Txn) error {
		prefix := []byte("u-066")

		// 设置前缀扫描选项
		prefixOptions := badger.DefaultIteratorOptions
		prefixOptions.Prefix = prefix

		// 使用前缀扫描选项创建迭代器
		it := txn.NewIterator(prefixOptions)
		defer it.Close()

		// 遍历匹配前缀的key
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			//item := it.Item()
			//key := item.Key()

			// ticker
			//fmt.Printf("key: %s\n", key)

			// 进行你的处理，例如统计数量等
			count++
		}

		return nil
	})

	if err != nil {
		t.Fatal("Error prefetching utxos from db ", err)
	}

	elapsed := time.Since(startTime).Milliseconds()
	fmt.Printf("initFromDB %d utxos in %d ms\n", count, elapsed)
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
