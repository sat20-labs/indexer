package nft

import (
	"github.com/sat20-labs/indexer/indexer/nft/pb"
)

const NFT_DB_VERSION = "1.0.1" // support on-chain collection and gallery
const NFT_DB_VERSION_KEY = "nsdbver"
const NFT_STATUS_KEY = "nftstatus"

const (
	DB_PREFIX_SAT      		= "a-" // sat -> NftsInSat
	DB_PREFIX_NFT      		= "b-" // nftId -> Nft
	DB_PREFIX_UTXO     		= "c-" // utxo -> []sat  所有存在资产的utxo
	DB_PREFIX_BUCK    		= "d-" // buck ->
	DB_PREFIX_INSC     		= "e-" // inscriptionId -> sat
	DB_PREFIX_INSCADDR 		= "f_" // addressId+nftId -> sat 
	DB_PREFIX_IT       		= "g-" // content type id -> content type
	DB_PREFIX_IC 	   		= "h-" // contentId -> content
	DB_PREFIX_CI 	   		= "i-" // content -> id
	DB_PREFIX_DISABLED_SAT 	= "j-" // disabled sat
	DB_PREFIX_COLLECTION 	= "k-" // parent->children
	DB_PREFIX_GALLERY   	= "l-" //
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
