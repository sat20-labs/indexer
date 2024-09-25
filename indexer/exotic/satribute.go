package exotic

const (
	Pizza            string = "pizza"
	Block9           string = "block9"
	Block78          string = "block78"
	Nakamoto         string = "nakamoto"
	FirstTransaction string = "1stTX"
	Vintage          string = "vintage"
	Common           string = "common"
	Uncommon         string = "uncommon"
	Rare             string = "rare"
	Epic             string = "epic"
	Legendary        string = "legendary"
	Mythic           string = "mythic"
	Black            string = "black"
	Alpha            string = "alpha"
	Omega            string = "omega"
	Hitman           string = "hitman"
	Jpeg             string = "jpeg"
	Fibonacci        string = "fibonacci"
	Customized       string = "customized"
)

var SatributeList = []string{
	Pizza,
	Block9,
	Block78,
	Nakamoto,
	FirstTransaction,
	Vintage,
	Uncommon,
	Rare,
	Epic,
	Legendary,
	Mythic,
	Black,  // 区块最后一聪, Uncommon-1
	//Alpha,  ordinals协议的编号有争议，暂时关闭
	//Omega,  ordinals协议的编号有争议，暂时关闭
	//Hitman, ordinals协议的编号有争议，暂时关闭
	//Jpeg, ordinals协议的编号有争议，暂时关闭
	//Fibonacci,  需要预先计算每一个sat再保存起来，目前没热度，先不支持
	//Customized,
}
