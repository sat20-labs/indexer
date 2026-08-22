package nft

import (
	"sort"

	"github.com/sat20-labs/indexer/common"
)

func satOffsetsForPersistence(utxoID uint64, sats map[int64]int64) []*SatOffset {
	keys := make([]int64, 0, len(sats))
	for sat := range sats {
		if sat == 0 {
			common.Log.Warnf("skip invalid zero sat in utxo %d", utxoID)
			continue
		}
		keys = append(keys, sat)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	result := make([]*SatOffset, 0, len(keys))
	for _, sat := range keys {
		result = append(result, &SatOffset{Sat: sat, Offset: sats[sat]})
	}
	return result
}
