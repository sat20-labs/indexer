package brc20

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"

	"github.com/sat20-labs/indexer/indexer/brc20/validate"
)


type CheckPoint struct {
	Height int
	TickerCount int
	Tickers map[string]*TickerStatus
}

type TickerStatus struct {
	Name string
	Max  string
	Minted string
	MintCount int
	HolderCount int
	TxCount int
	Holders map[string]string
}


var _enable_validate bool = true
var _validateData map[string]*validate.BRC20CSVRecord
var _heightToRecords map[int][]*validate.BRC20CSVRecord // 按顺序

var testnet4_checkpoint = map[int]*CheckPoint{
	27227: {
		Height: 27227,
		TickerCount: 0,
		Tickers: nil,
	},

	30000: {
		Height: 30000,
		TickerCount: 12,
		Tickers: map[string]*TickerStatus{
			"ordi": {
				Name: "ordi",
				Max: "2400000000",
				Minted: "110127",
				MintCount: 23,
				HolderCount: 4,
				TxCount: 66,
				Holders: map[string]string{
					"tb1pmm586mlhs35e8ns08trdejpzv02rupx0hp9j8arumg5c29dyrfnq2trqcw": "99000",
					"tb1p5pmdgkjk2dcpgmme2wx5q0uvwnzk6zhhfkpn5ldtuy3syn07hh4qqm2lsv": "10056",
					"tb1plts00urlmu2kf7gcnp5225dnh4f0tn7e0r2jlvnd0exrwd03xe4ssykzpj": "1000",
					"tb1papcjm9pgqvwxrjd2zzft4cr43rvsym7qup2y3cgq7tzhptm0xg6sg04td8": "71",
				},
			},

			"usdt": {
				Name: "usdt",
				Max: "24000000",
				Minted: "2000",
				MintCount: 2,
				HolderCount: 1,
				TxCount: 5,
				Holders: map[string]string{
					"tb1p48rat08qtandh564ld2fxf85evw5655q3eqd4ttt307c0lf80r9q29l04s": "2000",
				},
			},

			"GC  ": {
				Name: "GC  ",
				Max: "210000",
				Minted: "700",
				MintCount: 7,
				HolderCount: 2,
				TxCount: 10,
				Holders: map[string]string{
					"tb1pj2lgtsa5x9pg7vhugxgumpfs8uu867xhuw28spwrkjzqmmvjm24qwfut59": "470",
					"tb1pgc9wqc2df5t0ec2a25fv45zkh8sgpl8yks236s036jfnhk0jc8nq40kzj2": "230",
				},
			},
		},
	},

	60000: {
		Height: 60000,
		TickerCount: 18,
		Tickers: map[string]*TickerStatus{
			"ordi": {
				Name: "ordi",
				Max: "2400000000",
				Minted: "110137",
				MintCount: 24,
				HolderCount: 8,
				TxCount: 77,
				Holders: map[string]string{
					"tb1pmm586mlhs35e8ns08trdejpzv02rupx0hp9j8arumg5c29dyrfnq2trqcw": "98000",
					"tb1p5pmdgkjk2dcpgmme2wx5q0uvwnzk6zhhfkpn5ldtuy3syn07hh4qqm2lsv": "9986",
					"tb1prt46ejv34r2qaukk3wgnaghcfm7tzm26wt2hkxe95zzrnquacmsqgtmqyt": "1000",
					"tb1plts00urlmu2kf7gcnp5225dnh4f0tn7e0r2jlvnd0exrwd03xe4ssykzpj": "1000",
					"tb1papcjm9pgqvwxrjd2zzft4cr43rvsym7qup2y3cgq7tzhptm0xg6sg04td8": "71",
					"tb1pj2lgtsa5x9pg7vhugxgumpfs8uu867xhuw28spwrkjzqmmvjm24qwfut59": "60",
					"tb1p8f5r8ed5nmhw9xgwyus0f6mrp8f8npvszj4x9gee4azd0t94fn9q3rj745": "10",
					"tb1pfuqd6gadnlycmyas8nc8zgads69uhzhejjvx8epenqa7pcfxqtkqngv4q4": "10",
				},
			},

			"usdt": {
				Name: "usdt",
				Max: "24000000",
				Minted: "2000",
				MintCount: 2,
				HolderCount: 1,
				TxCount: 5,
				Holders: map[string]string{
					"tb1p48rat08qtandh564ld2fxf85evw5655q3eqd4ttt307c0lf80r9q29l04s": "2000",
				},
			},

			"GC  ": {
				Name: "GC  ",
				Max: "210000",
				Minted: "700",
				MintCount: 7,
				HolderCount: 2,
				TxCount: 10,
				Holders: map[string]string{
					"tb1pj2lgtsa5x9pg7vhugxgumpfs8uu867xhuw28spwrkjzqmmvjm24qwfut59": "470",
					"tb1pgc9wqc2df5t0ec2a25fv45zkh8sgpl8yks236s036jfnhk0jc8nq40kzj2": "230",
				},
			},

			"husk": {
				Name: "husk",
				Max: "210000000",
				Minted: "20000",
				MintCount: 10,
				HolderCount: 1801,
				TxCount: 5811,
				Holders: map[string]string{
					"tb1pclqddn5aed3wtq78mgekrfe5c7s3dcvdz0a2ylcxmdhmuualr90sr04sc4": "18200",
					"tb1pu0rx5g5v58mvegyqdxj64fkvdsjjgfcv9lyfp4eax02wunmurjhs2ls9uv": "1",
				},
			},
		},
	},

	100000: {
		Height: 100000,
		TickerCount: 203,
		Tickers: map[string]*TickerStatus{
			"ordi": {
				Name: "ordi",
				Max: "2400000000",
				Minted: "1211580869",
				MintCount: 121391,
				HolderCount: 133,
				TxCount: 121598,
				Holders: map[string]string{
					"tb1pgw439hxzr7vj0gzfqx69wl3plem4ne26kj7ktnuzj3lkpw5mmp3qhz7yv4": "230000000",
					"tb1p6eahny66039p30ntrp9ke0qpyyffgnkekf69js6d2qcjf8cdmu0shx273f": "230000000",
					"tb1pc2nqm8k0kwnctkr2amchtcys4fq4elkq8ezhtsrntlkfc92z5tssh68xzl": "190000000",
					"tb1qy6zm520mnla9894t4jqvwe9s2sjsn2sfude0r0": "50260000",
					"tb1plzvdzn3sagtlavxsrdv9kp65empk80j0ksmazzqdc6nqkarj238s4r5qwx": "50000000",
					"tb1p5cymzvgf87fgeuzfexwxgvlmuuq309gegfh4q6np8g4qq6lnlk3qpzf2rs": "50000000",
					"tb1qmtlvgn8fl8ug2kgu26r6j9gykxm90tv5v4f6zx": "40000000",
					"tb1qn5pvsgw32gshn365n93wzw606hfy9k6cuvkxmn": "30000000",
					"tb1qw3qp3d0m0ykl2v7yj4uvrp4gsw8pwqmghul8w8": "30000000",
					"tb1qw65mlex2hpv2py2pucysfrfe59h3acde3vtya9": "20260000",
					"tb1qffmg3mrgfwk4uhml0umffyhf0hwk8hyrn794jt": "20240000",
					"tb1qygnyv2pdvecdgvmp22sm7pezvs5qpj3npm0cdd": "20000000",
					"tb1q8qm4f4t0aezwpm0y6cmnzv98z47ua35e8mkgmg": "20000000",
					"tb1qs4hgfvgu87jd9k2gxpvmekd7duxelj8urkmkrn": "20000000",
					"tb1qvg9wgs68w35pevc6uewsxge3xqwg03qd0wnkag": "20000000",
					"tb1q0294eqavsdtmz38pq7gp9zxtx8cwftexky2395": "20000000",
					"tb1qtmrk4p3luatdjmk60p2nszmjn8gjqccer33ysh": "20000000",
					"tb1qcxcynntlpkmp6nleu3e3g3dt6nqy5vnjyeg0p6": "10010700",
					"tb1qrgrv3z660lfzjq59mva2at68qf99msg3j8qv7g": "10000000",
					"tb1q70le3xu5de783xrhvqfhhhxf26zttg39lgcckm": "10000000",
					"tb1qs2zkvg99lg5cn0p89qw826yqnxf4ljzkpgf5dm": "10000000",
				},
			},

			"usdt": {
				Name: "usdt",
				Max: "24000000",
				Minted: "24000000",
				MintCount: 24001,
				HolderCount: 13,
				TxCount: 24009,
				Holders: map[string]string{
					"tb1p5cymzvgf87fgeuzfexwxgvlmuuq309gegfh4q6np8g4qq6lnlk3qpzf2rs": "7910000",
					"tb1pgw439hxzr7vj0gzfqx69wl3plem4ne26kj7ktnuzj3lkpw5mmp3qhz7yv4": "4302000",
					"tb1p6eahny66039p30ntrp9ke0qpyyffgnkekf69js6d2qcjf8cdmu0shx273f": "4239000",
					"tb1ptkqd49ueqf25enjk7hkd4rquycamcgpe5z0p96rdypnh72ac7eeqgqeztp": "1915744",
					"tb1pxe4t7t8m2qcga5kvr8emgt2m9knyqssa0yl633m0ws4qmuagar6sj4xz56": "1285000",
					"tb1pmdpgajqsl46nlmuhhz0ey7ajf900j0d7g84c7q3htmtjyvuz53zssps35j": "1229000",
					"tb1pggnahraey96gua4kvzc3pkkd5ywewtk4n8lh3elfnuznnhyga2tqesxhlk": "1227000",
					"tb1pga4y002wqsfrtserdyv80tcrn8yrhzfdgl4cz4wrxrpv0r5knpks6fcffr": "640000",
					"tb1pr2xm453pl0zxuay5jar67nlr0m08ke4c3udsc0uxxfsg8un4wtxqyzkzs9": "628000",
					"tb1pde9jyp3zatwfj2r853a7asykwjg4x3ga4vsnyr4uuvfs66afrj3s2ft0vm": "611000",
					"tb1pcm5ccj039x4e558l8sxvspmhuxrn2zauz5k2rnt3zz0n6jzhcqkqd3grsv": "11000",
					"tb1p48rat08qtandh564ld2fxf85evw5655q3eqd4ttt307c0lf80r9q29l04s": "2000",
					"tb1pfw5q9yak92c9us4rwxv4c6hduk2ml7uc7et8mc8qdfj89aeeg8dstdqx57": "256",
				},
			},

			"Test": {
				Name: "Test",
				Max: "21000000",
				Minted: "14917000",
				MintCount: 14917,
				HolderCount: 32,
				TxCount: 14957,
				Holders: map[string]string{
					"tb1p5cymzvgf87fgeuzfexwxgvlmuuq309gegfh4q6np8g4qq6lnlk3qpzf2rs": "3000000",
					"tb1q0juamrh0s56hwzh9w5af2r9qfn986tym4h6yz5": "2450000",
					"tb1pgw439hxzr7vj0gzfqx69wl3plem4ne26kj7ktnuzj3lkpw5mmp3qhz7yv4": "2000000",
					"tb1plzvdzn3sagtlavxsrdv9kp65empk80j0ksmazzqdc6nqkarj238s4r5qwx": "2000000",
					"tb1p8k8cefngmd9jj4l98r4njc0e703969ufeqwq3sjvtvczmnrh6n8s5eyns3": "2000000",
					"tb1pxe4t7t8m2qcga5kvr8emgt2m9knyqssa0yl633m0ws4qmuagar6sj4xz56": "1000000",
					"tb1p57klsq8jaaxc00ryalqpfzaqgyjklndk203r4mqceuwms3dcwssslelm5f": "1000000",
					"tb1p3p2frpszwaq4vqm6mamu2t0u5n3n02p0rez0nhrgms29md4zrldqrladhc": "355000",
					"tb1qed3rnn7tlt0fjsur07rva0wf6qafvt8hud9n56": "291000",
					"tb1qp08qs59zlecmy32v2xl6jxcanduu0vprz9ys59": "273000",
					"tb1pdc76nva2m6lfxtvh76p3a85afnpyugdv36kdg082ekmvhq7yplxsy8sulq": "145000",
					"tb1qcspzht2al9u2xzn4g5g9wcegk3pc5gz6ql4t2l": "144000",
					"tb1qzm5t43xadvx2ev9ez8kzns3ephl4hwureudxz9": "93000",
					"tb1psftymfx22t26hgyuwv9mrp57jky0rz34frhnk9nargd4rr37hpls647k9j": "76000",
					"tb1pynkxrpt3nfz9n3w9rycsnumu37llpy8hd003e7zv56vn7jw68epquh5vkx": "58000",
					"tb1qx50a4v3rmyusf6srged0mpxk4z7pj3re96m2ax": "8000",
					"tb1pytpwuwfcqg0sz0udvsmj02ktpt5ksf2u22zruxk2nhxddm4zjasqw44e5l": "6000",
					"tb1qwsjf9v7all9tvn4f49c9s6wqedmv7mn86qpz2z": "4000",
					"tb1q6e5p8ceg080y3qrd87x059xjz2w8z489t2qaps": "4000",
					"tb1psf5knz52dexmy90krnry5agkph8cszvkpzhpll3a2whr5xalq33q33ajcm": "1100",
					"tb1pc7s0sykv4leu242d90ls0t883qjx5cs5fm0hdzzx686vnld2f3rsgjvvw8": "1000",
				},
			},

			"husk": {
				Name: "husk",
				Max: "210000000",
				Minted: "246000",
				MintCount: 123,
				HolderCount: 1806,
				TxCount: 5963,
				Holders: map[string]string{
					"tb1p8nr7kkfcp6g4l9m4mlaxn5wmfehht57er7ptekk44mrzq3uv9c4ql6z5l6": "198833.1559022",
					"tb1pclqddn5aed3wtq78mgekrfe5c7s3dcvdz0a2ylcxmdhmuualr90sr04sc4": "14178",
					"tb1pqq5k06wuhcstcr54aaw7rtnd4h59ns6ykpsmn572dncvd73400sqrsanjw": "4003",
					"tb1pdy9x2cxh7hnfam6ge9k7vtskydjqdz3sat0gp3ddytkmkczzslgspae5f9": "4003",
					"tb1pf0tcpgxr30kqhh2gn3pgvr4qyachzm0xxydnfxth5cxlkjl37ugspxp7tm": "3166.8440978",
					"tb1payznq05kum8wdj0dcy4z2qswj4z664fnpnfknkgae7na5tazvdns99zp9z": "2003",
					"tb1p8e3x8247p4gppwkjs6rn6ue76jhyagphwkz70ah8v2ghuleulw6szfglgr": "2003",
					"tb1pu4c3g4u00jncvaek0pu6efzuycyqxewrck8g0gkzaeewm7xeaqgqrm8j70": "2003",
					"tb1ppqlhvdf7anldme2kk7ydjj9hwl92wxs3xs4sjxcxdn2glq3rmypss8v9hd": "2003",
					"tb1puyu4eqrn48u4tkepv8raeeq6mpgu2rh9hew7p0cpcd5u5n4u2yxsra834y": "2003",
					"tb1pagjdz79jmng6l5wp3el3tsn3vlc5klf9kp5ldjhh5dysdn52yttqql94e0": "2003",
					"tb1pd27qtkvyw890va9fmel8vj5s7q6qns97kvv07935puw27ty7f2lq6vgkrp": "2003",
					"tb1pwzqrgypve5ghdzq4pyh7v9jgeztxtfygh3vtvjjch4jq57jl0v8sj9ag23": "2003",
					"tb1qk3l2xelnz4fuw5ezdm6lxaadgpgua2cfharshy": "2000",
					"tb1quhzurujsfl0e6q24dhr9qrly9pmvnsmpt29c6c": "2000",
					"mj6koXKt4BKb1TGdXEKURSTforYgg8pcPo": "2",
					"tb1pnjj9t6sf95wfd9u6acdmndsnhtf0vr7h4gk4ywzhvk706u5fkfxqyedu4f": "1",
					"tb1pvzyn3cvaldtwdatkty47e2lg4gfuevj783n6s9hv8u730rrhxsdsfwhzt6": "1",
					"tb1pk204h3uln4l7y905fh4d5n0tlr80ny7gtx5duv084te7g4cfvkfqv6cq7c": "1",
					"tb1p3zsmxd86t6jvmvznr2pl2l3jyxjzjqv9ldj5yws2y280rd5ehurs97umg7": "1",
					"tb1p6ej7x9yem5v25krlk658865ul69r3v2sczhwvm0ksjrcxsh5rhgq4dxzgj": "1",
				},
			},

			"GC  ": {
				Name: "GC  ",
				Max: "210000",
				Minted: "210000",
				MintCount: 2101,
				HolderCount: 6,
				TxCount: 2104,
				Holders: map[string]string{
					"tb1pz70t56u56kxr9hzeh3hx328y08e5ftlq4edtll59zalyg43mj9fq7wg9p9": "100000",
					"tb1pgw439hxzr7vj0gzfqx69wl3plem4ne26kj7ktnuzj3lkpw5mmp3qhz7yv4": "100000",
					"tb1p5cymzvgf87fgeuzfexwxgvlmuuq309gegfh4q6np8g4qq6lnlk3qpzf2rs": "9200",
					"tb1pj2lgtsa5x9pg7vhugxgumpfs8uu867xhuw28spwrkjzqmmvjm24qwfut59": "470",
					"tb1pgc9wqc2df5t0ec2a25fv45zkh8sgpl8yks236s036jfnhk0jc8nq40kzj2": "230",
					"tb1pfh9ragf49v76ewe46mlmadag92q4pttywm4pxkk7qswhyjwwcfjs8dh2uw": "100",
				},
			},

			"ttt3": {
				Name: "ttt3",
				Max: "100000000000",
				Minted: "210400",
				MintCount: 2104,
				HolderCount: 17,
				TxCount: 2220,
				Holders: map[string]string{
					"tb1pp7vrjrxg2m4mpxd2pjd60e8xr74plxm24vmxf7yxzcx3z7kmlk2s2vj2uz": "129104.666688888888888889",
					"tb1q6n55n8xuexk4gsp67htsp65kvvrnzmxzjv2ryf": "74461",
					"tb1pe2rf7engceepj0vjmkt2a28qr3hatv3suv8n7tl2cp2vas94f04sf0sck6": "3000",
					"tb1ph8m57xc93q4hsntj5085g84y3g87fry28ann5r98r7pq482vzsjqt6yfdc": "1043.333311111111111111",
					"tb1qgzqyaxd86s0jzfqnf7f2wzujt5lvchduc2mfwm": "1000",
					"tb1pppknmsy8mg6zwkts2pq7q72tewsf3rf8g28htzev2n5jg3cc9q6s7sw4lz": "1000",
					"tb1p45r5urlu2kwn8l7nx6zjhsp6zzkpdgzneng6ws9mmuc0e7xv07eqdxq3g5": "100",
					"tb1pttukeu2nfdqy5f60dwfy3tadds4defak7nj0wtgswk3tpgzmte6qwlevkw": "100",
					"tb1pnjjeudwz4m6t9400dfuz3xaug4r6fvwk07t3d4xx0wm8fecsgd6qvqd9cs": "100",
					"tb1p323mdjea9mnp8s3nw9j9spgqdugsup6zg7zdc2qqaw3m5mknhpvqhypz5p": "100",
					"tb1q8pzdma068pk5ushkyddhucwm8rfpywnuxww7mt": "100",
					"tb1pfteg0ynnhhagrvnlsse9k447d63g0gghawthaaaph9h5gt5fxnkq3n3smy": "100",
					"tb1plhgzs5js0kgl2ulwhflzkv8f8jmwcnm5gcngcvwq072jczgt6awsje4pnk": "80",
					"tb1pke0795uzrxqkw3tdeupszvvprgv0vcxd9haqtjw53frmjrwqknpsykkzlp": "50",
					"tb1pwh637kd9gutw0yq2d7k35lw34vn5akuz3x5k32wgxadynfmujp4sy9ghge": "50",
					"tb1qyvlfu3m2lcnaqzpt3neqact8gnyue26ur26vuy": "10",
					"tb1qgev3lxu4j58shxdj4gak5a4j2cunamfe2t484u": "1",
				},
			},

			"sats": {
				Name: "sats",
				Max: "2100000000000000000",
				Minted: "89218000",
				MintCount: 4249,
				HolderCount: 14,
				TxCount: 4341,
				Holders: map[string]string{
					"tb1plzvdzn3sagtlavxsrdv9kp65empk80j0ksmazzqdc6nqkarj238s4r5qwx": "42000000",
					"tb1p9vvpkrq6a2y7e5tlkadl2upf0njw9fmdd6ykl5r2g4fqvtpkxcwsjs4yv3": "23098754.42303111",
					"tb1qhafqkwhhl5rzwd5pwfzpc42udx0y6txd8rfkkj": "21000000",
					"tb1qulrg98cr3wntw5fw7wde2p60ryxkack0hy9d2t": "2100000",
					"tb1qcxcynntlpkmp6nleu3e3g3dt6nqy5vnjyeg0p6": "693000",
					"tb1q6n55n8xuexk4gsp67htsp65kvvrnzmxzjv2ryf": "272900",
					"tb1qwsjf9v7all9tvn4f49c9s6wqedmv7mn86qpz2z": "21000",
					"tb1pxwvdxnhqn2yepgkmyvptv50qz067wphetkz44gecgtuap6qzkp7s8whgew": "21000",
					"tb1p660jruap0r5m27rrhntgnq3r3f68mge9zl9rt5ahcwxvtyk47zasz0smx5": "9000",
					"tb1pupt6rkpqv77mkxwks85vewwhyakc430rzmelgw5dhg3v83wnf40slm7mqs": "1225.32591029",
					"tb1pr0f4qntxppatzsuh8qt2j2anfzxns05wn0dpzmj4xcqsgwgprr3qfj45ze": "1000",
					"tb1q76qc3rxsy5v764n24q6un3n9yzfker8y0tqsah": "100",
					"tb1pf0tcpgxr30kqhh2gn3pgvr4qyachzm0xxydnfxth5cxlkjl37ugspxp7tm": "19.31391875",
					"tb1qck26s8e99jd03rxtflkge9n70vsc7u8v8wyvls": "0.93713985",
				},
			},

			"TBTC": {
				Name: "TBTC",
				Max: "21000000",
				Minted: "10470.06391",
				MintCount: 251,
				HolderCount: 49,
				TxCount: 257,
				Holders: map[string]string{
					"tb1qulrg98cr3wntw5fw7wde2p60ryxkack0hy9d2t": "10000",
					"tb1q3whu504emtc8np7sg44xt8nhhlsxmscs9yx0ly": "145.52147",
					"tb1qw3wl4g2d6nljx9mcc7fpma322kksz2tsmn5uz4": "100",
					"tb1punpnvdyr2f7pmve0d8d39unkyacn4788vsjwwhnaq70n0ztwjausgna6hh": "100",
					"tb1qpj4newlzkh44psn8gnq9r9xrqde0cltn27y8n5": "100",
					"tb1ps5gq6eqnjchz8xdm65258h26jwara6nge4x7d4v8vlh928wew5rqejy8jp": "14.00012",
					"tb1pusrw9n97zrsy8vudp2am2tfzz342lasdptej7pwsnhaa73pem34sewddhe": "3.50002",
					"tb1p2dwjlkyqfq27xmnnxk37v87gnds2scq4n5fg0f405k9d22ft2q6qrc2f9d": "3.00004",
					"tb1ps3xlfe580w0yx7asj76pn4hmwwrres94e22vtp40cqzl9l2tk9eswrqz43": "1.02585",
					"tb1pgfpxj6qluljqw5dswauhnn8t8ycparz9fsj565hacqg8wtyyu4sqhnzvm6": "1.0001",
					"tb1pcr6e0mykey4rzyc83yvqtplrrcl388rcfd05903vw42r8wyjqgmq2ahyne": "1",
					"tb1qs909l05a5m82ncf0uty37c5x9gtx3a0d59fykw": "1",
					"tb1p0gqv7yzv3uxphf0anyw8gzsq38yru6p4r6wtr9pncwmwcf2aleqqn7ekxm": "0.01",
					"tb1pwv9g6yllr46rgnka424w2f8dgrzmetxr4sz3k6ydhwg5sn7fsy5s8g2uck": "0.0031",
					"tb1q6wmepejts8r5hdxqr6egjf9f2jg9vuedzv8shd": "0.0012",
					"tb1ptpqdyexm5xntu2hanh4zne84yzxjlpahvr4p5wau9tfzf8cxnkhq5ljdz5": "0.0003",
					"tb1pj5vxuz9mk0lxuw45kq6qr23m9mrpt4nn35d94z4cl3umtahf6p0sxdntqc": "0.00023",
					"tb1plw8tmkumsudmstx4pvcdv4heny3rw9alj5e9cknvrawhd2lxtz6s0gxl60": "0.0002",
					"tb1q7re8fr9qdmkchfjqga5uhe68669tfrqg2y2l5m": "0.00012",
					"tb1q3sl5m2vr4ep2a8qyxp66jsjymjkflm0pvkf40f": "0.00011",
					"tb1plc7flyj9kuqpuc7q79p9j2wwx2v0686x7p05flzshlstx0f8xp5qz7juew": "0.0001",
				},
			},
		},
	},

	113208: {
		Height: 113208,
		TickerCount: 254,
		Tickers: map[string]*TickerStatus{
			"GC  ": {Minted: "210000", HolderCount: 6, TxCount: 2104},
			"ordi": {Minted: "1251660992", HolderCount: 152, TxCount: 125664},
			"rats": {Minted: "2723000", HolderCount: 7, TxCount: 2730},
			"Usdt": {Minted: "24000000", HolderCount: 13, TxCount: 24009},
			"husk": {Minted: "2240000", HolderCount: 1809, TxCount: 6964},
			"sats": {Minted: "118639000", HolderCount: 19, TxCount: 5758},
			"Test": {Minted: "15023000", HolderCount: 34, TxCount: 15063},
			"ttt3": {Minted: "210400", HolderCount: 17, TxCount: 2220},
			"abcd": {Minted: "194999", HolderCount: 19, TxCount: 203},
			"TBTC": {Minted: "198570.06391", HolderCount: 55, TxCount: 2147},
			"⚽ ": {Minted: "100000000", HolderCount: 3, TxCount: 104},
			"cats": {Minted: "2088000", HolderCount: 4, TxCount: 2092},
			"bqb4": {Minted: "1000000", HolderCount: 4, TxCount: 1050},
			"💚": {Minted: "42000", HolderCount: 2, TxCount: 44},
			"DSWP": {Minted: "100000", HolderCount: 3, TxCount: 139},
			"doge": {Minted: "4347100", HolderCount: 11, TxCount: 4356},
			"ordx": {Minted: "36000000", HolderCount: 6, TxCount: 63},
		},
	},
}

