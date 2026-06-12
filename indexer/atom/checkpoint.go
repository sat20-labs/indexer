package atom

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
	atomValidate "github.com/sat20-labs/indexer/indexer/atom/validate"
)

type CheckPoint struct {
	Height         int
	TickerCount    int64
	AssetUtxoCount int
	Tickers        map[string]*TickerStatus
	RejectedTicker []string
}

type TickerStatus struct {
	AtomicalId   string
	DeployHeight int
	UtxoCount    int
	UtxoAmount   int64
	MintedTimes  int64
	MintedAmount int64
	MaxMints     int64
	HolderCount  int
	Holders      map[string]int64
}

var mainnetCheckpoint = map[int]*CheckPoint{
	808122: {
		Height:      808122,
		TickerCount: 1,
		Tickers: map[string]*TickerStatus{
			"atom": {
				AtomicalId: "56a8702bab3d2405eb9a356fd0725ca112a93a8efd1ecca06c6085e7278f0341i0",
			},
		},
	},
	808513: {
		Height:      808513,
		TickerCount: 1,
		Tickers: map[string]*TickerStatus{
			"atom": {
				AtomicalId: "56a8702bab3d2405eb9a356fd0725ca112a93a8efd1ecca06c6085e7278f0341i0",
				UtxoCount:  4,
				UtxoAmount: 4000,
			},
		},
	},
	808684: {
		Height:      808684,
		TickerCount: 6,
		Tickers: map[string]*TickerStatus{
			"abtc": {
				AtomicalId: "8888722295d5b9d38efbc262e8ac9a21356257ee49d23037eba6279c16b58c8bi0",
				UtxoCount:  15,
				UtxoAmount: 15000,
			},
			"atom": {
				AtomicalId: "56a8702bab3d2405eb9a356fd0725ca112a93a8efd1ecca06c6085e7278f0341i0",
				UtxoCount:  18976,
				UtxoAmount: 20689572,
			},
			"atombtc": {
				AtomicalId: "1200d535ed19e97345c2db6b264b16c42105c5ea9b0d245d2e49117dbe81dd01i0",
				UtxoCount:  8,
				UtxoAmount: 8000,
			},
			"doge": {
				AtomicalId: "00009b954c9f1358de9c089f95ec420132e4106a89c8fbb3cfda198ae1e5f9d5i0",
				UtxoCount:  18,
				UtxoAmount: 18000,
			},
			"ordi": {
				AtomicalId: "6969d53ee63166485d1d0c2dd96a4f2d1483c12cbe999447bc4cf6c73aeaa2b7i0",
			},
			"pepe": {
				AtomicalId: "9ba68637ba32edb6370bebceaac3df4341180cbf7bac210741b12a679692d716i0",
			},
		},
		RejectedTicker: []string{"rekt"},
	},
	808823: {
		Height:      808823,
		TickerCount: 7,
		Tickers: map[string]*TickerStatus{
			"shib": {
				AtomicalId:   "2574a2c35ab9bb2d5089f6482226390353d77bf307c485f1d3ce42fda06f1ab4i0",
				DeployHeight: 808823,
			},
		},
	},
	808947: {
		Height:      808947,
		TickerCount: 22,
		Tickers: map[string]*TickerStatus{
			"a": {
				AtomicalId: "66466d0207d9f50b5782678282e08cffb8e315e95da8d91b3d46f290b440a8bdi0",
			},
			"abtc": {
				AtomicalId: "8888722295d5b9d38efbc262e8ac9a21356257ee49d23037eba6279c16b58c8bi0",
			},
			"arcs": {
				AtomicalId: "9406ccceba2ef261f113fa14bfe156ec2b1d36d572f183e288ab4f4fb3e6cb8bi0",
			},
			"arcx": {
				AtomicalId: "983134562b186296060299fea8ac9799d6f1adef7dc81707a6cbd2241439a455i0",
			},
			"atm": {
				AtomicalId: "9ba6b2828464aaa14d3ce7054d028f8b7a4f83a09acf669ad531e17ffcc5e352i0",
			},
			"atom": {
				AtomicalId: "56a8702bab3d2405eb9a356fd0725ca112a93a8efd1ecca06c6085e7278f0341i0",
			},
			"atombtc": {
				AtomicalId: "1200d535ed19e97345c2db6b264b16c42105c5ea9b0d245d2e49117dbe81dd01i0",
			},
			"atomicals": {
				AtomicalId: "54799a35caf574c08d44216c354ef16344f9c5f6c8cf6e349468d90549c9ed5fi0",
			},
			"btc": {
				AtomicalId: "7296411f89e8e6171966a0b9d11e3fe12e86fdc3b0515b1f7bbb1c29d65f29adi0",
			},
			"btcs": {
				AtomicalId: "51184faf1a7a162db05a7311f92580748cc340521899d73cb779fa812b1d6848i0",
			},
			"doge": {
				AtomicalId: "00009b954c9f1358de9c089f95ec420132e4106a89c8fbb3cfda198ae1e5f9d5i0",
			},
			"domo": {
				AtomicalId: "54517e3c7a9c65b7f81fe6d373c5c71e7cb8b04a4e4d69bacc379c45af82e1b7i0",
			},
			"hash": {
				AtomicalId: "40178269a779aa15e8bf90d711c7eea95b59ffb4ef40db5d971e50be5cc28272i0",
			},
			"icals": {
				AtomicalId: "2679c605df1201f501b9827fa61e1405d19e37c8c9f8ac2dd8a67da2f87e76bfi0",
			},
			"ordi": {
				AtomicalId: "6969d53ee63166485d1d0c2dd96a4f2d1483c12cbe999447bc4cf6c73aeaa2b7i0",
			},
			"pepe": {
				AtomicalId: "9ba68637ba32edb6370bebceaac3df4341180cbf7bac210741b12a679692d716i0",
			},
			"rekt": {
				AtomicalId: "9ba6f71c6176ef7dab6751e4b71f6e6d13694d65134935bb275d89d1f0e9fdb2i0",
			},
			"sats": {
				AtomicalId: "8632ac61fc05d5a5e75a35e8a1b579d451c2695b285bd8d91edb390ff6f5c5dbi0",
			},
			"shib": {
				AtomicalId: "2574a2c35ab9bb2d5089f6482226390353d77bf307c485f1d3ce42fda06f1ab4i0",
			},
			"tor": {
				AtomicalId: "38001fbd83c763636f61693f28d7377ee743a9e2b38dc61c916d9bb1bd0a3568i0",
			},
			"wizz": {
				AtomicalId: "151663aed4c10516931258f097a5f00e747d99c3eb2e337813064d0eac47da75i0",
			},
			"x": {
				AtomicalId: "9265ee5b124b26ebd652b56a43b4ab9a322809f3ff909a86af2850aee4929839i0",
			},
		},
	},
	808989: {
		Height:      808989,
		TickerCount: 22,
		Tickers: map[string]*TickerStatus{
			"icals": {
				AtomicalId: "2679c605df1201f501b9827fa61e1405d19e37c8c9f8ac2dd8a67da2f87e76bfi0",
				UtxoCount:  4875,
				UtxoAmount: 4960000,
			},
		},
	},
	809041: {
		Height:      809041,
		TickerCount: 22,
		Tickers: map[string]*TickerStatus{
			"pepe": {
				AtomicalId:   "9ba68637ba32edb6370bebceaac3df4341180cbf7bac210741b12a679692d716i0",
				UtxoCount:    34029,
				UtxoAmount:   68388995,
				MintedTimes:  34500,
				MintedAmount: 69000000,
				MaxMints:     34500,
			},
		},
	},
	809182: {
		Height:      809182,
		TickerCount: 23,
		Tickers: map[string]*TickerStatus{
			"arc": {
				AtomicalId:   "0000dfbeb6bf0a0584969b04224607890ebf1dc167738c51fec61b89f01730eai0",
				DeployHeight: 809182,
			},
		},
	},
	809373: {
		Height:      809373,
		TickerCount: 24,
		Tickers: map[string]*TickerStatus{
			"runes": {
				AtomicalId:   "463373bc30c8cb3e63265f1ad1da1933e51e91f5e858b62911f51d21b63042c9i0",
				DeployHeight: 809373,
			},
		},
	},
	809407: {
		Height:      809407,
		TickerCount: 25,
		Tickers: map[string]*TickerStatus{
			"rune": {
				AtomicalId:   "9838fe12b7e9dd4e22c9baeb38b17ffa584edbca63f4744ce518269fc304e659i0",
				DeployHeight: 809407,
			},
		},
	},
	809439: {
		Height:      809439,
		TickerCount: 26,
		Tickers: map[string]*TickerStatus{
			"lvx": {
				AtomicalId:   "6794ee76fae9dc4d9f2f9bcdd5c7757b9d9f033c3f89dfea27470f2f0ff51e5di0",
				DeployHeight: 809439,
			},
		},
	},
	809552: {
		Height:      809552,
		TickerCount: 27,
		Tickers: map[string]*TickerStatus{
			"runeisdead": {
				AtomicalId:   "7081b7f6e18da3a41ab665c12c8dff5f07d165218298426efd02bc17d0aca571i0",
				DeployHeight: 809552,
			},
		},
	},
	809576: {
		Height:      809576,
		TickerCount: 28,
		Tickers: map[string]*TickerStatus{
			"coin": {
				AtomicalId:   "3d2a63f1b35716431c30dd2803afc67273b6fc0211d9baf38e323c25a800e711i0",
				DeployHeight: 809576,
			},
		},
	},
	809590: {
		Height:      809590,
		TickerCount: 29,
		Tickers: map[string]*TickerStatus{
			"pizza": {
				AtomicalId:   "50058e2e0ee267bcdd64a57f01c844c95bb19b3f42266b10a959eed6684eb4e8i0",
				DeployHeight: 809590,
			},
		},
	},
	810011: {
		Height:      810011,
		TickerCount: 30,
		Tickers: map[string]*TickerStatus{
			"vmpx": {
				AtomicalId:   "4845721f19c82a54ae3a4096248cb14319af182475c2ca61d63ae1773ea5ffd3i0",
				DeployHeight: 810011,
			},
		},
	},
	810013: {
		Height:      810013,
		TickerCount: 31,
		Tickers: map[string]*TickerStatus{
			"gold": {
				AtomicalId:   "2258c2531df921a591c9b8ee0e78d919fdc2b6a648e390ecd502c610beade96fi0",
				DeployHeight: 810013,
			},
		},
	},
	810306: {
		Height:      810306,
		TickerCount: 32,
		Tickers: map[string]*TickerStatus{
			"testtoken": {
				AtomicalId:   "161847f4d3671521ef55d6f19eb456f9bee64342f978e1c52361111868677fadi0",
				DeployHeight: 810306,
			},
		},
	},
	810814: {
		Height:      810814,
		TickerCount: 32,
		Tickers: map[string]*TickerStatus{
			"abtc": {
				AtomicalId: "8888722295d5b9d38efbc262e8ac9a21356257ee49d23037eba6279c16b58c8bi0",
				UtxoCount:  83,
				UtxoAmount: 83000,
			},
			"atom": {
				AtomicalId: "56a8702bab3d2405eb9a356fd0725ca112a93a8efd1ecca06c6085e7278f0341i0",
				UtxoCount:  12549,
				UtxoAmount: 20042938,
			},
			"atombtc": {
				AtomicalId: "1200d535ed19e97345c2db6b264b16c42105c5ea9b0d245d2e49117dbe81dd01i0",
				UtxoCount:  40,
				UtxoAmount: 40000,
			},
			"atomicals": {
				AtomicalId: "54799a35caf574c08d44216c354ef16344f9c5f6c8cf6e349468d90549c9ed5fi0",
				UtxoCount:  10,
				UtxoAmount: 10000,
			},
			"doge": {
				AtomicalId: "00009b954c9f1358de9c089f95ec420132e4106a89c8fbb3cfda198ae1e5f9d5i0",
				UtxoCount:  309,
				UtxoAmount: 309000,
			},
			"icals": {
				AtomicalId: "2679c605df1201f501b9827fa61e1405d19e37c8c9f8ac2dd8a67da2f87e76bfi0",
				UtxoCount:  4614,
				UtxoAmount: 4868351,
			},
			"pepe": {
				AtomicalId: "9ba68637ba32edb6370bebceaac3df4341180cbf7bac210741b12a679692d716i0",
				UtxoCount:  30389,
				UtxoAmount: 66757394,
			},
			"rune": {
				AtomicalId: "9838fe12b7e9dd4e22c9baeb38b17ffa584edbca63f4744ce518269fc304e659i0",
				UtxoCount:  261,
				UtxoAmount: 261000,
			},
		},
	},
	810916: {
		Height:      810916,
		TickerCount: 32,
		Tickers: map[string]*TickerStatus{
			"atom": {
				AtomicalId: "56a8702bab3d2405eb9a356fd0725ca112a93a8efd1ecca06c6085e7278f0341i0",
				UtxoCount:  12493,
				UtxoAmount: 20042938,
			},
			"pepe": {
				AtomicalId: "9ba68637ba32edb6370bebceaac3df4341180cbf7bac210741b12a679692d716i0",
				UtxoCount:  30373,
				UtxoAmount: 66668696,
			},
		},
	},
	811052: {
		Height:      811052,
		TickerCount: 33,
		Tickers: map[string]*TickerStatus{
			"gftk": {
				AtomicalId:   "3252d75e5b607f1129ebaf5d171c9727a147c67c8870542454c23648828815a6i0",
				DeployHeight: 811052,
			},
		},
	},
	811076: {
		Height:      811076,
		TickerCount: 34,
		Tickers: map[string]*TickerStatus{
			"honk": {
				AtomicalId:   "1576fd52da63e2f8f8fdd702d9d588c694786f31ca79b59fb90c97773af3613bi0",
				DeployHeight: 811076,
			},
		},
	},
	811084: {
		Height:      811084,
		TickerCount: 35,
		Tickers: map[string]*TickerStatus{
			"b": {
				AtomicalId:   "3509baa110c7ecaa8105dc5188cd9d0dcb68500560b080eef0e264c5405f780bi0",
				DeployHeight: 811084,
			},
		},
	},
	811095: {
		Height:      811095,
		TickerCount: 36,
		Tickers: map[string]*TickerStatus{
			"o": {
				AtomicalId:   "6926af7d645d52b433c13e9fe7487fb15f1190d8b40ba134d81922d700eb9cfdi0",
				DeployHeight: 811095,
			},
		},
	},
	811117: {
		Height:      811117,
		TickerCount: 37,
		Tickers: map[string]*TickerStatus{
			"build": {
				AtomicalId:   "1618dcd3b03bd06e77540f76b597a93134b226838827605c6a1dc1140e4c08a5i0",
				DeployHeight: 811117,
			},
		},
	},
	811233: {
		Height:      811233,
		TickerCount: 38,
		Tickers: map[string]*TickerStatus{
			"0": {
				AtomicalId:   "74354f5abd480379ff5346c2258bb87510c75859f9623734792000d9ff9cef81i0",
				DeployHeight: 811233,
			},
		},
	},
	811566: {
		Height:      811566,
		TickerCount: 39,
		Tickers: map[string]*TickerStatus{
			"sat": {
				AtomicalId:   "4170d1a3e8e925bd4b0021c26f38b08c6aba20cef492e1f48b741a7c107e0ae9i0",
				DeployHeight: 811566,
			},
		},
	},
	811570: {
		Height:      811570,
		TickerCount: 40,
		Tickers: map[string]*TickerStatus{
			"usd": {
				AtomicalId:   "3894fd450d98f9aec02c47e9616a5093719455dbd6de15f7c227a5abe0db3147i0",
				DeployHeight: 811570,
			},
		},
	},
	811604: {
		Height:      811604,
		TickerCount: 41,
		Tickers: map[string]*TickerStatus{
			"eve": {
				AtomicalId:   "aa3cf969a2c72c3a510d332c1258240c26746c9dce47d763b05855fa63fcadafi0",
				DeployHeight: 811604,
			},
		},
	},
	812105: {
		Height:      812105,
		TickerCount: 42,
		Tickers: map[string]*TickerStatus{
			"tx": {
				AtomicalId:   "7aa190582668aec03370119a3e18492744fb6a650cef04eca233a171ce1f7402i0",
				DeployHeight: 812105,
			},
		},
	},
	812260: {
		Height:      812260,
		TickerCount: 43,
		Tickers: map[string]*TickerStatus{
			"1": {
				AtomicalId:   "221236662c30949b81433f0bc69fabf55b0a5a0bea3e76b01dbd8e7afbc99c4bi0",
				DeployHeight: 812260,
			},
		},
	},
	812634: {
		Height:      812634,
		TickerCount: 44,
		Tickers: map[string]*TickerStatus{
			"546": {
				AtomicalId:   "8107947ce496504385d49d122327cc065a5be3e9ca828f2b2b30817346264b83i0",
				DeployHeight: 812634,
			},
		},
	},
	813924: {
		Height:      813924,
		TickerCount: 51,
		Tickers: map[string]*TickerStatus{
			"atomfi": {},
			"ibtc":   {},
			"pipe":   {},
			"realm":  {},
			"tap":    {},
		},
	},
	814924: {
		Height:      814924,
		TickerCount: 60,
		Tickers: map[string]*TickerStatus{
			"atoms":   {},
			"aton":    {},
			"bitvm":   {},
			"fish":    {},
			"rea1m":   {},
			"supreme": {},
		},
	},
	816924: {
		Height:      816924,
		TickerCount: 88,
	},
	817924: {
		Height:      817924,
		TickerCount: 94,
		Tickers: map[string]*TickerStatus{
			"2": {
				AtomicalId: "57799122cfded50a81835337b7b01d368be94677077a0fd8257ca6dc0a3a24cei0",
			},
			"3": {
				AtomicalId: "9135a933634304b26c5113d16af578f7579096305a60ae1dd09215803a32b54fi0",
			},
			"6": {
				AtomicalId: "3950e1c98f816ce8c3dbcc76b0a86a3f5580ef00b0c030813c1d790f65192aaai0",
			},
			"8": {
				AtomicalId: "2101f9ae1a90cc52f51db85e688459129a9e7f49f2623a4bc613d0085e0dafddi0",
			},
			"aa": {
				AtomicalId: "6909938815b923d4477923f76ada31754917b5f28fecd4e5c114c37edf881231i0",
			},
			"aaa": {
				AtomicalId: "34310bcf0c1f6b772d084b1ddfea88e4a26d982a88d791e34f5589038ac28be0i0",
			},
			"aaaa": {
				AtomicalId: "5571d14a696410838c97cc6eb408797f14af99c604869e04632d398157a888c5i0",
			},
			"afom": {
				AtomicalId: "2a498bdc322fbd0b4d3d78b17b2b0276e01442b84c7c52e1363e8b4af226a061i0",
			},
			"ai": {
				AtomicalId: "41203f4f1d97b5cda12acef1335afa019b23338229b78de088e9686bb212cd77i0",
			},
			"arcswap": {
				AtomicalId: "a091bc3620572d82a7b16bc3adaa10694f842af1282f0f2807845a7a9a6e3407i0",
			},
			"atmo": {
				AtomicalId: "39b7b4deb1b3e2dcbb886767907f3a0ef25e743da95cec6bd6f10e3490f00003i0",
			},
			"atom2": {
				AtomicalId: "136379859fdba9c8b4833ecbc29c3ae53e22590bf2b0460d5c806a3512302b70i0",
			},
			"atommap": {
				AtomicalId: "2993cf6622a408ef39f59c58ceeb4f4509d3f49a36adb8db01dfbf366451543ai0",
			},
			"beng": {
				AtomicalId: "801168229af434d496e5bc19c884ba930c9f64c398370fbeb841a5dedf8e1e9ai0",
			},
			"bitwork": {
				AtomicalId: "77777ed2e22644a34e164a00d5114cad4b60870e38588188ad65dd5c8fdb8a62i0",
			},
			"bobo": {
				AtomicalId: "6038643726dc35bd1606778a5d7de7bf7dbf70194afb2ce86c6be07e7b0e1564i0",
			},
			"dm1nt": {
				AtomicalId: "afe1d3b5235a8db8a76cdc2b30c43814948fcfca2ba923eca7e0ae5af17199b1i0",
			},
			"dmint": {
				AtomicalId: "9b90da6b5d209e8ea0ac6503966dbc48ac0c5a555fede28624ad86f0322f785di0",
			},
			"dnimt": {
				AtomicalId: "b36eb2cba9fbfa4c611dd9fd0864bfedbc36272b059cb08df36ac2aaa834b895i0",
			},
			"dune": {
				AtomicalId: "10246d36e367b36314271c2202ff51c9c5a7b8696446595bb1a03f06b92b1a98i0",
			},
			"electron": {
				AtomicalId: "536737aadfaffa17233bca342be2571e14916f6a29003ff4766d515283e68e90i0",
			},
			"feg": {
				AtomicalId: "45341f7094a0e31e9f784118c1a990e48fd3f9ca04621e829297e3dfe8c369dci0",
			},
			"ical": {
				AtomicalId: "0576bdc2241d4da8721eaa10a13f1e634e93fcb960d152d33081d53c774d2343i0",
			},
			"jesus": {
				AtomicalId: "7777a3648affc8ebd3a35d188a69672ad500eb208e0b86b71e51cdefca8475e0i0",
			},
			"luck": {
				AtomicalId: "10929e5496651989fd4fcb425d1cf0e3b06d96cf77ff4b7b26074719eea78491i0",
			},
			"meme": {
				AtomicalId: "66653cb07bbeff9e47c9b962c86c28245d29c4cdad0dcef6afb23325ce8094cdi0",
			},
			"mota": {
				AtomicalId: "1108e905068d3d2bfb932ff11eb33d830e12f290c9c1f39335899ccf47522daci0",
			},
			"nuke": {
				AtomicalId: "66579e16ea64b36252da05852910d9f3fbff82049953eac43dfbedb31340b607i0",
			},
			"pbpb": {
				AtomicalId: "151022eb5a815b8f4b615ae9246f64876fd8e4e2fb12063589543e8aae9a9d8fi0",
			},
			"pow": {
				AtomicalId: "441361acefcc7b925780309701ebfbd00c198e31d92eceaba4e541bdc091bdddi0",
			},
			"realms": {
				AtomicalId: "0507355b9637882eb98f0f8ec5c8819164e194faf1500b79d50738730adb975bi0",
			},
			"rexxie": {
				AtomicalId: "2222c0b117ad16100ee861be74447ec0bc77c99d6e63f0c2fef4db7c093254cai0",
			},
			"subrealm": {
				AtomicalId: "ab88afcf5c42cbbac141966714dfd14ad280feda3ba785ea9a3e52ca938231dci0",
			},
			"swap": {
				AtomicalId: "71a5c6bcbd66e80d540cee1d19e2c0de59be46a6cfdc969015b528e23b7bdcfci0",
			},
			"tao": {
				AtomicalId: "2d6dfbdf45b8f6edd7babc5d623fa64c4fdba5bb2995847879804377904b66f1i0",
			},
			"truth": {
				AtomicalId: "111116e91093e00d4df39bf58ebdc54b0e02e79561df9ed38824fe37e91ef2abi0",
			},
			"utxo": {
				AtomicalId: "220779505db257d24e92d8bf0de963965b4ab4fb9bb1ec9275ce79b094d1e08ci0",
			},
			"uxto": {
				AtomicalId: "1bb1fa5ed8609492dce572db305a8ab20aaaaf89b94fdd95a3a6ce0328a3b642i0",
			},
			"zoom": {
				AtomicalId: "1337122ab60cb5b8bd517a5d30ad39f6e15ba87dbcfc2e5012a5cce5a030b210i0",
			},
		},
	},
	820000: {
		Height:         820000,
		TickerCount:    101,
		AssetUtxoCount: 123281,
		Tickers: map[string]*TickerStatus{
			"atom": {
				AtomicalId:   "56a8702bab3d2405eb9a356fd0725ca112a93a8efd1ecca06c6085e7278f0341i0",
				UtxoCount:    7067,
				UtxoAmount:   19412302,
				MintedTimes:  21000,
				MintedAmount: 21000000,
				MaxMints:     21000,
				HolderCount:  1831,
				Holders: map[string]int64{
					"bc1pm7he53qa00jx2d04t3c3ehhen6lpkl3f0tltsxflj22687zlwe5sxtryrq": 375000,
					"bc1plzyj2qfp45v2t6vpccmgrn3a6gz7n3ta6qwdcwtex9c660g8m0eqx78uwe": 300000,
					"bc1p2dlszcwwhct0elzrrvhntzghg749u4z5wnwkj830we636pag8nsqas9h7f": 200000,
					"bc1pptsyhluekq60z5x5twq9nt5g9w3ymz6p4emrsdsz2p238vevfraqzt7k34": 200000,
					"bc1png9dtjgnnqacpx8w4hhrns3l6s23zpeev04wnztzzhlnd8k6qc9q7mu2tq": 185000,
					"bc1pgxkn7anuaxnc7rrl9h26t68w3jwwaw88hxr38lg2vvyjrxspty6qnays97": 180000,
					"bc1pae8s7wkeyqr2pcumzt3nyjka9tmagdattfe4mmpp570ms7jggwksapw9dg": 166520,
					"bc1pfl5alrk4sed3akfuq57ywa8fwzf6daquydmwp5pm2mrcvqvwn3mq2x0ncj": 150000,
					"bc1p9ppjyrcpa88kpxljh40tvchkngdnf23awfl6uejt64dka4mt9yss6gkzez": 146000,
					"bc1p24pvs7cahf08pncvmk602clmqrkf7q23eu5pg3dr5jvepvd7q5hs4ertpg": 143000,
				},
			},
		},
	},
	830000: {
		Height:         830000,
		TickerCount:    273,
		AssetUtxoCount: 816424,
		Tickers: map[string]*TickerStatus{
			"atom": {
				AtomicalId:   "56a8702bab3d2405eb9a356fd0725ca112a93a8efd1ecca06c6085e7278f0341i0",
				UtxoCount:    7458,
				UtxoAmount:   19345214,
				MintedTimes:  21000,
				MintedAmount: 21000000,
				MaxMints:     21000,
				HolderCount:  2482,
				Holders: map[string]int64{
					"bc1pm7he53qa00jx2d04t3c3ehhen6lpkl3f0tltsxflj22687zlwe5sxtryrq": 375000,
					"bc1pn84jwndjk5zenzjf0kmtytdk7h3rqznxzf7ku4nx4nmmwtygm59qsj2y4e": 366000,
					"bc1pry3mph3vy7u4spcjpmfpuemcmyxhahmftsaddfevtp4pheg3q2lsnytluu": 206000,
					"bc1p2dlszcwwhct0elzrrvhntzghg749u4z5wnwkj830we636pag8nsqas9h7f": 200000,
					"bc1pptsyhluekq60z5x5twq9nt5g9w3ymz6p4emrsdsz2p238vevfraqzt7k34": 200000,
					"bc1prqvme4naatxmde3zdkltjqsal0a5sttll5h2hpgf73p4jazv7tgswca9jj": 166520,
					"bc1p24pvs7cahf08pncvmk602clmqrkf7q23eu5pg3dr5jvepvd7q5hs4ertpg": 156000,
					"bc1p3eze9y3krkxk848t0ph4d0y4mml22ht3z7g5snr8npdecrfkmuzsm433rk": 153350,
					"bc1pgxkn7anuaxnc7rrl9h26t68w3jwwaw88hxr38lg2vvyjrxspty6qnays97": 152000,
					"bc1p9ppjyrcpa88kpxljh40tvchkngdnf23awfl6uejt64dka4mt9yss6gkzez": 146000,
				},
			},
			"quark": {
				AtomicalId:   "9125f03bcf9325f6071762b9aee00b461a0b43ed157c336e2e89e07f47ea6f66i0",
				UtxoCount:    216505,
				UtxoAmount:   9780398673,
				MintedTimes:  500000,
				MintedAmount: 10000000000,
				MaxMints:     500000,
				HolderCount:  13972,
				Holders: map[string]int64{
					"bc1psar3ak7efwncv048k9lxetg924edw84tryzddmm2srjmfjkuk7wszs0vct": 102878707,
					"bc1p7md5p8p9h5k0w38hzrnwessxh5uttevc4nz9t0snxqz0g564rgvqwhjchr": 68940000,
					"bc1p8xe6exmsh0p90agephzjrlen7thaquvacpwqdm3f28sjmy5s0uhqk7vw4z": 64700000,
					"bc1p2way0h8jf2wkdgkdhwkgk54tc7qsmkyzwe27lgdse2g7uem2t7uq77trml": 60800000,
					"bc1pw7t7h62dk76q2zu05psqnwg38qjvvf0rpej7lek0ynhthqd26ffqrzw4rq": 60360000,
					"bc1pqfvjxm6w0nx2c66nfp36a9gdcp6a4r4lchegup9769lrlsvq7f2szarhlr": 57200000,
					"bc1pggazu7wm4la3vt745qryfuj39hm202e5ljpar72nym4l9xvkhjhqd9pe3d": 55460000,
					"bc1pgfz630c2jp5ehwpcswzkrs85sddnj4xjastc7z92a7h6gurxfqhsknd732": 53680000,
					"bc1q8ksruedue2uwc7zrmg3c3ac55wflxclpaed4ys":                     48000000,
					"bc1prlgxsn7wqc8dfmrkjzxsyzp70hsts3tf937vmr76x37npkmekgxsqg6y5j": 47510258,
				},
			},
		},
	},
	840000: {
		Height:         840000,
		TickerCount:    393,
		AssetUtxoCount: 903100,
		Tickers: map[string]*TickerStatus{
			"atom": {
				AtomicalId:   "56a8702bab3d2405eb9a356fd0725ca112a93a8efd1ecca06c6085e7278f0341i0",
				UtxoCount:    8085,
				UtxoAmount:   19336214,
				MintedTimes:  21000,
				MintedAmount: 21000000,
				MaxMints:     21000,
				HolderCount:  3069,
				Holders: map[string]int64{
					"bc1pn84jwndjk5zenzjf0kmtytdk7h3rqznxzf7ku4nx4nmmwtygm59qsj2y4e": 450000,
					"bc1p3eze9y3krkxk848t0ph4d0y4mml22ht3z7g5snr8npdecrfkmuzsm433rk": 308896,
					"bc1pry3mph3vy7u4spcjpmfpuemcmyxhahmftsaddfevtp4pheg3q2lsnytluu": 206000,
					"bc1pptsyhluekq60z5x5twq9nt5g9w3ymz6p4emrsdsz2p238vevfraqzt7k34": 200000,
					"bc1prqvme4naatxmde3zdkltjqsal0a5sttll5h2hpgf73p4jazv7tgswca9jj": 185880,
					"bc1p3dfge99g2ulp8ry7almgr64fmwqzfxqclgh5mca39rzl2lw3zvasavfyzu": 176442,
					"bc1pgxkn7anuaxnc7rrl9h26t68w3jwwaw88hxr38lg2vvyjrxspty6qnays97": 152000,
					"bc1p544f4l28pydx8wja5zmx5ntzt97c70u6vrkaqughpwzfssc47a4qauvr4u": 129000,
					"bc1p9gl9ex8apc9h78d6rldmz92zxn2wugqfrjwt5k9htvsz7a3e07sqxhdm8s": 121300,
					"bc1pc503267c0x7amqy9ydguymasvh8hwj9e863ul5wlcf7zzh4vtfksdd9379": 119000,
				},
			},
			"quark": {
				AtomicalId:   "9125f03bcf9325f6071762b9aee00b461a0b43ed157c336e2e89e07f47ea6f66i0",
				UtxoCount:    182709,
				UtxoAmount:   9718714909,
				MintedTimes:  500000,
				MintedAmount: 10000000000,
				MaxMints:     500000,
				HolderCount:  15682,
				Holders: map[string]int64{
					"bc1p3dfge99g2ulp8ry7almgr64fmwqzfxqclgh5mca39rzl2lw3zvasavfyzu": 381426713,
					"bc1psq6xlaawpzpd0z68ef9r0kqt5lzkqy9zqwj2a2kent6p4z6etxtqy7eugc": 100000000,
					"bc1pm0jgthnm3rnmddyffn0qn769pele0r85f5aw2gc6we762e6kd9msz673pt": 80000000,
					"bc1p59jn8l62um3tf7rj3qskjty93c0c2ghwrrfd6dpuvjfn9tzp4p8qmgzwws": 75300000,
					"bc1p8xe6exmsh0p90agephzjrlen7thaquvacpwqdm3f28sjmy5s0uhqk7vw4z": 64700000,
					"bc1p2way0h8jf2wkdgkdhwkgk54tc7qsmkyzwe27lgdse2g7uem2t7uq77trml": 60800000,
					"bc1py6pu203c7sdyfv6zyclvaf4fpd0ghw4z003wwka83fwta0w2yckqsmn5zx": 60000000,
					"bc1pqfvjxm6w0nx2c66nfp36a9gdcp6a4r4lchegup9769lrlsvq7f2szarhlr": 54467108,
					"bc1pgfz630c2jp5ehwpcswzkrs85sddnj4xjastc7z92a7h6gurxfqhsknd732": 53680000,
					"bc1pyu56aw3c9uay46mvn5ecwpk048qnze44y4qlj6agvaeqm4k5dn6qwp00v8": 53066923,
				},
			},
		},
	},
	850000: {
		Height:         850000,
		TickerCount:    487,
		AssetUtxoCount: 857949,
		Tickers: map[string]*TickerStatus{
			"atom": {
				AtomicalId:   "56a8702bab3d2405eb9a356fd0725ca112a93a8efd1ecca06c6085e7278f0341i0",
				UtxoCount:    8229,
				UtxoAmount:   19330214,
				MintedTimes:  21000,
				MintedAmount: 21000000,
				MaxMints:     21000,
				HolderCount:  3233,
				Holders: map[string]int64{
					"bc1p3eze9y3krkxk848t0ph4d0y4mml22ht3z7g5snr8npdecrfkmuzsm433rk": 496348,
					"bc1pn84jwndjk5zenzjf0kmtytdk7h3rqznxzf7ku4nx4nmmwtygm59qsj2y4e": 341000,
					"bc1p3dfge99g2ulp8ry7almgr64fmwqzfxqclgh5mca39rzl2lw3zvasavfyzu": 225250,
					"bc1pry3mph3vy7u4spcjpmfpuemcmyxhahmftsaddfevtp4pheg3q2lsnytluu": 206000,
					"bc1pptsyhluekq60z5x5twq9nt5g9w3ymz6p4emrsdsz2p238vevfraqzt7k34": 200000,
					"bc1prqvme4naatxmde3zdkltjqsal0a5sttll5h2hpgf73p4jazv7tgswca9jj": 185880,
					"bc1pgxkn7anuaxnc7rrl9h26t68w3jwwaw88hxr38lg2vvyjrxspty6qnays97": 163851,
					"bc1pg2ktxj95uct8xj0j0p9vqm829ktdfwuaayhwtygnflg6qqgz8kms98plhc": 142990,
					"bc1pg6r0mdvzcf407vamm9670vtzcywlrcc6naj03r85qr5avvkuk6cql930r4": 140525,
					"bc1p544f4l28pydx8wja5zmx5ntzt97c70u6vrkaqughpwzfssc47a4qauvr4u": 129000,
				},
			},
			"quark": {
				AtomicalId:   "9125f03bcf9325f6071762b9aee00b461a0b43ed157c336e2e89e07f47ea6f66i0",
				UtxoCount:    166432,
				UtxoAmount:   9698976708,
				MintedTimes:  500000,
				MintedAmount: 10000000000,
				MaxMints:     500000,
				HolderCount:  18523,
				Holders: map[string]int64{
					"bc1p3dfge99g2ulp8ry7almgr64fmwqzfxqclgh5mca39rzl2lw3zvasavfyzu": 359780914,
					"bc1puxcfn62ceyxu4m5rrk89vh9eydd682avlqjphx57mxekglmuhj9snu77w4": 105175183,
					"bc1psq6xlaawpzpd0z68ef9r0kqt5lzkqy9zqwj2a2kent6p4z6etxtqy7eugc": 100000000,
					"bc1py6pu203c7sdyfv6zyclvaf4fpd0ghw4z003wwka83fwta0w2yckqsmn5zx": 97307600,
					"bc1pkskzpe3r9wt46y04ph23wqk4qn5g7uw4t2srshhseuh9fyvn6mfqp3sgtj": 94536502,
					"bc1pm0jgthnm3rnmddyffn0qn769pele0r85f5aw2gc6we762e6kd9msz673pt": 80000000,
					"bc1p59jn8l62um3tf7rj3qskjty93c0c2ghwrrfd6dpuvjfn9tzp4p8qmgzwws": 75300000,
					"bc1p8xe6exmsh0p90agephzjrlen7thaquvacpwqdm3f28sjmy5s0uhqk7vw4z": 64700000,
					"bc1p2way0h8jf2wkdgkdhwkgk54tc7qsmkyzwe27lgdse2g7uem2t7uq77trml": 60800000,
					"bc1p7ct25q76d7pgds4ds32kthj8zjrf8nsf78ysg7gxyapr5ldekpgqzftml4": 56195810,
				},
			},
		},
	},
	860000: {
		Height:         860000,
		TickerCount:    642,
		AssetUtxoCount: 832478,
		Tickers: map[string]*TickerStatus{
			"atom": {
				AtomicalId:   "56a8702bab3d2405eb9a356fd0725ca112a93a8efd1ecca06c6085e7278f0341i0",
				UtxoCount:    12899,
				UtxoAmount:   19330214,
				MintedTimes:  21000,
				MintedAmount: 21000000,
				MaxMints:     21000,
				HolderCount:  7680,
				Holders: map[string]int64{
					"bc1p3eze9y3krkxk848t0ph4d0y4mml22ht3z7g5snr8npdecrfkmuzsm433rk": 572773,
					"bc1p3dfge99g2ulp8ry7almgr64fmwqzfxqclgh5mca39rzl2lw3zvasavfyzu": 395362,
					"bc1pry3mph3vy7u4spcjpmfpuemcmyxhahmftsaddfevtp4pheg3q2lsnytluu": 206000,
					"bc1pptsyhluekq60z5x5twq9nt5g9w3ymz6p4emrsdsz2p238vevfraqzt7k34": 200000,
					"bc1pgxkn7anuaxnc7rrl9h26t68w3jwwaw88hxr38lg2vvyjrxspty6qnays97": 163851,
					"bc1pn84jwndjk5zenzjf0kmtytdk7h3rqznxzf7ku4nx4nmmwtygm59qsj2y4e": 161000,
					"bc1pg6r0mdvzcf407vamm9670vtzcywlrcc6naj03r85qr5avvkuk6cql930r4": 140525,
					"bc1pg2ktxj95uct8xj0j0p9vqm829ktdfwuaayhwtygnflg6qqgz8kms98plhc": 123083,
					"bc1p9gl9ex8apc9h78d6rldmz92zxn2wugqfrjwt5k9htvsz7a3e07sqxhdm8s": 121300,
					"bc1py8cgr0vznjz4dvrkp495y5vm74uqh2796lq4uys6ts9rvd0shtqqznpcyu": 117135,
				},
			},
			"quark": {
				AtomicalId:   "9125f03bcf9325f6071762b9aee00b461a0b43ed157c336e2e89e07f47ea6f66i0",
				UtxoCount:    154128,
				UtxoAmount:   9696961667,
				MintedTimes:  500000,
				MintedAmount: 10000000000,
				MaxMints:     500000,
				HolderCount:  21993,
				Holders: map[string]int64{
					"bc1p3dfge99g2ulp8ry7almgr64fmwqzfxqclgh5mca39rzl2lw3zvasavfyzu": 285969399,
					"bc1puxcfn62ceyxu4m5rrk89vh9eydd682avlqjphx57mxekglmuhj9snu77w4": 105175183,
					"bc1psq6xlaawpzpd0z68ef9r0kqt5lzkqy9zqwj2a2kent6p4z6etxtqy7eugc": 100000000,
					"bc1p337u20hy8a8m06664ntkx432m8ztdad4h30k9dw8rn84gw5dw24su7kty2": 97922701,
					"bc1py6pu203c7sdyfv6zyclvaf4fpd0ghw4z003wwka83fwta0w2yckqsmn5zx": 97307600,
					"bc1pkskzpe3r9wt46y04ph23wqk4qn5g7uw4t2srshhseuh9fyvn6mfqp3sgtj": 94536502,
					"bc1pm0jgthnm3rnmddyffn0qn769pele0r85f5aw2gc6we762e6kd9msz673pt": 80000000,
					"bc1p59jn8l62um3tf7rj3qskjty93c0c2ghwrrfd6dpuvjfn9tzp4p8qmgzwws": 75300000,
					"bc1p8xe6exmsh0p90agephzjrlen7thaquvacpwqdm3f28sjmy5s0uhqk7vw4z": 64700000,
					"bc1p2way0h8jf2wkdgkdhwkgk54tc7qsmkyzwe27lgdse2g7uem2t7uq77trml": 60800000,
				},
			},
		},
	},
	870000: {
		Height:         870000,
		TickerCount:    688,
		AssetUtxoCount: 810850,
		Tickers: map[string]*TickerStatus{
			"atom": {
				AtomicalId:   "56a8702bab3d2405eb9a356fd0725ca112a93a8efd1ecca06c6085e7278f0341i0",
				UtxoCount:    13980,
				UtxoAmount:   19328439,
				MintedTimes:  21000,
				MintedAmount: 21000000,
				MaxMints:     21000,
				HolderCount:  8782,
				Holders: map[string]int64{
					"bc1p3eze9y3krkxk848t0ph4d0y4mml22ht3z7g5snr8npdecrfkmuzsm433rk": 669435,
					"bc1pg2ktxj95uct8xj0j0p9vqm829ktdfwuaayhwtygnflg6qqgz8kms98plhc": 273083,
					"bc1pry3mph3vy7u4spcjpmfpuemcmyxhahmftsaddfevtp4pheg3q2lsnytluu": 206000,
					"bc1pptsyhluekq60z5x5twq9nt5g9w3ymz6p4emrsdsz2p238vevfraqzt7k34": 200000,
					"bc1pgxkn7anuaxnc7rrl9h26t68w3jwwaw88hxr38lg2vvyjrxspty6qnays97": 186893,
					"bc1pn84jwndjk5zenzjf0kmtytdk7h3rqznxzf7ku4nx4nmmwtygm59qsj2y4e": 161000,
					"bc1pvesnwzz63l3gru9l309yyppkrswl0u5y4s5mr95hxds3z5chmmls9ssr8q": 152466,
					"bc1pg6r0mdvzcf407vamm9670vtzcywlrcc6naj03r85qr5avvkuk6cql930r4": 141525,
					"bc1p9gl9ex8apc9h78d6rldmz92zxn2wugqfrjwt5k9htvsz7a3e07sqxhdm8s": 121300,
					"bc1py8cgr0vznjz4dvrkp495y5vm74uqh2796lq4uys6ts9rvd0shtqqznpcyu": 118135,
				},
			},
			"quark": {
				AtomicalId:   "9125f03bcf9325f6071762b9aee00b461a0b43ed157c336e2e89e07f47ea6f66i0",
				UtxoCount:    148249,
				UtxoAmount:   9688667436,
				MintedTimes:  500000,
				MintedAmount: 10000000000,
				MaxMints:     500000,
				HolderCount:  23326,
				Holders: map[string]int64{
					"bc1puxcfn62ceyxu4m5rrk89vh9eydd682avlqjphx57mxekglmuhj9snu77w4": 105175183,
					"bc1psq6xlaawpzpd0z68ef9r0kqt5lzkqy9zqwj2a2kent6p4z6etxtqy7eugc": 100000000,
					"bc1p337u20hy8a8m06664ntkx432m8ztdad4h30k9dw8rn84gw5dw24su7kty2": 98042701,
					"bc1py6pu203c7sdyfv6zyclvaf4fpd0ghw4z003wwka83fwta0w2yckqsmn5zx": 97307600,
					"bc1pkskzpe3r9wt46y04ph23wqk4qn5g7uw4t2srshhseuh9fyvn6mfqp3sgtj": 94536502,
					"bc1p0nkzg0qss837pfcyjc4qfr60vjlyrza8f4ss8krgladglmhlph3qrqwnqa": 88847966,
					"bc1pm0jgthnm3rnmddyffn0qn769pele0r85f5aw2gc6we762e6kd9msz673pt": 80000000,
					"bc1p59jn8l62um3tf7rj3qskjty93c0c2ghwrrfd6dpuvjfn9tzp4p8qmgzwws": 75300000,
					"bc1p8xe6exmsh0p90agephzjrlen7thaquvacpwqdm3f28sjmy5s0uhqk7vw4z": 64700000,
					"bc1p2way0h8jf2wkdgkdhwkgk54tc7qsmkyzwe27lgdse2g7uem2t7uq77trml": 60800000,
				},
			},
		},
	},
	880000: {
		Height:         880000,
		TickerCount:    695,
		AssetUtxoCount: 786252,
		Tickers: map[string]*TickerStatus{
			"atom": {
				AtomicalId:   "56a8702bab3d2405eb9a356fd0725ca112a93a8efd1ecca06c6085e7278f0341i0",
				UtxoCount:    14045,
				UtxoAmount:   19328439,
				MintedTimes:  21000,
				MintedAmount: 21000000,
				MaxMints:     21000,
				HolderCount:  8763,
				Holders: map[string]int64{
					"bc1p3eze9y3krkxk848t0ph4d0y4mml22ht3z7g5snr8npdecrfkmuzsm433rk": 838027,
					"bc1pvesnwzz63l3gru9l309yyppkrswl0u5y4s5mr95hxds3z5chmmls9ssr8q": 378302,
					"bc1p9uwfwtu6xpwynpd82tu04evjcygcgllyg0t3r9hqxm5l9cpyudvsjcc8sq": 283179,
					"bc1pg2ktxj95uct8xj0j0p9vqm829ktdfwuaayhwtygnflg6qqgz8kms98plhc": 273083,
					"bc1p7qtgvew6zpqklpsujjt8lhm9hwdk8y362dr9kj9mxfvnz2afa0yqp83ez2": 227997,
					"bc1pry3mph3vy7u4spcjpmfpuemcmyxhahmftsaddfevtp4pheg3q2lsnytluu": 206000,
					"bc1pptsyhluekq60z5x5twq9nt5g9w3ymz6p4emrsdsz2p238vevfraqzt7k34": 200000,
					"bc1pgxkn7anuaxnc7rrl9h26t68w3jwwaw88hxr38lg2vvyjrxspty6qnays97": 186893,
					"bc1pn84jwndjk5zenzjf0kmtytdk7h3rqznxzf7ku4nx4nmmwtygm59qsj2y4e": 161000,
					"bc1pg6r0mdvzcf407vamm9670vtzcywlrcc6naj03r85qr5avvkuk6cql930r4": 141525,
				},
			},
			"quark": {
				AtomicalId:   "9125f03bcf9325f6071762b9aee00b461a0b43ed157c336e2e89e07f47ea6f66i0",
				UtxoCount:    138695,
				UtxoAmount:   9601151819,
				MintedTimes:  500000,
				MintedAmount: 10000000000,
				MaxMints:     500000,
				HolderCount:  25930,
				Holders: map[string]int64{
					"bc1qwjv97kps9vszpq85jknepsvfs8qlvp3ytcnkml":                     109246306,
					"bc1p337u20hy8a8m06664ntkx432m8ztdad4h30k9dw8rn84gw5dw24su7kty2": 106457714,
					"bc1puxcfn62ceyxu4m5rrk89vh9eydd682avlqjphx57mxekglmuhj9snu77w4": 105175183,
					"bc1psq6xlaawpzpd0z68ef9r0kqt5lzkqy9zqwj2a2kent6p4z6etxtqy7eugc": 101319569,
					"bc1py6pu203c7sdyfv6zyclvaf4fpd0ghw4z003wwka83fwta0w2yckqsmn5zx": 97307600,
					"bc1pkskzpe3r9wt46y04ph23wqk4qn5g7uw4t2srshhseuh9fyvn6mfqp3sgtj": 94536502,
					"bc1p0nkzg0qss837pfcyjc4qfr60vjlyrza8f4ss8krgladglmhlph3qrqwnqa": 90839371,
					"bc1pm0jgthnm3rnmddyffn0qn769pele0r85f5aw2gc6we762e6kd9msz673pt": 80000000,
					"bc1p59jn8l62um3tf7rj3qskjty93c0c2ghwrrfd6dpuvjfn9tzp4p8qmgzwws": 75300000,
					"bc1p8xe6exmsh0p90agephzjrlen7thaquvacpwqdm3f28sjmy5s0uhqk7vw4z": 64700000,
				},
			},
		},
	},
	890000: {
		Height:         890000,
		TickerCount:    702,
		AssetUtxoCount: 778713,
		Tickers: map[string]*TickerStatus{
			"atom": {
				AtomicalId:   "56a8702bab3d2405eb9a356fd0725ca112a93a8efd1ecca06c6085e7278f0341i0",
				UtxoCount:    13912,
				UtxoAmount:   19327884,
				MintedTimes:  21000,
				MintedAmount: 21000000,
				MaxMints:     21000,
				HolderCount:  8599,
				Holders: map[string]int64{
					"bc1p3eze9y3krkxk848t0ph4d0y4mml22ht3z7g5snr8npdecrfkmuzsm433rk": 926214,
					"bc1pvesnwzz63l3gru9l309yyppkrswl0u5y4s5mr95hxds3z5chmmls9ssr8q": 327603,
					"bc1p9uwfwtu6xpwynpd82tu04evjcygcgllyg0t3r9hqxm5l9cpyudvsjcc8sq": 283179,
					"bc1p7qtgvew6zpqklpsujjt8lhm9hwdk8y362dr9kj9mxfvnz2afa0yqp83ez2": 274997,
					"bc1pfr28ahxfl4y4qq3xay9e4pgwck5kddqplfgu2ked0uzxc90yydxsqfgd33": 245068,
					"bc1pg2ktxj95uct8xj0j0p9vqm829ktdfwuaayhwtygnflg6qqgz8kms98plhc": 210093,
					"bc1pry3mph3vy7u4spcjpmfpuemcmyxhahmftsaddfevtp4pheg3q2lsnytluu": 206000,
					"bc1qxvtams3gj5ye8d2ju07dy7cnuzf8zjaz85y2n6":                     203379,
					"bc1pptsyhluekq60z5x5twq9nt5g9w3ymz6p4emrsdsz2p238vevfraqzt7k34": 200000,
					"bc1pgxkn7anuaxnc7rrl9h26t68w3jwwaw88hxr38lg2vvyjrxspty6qnays97": 186893,
				},
			},
			"quark": {
				AtomicalId:   "9125f03bcf9325f6071762b9aee00b461a0b43ed157c336e2e89e07f47ea6f66i0",
				UtxoCount:    138460,
				UtxoAmount:   9502002790,
				MintedTimes:  500000,
				MintedAmount: 10000000000,
				MaxMints:     500000,
				HolderCount:  31280,
				Holders: map[string]int64{
					"bc1qw8wrek2m7nlqldll66ajnwr9mh64syvkt67zlu":                     143264352,
					"bc1quhruqrghgcca950rvhtrg7cpd7u8k6svpzgzmrjy8xyukacl5lkq0r8l2d": 128080257,
					"bc1qwjv97kps9vszpq85jknepsvfs8qlvp3ytcnkml":                     109246306,
					"bc1p59jn8l62um3tf7rj3qskjty93c0c2ghwrrfd6dpuvjfn9tzp4p8qmgzwws": 75300000,
					"bc1p8xe6exmsh0p90agephzjrlen7thaquvacpwqdm3f28sjmy5s0uhqk7vw4z": 64700000,
					"bc1p2way0h8jf2wkdgkdhwkgk54tc7qsmkyzwe27lgdse2g7uem2t7uq77trml": 60800000,
					"bc1pgfz630c2jp5ehwpcswzkrs85sddnj4xjastc7z92a7h6gurxfqhsknd732": 52956774,
					"bc1psnrfefhlr2cyex8azk9qjw9fcwywzca2z78yg765nrsfw2082yqq2lhyxt": 52550012,
					"bc1pqfvjxm6w0nx2c66nfp36a9gdcp6a4r4lchegup9769lrlsvq7f2szarhlr": 52507108,
					"bc1qlx87a27s84ngdyveaqmcanxre8n4pk6lw998kfncqkndeqdmuuzs74zh0v": 51980345,
				},
			},
		},
	},
	900000: {
		Height:         900000,
		TickerCount:    702,
		AssetUtxoCount: 771371,
		Tickers: map[string]*TickerStatus{
			"sophon": {
				AtomicalId: "360533d31e6f3c535acf7a70686ab42cf477b3f7ceaf12ab1d30be218b1726a9i0",
				UtxoCount:  194054,
				UtxoAmount: 23140060735,
				MaxMints:   420000,
			},
			"quark": {
				AtomicalId:   "9125f03bcf9325f6071762b9aee00b461a0b43ed157c336e2e89e07f47ea6f66i0",
				UtxoCount:    136522,
				UtxoAmount:   9372835953,
				MintedTimes:  500000,
				MintedAmount: 10000000000,
				MaxMints:     500000,
				HolderCount:  36748,
				Holders: map[string]int64{
					"bc1qwjv97kps9vszpq85jknepsvfs8qlvp3ytcnkml":                     109246306,
					"bc1pgfz630c2jp5ehwpcswzkrs85sddnj4xjastc7z92a7h6gurxfqhsknd732": 102976762,
					"bc1q6z23472n4nu4khksm3j8ax5rwdjc30y5ygrle2":                     87686557,
					"bc1p59jn8l62um3tf7rj3qskjty93c0c2ghwrrfd6dpuvjfn9tzp4p8qmgzwws": 75300000,
					"bc1p8xe6exmsh0p90agephzjrlen7thaquvacpwqdm3f28sjmy5s0uhqk7vw4z": 64700000,
					"bc1p2way0h8jf2wkdgkdhwkgk54tc7qsmkyzwe27lgdse2g7uem2t7uq77trml": 60800000,
					"bc1puu8ej28nw90kj8fa34k4we88vazck3l8rys2kum5a07nleudah3sxmq67d": 54483052,
					"bc1psnrfefhlr2cyex8azk9qjw9fcwywzca2z78yg765nrsfw2082yqq2lhyxt": 52644100,
					"bc1pqfvjxm6w0nx2c66nfp36a9gdcp6a4r4lchegup9769lrlsvq7f2szarhlr": 52507108,
					"bc1qkl6tgpehvj00hmurkxd20hxusdhqn97p48z59a":                     51979790,
				},
			},
			"infinity": {
				AtomicalId: "0d5e64d42e4520e17bc204fe25662b0cf2d2a65c350766d6171facaadccb371bi0",
				UtxoCount:  87820,
				UtxoAmount: 2939142398,
				MaxMints:   3333,
			},
			"pepe": {
				AtomicalId: "9ba68637ba32edb6370bebceaac3df4341180cbf7bac210741b12a679692d716i0",
				UtxoCount:  20615,
				UtxoAmount: 59106269,
				MaxMints:   34500,
			},
			"a": {
				AtomicalId: "66466d0207d9f50b5782678282e08cffb8e315e95da8d91b3d46f290b440a8bdi0",
				UtxoCount:  15498,
				UtxoAmount: 20536387,
				MaxMints:   21000,
			},
			"atom": {
				AtomicalId:   "56a8702bab3d2405eb9a356fd0725ca112a93a8efd1ecca06c6085e7278f0341i0",
				UtxoCount:    13156,
				UtxoAmount:   19325316,
				MintedTimes:  21000,
				MintedAmount: 21000000,
				MaxMints:     21000,
				HolderCount:  8213,
				Holders: map[string]int64{
					"bc1pu62x0qzqn758srcmm0ctlxgum55a06am3njj3jgatkmyu9plmypsshzp45": 1151992,
					"bc1p3eze9y3krkxk848t0ph4d0y4mml22ht3z7g5snr8npdecrfkmuzsm433rk": 896192,
					"bc1p9uwfwtu6xpwynpd82tu04evjcygcgllyg0t3r9hqxm5l9cpyudvsjcc8sq": 252978,
					"bc1pvesnwzz63l3gru9l309yyppkrswl0u5y4s5mr95hxds3z5chmmls9ssr8q": 241048,
					"bc1pry3mph3vy7u4spcjpmfpuemcmyxhahmftsaddfevtp4pheg3q2lsnytluu": 206000,
					"bc1pptsyhluekq60z5x5twq9nt5g9w3ymz6p4emrsdsz2p238vevfraqzt7k34": 200000,
					"bc1pgxkn7anuaxnc7rrl9h26t68w3jwwaw88hxr38lg2vvyjrxspty6qnays97": 186893,
					"bc1pg6r0mdvzcf407vamm9670vtzcywlrcc6naj03r85qr5avvkuk6cql930r4": 141525,
					"bc1ptuwmztktn3xd9mvxu9xwk7ncdnvmc9nv64heglcp60ymk7aus64qs07z8s": 133985,
					"bc1py8cgr0vznjz4dvrkp495y5vm74uqh2796lq4uys6ts9rvd0shtqqznpcyu": 118135,
				},
			},
			"dragon": {
				AtomicalId: "dc0038f5313f5fbbcfc51aaab7370e43507bdc661760f55ba634aefb5ad15c57i0",
				UtxoCount:  15490,
				UtxoAmount: 1116007082,
				MaxMints:   21000,
			},
			"atoms": {
				AtomicalId: "6188a9840691e90b49d6e9a1c927c6a83ac282817a8b639ea0db17817307dea4i0",
				UtxoCount:  15330,
				UtxoAmount: 20762370,
				MaxMints:   21000,
			},
			"icals": {
				AtomicalId: "2679c605df1201f501b9827fa61e1405d19e37c8c9f8ac2dd8a67da2f87e76bfi0",
				UtxoCount:  15218,
				UtxoAmount: 18170707,
				MaxMints:   21000,
			},
			"nucleus": {
				AtomicalId: "9198d994c43d6214c062b9c12317c9740fdb6a73a3d7a7ebd68b962abe802d8bi0",
				UtxoCount:  14826,
				UtxoAmount: 20408016,
				MaxMints:   21000,
			},
			"neutron": {
				AtomicalId: "1d00ffffa6d003a0aaa9af6d03a793adbb7124c8c9ad8d6df5910e9ee2f912abi0",
				UtxoCount:  14670,
				UtxoAmount: 1642582776,
				MaxMints:   21000,
			},
			"btc": {
				AtomicalId: "7296411f89e8e6171966a0b9d11e3fe12e86fdc3b0515b1f7bbb1c29d65f29adi0",
				UtxoCount:  14637,
				UtxoAmount: 20297477,
				MaxMints:   21000,
			},
			"fanshood": {
				AtomicalId: "923d5f127fae7abcbe0d171e3d01d9cb7a4b5c2f2d7e1bf5ebdc8f361744309di0",
				UtxoCount:  14439,
				UtxoAmount: 19516133,
				MaxMints:   21000,
			},
			"coloredbitcoin": {
				AtomicalId: "00002cf05244e8c97f4bdee853ab3fc931a7ca61b79fd02e56507f90327245b7i0",
				UtxoCount:  12375,
				UtxoAmount: 20685701,
				MaxMints:   21000,
			},
			"atomical": {
				AtomicalId: "0000d816b114585b45bf29e2ed0c2fa3c846f01f6ae44ee985d78f2f4acfb18di0",
				UtxoCount:  13125,
				UtxoAmount: 20386959,
				MaxMints:   21000,
			},
			"electron": {
				AtomicalId: "536737aadfaffa17233bca342be2571e14916f6a29003ff4766d515283e68e90i0",
				UtxoCount:  6802,
				UtxoAmount: 913877769,
				MaxMints:   18400,
			},
			"quantum": {
				AtomicalId: "37086fce3b535f1c9033c61fab4f45bf6aed67cf737f2f510803c66ccb00c9a2i0",
				UtxoCount:  1744,
				UtxoAmount: 177262574,
				MaxMints:   2100,
			},
			"fishmask": {
				AtomicalId: "00001bc4b6f1e452fb601a05cb0711a2cf38fb1d0f3ed2d36ac5fd02a2b77710i0",
				UtxoCount:  9577,
				UtxoAmount: 172026172,
				MaxMints:   1000000,
			},
			"games": {
				AtomicalId: "000077a5fb242ae6337c22d99d0519f321063b0025181b6561f5078d9bad6e53i0",
				UtxoCount:  724,
				UtxoAmount: 99916721,
				MaxMints:   1,
			},
			"dots": {
				AtomicalId: "0000040ef0b5cd5d5d63ae82d3a143a8a46a504a5e304eed169ecc260cadcfc8i0",
				UtxoCount:  2746,
				UtxoAmount: 36851838,
				MaxMints:   1,
			},
			"uxon": {
				AtomicalId: "00005542ddb59645a70129e934882ffcb6275234632d60c98c14ec304d63ac5di0",
				UtxoCount:  1559,
				UtxoAmount: 24142790,
				MaxMints:   500,
			},
		},
	},
}

