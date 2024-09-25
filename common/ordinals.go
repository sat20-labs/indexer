package common

import (
	"fmt"
	"math"
)

func NewUTXOIndex() *UTXOIndex {
	return &UTXOIndex{
		Index: make(map[string]*Output),
	}
}

func SubsidyInTheory(height int) int64 {

	epoch := int64(math.Floor(float64(height) / 210000))
	ret := int64(50 * 100000000 >> epoch)

	// the first leak block
	// if height == 124724 {
	// 	return ret - 1000001
	// }
	// the biggest leak block
	// if height == 501726 {
	// 	return ret - 1250000000
	// }
	// ...
	// the last leak block
	// if height == 626205 {
	// 	return ret - 68
	// }

	return ret
}

func FirstOrdinalInTheory(height int) int64 {
	start := int64(0)
	for i := 0; i < height; i++ {
		start += SubsidyInTheory(i)
	}
	return start
}

func TransferRanges(ordinals []*Range, value int64) ([]*Range, []*Range) {
	remainingValue := value
	remaining := ordinals
	transferred := make([]*Range, 0)
	for remainingValue > 0 {
		currentRange := remaining[0]
		start := currentRange.Start
		transferSize := currentRange.Size
		if transferSize > remainingValue {
			transferSize = remainingValue
		}

		transferred = append(transferred, &Range{Start: start, Size: transferSize})
		remainingSize := currentRange.Size - transferSize

		if remainingSize == 0 {
			remaining = remaining[1:]
		} else {
			remaining[0] = &Range{Start: start + transferSize, Size: remainingSize}
		}

		remainingValue = remainingValue - transferSize
	}

	return transferred, remaining
}


/*
https://blog.csdn.net/u014633283/article/details/104759834
91812与91842，91722与91880，有两对txid相同的交易，需要做特殊处理

91812和91842：
d5d27987d2a3dfc724e359870c6644b40e497bdc0589a033220fe15429d88599:0 50BTC

91722与91880
e3bf3d07d4b0375638d5f1db5255fe07ba2c4cb067cd81b84ee974b6585fb468:0 50BTC
*/
func GetUtxo(height int, tx string, vout int) string {
	u := ""
	if height == 91842 && tx == "d5d27987d2a3dfc724e359870c6644b40e497bdc0589a033220fe15429d88599" && vout == 0 {
		// 将第二个tx的输出，当做 d5d27987d2a3dfc724e359870c6644b40e497bdc0589a033220fe15429d88599:1
		u = fmt.Sprintf("%s:1", tx)
	} else if height == 91880 && tx == "e3bf3d07d4b0375638d5f1db5255fe07ba2c4cb067cd81b84ee974b6585fb468" && vout == 0 {
		// 将第二个tx的输出，当做 e3bf3d07d4b0375638d5f1db5255fe07ba2c4cb067cd81b84ee974b6585fb468:1
		u = fmt.Sprintf("%s:1", tx)
	} else {
		u = fmt.Sprintf("%s:%d", tx, vout)
	}
	return u
}

func InterRange(r1, r2 *Range) *Range {
	start := max(r1.Start, r2.Start)
	end := min(r1.Start+r1.Size-1, r2.Start+r2.Size-1)
	if start > end {
		// 无相交部分
		return &Range{Start: -1, Size: 0}
	}
	return &Range{
		Start: start,
		Size:  end - start + 1,
	}
}

func RangeComparator(rangeA, rangeB *Range) int {
	// 范围有相交即相等
	if rangeA.Start+rangeA.Size-1 < rangeB.Start {
		return -1
	} else if rangeA.Start > rangeB.Start+rangeB.Size-1 {
		return 1
	}
	return 0
}

func GetOrdinalsSize(ordinals []*Range) int64 {
	size := int64(0)
	for _, rng := range ordinals {
		size += (rng.Size)
	}
	return size
}

// rng1 contains rng2
func RangesContained(rng1, rng2 []*Range) bool {

	for _, it1 := range rng2 {
		bFound := false
		for _, it2 := range rng1 {
			inter := InterRange(it1, it2)
			if inter.Size == it1.Size {
				bFound = true
				break
			}
		}
		if !bFound {
			return false
		}
	}
	return true
}

// check if sat in the range
func IsSatInRanges(sat int64, rng []*Range) bool {
	for _, it := range rng {
		if sat >= it.Start && sat < it.Start+it.Size {
			return true
		}
	}
	return false
}
