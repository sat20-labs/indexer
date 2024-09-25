package nft

import (
	"sort"

	indexer "github.com/sat20-labs/ordx/indexer/common"
)

func BindIdstoSat(p *indexer.SatRBTree, sat int64, ids []int64) error {
	value := p.FindNode((sat))
	if value != nil {
		value.(*RBTreeValue_NFTs).Ids = mergeToVector(value.(*RBTreeValue_NFTs).Ids, ids)
	} else {
		value = &RBTreeValue_NFTs{Ids: mergeToVector(nil, ids)}
		p.Put(sat, value)
	}

	return nil
}

func mergeToVector(vect1, vect2 []int64) []int64 {
	idsmap := make(map[int64]bool)
	for _, v := range vect1 {
		idsmap[v] = true
	}
	for _, v := range vect2 {
		idsmap[v] = true
	}
	result := make([]int64, 0)
	for k := range idsmap {
		result = append(result, k)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i] > result[j]
	})

	return result
}

func mapToVector(map1 map[int64]bool) []int64 {
	result := make([]int64, 0)
	for k := range map1 {
		result = append(result, k)
	}
	return result
}