var testnet4Checkpoint = map[int]*CheckPoint{}

var atomHolderStartHeight, atomHolderEndHeight int
var atomHeightToHolderRecords map[int]map[string]map[string]*atomValidate.HolderCSVRecord

func (s *Indexer) CheckPointWithBlockHeight(height int) {
	startTime := time.Now()

	s.mutex.RLock()
	defer s.mutex.RUnlock()
	s.checkPointWithBlockHeightLocked(height, startTime)
}

func (s *Indexer) checkPointWithBlockHeightLocked(height int, startTime time.Time) {
	checkpoint := s.checkPoint(height)
	if checkpoint == nil {
		return
	}

	if checkpoint.TickerCount != 0 && s.status.TickerCount != checkpoint.TickerCount {
		common.Log.Panicf("atom ticker count different at %d: %d %d", height, s.status.TickerCount, checkpoint.TickerCount)
	}
	if checkpoint.AssetUtxoCount != 0 {
		count := s.assetUtxoCountLocked()
		if count != checkpoint.AssetUtxoCount {
			common.Log.Infof("atom asset utxo summary at %d: %s", height, s.assetUtxoSummaryLocked(20))
			common.Log.Panicf("atom asset utxo count different at %d: %d %d", height, count, checkpoint.AssetUtxoCount)
		}
	}
	for _, name := range checkpoint.RejectedTicker {
		name = strings.ToLower(name)
		if s.getTickerLocked(name) != nil {
			common.Log.Panicf("atom rejected ticker %s exists at %d", name, height)
		}
	}
	for name, tickerStatus := range checkpoint.Tickers {
		name = strings.ToLower(name)
		if tickerStatus.DeployHeight != 0 && height < tickerStatus.DeployHeight {
			continue
		}
		ticker := s.getTickerLocked(name)
		if ticker == nil {
			common.Log.Panicf("atom checkpoint can't find ticker %s at %d", name, height)
		}
		if tickerStatus.AtomicalId != "" && ticker.AtomicalId != tickerStatus.AtomicalId {
			common.Log.Panicf("atom %s atomical id different at %d: %s %s", name, height, ticker.AtomicalId, tickerStatus.AtomicalId)
		}
		if tickerStatus.MintedTimes != 0 && ticker.MintedTimes != tickerStatus.MintedTimes {
			common.Log.Panicf("atom %s minted times different at %d: %d %d", name, height, ticker.MintedTimes, tickerStatus.MintedTimes)
		}
		if tickerStatus.MintedAmount != 0 && ticker.MintedAmount != tickerStatus.MintedAmount {
			common.Log.Panicf("atom %s minted amount different at %d: %d %d", name, height, ticker.MintedAmount, tickerStatus.MintedAmount)
		}
		if tickerStatus.MaxMints != 0 && ticker.MaxMints != tickerStatus.MaxMints {
			common.Log.Panicf("atom %s max mints different at %d: %d %d", name, height, ticker.MaxMints, tickerStatus.MaxMints)
		}
		if tickerStatus.HolderCount != 0 && ticker.HolderCount != tickerStatus.HolderCount {
			common.Log.Panicf("atom %s holder count different at %d: %d %d", name, height, ticker.HolderCount, tickerStatus.HolderCount)
		}
		if tickerStatus.UtxoCount != 0 || tickerStatus.UtxoAmount != 0 {
			count, amount := s.tickerUtxoSummaryLocked(name)
			if tickerStatus.UtxoCount != 0 && count != tickerStatus.UtxoCount {
				common.Log.Panicf("atom %s utxo count different at %d: %d %d", name, height, count, tickerStatus.UtxoCount)
			}
			if tickerStatus.UtxoAmount != 0 && amount != tickerStatus.UtxoAmount {
				common.Log.Panicf("atom %s utxo amount different at %d: %d %d", name, height, amount, tickerStatus.UtxoAmount)
			}
		}
		for address, amount := range tickerStatus.Holders {
			addressId := s.baseIndexer.GetAddressIdFromDB(address)
			if addressId == common.INVALID_ID {
				common.Log.Panicf("atom %s can't find holder address %s at %d", name, address, height)
			}
			holderAmount := s.tickerHolders[name][addressId]
			if holderAmount != amount {
				common.Log.Panicf("atom %s holder %s amount different at %d: %d %d", name, address, height, holderAmount, amount)
			}
		}
	}
	s.validateHolderDataLocked(height)
	common.Log.Infof("AtomIndexer.CheckPointWithBlockHeight %d checked, takes %v", height, time.Since(startTime))
}

