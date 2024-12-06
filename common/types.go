package common

import (
	"errors"
	"fmt"
	"math"
	"sort"
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
	TxAmount   int
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
// 白聪
var ASSET_PLAIN_SAT TickerName = TickerName{}

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

// 只有一种资产存在
func (p *AssetInfo_MainNet) PickUp(offset, amt int64) (*AssetInfo_MainNet, error) {
	result := &AssetInfo_MainNet{}
	result.Name = p.Name
	
	if amt > p.Amount {
		err := errors.New("pickup count is too big")
		return nil, err
	}

	if amt == 0 {
		// Nothing to pickup
		return result, nil
	}

	if amt == p.Amount {
		//All ranges are pickup
		return p.Clone(), nil
	}

	pickupRanges := make([]*OffsetRange, 0)
	remainingValue := amt
	for _, currentRange := range p.AssetOffsets {
		if currentRange.End <= offset {
			continue
		}
		var start int64
		if currentRange.Start <= offset {
			start = offset
		} else {
			start = currentRange.Start
		}
		rangeSize := currentRange.End - start
		if rangeSize > remainingValue {
			rangeSize = remainingValue
		}
		newRange := &OffsetRange{Start: start, End: start+rangeSize}
		pickupRanges = append(pickupRanges, newRange)
		remainingValue = remainingValue - rangeSize

		if remainingValue <= 0 {
			break
		}
	}
	
	// check valid
	if remainingValue != 0 {
		err := errors.New("pickup count is wrong")
		return nil, err
	}
	result.Amount = amt
	result.AssetOffsets = pickupRanges

	return result, nil
}


type TxAssets = swire.TxAssets
type TxAssets_MainNet []AssetInfo_MainNet

func (p *TxAssets_MainNet) ToTxAssets() TxAssets {
	result := make([]swire.AssetInfo, len(*p))
	for i, a := range *p {
		result[i] = a.AssetInfo
	}
	return result
}


func (p *TxAssets_MainNet) Clone() TxAssets_MainNet {
	if p == nil {
		return nil
	}

	newAssets := make(TxAssets_MainNet, len(*p))
	for i, asset := range *p {
		newAssets[i] = *asset.Clone()
	}

	return newAssets
}


// Add 将另一个资产列表合并到当前列表中
func (p *TxAssets_MainNet) Add(asset *AssetInfo_MainNet) error {
	if asset == nil {
		return nil
	}

	index, found := p.findIndex(&asset.Name)
	if found {
		if (*p)[index].Amount+asset.Amount < 0 {
			return fmt.Errorf("out of bounds")
		}
		(*p)[index].Amount += asset.Amount
	} else {
		*p = append(*p, AssetInfo_MainNet{}) // Extend slice
		copy((*p)[index+1:], (*p)[index:])
		(*p)[index] = *asset
		// TODO 调整offset？
	}
	return nil
}

// Subtract 从当前列表中减去另一个资产列表
func (p *TxAssets_MainNet) Subtract(asset *AssetInfo_MainNet) error {
	if asset == nil {
		return nil
	}
	if asset.Amount == 0 {
		return nil
	}

	index, found := p.findIndex(&asset.Name)
	if !found {
		return errors.New("asset not found")
	}
	if (*p)[index].Amount < asset.Amount {
		return errors.New("insufficient asset amount")
	}
	(*p)[index].Amount -= asset.Amount
	if (*p)[index].Amount == 0 {
		*p = append((*p)[:index], (*p)[index+1:]...)
	}
	return nil
}

// Binary search to find the index of an AssetName
func (p *TxAssets_MainNet) findIndex(name *swire.AssetName) (int, bool) {
	index := sort.Search(len(*p), func(i int) bool {
		if (*p)[i].Name.Protocol != name.Protocol {
			return (*p)[i].Name.Protocol >= name.Protocol
		}
		if (*p)[i].Name.Type != name.Type {
			return (*p)[i].Name.Type >= name.Type
		}
		return (*p)[i].Name.Ticker >= name.Ticker
	})
	if index < len(*p) && (*p)[index].Name == *name {
		return index, true
	}
	return index, false
}


func (p *TxAssets_MainNet) Find(asset *swire.AssetName) (*AssetInfo_MainNet, error) {
	index, found := p.findIndex(asset)
	if !found {
		return nil, errors.New("asset not found")
	}
	return &(*p)[index], nil
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
	n.Assets = make([]AssetInfo_MainNet, len(p.Assets))
	for i, u := range p.Assets {
		n.Assets[i] = *u.Clone()
	}
	return n
}

func (p *TxOutput) Value() int64 {
	return p.OutValue.Value
}

func (p *TxOutput) Zero() bool {
	return p.OutValue.Value == 0 && len(p.Assets) == 0
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

// should fill out Sats and Assets parameters.
func GenerateTxOutput(tx *wire.MsgTx, index int) *TxOutput {
	return &TxOutput{
		OutPointStr: tx.TxHash().String() + ":" + strconv.Itoa(index),
		OutValue:    *tx.TxOut[index],
	}
}

func (p *TxOutput) GetAsset(assetName *swire.AssetName) int64 {
	if assetName == nil || *assetName == ASSET_PLAIN_SAT {
		return p.Value()
	}
	asset, err := p.Assets.Find(assetName)
	if err != nil {
		return 0
	}
	return asset.Amount
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
