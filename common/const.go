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