func readAtomHolderDataToMap(dir string) (int, int) {
	records, err := atomValidate.ReadHolderCSVDir(dir)
	if err != nil {
		common.Log.Panicf("ReadAtomHolderCSVDir %s failed, %v", dir, err)
	}

	startHeight := int(^uint(0) >> 1)
	endHeight := 0
	atomHeightToHolderRecords = make(map[int]map[string]map[string]*atomValidate.HolderCSVRecord)
	for _, record := range records {
		tickerToHolders := atomHeightToHolderRecords[record.Height]
		if tickerToHolders == nil {
			tickerToHolders = make(map[string]map[string]*atomValidate.HolderCSVRecord)
			atomHeightToHolderRecords[record.Height] = tickerToHolders
		}
		holders := tickerToHolders[record.Ticker]
		if holders == nil {
			holders = make(map[string]*atomValidate.HolderCSVRecord)
			tickerToHolders[record.Ticker] = holders
		}
		holders[record.Address] = record

		if record.Height < startHeight {
			startHeight = record.Height
		}
		if record.Height > endHeight {
			endHeight = record.Height
		}
	}
	if len(records) == 0 {
		startHeight = 0
	}
	common.Log.Infof("readAtomHolderDataToMap height %d %d, records %d", startHeight, endHeight, len(records))
	return startHeight, endHeight
}

