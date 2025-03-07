package common

import "math"

const (

	// 暂时将所有nft当做一类ticker来管理，以后扩展时，使用合集名称来区分
	ASSET_TYPE_NFT    = "o"
	ASSET_TYPE_FT     = "f"
	ASSET_TYPE_EXOTIC = "e"
	ASSET_TYPE_NS     = "n"
)

const INVALID_INSCRIPTION_NUM = int64(math.MaxInt64) // 9223372036854775807

const MIN_BLOCK_INTERVAL = 1000


const (
	// TODO 正式发布前需要修改pubkey，使用全新的pubkey
	// tb1p62gjhywssq42tp85erlnvnumkt267ypndrl0f3s4sje578cgr79sekhsua
	BootstrapPubKey = "025fb789035bc2f0c74384503401222e53f72eefdebf0886517ff26ac7985f52ad" //
	BootStrapNodeId = 1

	// tb1pdw8xjqphyntnvgl3w0vmzkzd7dx266jwcprzwt0qen62pyzpdqhsdvr26h
	CoreNodePubKey = "0367f26af23dc40fdad06752c38264fe621b7bbafb1d41ab436b87ded192f1336e" //
	CoreNodeId = 100

	CORENODE_STAKING_ASSET_NAME string = "ordx:f:pearl"
	CORENODE_STAKING_ASSET_AMOUNT int64 = 1000000

	// 0 invalid
	// 1-99 boostrap
	// 100-999 core
	// 1000-99999 normal miner
	MIN_BOOTSTRAP_NODEID = 1
	MAX_BOOTSTRAP_NODEID = 9
	MIN_CORE_NODEID      = 100
	MAX_CORE_NODEID      = 999
	MIN_NORMAL_NODEID    = 100000
)

