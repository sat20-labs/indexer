package runes

import (
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"lukechampine.com/uint128"
)

// TODO 1 数据库分页（1000？,应该只能用原来的方法，因为需要排序） 2 默认值
// TODO 3 is_reserved 需要判断 rune 在高度内是不是合法，包括 reserved
// TODO 4 统计信息， 如 当前 rune 数量还有当前的block height
// TODO 5 burned runemap 也需要定义和存储数据库，每个RUNDid 对应的 burned amount
// TODO 6 发现 Term没有的Etching,是不能被mint的
type RunesStatus struct {
	Version string
	Count   uint64
}

type Address string

type RuneInfo struct {
	Etching *runestone.Etching
	Parent  *runestone.InscriptionId
}
type RuneInfoMap map[runestone.Rune]*RuneInfo
type RuneMap map[runestone.RuneId]*runestone.Rune
type RuneMintMap map[runestone.Rune][]*runestone.RuneId
type RuneTransferMap map[runestone.Rune][]*runestone.Edict
type RuneCenotaphMap map[runestone.Rune][]*runestone.Cenotaph

type RuneAsset struct {
	Amount    uint128.Uint128
	IsEtching bool
}
type RuneAssetMap map[runestone.Rune]*RuneAsset
type RuneAddressAsset struct {
	Assets    *RuneAssetMap
	Mints     *RuneMintMap
	Transfers *RuneTransferMap
	Cenotaphs *RuneCenotaphMap
}
type AddressAssetMap map[Address]*RuneAddressAsset