func (s *Indexer) validateHolderDataLocked(height int) {
	if s.chaincfgParam == nil || s.chaincfgParam.Net != wire.MainNet {
		return
	}
	if atomHeightToHolderRecords == nil {
		atomHolderStartHeight, atomHolderEndHeight = readAtomHolderDataToMap("./indexer/atom/validate/holders")
	}
	if len(atomHeightToHolderRecords) == 0 || height < atomHolderStartHeight || height > atomHolderEndHeight {
		return
	}

	tickerToHolders := atomHeightToHolderRecords[height]
	if len(tickerToHolders) == 0 {
		return
	}

	var failed []string
	for ticker, holders := range tickerToHolders {
		verified := true
		for address, record := range holders {
			addressId := s.baseIndexer.GetAddressIdFromDB(address)
			if addressId == common.INVALID_ID {
				common.Log.Errorf("AtomIndexer.validateHolderData GetAddressIdFromDB %s failed", address)
				failed = append(failed, ticker)
				verified = false
				continue
			}
			amount := s.tickerHolders[ticker][addressId]
			if amount != record.Amount {
				common.Log.Errorf("AtomIndexer.validateHolderData %s %s amount different %d %d",
					address, ticker, record.Amount, amount)
				failed = append(failed, ticker)
				verified = false
			}
		}
		if verified {
			common.Log.Infof("AtomIndexer.validateHolderData %s %d check succeeded.", ticker, len(holders))
		} else {
			common.Log.Infof("AtomIndexer.validateHolderData %s check failed.", ticker)
		}
	}

	if len(failed) > 0 {
		common.Log.Panicf("check atom %v holders failed", failed)
	}
}

