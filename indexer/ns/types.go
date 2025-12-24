package ns

import (
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/ns/pb"
)

const NS_DB_VERSION = "1.0.1"
const NS_DB_VERSION_KEY = "nsdbver"
const NS_STATUS_KEY = "nsstatus"

const (
	DB_PREFIX_NAME = "r-" // name  NameRegister
	DB_PREFIX_SAT  = "s-" // sat -> name
	DB_PREFIX_KV   = "k-" // key-value  KeyValueInDB
	DB_PREFIX_BUCK = "bk-"
)

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type NameUpdate struct {
	InscriptionId string `json:"inscriptionId"`
	BlockHeight   int    `json:"height"`

	Sat  int64       `json:"sat"`
	Name string      `json:"name"`
	KVs  []*KeyValue `json:"kvs"`
}

type TransferAction struct {
	UtxoId    uint64
	AddressId uint64
	Names     map[string]bool
	Action    int // -1 删除; 1 增加
}

type NameValueInDB = pb.NameValueInDB

// 由nft维持实时状态
type NameRegister struct {
	Nft  *common.Nft
	Id   int64
	Name string
}

type NameProperties struct {
	NameRegister
	KVs map[string]*common.KeyValueInDB // key->value
}

// 一个聪只能绑定一个名字。再次绑定会报错。
type RBTreeValue_Name struct {
	Name string
}