var mainnet_checkpoint = map[int]*CheckPoint{
	0: {
		Tickers: map[string]*TickerStatus{
			"ordi": {},
		},
	},

	779831: {
		Height: 779831,
		TickerCount: 0,
		Tickers: nil,
	},
	780070: {
		Tickers: map[string]*TickerStatus{
			"ordi": {Holders: map[string]string{
			}},
		},
	},

	790693: {
		Tickers: map[string]*TickerStatus{
			"ordi": {Holders: map[string]string{
				"16G1xYBbiNG78LSuZdMqp6tux5xvVp9Wxh": "1677449",
			}},
		},
	},

	790694: {
		Tickers: map[string]*TickerStatus{
			"ordi": {Holders: map[string]string{
				"16G1xYBbiNG78LSuZdMqp6tux5xvVp9Wxh": "1669003",
			}},
		},
	},

	800000: {
		Tickers: map[string]*TickerStatus{
			"ordi": {Holders: map[string]string{
				"bc1qggf48ykykz996uv5vsp5p9m9zwetzq9run6s64hm6uqfn33nhq0ql9t85q": "757425.92310402",
				"bc1q6tj4wm295pndmx4dywkg27rj6vqfxl5gn8j7zr": "183763.73281121",
			}},
		},
	},

	813844: {
		Tickers: map[string]*TickerStatus{
			"ordi": {Holders: map[string]string{"bc1qhuv3dhpnm0wktasd3v0kt6e4aqfqsd0uhfdu7d": "51"}},
		},
	},
	815725: {
		Tickers: map[string]*TickerStatus{
			"ordi": {Holders: map[string]string{"bc1qhuv3dhpnm0wktasd3v0kt6e4aqfqsd0uhfdu7d": "567663.0989"}},
		},
	},
	815875: {
		Tickers: map[string]*TickerStatus{
			"ordi": {Holders: map[string]string{"bc1qhuv3dhpnm0wktasd3v0kt6e4aqfqsd0uhfdu7d": "2235818.51"}},
		},
	},
	815993: {
		Tickers: map[string]*TickerStatus{
			"ordi": {Holders: map[string]string{"bc1qhuv3dhpnm0wktasd3v0kt6e4aqfqsd0uhfdu7d": "2118615.09"}},
		},
	},
	815999: {
		Tickers: map[string]*TickerStatus{
			"ordi": {Holders: map[string]string{"bc1qhuv3dhpnm0wktasd3v0kt6e4aqfqsd0uhfdu7d": "2118609.09"}},
		},
	},
	816000: {
		Tickers: map[string]*TickerStatus{
			"ordi": {Holders: map[string]string{"bc1qhuv3dhpnm0wktasd3v0kt6e4aqfqsd0uhfdu7d": "2111923.4701435"}},
		},
	},
	824529: {
		Tickers: map[string]*TickerStatus{
			"ordi": {Holders: map[string]string{"bc1qhuv3dhpnm0wktasd3v0kt6e4aqfqsd0uhfdu7d": "9128777.40996118"}},
		},
	},

	/*
	780000: {
		Height: 780000,
		TickerCount: 12,
		Tickers: map[string]*TickerStatus{
		},
	},

	827306: { // ordx
		Height: 827306,
		TickerCount: 14,
		Tickers: map[string]*TickerStatus{
		},
	},

	839999: { // runes
		Height: 839999,
		TickerCount: 14,
		Tickers: map[string]*TickerStatus{
		},
	},

	920000: { // latest
		Height: 920000,
		TickerCount: 14,
		Tickers: map[string]*TickerStatus{
		},
	},
	*/
}



