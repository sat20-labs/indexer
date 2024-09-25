package indexer

import (
	"fmt"

	"github.com/sat20-labs/ordx/common"
)

func (b *IndexerMgr) GetExoticsWithRanges(ranges []*common.Range) []*common.ExoticRange {
	return b.exotic.GetExoticsWithRanges(ranges)
}

func (b *IndexerMgr) GetExoticsWithType(ranges []*common.Range, typ string) []*common.ExoticRange {
	return b.exotic.GetExoticsWithType(ranges, typ)
}

func (b *IndexerMgr) HasExoticInRanges(ranges []*common.Range) bool {
	return b.exotic.HasExoticInRanges(ranges)
}

func (b *IndexerMgr) getExoticsWithUtxo(utxoId uint64) map[string]map[string][]*common.Range {
	_, rngs, err := b.rpcService.GetOrdinalsWithUtxoId(utxoId)
	if err != nil {
		return nil
	}
	result := make(map[string]map[string][]*common.Range)
	rngmap := b.exotic.GetExoticsWithRanges2(rngs)
	for k, v := range rngmap {
		info := make(map[string][]*common.Range)
		key := fmt.Sprintf("%s:%s:%x", common.ASSET_TYPE_EXOTIC, k, utxoId)
		info[key] = v
		result[k] = info
	}
	return result
}

// return: name -> utxoId
func (b *IndexerMgr) getExoticUtxos(utxos map[uint64]int64) map[string][]uint64 {
	result := make(map[string][]uint64, 0)
	for utxoId := range utxos {
		_, rng, err := b.GetOrdinalsWithUtxoId(utxoId)
		if err != nil {
			common.Log.Errorf("GetOrdinalsWithUtxoId failed, %d", utxoId)
			continue
		}

		sr := b.exotic.GetExoticsWithRanges2(rng)
		for t := range sr {
			result[t] = append(result[t], utxoId)
		}
	}

	return result
}

// return: name -> utxoId
func (b *IndexerMgr) getExoticSummaryByAddress(utxos map[uint64]int64) map[string]int64 {
	result := make(map[string]int64, 0)
	for utxoId := range utxos {
		_, rng, err := b.GetOrdinalsWithUtxoId(utxoId)
		if err != nil {
			common.Log.Errorf("GetOrdinalsWithUtxoId failed, %d", utxoId)
			continue
		}

		sr := b.exotic.GetExoticsWithRanges2(rng)
		for t, rngs := range sr {
			result[t] += common.GetOrdinalsSize(rngs)
		}
	}

	return result
}
