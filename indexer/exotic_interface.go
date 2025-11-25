package indexer

import (
	"fmt"

	"github.com/sat20-labs/indexer/common"
)


func (b *IndexerMgr) GetExotics(utxoId uint64) []*common.ExoticRange {
	return b.exotic.GetExotics(utxoId)
}


func (b *IndexerMgr) GetExoticsWithType(utxoId uint64, typ string) []*common.ExoticRange {
	return b.exotic.GetExoticsWithTypeV2(utxoId, typ)
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
func (b *IndexerMgr) getExoticSummaryByAddress(utxos map[uint64]int64) (map[string]int64, []uint64) {
	result := make(map[string]int64, 0)
	var plainUtxo []uint64
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

		if len(sr) == 0 {
			plainUtxo = append(plainUtxo, utxoId)
		}
	}

	return result, plainUtxo
}
