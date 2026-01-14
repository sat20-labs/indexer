package indexer

import (

	"github.com/sat20-labs/indexer/common"
)


func (b *IndexerMgr) GetExotics(utxoId uint64) map[string]common.AssetOffsets {
	return b.exotic.GetAssetsWithUtxo(utxoId)
}


func (b *IndexerMgr) GetExoticsWithType(utxoId uint64, typ string) common.AssetOffsets {
	return b.exotic.GetExoticsWithType(utxoId, typ)
}


func (b *IndexerMgr) getExoticsWithUtxo(utxoId uint64) map[string]common.AssetOffsets {
	return b.exotic.GetAssetsWithUtxo(utxoId)
}

// return: name -> utxoId
func (b *IndexerMgr) getExoticUtxos(utxos map[uint64]int64) map[string][]uint64 {
	result := make(map[string][]uint64, 0)
	// for utxoId := range utxos {
	// 	_, rng, err := b.GetOrdinalsWithUtxoId(utxoId)
	// 	if err != nil {
	// 		common.Log.Errorf("GetOrdinalsWithUtxoId failed, %d", utxoId)
	// 		continue
	// 	}

	// 	sr := b.exotic.GetExoticsWithRanges2(rng)
	// 	for t := range sr {
	// 		result[t] = append(result[t], utxoId)
	// 	}
	// }

	for utxo := range utxos {
		assets := b.getExoticsWithUtxo(utxo)
		if len(assets) != 0 {
			for name := range assets {
				result[name] = append(result[name], utxo)
			}
		}
	}

	return result
}

// return: name -> utxoId
func (b *IndexerMgr) getExoticSummaryByAddress(utxos map[uint64]int64) (map[string]int64, []uint64) {
	result := make(map[string]int64, 0)
	var plainUtxo []uint64

	for utxo := range utxos {
		assets := b.getExoticsWithUtxo(utxo)
		if len(assets) != 0 {
			for name, offsets := range assets {
				result[name] += offsets.Size()
			}
		} else {
			plainUtxo = append(plainUtxo, utxo)
		}
	}

	return result, plainUtxo
}
