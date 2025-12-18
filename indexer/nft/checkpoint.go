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
	780000: {
		InscriptionMap: map[string]int64{
			"c76fc75b8efd1bc4224486fe43384022c10256894f2473c439677a8523cd214ai0":362032,
		},
	},

	790000: {
		InscriptionMap: map[string]int64{
			"b1bd6f1b06a71f9e1cdda0c326cc308801fd2261be03f1fcbaa3a0cee017a793i0":7201580,
		},
	},

	800000: {
		InscriptionMap: map[string]int64{
			"965f866bf8623bbf956c1b2aeec1efc1ad162fd428ab7fb89f128a0754ebbc32i0":19003884,
		},
	},

	810000: {
		InscriptionMap: map[string]int64{
			"93cba414f354804c361266cd7c3295e95338792ba7ea600ed97256580d7ff429i0":34888874,
			"a919b31709835eeabb880d74f228ad944da95b1de0b9712cc881bd37448a5c5ai0":-214734,
		},
	},

	820000: {
		InscriptionMap: map[string]int64{
			"0187d7a3f4bc4147ac684456a4c83d78118fc18dbc3023754d269091b7c17ee9i0":46274954,
		},
	},

	830000: {
		InscriptionMap: map[string]int64{
			"1754cddb83ba997b7c67bf77220de94343af85ba48690a7c8cacce7bc7338297i0":60538260,
		},
	},

	840000: {
		InscriptionMap: map[string]int64{
			"152b928e97bb9e874da1bd4abdf766ae0cdc7a2f260dad5542967cb414c58489i0":70279589,
		},
	},

	850000: {
		InscriptionMap: map[string]int64{
			"e6a854c10dd771088a84466c4f659518780ef34a62615cb27a07d61f1ecbcc44i0":72037579,
		},
	},

	860000: {
		InscriptionMap: map[string]int64{
			"1e2bc81be65a49ebea824e23ca0b76d749ba36478bb35a0c9b467cc49f183cb1i0":75510934,
		},
	},

	870000: {
		InscriptionMap: map[string]int64{
			"2f7f50d40f1bb85ff825d3bfb6e6fdb9c76d2064a656fa9b71849d315350fa90i0":77323828,
		},
	},

	880000: {
		InscriptionMap: map[string]int64{
			"020a9bc259913b0fd081a31aab41b1aee4029a2bc947d332e2c9accf624d7543i0":83273416,
		},
	},

	890000: {
		InscriptionMap: map[string]int64{
			"1752e4c19d2307b219a14fb14c5595a63a6c7cb34366f8d5d8d42c3ca7f4eda8i0":91815319,
		},
	},

	900000: {
		InscriptionMap: map[string]int64{
			"34bb038077fc5c45ee8a15e8ded26051ed804342357d4dccd8e4ac4e3168a8f9i0":97183740,
		},
	},

	910000: {
		InscriptionMap: map[string]int64{
			"44284c209d99fe2571129a1a43ee1082adab7ec8a4158fdc0cbd1934ed66d3f9i0":103456684,
		},
	},

	920000: {
		InscriptionMap: map[string]int64{
			"061c1c318d463c0beefdd0169c34d855a9529470d85cf94da6430e64a4fc31b8i0":108033293,
		},
	},

	930000: {
		InscriptionMap: map[string]int64{
			"061c1c318d463c0beefdd0169c34d855a9529470d85cf94da6430e64a4fc31b8i0":108033293,
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
