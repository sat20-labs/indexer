package common

import (
	"math"
	"strconv"

	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common/pb"

	swire "github.com/sat20-labs/satsnet_btcd/wire"
)

const (
	DB_KEY_UTXO         = "u-"  // utxo -> UtxoValueInDB
	DB_KEY_ADDRESS      = "a-"  // address -> addressId
	DB_KEY_ADDRESSVALUE = "av-" // addressId-utxoId -> value
	DB_KEY_UTXOID       = "ui-" // utxoId -> utxo
	DB_KEY_ADDRESSID    = "ai-" // addressId -> address
	DB_KEY_BLOCK        = "b-"  // height -> block
)

// Address Type defined in txscript.ScriptClass

type UtxoValueInDB = pb.MyUtxoValueInDB

func ToAddrType(tp, reqSig int) uint32 {
	return uint32(tp << 16 + reqSig)
}

func FromAddrType(u uint32) (int, int) {
	return int(u >> 16), int(0xffff & u) 
}

type UtxoIdInDB struct {
	UtxoId uint64
	Value  int64
}

type UtxoValue struct {
	Op    int // -1 deleted; 0 read from db; 1 added
	Value int64
}

type AddressValueInDB struct {
	AddressType uint32
	AddressId   uint64
	Op          int                   // -1 deleted; 0 read from db; 1 added
	Utxos       map[uint64]*UtxoValue // utxoid -> value
}

type AddressValue struct {
	AddressType uint32
	AddressId   uint64
	Utxos       map[uint64]int64 // utxoid -> value
}

type BlockValueInDB struct {
	Height     int
	Timestamp  int64
	InputUtxo  int
	OutputUtxo int
	InputSats  int64
	OutputSats int64
	Ordinals   Range
	LostSats   []*Range // ordinals protocol issue
}

type BlockInfo struct {
	Height     int   `json:"height"`
	Timestamp  int64 `json:"timestamp"`
	TotalSats  int64 `json:"totalsats"`
	RewardSats int64 `json:"rewardsats"`
}

const INVALID_ID = math.MaxUint64

const ALL_TICKERS = "*"

type TickerName = swire.AssetName


type AssetOffsetRange struct {
	Range  *Range        `json:"range"`
	Offset int64         `json:"offset"`
	Assets []*TickerName `json:"assets"`
}

func (a *AssetOffsetRange) Clone() *AssetOffsetRange {
	asset := make([]*TickerName, len(a.Assets))
	copy(asset, a.Assets)
	return &AssetOffsetRange{
		Range:  &Range{Start: a.Range.Start, Size: a.Range.Size},
		Offset: a.Offset,
		Assets: asset,
	}
}

type UtxoInfo struct {
	UtxoId   uint64
	Value    int64
	PkScript []byte
	Ordinals []*Range
}

// offset range in a UTXO, not satoshi ordinals
type OffsetRange struct {
	Start int64
	End   int64  // 不包括End
}

type AssetInfo_MainNet struct {
	swire.AssetInfo
	AssetOffsets []*OffsetRange // 绑定了资产的聪的位置
}

func (p *AssetInfo_MainNet) Clone() *AssetInfo_MainNet {
	n := &AssetInfo_MainNet{
		AssetInfo: swire.AssetInfo{
			Name: p.Name,
			Amount: p.Amount,
			BindingSat: p.BindingSat,
		},
	}
	n.AssetOffsets = make([]*OffsetRange, len(p.AssetOffsets))
	for i := 0; i < len(p.AssetOffsets); i++ {
		n.AssetOffsets[i] = &OffsetRange{Start: p.AssetOffsets[i].Start, End: p.AssetOffsets[i].End}
	}
	return n
}

type TxOutput struct {
	OutPoint string
	OutValue wire.TxOut
	Sats     []*Range
	Assets   []*AssetInfo_MainNet 
	// 注意BindingSat属性，TxOutput.OutValue.Value必须大于等于
	// Assets数组中任何一个AssetInfo.BindingSat
}


func (p *TxOutput) Clone() *TxOutput {
	n := &TxOutput{
		OutPoint: p.OutPoint,
		OutValue: p.OutValue,
	}
	for _, u := range p.Sats {
		n.Sats = append(n.Sats, &Range{Start: u.Start, Size:u.Size})
	}
	for _, u := range p.Assets {
		n.Assets = append(n.Assets, u.Clone())
	}
	return n
}

func (p *TxOutput) Value() int64 {
	return p.OutValue.Value
}

func (p *TxOutput) SizeOfBindingSats() int64 {
	bindingSats := int64(0)
	for _, asset := range p.Assets {
		amount := int64(0)
		if asset.BindingSat != 0 {
			amount = (asset.Amount)
		}
		
		if amount > (bindingSats) {
			bindingSats = amount
		}
	}
	return bindingSats
}

type TxOutput_SatsNet struct {
	OutPoint string
	OutValue swire.TxOut
}

func (p *TxOutput_SatsNet) Value() int64 {
	return p.OutValue.Value
}

func (p *TxOutput_SatsNet) SizeOfBindingSats() int64 {
	bindingSats := int64(0)
	for _, asset := range p.OutValue.Assets {
		amount := int64(0)
		if asset.BindingSat != 0 {
			amount = (asset.Amount)
		}
		
		if amount > (bindingSats) {
			bindingSats = amount
		}
	}
	return bindingSats
}

// should fill out Sats and Assets parameters.
func GenerateTxOutput(tx *wire.MsgTx, index int) *TxOutput {
	return &TxOutput{
		OutPoint: tx.TxHash().String()+":"+strconv.Itoa(index),
		OutValue: *tx.TxOut[index],
	}
}

func GenerateTxOutput_SatsNet(tx *swire.MsgTx, index int) *TxOutput_SatsNet {
	return &TxOutput_SatsNet{
		OutPoint: tx.TxHash().String()+":"+strconv.Itoa(index),
		OutValue: *tx.TxOut[index],
	}
}
