package exotic

import (
	"fmt"

	"github.com/sat20-labs/ordx/common"
)

func (p *ExoticIndexer) GetExoticsWithRanges(ranges []*common.Range) []*common.ExoticRange {
	result := []*common.ExoticRange{}
	if p.exoticTickerMap == nil {
		return nil
	}

	// 需要保持range的顺序，同时尽可能让每一段range的属性能累加在一起
	offset := int64(0)
	for _, rng := range ranges {
		resmap := make(map[string][]*common.ExoticRange, 0)
		for name, tickinfo := range p.exoticTickerMap {
			intersections := tickinfo.MintInfo.FindIntersections(rng)
			for _, it := range intersections {
				exr := common.ExoticRange{Range: it.Rng, Offset: offset + it.Rng.Start - rng.Start,
					Satributes: []string{string(name)}}
				key := fmt.Sprintf("%d-%d", exr.Range.Start, exr.Range.Size)
				resmap[key] = append(resmap[key], &exr)
			}
		}

		for _, exranges := range resmap {
			satributes := make([]string, 0)
			for _, exr := range exranges {
				satributes = append(satributes, exr.Satributes...)
			}
			exr := exranges[0]
			exr.Satributes = satributes
			result = append(result, exr)
		}

		offset += rng.Size
	}

	return result
}

func (p *ExoticIndexer) HasExoticInRanges(ranges []*common.Range) bool {

	if p.exoticTickerMap == nil {
		return false
	}

	for _, rng := range ranges {
		for _, tickinfo := range p.exoticTickerMap {
			intersections := tickinfo.MintInfo.FindIntersections(rng)
			if len(intersections) > 0 {
				return true
			}
		}
	}

	return false
}

func (p *ExoticIndexer) GetExoticsWithType(ranges []*common.Range, typ string) []*common.ExoticRange {
	result := make([]*common.ExoticRange, 0)
	exoticRanges := p.GetExoticsWithRanges(ranges)
	for _, rng := range exoticRanges {
		for _, satr := range rng.Satributes {
			if typ == string(satr) {
				result = append(result, &common.ExoticRange{Range: rng.Range, Offset: rng.Offset})
				break
			}
		}
	}
	return result
}

func (p *ExoticIndexer) GetExoticsWithRanges2(ranges []*common.Range) map[string][]*common.Range {
	res := make(map[string][]*common.Range)

	if p.exoticTickerMap == nil {
		return nil
	}
	for _, rng := range ranges {
		for name, tickinfo := range p.exoticTickerMap {
			intersections := tickinfo.MintInfo.FindIntersections(rng)
			for _, it := range intersections {
				res[name] = append(res[name], it.Rng)
			}
		}
	}

	return res
}

func (p *ExoticIndexer) GetExoticsWithType2(ranges []*common.Range, typ string) []*common.Range {

	exoticRanges := p.GetExoticsWithRanges2(ranges)
	return exoticRanges[typ]
}
