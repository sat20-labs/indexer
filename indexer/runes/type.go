package runes

import (
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"lukechampine.com/uint128"
)

// TODO 1 数据库分页（1000？） 2 默认值
// TODO 3 is_reserved 需要判断 rune 在高度内是不是合法，包括 reserved
// TODO 4 统计信息， 如 当前 rune 数量

type Address string
type InscriptionId string

type RuneInfo struct {
	Etching *runestone.Etching
	Parent  *InscriptionId
}
type RuneInfoMap map[runestone.Rune]*RuneInfo
type RuneMap map[runestone.RuneId]*runestone.Rune
type MintMap map[runestone.Rune][]*runestone.RuneId
type TransferMap map[runestone.Rune][]*runestone.Edict
type CenotaphMap map[runestone.Rune][]*runestone.Cenotaph

type Asset struct {
	Amount    uint128.Uint128
	IsEtching bool
}
type AssetMap map[runestone.Rune]*Asset
type AddressAsset struct {
	Assets    *AssetMap
	Mints     *MintMap
	Transfers *TransferMap
	Cenotaphs *CenotaphMap
}
type AddressAssetMap map[Address]*AddressAsset
