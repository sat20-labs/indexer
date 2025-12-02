package ord0_14_1

import (
	"fmt"
	"math"
)

type Txid [32]byte

type OutPoint struct {
	Hash  Txid   `json:"txid"`
	Index uint32 `json:"vout"`
}

func NewOutPoint(hash Txid, index uint32) OutPoint {
	return OutPoint{
		Hash:  hash,
		Index: index,
	}
}

func NewNullOutPoint() OutPoint {
	return OutPoint{
		Hash:  Txid{},
		Index: math.MaxUint32,
	}
}

func (op OutPoint) IsNull() bool {
	return op.Hash == Txid{} && op.Index == math.MaxUint32
}

func (op OutPoint) String() string {
	return fmt.Sprintf("%x:%d", op.Hash, op.Index)
}