func (p *BRC20Indexer) CheckPointWithBlockHeight(height int) {

	p.validateHistory(height)

	startTime := time.Now()
	var checkpoint *CheckPoint
	matchHeight := height
	isMainnet := p.nftIndexer.GetBaseIndexer().IsMainnet()
	if isMainnet {
		checkpoint = mainnet_checkpoint[height]
		if checkpoint == nil {
			matchHeight = 0
			checkpoint = mainnet_checkpoint[0]
		}
	} else {
		checkpoint = testnet4_checkpoint[height]
		if checkpoint == nil {
			matchHeight = 0
			checkpoint = testnet4_checkpoint[0]
		}
	}
	if checkpoint == nil {
		return
	}

	if matchHeight != 0 {
		tickers := p.getAllTickers()
		if checkpoint.TickerCount != 0 && len(tickers) != checkpoint.TickerCount {
			common.Log.Panicf("ticker count different")
		}
	}
	// 太花时间
	//rpc := base.NewRpcIndexer(p.nftIndexer.GetBaseIndexer())
	baseIndexer := p.nftIndexer.GetBaseIndexer()
	for name, tickerStatus := range checkpoint.Tickers {
		name = strings.ToLower(name)
		tickerInfo := p.loadTickInfo(name)
		if tickerInfo == nil {
			common.Log.Panicf("CheckPointWithBlockHeight can't find ticker %s", name)
		}
		ticker := tickerInfo.Ticker
		if tickerStatus.Max != "" && ticker.Max.String() != tickerStatus.Max {
			common.Log.Panicf("%s Max different, %s %s", name, ticker.Max.String(), tickerStatus.Max)
		}
		if tickerStatus.Minted != "" && ticker.Minted.String() != tickerStatus.Minted {
			common.Log.Panicf("%s Minted different, %s %s", name, ticker.Minted.String(), tickerStatus.Minted)
		}
		if tickerStatus.MintCount!= 0 && ticker.MintCount != uint64(tickerStatus.MintCount) {
			common.Log.Panicf("%s MinteMintCountd different, %d %d", name, ticker.MintCount, tickerStatus.MintCount)
		}
		if tickerStatus.HolderCount != 0 && ticker.HolderCount != uint64(tickerStatus.HolderCount) {
			common.Log.Panicf("%s HolderCount different, %d %d", name, ticker.HolderCount, tickerStatus.HolderCount)
		}
		if tickerStatus.TxCount != 0 && ticker.TransactionCount != uint64(tickerStatus.TxCount) {
			common.Log.Panicf("%s TxCount different, %d %d", name, ticker.TransactionCount, tickerStatus.TxCount)
		}

		for address, amt := range tickerStatus.Holders {
			addressId := baseIndexer.GetAddressIdFromDB(address)
			if addressId == common.INVALID_ID {
				common.Log.Panicf("%s GetAddressIdFromDB %s failed", name, address)
			}
			abbrInfo := p.getHolderAbbrInfo(addressId, name)
			if abbrInfo == nil {
				common.Log.Panicf("%s getHolderAbbrInfo %x %s failed", name, addressId, address)
			}
			if abbrInfo.AssetAmt().String() != amt {
				p.printHistoryWithAddress(name, addressId)
				common.Log.Panicf("%s holder %s amt different, %s %s", name, address, abbrInfo.AssetAmt().String(), amt)
			}
		}

		holdermap := p.getHoldersWithTick(name)
		var holderAmount *common.Decimal
		for _, amt := range holdermap {
			holderAmount = holderAmount.Add(amt)
		}
		if holderAmount.Cmp(&ticker.Minted) != 0 {
			common.Log.Infof("block %d, ticker %s, asset amount different %s %s", 
				height, name, ticker.Minted.String(), holderAmount.String())
			
			printAddress := make(map[uint64]bool)
			for k, v := range holdermap {
				old, ok := p.holdermap[k]
				if ok {
					if old.Cmp(v) != 0 {
						common.Log.Infof("%x changed %s -> %s", k, old.String(), v.String())
						printAddress[k] = true
					}
				} else {
					common.Log.Infof("%x added %s -> %s", k, old.String(), v.String())
					printAddress[k] = true
				}
			}
			for k := range printAddress {
				p.printHistoryWithAddress(name, k)
			}

			//p.printHistory(name)
			//p.printHistoryWithAddress(name, 0x52b1777c)
			common.Log.Panicf("%s amount different %s %s", name, ticker.Minted.String(), holderAmount.String())
		}
		p.holdermap = holdermap
		
	}
	common.Log.Infof("CheckPointWithBlockHeight %d checked, takes %v", height, time.Since(startTime))
}

