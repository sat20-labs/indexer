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
	// bc1pr0fst3mzyvdkpzatx2cs2cahqwsrnr4nm403kfnv2ec4nys6h5qqt77vxn
	_bootstrapPubKey = "03ab606f4dffd65965b4a9db957361800f8b03ed16acac11d5a4672801554596d0"
	// tb1p62gjhywssq42tp85erlnvnumkt267ypndrl0f3s4sje578cgr79sekhsua
	_bootstrapPubKey_test = "025fb789035bc2f0c74384503401222e53f72eefdebf0886517ff26ac7985f52ad"
	// tb1p62gjhywssq42tp85erlnvnumkt267ypndrl0f3s4sje578cgr79sekhsua
	_bootstrapPubKey_dev = "025fb789035bc2f0c74384503401222e53f72eefdebf0886517ff26ac7985f52ad"

	// bc1pv7cceqgdz8303mpaaaz8a8zxc773chakqzwyrjuq7xyvvmyqsgzq0j92gc
	_coreNodePubKey = "03da2321f4500d7bc7075cb04c3105fda024379fa66ebb4cba9a15af91d88ca54f"
	// tb1pdw8xjqphyntnvgl3w0vmzkzd7dx266jwcprzwt0qen62pyzpdqhsdvr26h
	_coreNodePubKey_test = "0367f26af23dc40fdad06752c38264fe621b7bbafb1d41ab436b87ded192f1336e"
	// tb1pdw8xjqphyntnvgl3w0vmzkzd7dx266jwcprzwt0qen62pyzpdqhsdvr26h
	_coreNodePubKey_dev = "0367f26af23dc40fdad06752c38264fe621b7bbafb1d41ab436b87ded192f1336e"
	

	CORENODE_STAKING_ASSET_NAME string = "ordx:f:pearl"
	CORENODE_STAKING_ASSET_AMOUNT int64 = 1000000

	TESTNET_CORENODE_STAKING_ASSET_NAME string = "ordx:f:dogcoin"
	TESTNET_CORENODE_STAKING_ASSET_AMOUNT int64 = 1000

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

var ENV string = "prd"

func GetBootstrapPubKey() string {
	switch ENV {
	case "prd":
		return _bootstrapPubKey
	case "test": 
		return _bootstrapPubKey_test
	case "dev": 
		return _bootstrapPubKey_dev
	}
	return _bootstrapPubKey
}

func GetCoreNodePubKey() string {
	switch ENV {
	case "prd":
		return _coreNodePubKey
	case "test": 
		return _coreNodePubKey_test
	case "dev": 
		return _coreNodePubKey_dev
	}
	return _coreNodePubKey
}
