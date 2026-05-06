package common

import "math"

const (

	// 暂时将所有nft当做一类ticker来管理，以后扩展时，使用合集名称来区分
	ASSET_TYPE_NFT    = "o"  // 原始的铭文类型，一般作为还没有被识别为其他资产的类型，也就是基本都是垃圾。
	ASSET_TYPE_FT     = "f"
	ASSET_TYPE_EXOTIC = "e"
	ASSET_TYPE_NS     = "n"
)

const INVALID_INSCRIPTION_NUM = int64(math.MaxInt64) // 9223372036854775807

const MIN_BLOCK_INTERVAL = 1000


const (
	// TODO 正式发布前需要修改pubkey，使用全新的pubkey
	// bc1pr0fst3mzyvdkpzatx2cs2cahqwsrnr4nm403kfnv2ec4nys6h5qqt77vxn
	_bootstrapPubKey = "03ab606f4dffd65965b4a9db957361800f8b03ed16acac11d5a4672801554596d0"
	// tb1p62gjhywssq42tp85erlnvnumkt267ypndrl0f3s4sje578cgr79sekhsua
	_bootstrapPubKey_testnet = "025fb789035bc2f0c74384503401222e53f72eefdebf0886517ff26ac7985f52ad"

	// bc1pv7cceqgdz8303mpaaaz8a8zxc773chakqzwyrjuq7xyvvmyqsgzq0j92gc
	_coreNodePubKey = "03da2321f4500d7bc7075cb04c3105fda024379fa66ebb4cba9a15af91d88ca54f"
	// tb1pdw8xjqphyntnvgl3w0vmzkzd7dx266jwcprzwt0qen62pyzpdqhsdvr26h
	_coreNodePubKey_testnet = "0367f26af23dc40fdad06752c38264fe621b7bbafb1d41ab436b87ded192f1336e"
	

	CORENODE_STAKING_ASSET_NAME string = "ordx:f:pearl"
	CORENODE_STAKING_ASSET_AMOUNT int64 = 1000000
	CORENODE_STAKING_ASSET_NAME_V2 string = "ordx:f:pearl"
	CORENODE_STAKING_ASSET_AMOUNT_V2 int64 = 1000000
	STAKING_ASSET_SWITCH_HEIGHT int = 942000
	STAKING_ASSET_SWITCH_HEIGHT_L2 int = 38000

	TESTNET_CORENODE_STAKING_ASSET_NAME string = "ordx:f:dogcoin"
	TESTNET_CORENODE_STAKING_ASSET_AMOUNT int64 = 1000
	TESTNET_CORENODE_STAKING_ASSET_NAME_V2 string = "ordx:f:rarepizza"
	TESTNET_CORENODE_STAKING_ASSET_AMOUNT_V2 int64 = 1000
	TESTNET_STAKING_ASSET_SWITCH_HEIGHT int = 128000
	TESTNET_STAKING_ASSET_SWITCH_HEIGHT_L2 int = 3400

	NODE_TYPE_NORMAL    int = 0 // 普通聪网节点，只同步数据，不挖矿
	NODE_TYPE_LIGHT     int = 1 // 轻节点，暂时没有实现
	NODE_TYPE_MINER     int = 10 // 矿机，挖矿，不提供通道接入服务
	NODE_TYPE_CORE      int = 11 // 核心节点，挖矿，提供通道接入服务
	NODE_TYPE_BOOTSTRAP int = 20 // 引导节点
)

var ENABLE_TESTING bool = false
var CHAIN string = "mainnet"

func IsMainnet() bool {
	return CHAIN == "mainnet"
}

func GetBootstrapPubKey() string {
	if ENABLE_TESTING {
		return _bootstrapPubKey_testnet
	}
	switch CHAIN {
	case "mainnet":
		return _bootstrapPubKey
	case "testnet": 
		return _bootstrapPubKey_testnet
	}
	return _bootstrapPubKey
}

