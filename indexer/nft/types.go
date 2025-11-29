package nft

import (
	"github.com/sat20-labs/indexer/indexer/nft/pb"
)

const NFT_DB_VERSION = "1.0.0"
const NFT_DB_VERSION_KEY = "nsdbver"
const NFT_STATUS_KEY = "nftstatus"

const (
	DB_PREFIX_SAT      = "s-"  // sat -> NftsInSat
	DB_PREFIX_NFT      = "n-"  // nftId -> Nft
	DB_PREFIX_UTXO     = "u-"  // utxo -> []sat  所有存在资产的utxo
	DB_PREFIX_BUCK     = "bk-" // buck ->
	DB_PREFIX_INSC     = "i-"  // inscriptionId -> sat
	DB_PREFIX_INSCADDR = "a-"  // addressId -> sat
	DB_PREFIX_IT       = "it-" // content type id -> content type

	DB_PREFIX_IC = "ic-" // contentId -> content
	DB_PREFIX_CI = "ci-" // content -> id

	DB_PREFIX_DISABLED_SAT = "ds-" // disabled sat
)

type TransferAction struct {
	UtxoId    uint64
	AddressId uint64
	Sats      []int64 // sats
	Action    int     // -1 删除; 1 增加
}

type InscriptionInDB struct {
	Sat int64
	Id  int64
}

type SatOffset = pb.SatOffset
type NftsInUtxo = pb.NftsInUtxo

// 一个聪可以有多个nft
type RBTreeValue_NFTs struct {
	Ids []int64
}
