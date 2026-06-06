package atom

import (
	"strings"
	"time"

	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
)

type CheckPoint struct {
	Height         int
	TickerCount    int64
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
			"gold": {
				AtomicalId:   "2258c2531df921a591c9b8ee0e78d919fdc2b6a648e390ecd502c610beade96fi0",
				DeployHeight: 810011,
			},
		},
	},
	810013: {
		Height:      810013,
		TickerCount: 31,
		Tickers: map[string]*TickerStatus{
			"vmpx": {
				AtomicalId:   "4845721f19c82a54ae3a4096248cb14319af182475c2ca61d63ae1773ea5ffd3i0",
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
}

var testnet4Checkpoint = map[int]*CheckPoint{}

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
		if tickerStatus.UtxoCount != 0 || tickerStatus.UtxoAmount != 0 {
			count, amount := s.tickerUtxoSummaryLocked(name)
			if tickerStatus.UtxoCount != 0 && count != tickerStatus.UtxoCount {
				common.Log.Panicf("atom %s utxo count different at %d: %d %d", name, height, count, tickerStatus.UtxoCount)
			}
			if tickerStatus.UtxoAmount != 0 && amount != tickerStatus.UtxoAmount {
				common.Log.Panicf("atom %s utxo amount different at %d: %d %d", name, height, amount, tickerStatus.UtxoAmount)
			}
		}
	}
	common.Log.Infof("AtomIndexer.CheckPointWithBlockHeight %d checked, takes %v", height, time.Since(startTime))
}

func (s *Indexer) checkPoint(height int) *CheckPoint {
	if s.chaincfgParam != nil && s.chaincfgParam.Net == wire.MainNet {
		return mainnetCheckpoint[height]
	}
	return testnet4Checkpoint[height]
}

func (s *Indexer) tickerUtxoSummaryLocked(ticker string) (int, int64) {
	var amount int64
	utxos := s.tickerUtxos[strings.ToLower(ticker)]
	for _, value := range utxos {
		if value <= 0 {
			continue
		}
		amount += value
	}
	return len(utxos), amount
}
