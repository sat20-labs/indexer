package exotic

import "github.com/sat20-labs/indexer/common"

// satingRange is a range of sats as defined by sating.io
type satingRange struct {
	start int64
	end   int64
}

func (r *satingRange) ToOrdinalsRange() *common.Range {
	return &common.Range{
		Start: r.start,
		Size:  r.end - r.start,
	}
}

func SatingRangesToOrdinalsRanges(ranges []*satingRange) []*common.Range {
	var result []*common.Range
	for _, r := range ranges {
		result = append(result, r.ToOrdinalsRange())
	}
	return result
}
