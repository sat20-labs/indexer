package common

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/wire"
)

// 所有聪
var ASSET_ALL_SAT AssetName = AssetName{
	Protocol: "",
	Type: "*",
	Ticker: "",
}

// 白聪
var ASSET_PLAIN_SAT AssetName = AssetName{}

// offset range in a UTXO, not satoshi ordinals
type OffsetRange struct {
	Start int64
	End   int64 // 不包括End
}

func (p *OffsetRange) Clone() *OffsetRange {
	if p == nil {
		return nil
	}
	n := *p
	return &n
}

type AssetOffsets []*OffsetRange

func (p *AssetOffsets) Clone() AssetOffsets {
	if p == nil {
		return nil
	}

	result := make([]*OffsetRange, len(*p))
	for i, u := range *p {
		result[i] = &OffsetRange{Start: u.Start, End: u.End}
	}
	return result
}

func (p *AssetOffsets) Size() int64 {
	if p == nil {
		return 0
	}
	total := int64(0)
	for _, rng := range *p {
		total += rng.End - rng.Start
	}
	return total
}

// 按资产分割
func (p *AssetOffsets) Split(amt int64) (AssetOffsets, AssetOffsets) {
	var left, right []*OffsetRange

	if p == nil {
		return nil, nil
	}
	if amt == 0 {
		return nil, p.Clone()
	}

	remaining := amt
	offset := int64(0)
	for _, r := range *p {
		if remaining > 0 {
			if r.End-r.Start <= remaining {
				// 完全在左边
				left = append(left, r)
				offset = r.End
			} else {
				// 跨越 offset，需要拆分
				left = append(left, &OffsetRange{Start: r.Start, End: r.Start + remaining})
				offset = r.Start + remaining
				right = append(right, &OffsetRange{Start: 0, End: r.End - offset})
			}
			remaining -= r.End - r.Start
		} else {
			n := r.Clone()
			n.Start -= offset
			n.End -= offset
			right = append(right, n)
		}
	}

	return left, right
}

// 按聪数量分割, Append 的逆操作
func (p *AssetOffsets) Cut(value int64) (AssetOffsets, AssetOffsets) {
	var left, right []*OffsetRange

	if p == nil {
		return nil, nil
	}
	if value == 0 {
		return nil, p.Clone()
	}

	offset := value
	for _, r := range *p {
		if r.Start >= offset {
			// 完全在右边
			n := r.Clone()
			n.Start -= offset
			n.End -= offset
			right = append(right, n)
		} else if r.End <= offset {
			// 完全在左边
			left = append(left, r.Clone())
		} else {
			// 跨越 offset，需要拆分
			left = append(left, &OffsetRange{Start: r.Start, End: offset})
			right = append(right, &OffsetRange{Start: 0, End: r.End - offset})
		}
	}

	return left, right
}

// 同一个utxo中的offset合并
func (p *AssetOffsets) Cat(r2 *OffsetRange) {
	if r2 == nil {
		return
	}
	var r1 *OffsetRange
	len1 := len(*p)
	if len1 > 0 {
		r1 = (*p)[len1-1]
		if r1.End == r2.Start {
			r1.End = r2.End
		} else {
			*p = append(*p, r2)
		}
	} else {
		*p = append(*p, r2)
	}
}

// Insert 将一个新的 OffsetRange 插入到 AssetOffsets 中，保持排序，并合并相邻的区间
func (p *AssetOffsets) Insert(r2 *OffsetRange) {
	// 找到插入的位置
	i := 0
	for i < len(*p) && (*p)[i].End <= r2.Start {
		i++
	}

	// 将新范围插入到合适的位置
	*p = append(*p, nil)       // 扩展切片
	copy((*p)[i+1:], (*p)[i:]) // 将插入位置后的元素向后移动
	(*p)[i] = r2               // 插入新元素

	// 合并相邻的区间
	if i > 0 && (*p)[i-1].End >= (*p)[i].Start { // 如果与前一个区间相邻，合并
		(*p)[i-1].End = max((*p)[i-1].End, (*p)[i].End)
		*p = append((*p)[:i], (*p)[i+1:]...) // 移除合并后的区间
		i--                                  // 退回到合并后的区间
	}
	if i < len(*p)-1 && (*p)[i].End >= (*p)[i+1].Start { // 如果与后一个区间相邻，合并
		(*p)[i].End = max((*p)[i].End, (*p)[i+1].End)
		*p = append((*p)[:i+1], (*p)[i+2:]...) // 移除合并后的区间
	}
}

