package exotic

import (
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func (p *ExoticIndexer) getBlockInBuffer(height int) *common.BlockValueInDB {
	return p.baseIndexer.GetBlockInBuffer(height)
}

func (p *ExoticIndexer) getBlockRange(height int, txn common.ReadBatch) *common.Range {

	if height < 0 || height > p.baseIndexer.GetHeight() {
		return nil
	}

	block := p.getBlockInBuffer(height)
	if block != nil {
		return &block.Ordinals
	}

	key := db.GetBlockDBKey(height)
	block = &common.BlockValueInDB{}
	err := db.GetValueFromTxn([]byte(key), block, txn)
	if err != nil {
		common.Log.Panicf("GetValueFromDB %s failed. %v", key, err)
	}
	return &block.Ordinals
}

func (p *ExoticIndexer) getRangeForBlock(height int, txn common.ReadBatch) []*common.Range {
	rng := p.getBlockRange(height, txn)
	return []*common.Range{rng}
}

func (p *ExoticIndexer) getRangeToBlock(height int, txn common.ReadBatch) []*common.Range {
	rng := p.getBlockRange(height, txn)
	r := &common.Range{
		Start: 0,
		Size:  rng.Start + rng.Size,
	}
	return []*common.Range{r}
}

func (p *ExoticIndexer) getRangesForBlocks(blocks []int, txn common.ReadBatch) []*common.Range {
	ranges := []*common.Range{}
	for _, b := range blocks {
		ranges = append(ranges, p.getRangeForBlock(b, txn)...)
	}
	return ranges
}

// 速度很慢，最好是在跑完数据才更新
func (p *ExoticIndexer) InitRarityDB(height int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	start := time.Now()
	bs := NewBuckStore(p.db, string(Uncommon))
	syncHeight := bs.GetLastKey()
	if syncHeight == height {
		return
	} else if syncHeight > height {
		syncHeight = -1
		bs.Reset()
	}

	Uncommon := make(map[int]*common.Range, 0)
	p.db.View(func(txn common.ReadBatch) error {
		for i := syncHeight + 1; i <= height; i++ {
			rng := p.getBlockRange(i, txn)
			r := common.Range{
				Start: rng.Start,
				Size:  1,
			}
			Uncommon[i] = &r
		}
		return nil
	})

	bs.BatchPut(Uncommon)

	common.Log.Infof("InitRarityDB %d takes %v", height, time.Since(start))
}

func AddAsset(output *common.TxOutput, name string, offset int64, amt int64) {
	asset := common.AssetInfo{
		Name: common.AssetName{
			Protocol: common.PROTOCOL_NAME_ORD,
			Type: common.ASSET_TYPE_EXOTIC,
			Ticker: name,
		},
		Amount: *common.NewDecimal(amt, 0),
		BindingSat: 1,
	}
	output.Assets.Add(&asset)
	output.Offsets[asset.Name] = common.AssetOffsets{
		{
			Start: offset,
			End: offset+amt,
		},
	}
}

func (p *ExoticIndexer) GenerateRodarmorRarityAssets(block *common.Block, 
	coinbase []*common.Range)  {

	// // uncommon
	// tx := block.Transactions[0]
	
	
	// for height, rng := range rngsInBlock {
	// 	firstSatInBlock = append(firstSatInBlock, rng)
	// 	p.firstSatInBlock.Put(rng.Start, height)
	// }
	// result[Uncommon] = firstSatInBlock

	// // black
	// lastSatInBlock := make([]*common.Range, 0)
	// for _, rng := range firstSatInBlock {
	// 	lastSatInBlock = append(lastSatInBlock, &common.Range{Start: rng.Start - 1, Size: 1})
	// }
	// result[Black] = lastSatInBlock

	// // mythic
	// rng := p.getBlockRange(0, txn)
	// r := common.Range{Start: rng.Start, Size: 1}
	// result[Mythic] = append(result[Mythic], &r)

	// for i := CycleInterval; i <= height; i += CycleInterval {
	// 	rng := p.getBlockRange(i, txn)
	// 	r := common.Range{Start: rng.Start, Size: 1}
	// 	result[Legendary] = append(result[Legendary], &r)
	// }

	// for i := HalvingInterval; i <= height; i += HalvingInterval {
	// 	if i == 0 {
	// 		continue
	// 	} else if i%CycleInterval == 0 {
	// 		continue
	// 	}
	// 	rng := p.getBlockRange(i, txn)
	// 	r := common.Range{Start: rng.Start, Size: 1}
	// 	result[Legendary] = append(result[Legendary], &r)
	// }

	// for i := DificultyAdjustmentInterval; i <= height; i += DificultyAdjustmentInterval {
	// 	if i == 0 {
	// 		continue
	// 	} else if i%CycleInterval == 0 {
	// 		continue
	// 	} else if i%HalvingInterval == 0 {
	// 		continue
	// 	}
	// 	rng := p.getBlockRange(i, txn)
	// 	r := common.Range{Start: rng.Start, Size: 1}
	// 	result[Legendary] = append(result[Legendary], &r)
	// }

	// common.Log.Infof("GetRangesForRodarmorRarity %d takes %v", height, time.Since(start))

	// // for i := 0; i <= height; i++ {
	// // 	rng := p.getBlockRange(i, txn)
	// // 	r := define.Range{
	// // 		Start: rng.Start,
	// // 		Size:  1,
	// // 	}

	// // 	if i == 0 {
	// // 		result[Mythic] = append(result[Mythic], &r)
	// // 	} else if i%CycleInterval == 0 {
	// // 		result[Legendary] = append(result[Legendary], &r)
	// // 	} else if i%HalvingInterval == 0 {
	// // 		result[Epic] = append(result[Epic], &r)
	// // 	} else if i%DificultyAdjustmentInterval == 0 {
	// // 		result[Rare] = append(result[Rare], &r)
	// // 	} else {
	// // 		result[Uncommon] = append(result[Uncommon], &r)
	// // 	}
	// // }

	// return result
}

func (p *ExoticIndexer) getMoreRodarmorRarityRangesToHeight(startHeight, endHeight int, txn common.ReadBatch) map[string][]*common.Range {
	result := make(map[string][]*common.Range, 0)

	for i := startHeight; i <= endHeight; i++ {
		rng := p.getBlockRange(i, txn)
		if rng == nil {
			break
		}

		r := common.Range{
			Start: rng.Start,
			Size:  1,
		}

		if i == 0 {
			result[Mythic] = append(result[Mythic], &r)
		} else if i%CycleInterval == 0 {
			result[Legendary] = append(result[Legendary], &r)
		} else if i%HalvingInterval == 0 {
			result[Epic] = append(result[Epic], &r)
		} else if i%DificultyAdjustmentInterval == 0 {
			result[Rare] = append(result[Rare], &r)
		} else {
			result[Uncommon] = append(result[Uncommon], &r)

			r2 := common.Range{
				Start: r.Start - 1,
				Size:  1,
			}
			result[Black] = append(result[Black], &r2)
		}
	}
	return result
}

func (p *ExoticIndexer) getRangesForAlpha(startHeight, endHeight int, txn common.ReadBatch) []*common.Range {
	ranges := []*common.Range{}
	rng1 := p.getBlockRange(startHeight, txn)
	rng2 := p.getBlockRange(endHeight, txn)
	sat1 := rng1.Start
	sat2 := rng2.Start + rng2.Size
	sat1 = (sat1) / 1e8
	sat2 = (sat2) / 1e8
	for i := sat1; i < sat2; i++ {
		r := &common.Range{
			Start: i * 1e8,
			Size:  1,
		}
		ranges = append(ranges, r)
	}
	return ranges
}

func (p *ExoticIndexer) getRangesForOmega(startHeight, endHeight int, txn common.ReadBatch) []*common.Range {
	ranges := []*common.Range{}
	rng1 := p.getBlockRange(startHeight, txn)
	rng2 := p.getBlockRange(endHeight, txn)
	sat1 := rng1.Start
	sat2 := rng2.Start + rng2.Size
	sat1 = (sat1) / 1e8
	sat2 = (sat2) / 1e8
	for i := sat1; i < sat2; i++ {
		if i == 0 {
			continue
		}
		r := &common.Range{
			Start: i*1e8 - 1,
			Size:  1,
		}
		ranges = append(ranges, r)
	}
	return ranges
}

// 所有事先定义的稀有聪
// ticker->utxo->offset
func (p *ExoticIndexer) getDefaultExoticTickerMap() map[string]map[string]common.AssetOffsets {
	result := make(map[string]map[string]common.AssetOffsets)
	

	
	result[Pizza] = map[string]common.AssetOffsets{
		PIZZA_UTXO: common.AssetOffsets{
			{
				Start: 0,
				End: PIZZA_VALUE,
			},
		},
	}

	result[Block9] = map[string]common.AssetOffsets{
		"0437cd7f8525ceed2324359c2d0ba26006d92d856a9c20fa0241106ee5a597c9:0": common.AssetOffsets{
			{
				Start: 0,
				End: 50*100000000,
			},
		},
	}

	result[Block78] = map[string]common.AssetOffsets{
		"7ea1d2304f1f95fae773ed8ef67b51cfd5ab33ea8b6ab0a932ee3e248b7ba74c:0": common.AssetOffsets{
			{
				Start: 0,
				End: 50*100000000,
			},
		},
	}

	result[FirstTransaction] = map[string]common.AssetOffsets{
		"f4184fc596403b9d638783cf57adfe4c75c605f6356fbc91338530e9831e9e16:0": common.AssetOffsets{
			{
				Start: 0,
				End: 10*100000000,
			},
		},
	}


		// validBlock := make([]int, 0)
		// for h := range NakamotoBlocks {
		// 	if h <= height {
		// 		validBlock = append(validBlock, h)
		// 	}
		// }
		// result[Nakamoto] = p.getRangesForBlocks(validBlock, txn)

		// if height >= 1000 {
		// 	result[Vintage] = p.getRangeToBlock(1000, txn)
		// }

		// // TODO
		// //result[Alpha] = GetRangesForAlpha(0, height)
		// //result[Omega] = GetRangesForOmega(0, height)

		// //result[Hitman] = HitmanRanges
		// //result[Jpeg] = JpegRanges
		// //result[Fibonacci] =

		// if IsTestNet {
		// 	result[Customized] = CustomizedRange
		// }
		// return nil


	return result
}
