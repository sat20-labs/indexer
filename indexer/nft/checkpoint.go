package nft

import (
	"math"
	"time"

	"github.com/sat20-labs/indexer/common"
)

type CheckPoint struct {
	NftCount       uint64
	InscriptionMap map[string]int64
}

var testnet4_checkpoint = map[int]*CheckPoint{
	27227: {
		NftCount:       0,
		InscriptionMap: nil,
	},

	30000: {
		NftCount: 194,
		InscriptionMap: map[string]int64{
			"0ab6fc4e95b0bfcf31a242d6cde21421586372c6bbf5bc3e23432df606363f3fi0": 142,
			"103a16e73c97bca1df6ec280af4cf6b7e9540b45791e6634f362cc42f4d4ac76i0": 193,
		},
	},
}

var mainnet_checkpoint = map[int]*CheckPoint{
	780000: {
		InscriptionMap: map[string]int64{
			"c76fc75b8efd1bc4224486fe43384022c10256894f2473c439677a8523cd214ai0": 362032,
		},
	},
	
	788200: {
		InscriptionMap: map[string]int64{
			"c1e0db6368a43f5589352ed44aa1ff9af33410e4a9fd9be0f6ac42d9e4117151i0": 3492721, // unbound
		},
	},

	788312: {
		InscriptionMap: map[string]int64{
			"99e70421ab229d1ccf356e594512da6486e2dd1abdf6c2cb5014875451ee8073i0": 3644015, // unbound
		},
	},

	790000: {
		InscriptionMap: map[string]int64{
			"b1bd6f1b06a71f9e1cdda0c326cc308801fd2261be03f1fcbaa3a0cee017a793i0": 7201580,
		},
	},

	800000: {
		InscriptionMap: map[string]int64{
			"965f866bf8623bbf956c1b2aeec1efc1ad162fd428ab7fb89f128a0754ebbc32i0": 19003884,
		},
	},

	810000: {
		InscriptionMap: map[string]int64{
			"93cba414f354804c361266cd7c3295e95338792ba7ea600ed97256580d7ff429i0": 34888874,
			"a919b31709835eeabb880d74f228ad944da95b1de0b9712cc881bd37448a5c5ai0": -214734,
		},
	},
	816000: {
		InscriptionMap: map[string]int64{
			"6bf226eb80e0aa9032b11a7b101697726e008a9c44e1cf89cd666c06d10e525fi0": 38730360,
		},
	},
	816001: {
		InscriptionMap: map[string]int64{
			"028f955607b7fb7c645a2fd305d742009324095fe452329ac6dcc5ad465049a9i0": 38731385,
		},
	},

	820000: {
		InscriptionMap: map[string]int64{
			"0187d7a3f4bc4147ac684456a4c83d78118fc18dbc3023754d269091b7c17ee9i0": 46274954,
		},
	},

	824290: {
		InscriptionMap: map[string]int64{
			"8a8a53e144bf78c5dc16364a89765af84c2e6221b349952d78d3313468d74291i0": 53166694,
			"67477f68d5b0505c54456dfcc3fe054950abf6ff17e5c4d5d608f0818f7e2155i0": 53166800,
		},
	},

	824291: {
		InscriptionMap: map[string]int64{
			"21752d77a4e25cd4a6dc28c92395124c8337fe3951a6397922b16ed19055fb94i0": 53166900,
			"8edee91e3eb18dea399d8da29550782e5d7a15d22a1ccd60ae82169ef7ed25b0i0": 53167799,
			"fef75422456e29c39238abc145abbf38c644809378def4156b19a9a8ce0da118i0": 53167000,
			"652ed241027b95ffaf34000fed3bf99d50c95df2550c0edc7a2ea8ba42587695i0": 53167100,
			"695fae12fef95a74ee26ad39e70ad951fa742cc069d5d7554bd29c36a308cf3ci0": 53167200,
			"c33723a7f6f507a57315bf6a404a0aadf0d39687f34c9149207c9f691a0433c2i0": 53167801, // fee spent
			"63226a8984f09d6ef436afa9bd918f084976e6dadbef6eba80463fe94d8b83bei0": 53167300,
			"4b359bce14a930e38a2dfa903550556438e251b5f1297480950f8f2811f9c254i0": 53167400,
			"18b55648d8bbc60ca0f5997b4c48eb6edc074c4754c5311a3f1ffa4b4a7836dbi0": 53167500,
			"47898ac9f86f35807f25f02e6834dee9c7435b9f6005e612eadf8b1ac816b300i0": 53167600,
			
		},
	},
	
	824361: {
		InscriptionMap: map[string]int64{
			"49b3e21c2652d1ddc9e574c2b1cefb37ce2be63c0caf1800e6c627e12c429a43i0": -419018,
			"cd36e1d3c3a9a763a579d07c1cc80b242d655894b51a8a2f008a22c34a5c9300i0": 53217736,
		},
	},

	830000: {
		InscriptionMap: map[string]int64{
			"1754cddb83ba997b7c67bf77220de94343af85ba48690a7c8cacce7bc7338297i0": 60538260,
		},
	},

	840000: {
		InscriptionMap: map[string]int64{
			"152b928e97bb9e874da1bd4abdf766ae0cdc7a2f260dad5542967cb414c58489i0": 70279589,
		},
	},

	850000: {
		InscriptionMap: map[string]int64{
			"e6a854c10dd771088a84466c4f659518780ef34a62615cb27a07d61f1ecbcc44i0": 72037579,
		},
	},

	860000: {
		InscriptionMap: map[string]int64{
			"1e2bc81be65a49ebea824e23ca0b76d749ba36478bb35a0c9b467cc49f183cb1i0": 75510934,
		},
	},

	870000: {
		InscriptionMap: map[string]int64{
			"2f7f50d40f1bb85ff825d3bfb6e6fdb9c76d2064a656fa9b71849d315350fa90i0": 77323828,
		},
	},

	880000: {
		InscriptionMap: map[string]int64{
			"020a9bc259913b0fd081a31aab41b1aee4029a2bc947d332e2c9accf624d7543i0": 83273416,
		},
	},

	890000: {
		InscriptionMap: map[string]int64{
			"1752e4c19d2307b219a14fb14c5595a63a6c7cb34366f8d5d8d42c3ca7f4eda8i0": 91815319,
		},
	},

	900000: {
		InscriptionMap: map[string]int64{
			"34bb038077fc5c45ee8a15e8ded26051ed804342357d4dccd8e4ac4e3168a8f9i0": 97183740,
		},
	},

	910000: {
		InscriptionMap: map[string]int64{
			"44284c209d99fe2571129a1a43ee1082adab7ec8a4158fdc0cbd1934ed66d3f9i0": 103456684,
		},
	},

	920000: {
		InscriptionMap: map[string]int64{
			"061c1c318d463c0beefdd0169c34d855a9529470d85cf94da6430e64a4fc31b8i0": 108033293,
		},
	},

	930000: {
		InscriptionMap: map[string]int64{
			"061c1c318d463c0beefdd0169c34d855a9529470d85cf94da6430e64a4fc31b8i0": 108033293,
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

	if isMainnet {
		if height >= common.Jubilee_Height && p.status.CurseCount != 472043 {
			common.Log.Panicf("invalid curse count %d", p.status.CurseCount)
		}
	} else {
		if p.status.CurseCount > 0 {
			common.Log.Panicf("invalid curse count %d", p.status.CurseCount)
		}
	}

	if checkpoint.NftCount != 0 && checkpoint.NftCount != p.status.Count+p.status.CurseCount {
		common.Log.Panicf("inscription count different %d %d", checkpoint.NftCount, p.status.Count+p.status.CurseCount)
	}

	dismatch := false
	minNum := int64(math.MaxInt64)
	for inscriptionId, num := range checkpoint.InscriptionMap {
		nft := p.getNftWithInscriptionId(inscriptionId)
		if nft == nil {
			common.Log.Panicf("getNftWithInscriptionId %s failed", inscriptionId)
		}
		if nft.Base.Id != num {
			common.Log.Errorf("%s inscription num different %d %d", inscriptionId, num, nft.Base.Id)
			dismatch = true
			if minNum > num {
				minNum = num
			}
		}
	}
	if dismatch {
		for i := minNum - 100; i <= minNum; i++ {
			nft := p.getNftWithId(i)
			common.Log.Infof("%d %s %d", i, nft.Base.InscriptionId, nft.Base.Sat)
		}

		common.Log.Panic("")
	}

	common.Log.Infof("NftIndexer.CheckPointWithBlockHeight %d checked, takes %v", height, time.Since(startTime))
}
