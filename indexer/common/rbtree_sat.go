package common

import (
	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"
	"github.com/sat20-labs/indexer/common"
)

type SatRBTree struct {
	tree *redblacktree.Tree
}

func NewSatRBTress() *SatRBTree {
	tree := redblacktree.NewWith(utils.Int64Comparator)
	return &SatRBTree{tree: tree}
}

func (p *SatRBTree) RemoveSatsWithRange(key *common.Range) []int64 {

	result := make([]int64, 0)
	namemap := p.FindSatValuesWithRange(key)
	for sat := range namemap {
		result = append(result, sat)
		p.tree.Remove(sat)
	}

	return result
}

// findNodeGreaterOrEqual 从 Red-Black Tree 中查找大于等于给定键的节点
func (p *SatRBTree) findNodeGreaterOrEqual(key *common.Range) *redblacktree.Node {
	node := p.tree.Root
	var result *redblacktree.Node
	start := key.Start
	end := key.Start + key.Size
	for node != nil {
		if p.tree.Comparator(node.Key, start) >= 0 {
			if p.tree.Comparator(node.Key, end) <= 0 {
				result = node
			}
			node = node.Left
		} else {
			node = node.Right
		}
	}
	return result
}

// findNodeLesser 从 Red-Black Tree 中查找小于给定键的节点
func (p *SatRBTree) findNodeLesser(key *common.Range) *redblacktree.Node {
	node := p.tree.Root
	var result *redblacktree.Node
	start := key.Start
	end := key.Start + key.Size
	for node != nil {
		if p.tree.Comparator(node.Key, end) < 0 {
			if p.tree.Comparator(node.Key, start) >= 0 {
				result = node
			}
			node = node.Right
		} else {
			node = node.Left
		}
	}
	return result
}

func (p *SatRBTree) FindFirstSmaller(key int64) (interface{}, bool) {
	node := p.tree.Root
	var result interface{}
	found := false

	for node != nil {
		nodeKey := node.Key.(int64)
		if nodeKey < key {
			result = node.Value
			found = true
			node = node.Right
		} else if nodeKey == key {
			return node.Value, true
		} else {
			node = node.Left
		}
	}

	return result, found
}

func (p *SatRBTree) FindNode(sat int64) interface{} {
	value, bFound := p.tree.Get(sat)
	if bFound {
		return value
	}
	return nil
}

// key: sat
func (p *SatRBTree) FindSatValuesWithRange(key *common.Range) map[int64]interface{} {

	result := make(map[int64]interface{}, 0)

	left := p.findNodeGreaterOrEqual(key)
	if left == nil {
		return nil
	}
	right := p.findNodeLesser(key)
	if right == nil {
		return nil
	}
	it := p.tree.IteratorAt(left)
	it2 := p.tree.IteratorAt(right)
	for true {
		result[it.Key().(int64)] = it.Value()
		if it == it2 {
			break
		}
		if !it.Next() {
			break
		}
	}

	return result
}

func (p *SatRBTree) Put(key int64, value interface{}) {
	p.tree.Put(key, value)
}

func (p *SatRBTree) Get(start, end int) []interface{} {
	result := make([]interface{}, 0)
	it := p.tree.Iterator()
	it.Begin()

	i := 0
	for i < start && it.Next() {
		i++
	}

	for i <= end && it.Next() {
		value := it.Value()
		result = append(result, value)
		i++
	}

	return result
}

func (p *SatRBTree) Delete(key int64) {
	p.tree.Remove(key)
}

func (p *SatRBTree) View(f func(k int64, v interface{}) error) {
	it := p.tree.Iterator()
	for it.Next() {
		k := it.Key().(int64)
		v := it.Value()
		err := f(k, v)
		if err != nil {
			break
		}
	}
}
