package nft

import (
	"time"

	"github.com/sat20-labs/indexer/common"
)


type CheckPoint struct {
	NftCount uint64
	InscriptionMap map[string]int64
}

var testnet4_checkpoint = map[int]*CheckPoint{
	27227: {
		NftCount: 0,
		InscriptionMap: nil,
	},

	30000: {
		NftCount: 194,
		InscriptionMap: map[string]int64{
			"ordi": 0,
		},
	},
}

var mainnet_checkpoint = map[int]*CheckPoint{
	27227: {
		NftCount: 0,
		InscriptionMap: nil,
	},

	30000: {
		NftCount: 12,
		InscriptionMap: map[string]int64{
			"ordi": 0,
		},
	},
}



func (p *NftIndexer) CheckPointWithBlockHeight(height int) {

	startTime := time.Now()
	var checkpoint *CheckPoint
	isMainnet := p.baseIndexer.IsMainnet()
	if isMainnet {
		checkpoint = mainnet_checkpoint[height]
	} else {
		checkpoint = testnet4_checkpoint[height]
	}
	if checkpoint == nil {
		return
	}

	if checkpoint.NftCount != 0 && checkpoint.NftCount != p.status.Count + p.status.CurseCount {
		common.Log.Panicf("inscription count different %d %d", checkpoint.NftCount, p.status.Count + p.status.CurseCount)
	}

	for inscriptionId, num := range checkpoint.InscriptionMap {
		nft := p.getNftWithInscriptionId(inscriptionId)
		if nft == nil {
			common.Log.Panicf("getNftWithInscriptionId %s failed", inscriptionId)
		}
		if nft.Base.Id != num {
			common.Log.Panicf("%s inscription num different %d %d", inscriptionId, num, nft.Base.Id)
		}
	}
	
	common.Log.Infof("NftIndexer.CheckPointWithBlockHeight %d checked, takes %v", height, time.Since(startTime))
}