// 逐个区块对比某个brc20 ticker的相关事件，效率很低，只适合开发阶段做数据的校验，后续要关闭该校验
func (p *BRC20Indexer) validateHistory(height int) {
	if !_enable_validate {
		return
	}
	if height < 779832 {
		return
	}
	name := "ordi"
	if _validateData == nil {
		var err error
		_validateData, err = validate.ReadBRC20CSV("./indexer/brc20/validate/ordi.csv")
		if err != nil {
			common.Log.Panicf("ReadBRC20CSV failed, %v", err)
		}
		
		_heightToRecords = make(map[int][]*validate.BRC20CSVRecord)
		for _, record := range _validateData {
			v := _heightToRecords[record.Height]
			if len(v) == 0 {
				_heightToRecords[record.Height] = append([]*validate.BRC20CSVRecord(nil), record)
			} else {
				_heightToRecords[record.Height] = validate.InsertByInscriptionNumber(v, record)
			}
		}
	}
	if len(_validateData) == 0 {
		return
	}

	tobeValidating := make([]*HolderAction, 0)
	for _, v := range p.holderActionList {
		if v.Height != height || v.Ticker != name {
			continue
		}
		if v.Action == common.BRC20_Action_Transfer_Spent {
			continue
		}
		tobeValidating = append(tobeValidating, v)
	}
	
	sort.Slice(tobeValidating, func(i, j int) bool {
		if tobeValidating[i].Height == tobeValidating[j].Height {
			if tobeValidating[i].TxIndex == tobeValidating[j].TxIndex {
				return tobeValidating[i].TxInIndex < tobeValidating[j].TxInIndex
			}
			return tobeValidating[i].TxIndex < tobeValidating[j].TxIndex
		}
		return tobeValidating[i].Height < tobeValidating[j].Height
	})
	
	tobeMap := make(map[string]*HolderAction)
	for _, item := range tobeValidating {
		key := fmt.Sprintf("%d-%x", item.NftId, item.TxIndex)
		tobeMap[key] = item
	}

	// 执行验证
	validateRecords := _heightToRecords[height]
	validateMap := make(map[string]*validate.BRC20CSVRecord)
	for _, item := range validateRecords {
		key := fmt.Sprintf("%d-%x", item.InscriptionNumber, item.TxIdx)
		validateMap[key] = item
	}

	if len(validateRecords) != len(tobeValidating) {
		more := p.loadTransferHistoryWithHeightFromDB(name, height)
		for _, item := range more {
			if item.Action == common.BRC20_Action_Transfer_Spent {
				continue
			}
			key := fmt.Sprintf("%d-%x", item.NftId, item.TxIndex)
			tobeMap[key] = item
			tobeValidating = append(tobeValidating, item)
		}
		sort.Slice(tobeValidating, func(i, j int) bool {
			if tobeValidating[i].Height == tobeValidating[j].Height {
				if tobeValidating[i].TxIndex == tobeValidating[j].TxIndex {
					return tobeValidating[i].TxInIndex < tobeValidating[j].TxInIndex
				}
				return tobeValidating[i].TxIndex < tobeValidating[j].TxIndex
			}
			return tobeValidating[i].Height < tobeValidating[j].Height
		})
		if len(validateRecords) != len(tobeValidating) {
			diff1 := findDiffInMap(validateMap, tobeMap)
			if len(diff1) > 0 {
				common.Log.Infof("in validate data but missing in our process")
				for _, v := range diff1 {
					common.Log.Infof("%v", validateMap[v])
				}
			}

			diff2 := findDiffInMap(tobeMap, validateMap)
			if len(diff2) > 0 {
				common.Log.Infof("not in validate data but occur in our process")
				for _, v := range diff2 {
					common.Log.Infof("%v", tobeMap[v])
				}
			}

			common.Log.Panicf("transfer count different in block %d, %d %d", height, 
				len(validateRecords), len(tobeValidating))
		}
	}

	// 按顺序检查，顺序严格一致
	for i, valid := range validateRecords {
		item := tobeValidating[i]
		
		if item.TxIndex != valid.TxIdx ||
		(item.NftId) != (valid.InscriptionNumber) {
			if valid.Value == 0 && valid.Offset == 0 && valid.To == valid.From {
				// cancel-transfer 的例子，暂时没有准确排序，需要搜索查找
				for _, t := range tobeValidating {
					if t.TxIndex == valid.TxIdx &&
					t.NftId == valid.InscriptionNumber {
						item = t
						break
					}
				}
			} else {
				common.Log.Errorf("%d #%d %s different item in tx  %s, %d", height, 
					valid.InscriptionNumber, valid.InscriptionID, valid.TxID, valid.TxIdx)
				nft := p.nftIndexer.GetNftWithId(item.NftId)
				if nft != nil {
					common.Log.Infof("local: %d -> %s", nft.Base.Id, nft.Base.InscriptionId)
				}

				nft2 := p.nftIndexer.GetNftWithInscriptionId(valid.InscriptionID)
				if nft2 != nil {
					common.Log.Infof("validate: %d -> %s", nft2.Base.Id, nft2.Base.InscriptionId)
				}
				
				for _, tobe := range tobeValidating {
					if tobe.TxIndex == valid.TxIdx {
						common.Log.Infof("id: %d", tobe.NftId)
					}
				}
				common.Log.Infof("validate:")
				for _, v := range validateRecords {
					if v.TxIdx == valid.TxIdx {
						common.Log.Infof("id: %d", v.InscriptionNumber)
					}
				}
				
				common.Log.Panic("")
				

				// nft := p.nftIndexer.GetNftWithId(item.NftId)
				// if nft == nil {
				// 	common.Log.Panicf("GetNftWithId %d failed", item.NftId)
				// }
				// common.Log.Panicf("%d #%d %s different inscription number %d %s", 
				// height, valid.InscriptionNumber, valid.InscriptionID, item.NftId, nft.Base.InscriptionId)
			}
		}

		if item.Ticker != valid.Ticker {
			common.Log.Panicf("%d #%d %s different asset in tx  %s, %d", height, 
					valid.InscriptionNumber, valid.InscriptionID, valid.TxID, valid.TxIdx)
		}

		if item.Amount.String() != valid.Amount {
			// validate的数据，最多8位小数，并且做了4舍5入
			parts := strings.Split(valid.Amount, ".")
			d := 8
			if len(parts) == 2 {
				d = len(parts[1])
			}
			if item.Amount.Precision > d {
				amt := item.Amount.SetPrecisionWithRound(d)
				if amt.String() != valid.Amount {
					common.Log.Panicf("%d #%d %s different asset amount %s-%s in tx  %s, %d", height, 
						valid.InscriptionNumber, valid.InscriptionID, amt.String(), valid.Amount, valid.TxID, valid.TxIdx)
				}
			} else {
				common.Log.Panicf("%d #%d %s different asset amount %s in tx  %s, %d", height, 
						valid.InscriptionNumber, valid.InscriptionID, item.Amount.String(), valid.TxID, valid.TxIdx)
			}
		}

		if item.Action != validate.ActionToInt[valid.Type] {
			common.Log.Panicf("%d #%d %s different action in tx  %s, %d", height, 
					valid.InscriptionNumber, valid.InscriptionID, valid.TxID, valid.TxIdx)
		}

		// nft := p.nftIndexer.GetNftWithId(item.NftId)
		// if nft == nil {
		// 	common.Log.Panicf("GetNftWithId %d failed", item.NftId)
		// }
		// if nft.Base.InscriptionId != valid.InscriptionID {
		// 	common.Log.Panicf("inscription Id different in block %d, %s %s", height,
		// 		valid.InscriptionID, nft.Base.InscriptionId)
		// }
	}

}

// 找出T1中key不在T2的元素
func findDiffInMap[T1 any, T2 any](t1 map[string]T1, t2 map[string]T2) []string {
	result := make([]string, 0)
	for k := range t1 {
		_, ok := t2[k]
		if !ok {
			result = append(result, k)
		}
	}
	return result
}