func GetCoreNodePubKey() string {
	if ENABLE_TESTING {
		return _coreNodePubKey_testnet
	}
	switch CHAIN {
	case "mainnet":
		return _coreNodePubKey
	case "testnet": 
		return _coreNodePubKey_testnet
	}
	return _coreNodePubKey
}

// L1 height
func GetStakeAssetName(height int) string {
	if ENABLE_TESTING {
		if height < TESTNET_STAKING_ASSET_SWITCH_HEIGHT {
			return TESTNET_CORENODE_STAKING_ASSET_NAME
		} else {
			return TESTNET_CORENODE_STAKING_ASSET_NAME_V2
		}
	}
	if CHAIN == "mainnet" {
		if height < STAKING_ASSET_SWITCH_HEIGHT {
			return CORENODE_STAKING_ASSET_NAME
		} else {
			return CORENODE_STAKING_ASSET_NAME_V2
		}
	} else {
		if height < TESTNET_STAKING_ASSET_SWITCH_HEIGHT {
			return TESTNET_CORENODE_STAKING_ASSET_NAME
		} else {
			return TESTNET_CORENODE_STAKING_ASSET_NAME_V2
		}
	}
}

func GetStakeAssetNameWithHeightL2(height int) string {
	if ENABLE_TESTING {
		if height < TESTNET_STAKING_ASSET_SWITCH_HEIGHT_L2 {
			return TESTNET_CORENODE_STAKING_ASSET_NAME
		} else {
			return TESTNET_CORENODE_STAKING_ASSET_NAME_V2
		}
	}
	if CHAIN == "mainnet" {
		if height < STAKING_ASSET_SWITCH_HEIGHT_L2 {
			return CORENODE_STAKING_ASSET_NAME
		} else {
			return CORENODE_STAKING_ASSET_NAME_V2
		}
	} else {
		if height < TESTNET_STAKING_ASSET_SWITCH_HEIGHT_L2 {
			return TESTNET_CORENODE_STAKING_ASSET_NAME
		} else {
			return TESTNET_CORENODE_STAKING_ASSET_NAME_V2
		}
	}
}

// L1 height
func GetStakeAssetAmt(height int) int64 {
	if ENABLE_TESTING {
		if height < TESTNET_STAKING_ASSET_SWITCH_HEIGHT {
			return TESTNET_CORENODE_STAKING_ASSET_AMOUNT
		} else {
			return TESTNET_CORENODE_STAKING_ASSET_AMOUNT_V2
		}
	}
	if CHAIN == "mainnet" {
		if height < STAKING_ASSET_SWITCH_HEIGHT {
			return CORENODE_STAKING_ASSET_AMOUNT
		} else {
			return CORENODE_STAKING_ASSET_AMOUNT_V2
		}
	} else {
		if height < TESTNET_STAKING_ASSET_SWITCH_HEIGHT {
			return TESTNET_CORENODE_STAKING_ASSET_AMOUNT
		} else {
			return TESTNET_CORENODE_STAKING_ASSET_AMOUNT_V2
		}
	}
}


func GetStakeAssetAmtWithHeightL2(height int) int64 {
	if ENABLE_TESTING {
		if height < TESTNET_STAKING_ASSET_SWITCH_HEIGHT_L2 {
			return TESTNET_CORENODE_STAKING_ASSET_AMOUNT
		} else {
			return TESTNET_CORENODE_STAKING_ASSET_AMOUNT_V2
		}
	}
	if CHAIN == "mainnet" {
		if height < STAKING_ASSET_SWITCH_HEIGHT_L2 {
			return CORENODE_STAKING_ASSET_AMOUNT
		} else {
			return CORENODE_STAKING_ASSET_AMOUNT_V2
		}
	} else {
		if height < TESTNET_STAKING_ASSET_SWITCH_HEIGHT_L2 {
			return TESTNET_CORENODE_STAKING_ASSET_AMOUNT
		} else {
			return TESTNET_CORENODE_STAKING_ASSET_AMOUNT_V2
		}
	}
}