func (s *Indexer) checkPoint(height int) *CheckPoint {
	if s.chaincfgParam != nil && s.chaincfgParam.Net == wire.MainNet {
		return mainnetCheckpoint[height]
	}
	return testnet4Checkpoint[height]
}

func (s *Indexer) tickerUtxoSummaryLocked(ticker string) (int, int64) {
	ticker = strings.ToLower(ticker)
	var count int
	var amount int64
	for _, balances := range s.utxoBalances {
		for _, balance := range balances {
			if strings.ToLower(balance.Ticker) != ticker || balance.Amount <= 0 {
				continue
			}
			count++
			amount += balance.Amount
		}
	}
	return count, amount
}

func (s *Indexer) assetUtxoCountLocked() int {
	var count int
	for _, balances := range s.utxoBalances {
		for _, balance := range balances {
			if balance.Amount > 0 {
				count++
			}
		}
	}
	return count
}

func (s *Indexer) assetUtxoSummaryLocked(limit int) string {
	counts := make(map[string]int)
	for _, balances := range s.utxoBalances {
		for _, balance := range balances {
			if balance.Amount <= 0 {
				continue
			}
			counts[strings.ToLower(balance.Ticker)]++
		}
	}
	type row struct {
		ticker string
		count  int
	}
	rows := make([]row, 0, len(counts))
	for ticker, count := range counts {
		rows = append(rows, row{ticker: ticker, count: count})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].count == rows[j].count {
			return rows[i].ticker < rows[j].ticker
		}
		return rows[i].count > rows[j].count
	})
	if limit > len(rows) || limit <= 0 {
		limit = len(rows)
	}
	parts := make([]string, 0, limit)
	for _, row := range rows[:limit] {
		parts = append(parts, fmt.Sprintf("%s=%d", row.ticker, row.count))
	}
	return strings.Join(parts, ",")
}
