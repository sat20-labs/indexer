package common

import (
	"errors"
	"math"
	"strconv"
	"strings"

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
	return uint32(tp<<16 + reqSig)
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
	End   int64 // 不包括End
}

type AssetInfo_MainNet struct {
	swire.AssetInfo
	AssetOffsets []*OffsetRange // 绑定了资产的聪的位置
}

func (p *AssetInfo_MainNet) Clone() *AssetInfo_MainNet {
	n := &AssetInfo_MainNet{
		AssetInfo: swire.AssetInfo{
			Name:       p.Name,
			Amount:     p.Amount,
			BindingSat: p.BindingSat,
		},
	}
	n.AssetOffsets = make([]*OffsetRange, len(p.AssetOffsets))
	for i := 0; i < len(p.AssetOffsets); i++ {
		n.AssetOffsets[i] = &OffsetRange{Start: p.AssetOffsets[i].Start, End: p.AssetOffsets[i].End}
	}
	return n
}

type TxAssets = swire.TxAssets
type TxAssets_MainNet []*AssetInfo_MainNet

func (p *TxAssets_MainNet) ToTxAssets() TxAssets {
	result := make([]swire.AssetInfo, len(*p))
	for i, a := range *p {
		result[i] = a.AssetInfo
	}
	return result
}

type TxOutput struct {
	OutPointStr string
	OutValue    wire.TxOut
	Sats        TxRanges
	Assets      TxAssets_MainNet
	// 注意BindingSat属性，TxOutput.OutValue.Value必须大于等于
	// Assets数组中任何一个AssetInfo.BindingSat
}

func (p *TxOutput) Clone() *TxOutput {
	n := &TxOutput{
		OutPointStr: p.OutPointStr,
		OutValue:    p.OutValue,
	}
	n.Sats = make(TxRanges, len(p.Sats))
	for i, u := range p.Sats {
		n.Sats[i] = &Range{Start: u.Start, Size: u.Size}
	}
	n.Assets = make([]*AssetInfo_MainNet, len(p.Assets))
	for i, u := range p.Assets {
		n.Assets[i] = u.Clone()
	}
	return n
}

func (p *TxOutput) Value() int64 {
	return p.OutValue.Value
}

func (p *TxOutput) OutPoint() *wire.OutPoint {
	outpoint, _ := wire.NewOutPointFromString(p.OutPointStr)
	return outpoint
}

func (p *TxOutput) TxID() string {
	parts := strings.Split(p.OutPointStr, ":")
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}

func (p *TxOutput) TxIn() *wire.TxIn {
	outpoint, err := wire.NewOutPointFromString(p.OutPointStr)
	if err != nil {
		return nil
	}
	return wire.NewTxIn(outpoint, nil, nil)
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
	OutPointStr string
	OutValue swire.TxOut
}

func (p *TxOutput_SatsNet) Value() int64 {
	return p.OutValue.Value
}

func (p *TxOutput_SatsNet) OutPoint() *swire.OutPoint {
	outpoint, _ := swire.NewOutPointFromString(p.OutPointStr)
	return outpoint
}


func (p *TxOutput_SatsNet) TxID() string {
	parts := strings.Split(p.OutPointStr, ":")
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}

func (p *TxOutput_SatsNet) TxIn() *swire.TxIn {
	outpoint, err := swire.NewOutPointFromString(p.OutPointStr)
	if err != nil {
		return nil
	}
	return swire.NewTxIn(outpoint, nil, nil)
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
		OutPointStr: tx.TxHash().String() + ":" + strconv.Itoa(index),
		OutValue:    *tx.TxOut[index],
	}
}

func GenerateTxOutput_SatsNet(tx *swire.MsgTx, index int) *TxOutput_SatsNet {
	return &TxOutput_SatsNet{
		OutPointStr: tx.TxHash().String() + ":" + strconv.Itoa(index),
		OutValue: *tx.TxOut[index],
	}
}


type TxRanges []*Range

// t被改变， ranges不会被改变
func TxRangesAppend(t TxRanges, ranges TxRanges) TxRanges {
	len1 := len(t)
	len2 := len(ranges)
	if len1 == 0 {
		return ranges.Clone()
	}
	if len2 == 0 {
		return t
	}

	t.Append(ranges)
	return t
}

func (p *TxRanges) GetSize() int64 {
	size := int64(0)
	for _, rng := range *p {
		size += (rng.Size)
	}
	return size
}

func (p *TxRanges) Clone() TxRanges {
	result := make([]*Range, len(*p))
	for i, u := range *p {
		result[i] = &Range{Start: u.Start, Size: u.Size}
	}
	return result
}