// another 已经调整过偏移值
func (p *AssetOffsets) Append(another AssetOffsets) {
	var r1, r2 *OffsetRange
	len1 := len(*p)
	len2 := len(another)
	if len1 > 0 {
		if len2 == 0 {
			return
		}
		r1 = (*p)[len1-1]
		r2 = another[0]
		if r1.End == r2.Start {
			r1.End = r2.End
			*p = append(*p, another[1:]...)
		} else {
			*p = append(*p, another...)
		}
	} else {
		*p = append(*p, another...)
	}
}

type TxOutput struct {
	UtxoId      uint64
	OutPointStr string
	OutValue    wire.TxOut
	//Sats        TxRanges  废弃。需要时重新获取
	Assets  TxAssets
	Offsets map[AssetName]AssetOffsets
	// 注意BindingSat属性，TxOutput.OutValue.Value必须大于等于
	// Assets数组中任何一个AssetInfo.BindingSat
}

func NewTxOutput(value int64) *TxOutput {
	return &TxOutput{
		UtxoId:      INVALID_ID,
		OutPointStr: "",
		OutValue:    wire.TxOut{Value: value},
		Assets:      nil,
		Offsets:     make(map[AssetName]AssetOffsets),
	}
}

func (p *TxOutput) Clone() *TxOutput {
	n := &TxOutput{
		UtxoId:      p.UtxoId,
		OutPointStr: p.OutPointStr,
		OutValue:    p.OutValue,
		Assets:      p.Assets.Clone(),
	}

	n.Offsets = make(map[AssetName]AssetOffsets)
	for i, u := range p.Offsets {
		n.Offsets[i] = u.Clone()
	}
	return n
}

func (p *TxOutput) Height() int {
	if p.UtxoId == INVALID_ID {
		return -1
	}
	h, _, _ := FromUtxoId(p.UtxoId)
	return h
}

func (p *TxOutput) Value() int64 {
	return p.OutValue.Value
}

func (p *TxOutput) Zero() bool {
	return p.OutValue.Value == 0 && len(p.Assets) == 0
}

func (p *TxOutput) HasPlainSat() bool {
	if len(p.Assets) == 0 {
		return true
	}
	assetAmt := p.Assets.GetBindingSatAmout()
	return p.OutValue.Value > assetAmt
}

func (p *TxOutput) GetPlainSat() int64 {
	if len(p.Assets) == 0 {
		return p.OutValue.Value
	}
	assetAmt := p.Assets.GetBindingSatAmout()
	return p.OutValue.Value - assetAmt
}

func (p *TxOutput) OutPoint() *wire.OutPoint {
	outpoint, _ := wire.NewOutPointFromString(p.OutPointStr)
	return outpoint
}

func (p *TxOutput) TxOut() *wire.TxOut {
	return &wire.TxOut{
		Value:    p.Value(),
		PkScript: p.OutValue.PkScript,
	}
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
	return p.Assets.GetBindingSatAmout()
}

func (p *TxOutput) Append(another *TxOutput) error {
	if another == nil {
		return nil
	}

	if p.OutValue.Value+another.OutValue.Value < 0 {
		return fmt.Errorf("out of bounds")
	}
	value := p.OutValue.Value
	for _, asset := range another.Assets {
		p.Assets.Add(&asset)

		offsets, ok := another.Offsets[asset.Name]
		if !ok {
			// 非绑定资产没有offset
			continue
		}
		newOffsets := offsets.Clone()
		for j := 0; j < len(newOffsets); j++ {
			newOffsets[j].Start += value
			newOffsets[j].End += value
		}
		existingOffsets, ok := p.Offsets[asset.Name]
		if ok {
			existingOffsets.Append(newOffsets)
		} else {
			existingOffsets = newOffsets
		}
		p.Offsets[asset.Name] = existingOffsets
	}
	p.OutValue.Value += another.OutValue.Value

	p.UtxoId = INVALID_ID
	p.OutPointStr = ""

	return nil
}

