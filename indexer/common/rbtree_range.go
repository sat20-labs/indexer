package common

import (
	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/sat20-labs/indexer/common"
)

type SatRange2Value struct {
	Rng   *common.Range
	Value interface{}
}

type RangeRBTree struct {
	tree *redblacktree.Tree
}

func NewRBTress() *RangeRBTree {
	tree := redblacktree.NewWith(RangeComparator)
	return &RangeRBTree{tree: tree}
}

// RangeComparator 是用于比较区间的比较器
func RangeComparator(a, b interface{}) int {
	rangeA := a.(*common.Range)
	rangeB := b.(*common.Range)
	return common.RangeComparator(rangeA, rangeB)
}

func (p *RangeRBTree) RemoveRange(key *common.Range) []*common.Range {
	// 从相交的区间，获取
	result := make([]*common.Range, 0)
	node := p.tree.GetNode(key)
	for node != nil {
		result = append(result, node.Key.(*common.Range))
		p.tree.Remove(node.Key)
		node = p.tree.GetNode(key)
	}
	return result
}

func (p *RangeRBTree) Put(key *common.Range, value interface{}) {
	p.tree.Put(key, value)
}

func (p *RangeRBTree) Size() int {
	return p.tree.Size()
}

// 不用递归的方式
func (p *RangeRBTree) FindIntersections(key *common.Range) []*SatRange2Value {

	result := make([]*SatRange2Value, 0)

	stack := []*redblacktree.Node{p.tree.GetNode(key)}
	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if node != nil {
			ret := RangeComparator(key, node.Key)
			if ret == 0 {
				var sats SatRange2Value
				sats.Value = node.Value
				sats.Rng = p.Intersection(key, node.Key.(*common.Range))
				result = append(result, &sats)

				stack = append(stack, node.Right, node.Left)
			} else if ret < 0 {
				stack = append(stack, node.Left)
			} else if ret > 0 {
				stack = append(stack, node.Right)
			}
		}
	}

	if len(result) > 0 {
		return result
	}
	return nil
}

// 保持tree中的key不变
func (p *RangeRBTree) FindIntersections_OriginalKey(key *common.Range) []*SatRange2Value {

	result := make([]*SatRange2Value, 0)

	stack := []*redblacktree.Node{p.tree.GetNode(key)}
	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if node != nil {
			ret := RangeComparator(key, node.Key)
			if ret == 0 {
				var sats SatRange2Value
				sats.Value = node.Value
				sats.Rng = node.Key.(*common.Range)
				result = append(result, &sats)

				stack = append(stack, node.Right, node.Left)
			} else if ret < 0 {
				stack = append(stack, node.Left)
			} else if ret > 0 {
				stack = append(stack, node.Right)
			}
		}
	}

	if len(result) > 0 {
		return result
	}
	return nil
}

func (p *RangeRBTree) CheckIntersection(key *common.Range) bool {
	return p.tree.GetNode(key) != nil
}

func (p *RangeRBTree) Intersection(r1, r2 *common.Range) *common.Range {
	return common.InterRange(r1, r2)
}

func (p *RangeRBTree) AddMintInfo(rng *common.Range, inscriptionId string) {
	// 处理重复铭刻在同一段satrange的情况
	interRanges := p.FindIntersections_OriginalKey(rng)

	newValue := &common.RBTreeValue_Mint{InscriptionIds: []string{inscriptionId}}
	if interRanges == nil {
		p.tree.Put(&common.Range{Start: rng.Start, Size: rng.Size}, newValue)
		return
	}

	for _, ir := range interRanges {

		// 分割原来的数据
		start1 := max(ir.Rng.Start, rng.Start)
		end1 := min(ir.Rng.Start+ir.Rng.Size-1, rng.Start+rng.Size-1)

		start2 := min(ir.Rng.Start, rng.Start)
		end2 := max(ir.Rng.Start+ir.Rng.Size-1, rng.Start+rng.Size-1)

		// 相交部分
		rng1 := common.Range{Start: start1, Size: end1 - start1 + 1}
		// 不相交1
		rng2 := common.Range{Start: start2, Size: start1 - start2}
		// 不相交2
		rng3 := common.Range{Start: end1 + 1, Size: end2 - end1}

		if rng1.Size != 0 {
			// copy a new value
			value := *ir.Value.(*common.RBTreeValue_Mint)
			value.InscriptionIds = append(value.InscriptionIds, inscriptionId)
			p.tree.Put(&rng1, &value)
		}
		if rng2.Size != 0 {
			if rng2.Start >= rng.Start && rng2.Start+rng2.Size-1 <= rng.Start+rng.Size-1 {
				p.tree.Put(&rng2, newValue)
			} else {
				value := ir.Value.(*common.RBTreeValue_Mint)
				p.tree.Put(&rng2, value)
			}
		}
		if rng3.Size != 0 {
			if rng3.Start >= rng.Start && rng3.Start+rng3.Size-1 <= rng.Start+rng.Size-1 {
				p.tree.Put(&rng3, newValue)
			} else {
				value := ir.Value.(*common.RBTreeValue_Mint)
				p.tree.Put(&rng3, value)
			}
		}
	}
}