// Pickup an TxRanges with the given offset and count from the current TxRanges, 
// current TxRanges is not changed
func (p *TxRanges) PickUp(offset, count int64) (TxRanges, error) {
	size := p.GetSize()
	if offset+count > size {
		err := errors.New("pickup count is too big")
		return nil, err
	}

	if count == 0 {
		// Nothing to pickup
		return TxRanges{}, nil
	}

	if offset == 0 && count == size {
		//All ranges are pickup
		return p.Clone(), nil
	}

	pickupRanges := make(TxRanges, 0)
	remainingValue := count
	pos := int64(0)
	for _, currentRange := range *p {
		if pos < offset {
			if pos+currentRange.Size <= offset {
				pos = pos + currentRange.Size
				continue
			}
			// Will pickup from current range

			start := currentRange.Start + (offset - pos)
			rangeSize := currentRange.Size - (offset - pos)
			if rangeSize > remainingValue {
				rangeSize = remainingValue
			}
			newRange := &Range{Start: start, Size: rangeSize}
			pickupRanges = append(pickupRanges, newRange)
			remainingValue = remainingValue - rangeSize
		} else {
			// Will pickup from current range
			start := currentRange.Start
			rangeSize := currentRange.Size
			if rangeSize > remainingValue {
				rangeSize = remainingValue
			}
			newRange := &Range{Start: start, Size: rangeSize}
			pickupRanges = append(pickupRanges, newRange)
			remainingValue = remainingValue - rangeSize
		}

		pos = pos + currentRange.Size

		if remainingValue <= 0 {
			break
		}
	}

	// check valid
	pickupSize := pickupRanges.GetSize()
	if count != pickupSize {
		err := errors.New("pickup count is wrong")
		return nil, err
	}

	return pickupRanges, nil
}


func (p *TxRanges) Resize(amt int64)  {
	result := make([]*Range, 0)
	size := int64(0)
	for _, rng := range *p {
		if size+(rng.Size) <= amt {
			result = append(result, rng)
			size += (rng.Size)
		} else {
			newRng := Range{Start: rng.Start, Size: (amt - size)}
			result = append(result, &newRng)
			size += (newRng.Size)
		}

		if size == amt {
			break
		}
	}
	*p = result
}

func (p *TxRanges) Split(amount int64) ([]*Range, []*Range) {
	var front, end []*Range
	var sum int64

	for _, r := range *p {
		if sum >= amount {
			// 如果已经达到或超过了 amount，将剩余的范围添加到 after
			end = append(end, r)
		} else if sum+r.Size <= amount {
			// 如果当前范围完全在 amount 之前，添加到 before
			front = append(front, r)
			sum += r.Size
		} else {
			// 需要分割当前范围
			splitPoint := amount - sum
			front = append(front, &Range{Start: r.Start, Size: splitPoint})
			end = append(end, &Range{Start: r.Start + splitPoint, Size: r.Size - splitPoint})
			sum = amount
		}
	}

	return front, end
}

// 确保输出是第一个参数。只需要检查第一组的最后一个和第二组的第一个
func (p *TxRanges) Append(rngs2 TxRanges) {
	var r1, r2 *Range
	len1 := len(*p)
	len2 := len(rngs2)
	if len1 > 0 {
		if len2 == 0 {
			return
		}
		r1 = (*p)[len1-1]
		r2 = rngs2[0]
		if r1.Start+r1.Size == r2.Start {
			r1.Size += r2.Size
			*p = append(*p, rngs2[1:]...)
		} else {
			*p = append(*p, rngs2...)
		}
	} else {
		*p = append(*p, rngs2...)
	}
}

// 确保输出是第一个参数。只需要检查第一组的最后一个和第二组的第一个
func (p *TxRanges) AppendRange(rngs2 *Range) {
	var r1, r2 *Range
	len1 := len(*p)
	if len1 > 0 {
		r1 = (*p)[len1-1]
		r2 = rngs2
		if r1.Start+r1.Size == r2.Start {
			r1.Size += r2.Size
		} else {
			*p = append(*p, rngs2)
		}
	} else {
		*p = append(*p, rngs2)
	}
}


// 确保输出是第一个参数。只需要检查第一组的最后一个和第二组的第一个
func AppendOffsetRange(rngs1 []*OffsetRange, rngs2 *OffsetRange) []*OffsetRange {
	var r1, r2 *OffsetRange
	len1 := len(rngs1)
	if len1 > 0 {
		r1 = rngs1[len1-1]
		r2 = rngs2
		if r1.End == r2.Start {
			r1.End = r2.End
		} else {
			rngs1 = append(rngs1, rngs2)
		}
		return rngs1
	} else {
		rngs1 = append(rngs1, rngs2)
		return rngs1
	}
}