// 按照offset将TxOut分割为两个，是Append的反操作
func (p *TxOutput) Cut(offset int64) (*TxOutput, *TxOutput, error) {

	if p.Value() < offset {
		return nil, nil, fmt.Errorf("offset too large")
	}
	if p.Value() == offset {
		return p.Clone(), nil, nil
	}

	var value1, value2 int64
	value1 = offset
	value2 = p.Value() - value1
	part1 := NewTxOutput(value1)
	part2 := NewTxOutput(value2)

	for _, asset := range p.Assets {
		if asset.BindingSat > 0 {
			// cut
			newOffsets := p.Offsets[asset.Name]
			offset1, offset2 := newOffsets.Cut(offset)

			amt1 := offset1.Size() * int64(asset.BindingSat)
			if amt1 > 0 {
				asset1 := AssetInfo{
					Name:       asset.Name,
					Amount:     *NewDefaultDecimal(amt1),
					BindingSat: asset.BindingSat,
				}
				part1.Assets.Add(&asset1)
				part1.Offsets[asset.Name] = offset1
			}

			amt2 := offset2.Size() * int64(asset.BindingSat)
			if amt2 > 0 {
				asset2 := AssetInfo{
					Name:       asset.Name,
					Amount:     *NewDefaultDecimal(amt2),
					BindingSat: asset.BindingSat,
				}
				part2.Assets.Add(&asset2)
				part2.Offsets[asset.Name] = offset2
			}
		} else {
			part1.Assets.Add(&asset)
		}
	}

	return part1, part2, nil
}

// 主网utxo，在处理过程中只允许处理一种资产，所以这里最多只有一种资产
func (p *TxOutput) Split(name *AssetName, value int64, amt *Decimal) (*TxOutput, *TxOutput, error) {

	if value == 0 {
		// 按照资产数量确定value
		if name == nil || *name == ASSET_PLAIN_SAT {
			value = amt.Int64()
			if value < 330 {
				return nil, nil, fmt.Errorf("not allow send %d sats", value)
			}
		} else {
			asset, err := p.Assets.Find(name)
			if err != nil {
				return nil, nil, err
			}
			n := asset.BindingSat
			if n != 0 {
				if amt.Int64()%int64(n) != 0 {
					return nil, nil, fmt.Errorf("amt must be times of %d", n)
				}
				
				offsets := p.Offsets[asset.Name]
				if offsets == nil {
					return nil, nil, fmt.Errorf("can't find offset for asset %s", asset.Name.String())
				}
				tmp := offsets.Clone()
				satsNum := GetBindingSatNum(amt, n)
				offset1, offset2 := tmp.Split(satsNum)
				value = offset1[len(offset1)-1].End
				if value < 330 {
					if len(offset2) == 0 {
						value = 330
					} else {
						if offset2[0].Start + value < 330 {
							return nil, nil, fmt.Errorf("no 330 plain sat, %d", offset2[0].Start + value)
						} else {
							value = 330
						}
					}
				}
			} else {
				value = 330
			}
		}
	}

	if p.Value() < value {
		return nil, nil, fmt.Errorf("output value too small")
	}
	if len(p.Assets) > 1 {
		return nil, nil, fmt.Errorf("only one asset can be processed in mainnet utxo")
	}

	var value1, value2 int64
	value1 = value
	value2 = p.Value() - value1
	part1 := NewTxOutput(value1)
	part2 := NewTxOutput(value2)

	if name == nil || *name == ASSET_PLAIN_SAT {
		if p.Value() < amt.Int64() {
			return nil, nil, fmt.Errorf("amount too large")
		}
		part2.Assets = p.Assets
		for k, v := range p.Offsets {
			_, part2.Offsets[k] = v.Cut(value1)
		}
		return part1, part2, nil
	}

	asset, err := p.Assets.Find(name)
	if err != nil {
		return nil, nil, err
	}
	n := asset.BindingSat
	if n != 0 {
		if amt.Int64()%int64(n) != 0 {
			return nil, nil, fmt.Errorf("amt must be times of %d", n)
		}
		requiredValue := GetBindingSatNum(amt, asset.BindingSat)
		if requiredValue > value {
			return nil, nil, fmt.Errorf("value too small")
		}
	}

	if asset.Amount.Cmp(amt) < 0 {
		return nil, nil, fmt.Errorf("amount too large")
	}
	asset1 := asset.Clone()
	asset1.Amount = *amt.Clone()
	assets2 := p.Assets.Clone()
	assets2.Subtract(asset1)

	part1.Assets = TxAssets{*asset1}
	part2.Assets = assets2

	if !IsBindingSat(name) {
		// runes：no offsets
		part2.Offsets = p.Offsets
		return part1, part2, nil
	}

	offsets, ok := p.Offsets[*name]
	if !ok {
		return nil, nil, fmt.Errorf("can't find asset offset")
	}
	if asset.Amount.Cmp(amt) == 0 {
		part1.Offsets[*name] = offsets.Clone()
		if part2.Value() == 0 {
			part2 = nil
		}
		return part1, part2, nil
	}
	offset1, offset2 := offsets.Split(GetBindingSatNum(amt, n))
	part1.Offsets[*name] = offset1
	part2.Offsets[*name] = offset2

	return part1, part2, nil
}

func (p *TxOutput) GetAssetOffset(name *AssetName, amt *Decimal) (int64, error) {

	if !IsBindingSat(name) {
		return 330, nil
	}

	if IsPlainAsset(name) {
		if p.Value() < amt.Int64() {
			return 0, fmt.Errorf("amount too large")
		}
		return amt.Int64(), nil
	}

	offsets, ok := p.Offsets[*name]
	if !ok {
		return 0, fmt.Errorf("no asset in %s", p.OutPointStr)
	}
	if len(offsets) == 0 {
		return 0, fmt.Errorf("no asset in %s", p.OutPointStr)
	}

	asset, err := p.Assets.Find(name)
	if err != nil {
		return 0, err
	}

	total := asset.Amount
	cmp := amt.Cmp(&total)
	if cmp > 0 {
		return 0, fmt.Errorf("amt too large")
	} else if cmp == 0 {
		return offsets[len(offsets)-1].End, nil
	}

	satsNum := GetBindingSatNum(amt, asset.BindingSat)
	for _, off := range offsets {
		if satsNum > off.End-off.Start {
			satsNum -= off.End - off.Start
		} else if satsNum == off.End-off.Start {
			return off.End, nil
		} else {
			return off.Start + satsNum, nil
		}
	}

	return 0, fmt.Errorf("offsets are wrong")
}

func (p *TxOutput) GetAsset(assetName *AssetName) *Decimal {
	if assetName == nil || *assetName == ASSET_PLAIN_SAT {
		return NewDefaultDecimal(p.GetPlainSat())
	}
	asset, err := p.Assets.Find(assetName)
	if err != nil {
		return nil
	}
	return asset.Amount.Clone()
}

// should fill out Assets parameters.
func GenerateTxOutput(tx *wire.MsgTx, index int) *TxOutput {
	return &TxOutput{
		UtxoId:      INVALID_ID,
		OutPointStr: tx.TxHash().String() + ":" + strconv.Itoa(index),
		OutValue:    *tx.TxOut[index],
		Offsets:     make(map[AssetName]AssetOffsets),
	}
}

func IsNft(assetType string) bool {
	return assetType == ASSET_TYPE_NFT || assetType == ASSET_TYPE_NS
}

func IsPlainAsset(assetName *AssetName) bool {
	if assetName == nil {
		return true
	}
	return ASSET_PLAIN_SAT == *assetName
}

func IsBindingSat(name *AssetName) bool {
	if name == nil {
		return true // ordx asset
	}
	if name.Protocol == PROTOCOL_NAME_ORD ||
		name.Protocol == PROTOCOL_NAME_ORDX ||
		name.Protocol == "" {
		return true
	}
	return false
}

func IsFungibleToken(name *AssetName) bool {
	if name == nil {
		return true
	}

	return name.Type == ASSET_TYPE_FT
}

func IsOrdxFT(name *AssetName) bool {
	if name == nil {
		return false
	}

	return name.Protocol == PROTOCOL_NAME_ORDX && name.Type == ASSET_TYPE_FT
}

// amt的资产需要多少聪
func GetBindingSatNum(amt *Decimal, n uint32) int64 {
	if n == 0 {
		return 0
	}
	return (amt.Int64() + int64(n) - 1) / int64(n)
}

// amt的资产需要多少聪
func GetBindingSatNumV2(amt int64, n uint32) int64 {
	if n == 0 {
		return 0
	}
	return (amt + int64(n) - 1) / int64(n)
}
