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
	Assets  	TxAssets
	Offsets 	map[AssetName]AssetOffsets
	SatBindingMap map[int64]*AssetInfo // 用于brc20，key是sat的offset, 只有brc20才赋值
	Invalids 	map[AssetName]bool // 表示该Utxo中对应的资产数据只能看，不能用。用于brc20: inscribe-transfer用过后，默认都是有效的
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
		SatBindingMap: make(map[int64]*AssetInfo),
		Invalids: 	make(map[AssetName]bool),
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

	n.SatBindingMap = make(map[int64]*AssetInfo)
	for k, v := range p.SatBindingMap {
		n.SatBindingMap[k] = v.Clone()
	}

	n.Invalids = make(map[AssetName]bool)
	for k, v := range p.Invalids {
		n.Invalids[k] = v
	}

	return n
}

// brc20 专属
func (p *TxOutput) getAssetOffsetMap() map[AssetName][]*OffsetToAmount {
	if p == nil {
		return nil
	}

	assetOffsetMap := make(map[AssetName][]*OffsetToAmount)
	for offset, asset := range p.SatBindingMap {
		o := OffsetToAmount{
			Offset: offset,
			Amount: asset.Amount.String(),
		}
		offsetToAmts := assetOffsetMap[asset.Name]
		assetOffsetMap[asset.Name] = append(offsetToAmts, &o)
	}
	return assetOffsetMap
}

func (p *TxOutput) ToAssetsInUtxo() *AssetsInUtxo{
	if p == nil {
		return nil
	}

	var assets []*DisplayAsset
	if len(p.Assets) != 0 {
		assetOffsetMap := p.getAssetOffsetMap()
		assets = make([]*DisplayAsset, 0)
		for _, asset := range p.Assets {
			display := DisplayAsset{
				AssetName: asset.Name,
				Amount: asset.Amount.String(),
				Precision: asset.Amount.Precision,
				BindingSat: int(asset.BindingSat),
				Offsets: p.Offsets[asset.Name],
				OffsetToAmts: assetOffsetMap[asset.Name],
				Invalid: p.Invalids[asset.Name],
			}
			assets = append(assets, &display)
		}
	}
	return &AssetsInUtxo{
		UtxoId: p.UtxoId,
		OutPoint: p.OutPointStr,
		Value: p.OutValue.Value,
		PkScript: p.OutValue.PkScript,
		Assets: assets,
	}
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
	assetAmt := p.SizeOfBindingSats()
	return p.OutValue.Value > assetAmt
}

// 考虑同一个聪绑定多种资产的情况
func (p *TxOutput) GetPlainSat() int64 {
	if len(p.Assets) == 0 {
		return p.OutValue.Value
	}
	assetAmt := p.SizeOfBindingSats()
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

// 考虑同一个聪绑定多种资产的情况
func (p *TxOutput) SizeOfBindingSats() int64 {
	// 如果是聪网转换过来的，其offset为零，这个时候需要采用assets的结果
	if len(p.Offsets) == 0 {
		return p.Assets.GetBindingSatAmout()
	}
	offset := make(AssetOffsets, 0)
	for _, assetOffset := range p.Offsets {
		for _, off := range assetOffset {
			offset.Insert(off)
		}
	}
	n := int64(len(p.SatBindingMap))
	// 每个brc20的transfer nft，都看作是占用330聪
	
	return offset.Size() - n + n * 330
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
		if invalid, ok := another.Invalids[asset.Name]; ok && invalid {
			continue
		}

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
	for k, v := range another.SatBindingMap {
		p.SatBindingMap[k+value] = v.Clone()
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
		if invalid, ok := p.Invalids[asset.Name]; ok && invalid {
			continue
		}

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
			newOffsets, ok := p.Offsets[asset.Name]
			if ok {
				// brc20 
				offset1, offset2 := newOffsets.Cut(offset)
				satmap1 := make(map[int64]*AssetInfo)
				satmap2 := make(map[int64]*AssetInfo)
				for k, v := range p.SatBindingMap {
					if k < offset {
						satmap1[k] = v.Clone()
					} else {
						satmap2[k-offset] = v.Clone()
					}
				}

				if len(satmap1) > 0 {
					var amt *Decimal
					for _, asset := range satmap1 {
						amt = amt.Add(&asset.Amount)
					}
					asset := AssetInfo{
						Name:       asset.Name,
						Amount:     *amt,
						BindingSat: asset.BindingSat,
					}
					part1.Assets.Add(&asset)
					part1.Offsets[asset.Name] = offset1
					part1.SatBindingMap = satmap1
				}

				if len(satmap2) > 0 {
					var amt *Decimal
					for _, asset := range satmap2 {
						amt = amt.Add(&asset.Amount)
					}
					asset := AssetInfo{
						Name:       asset.Name,
						Amount:     *amt,
						BindingSat: asset.BindingSat,
					}
					part2.Assets.Add(&asset)
					part2.Offsets[asset.Name] = offset2
					part2.SatBindingMap = satmap2
				}
			} else {
				part1.Assets.Add(&asset) // runes
			}
		}
	}

	return part1, part2, nil
}

// 根据value或者amt切分
// 主网utxo，在处理过程中只允许处理一种资产，所以这里最多只有一种资产
func (p *TxOutput) Split(name *AssetName, value int64, amt *Decimal) (*TxOutput, *TxOutput, error) {

	if value == 0 && amt.Sign() == 0 {
		return nil, nil, fmt.Errorf("should provide at least one asset amount")
	}

	if invalid, ok := p.Invalids[*name]; ok && invalid {
		return nil, nil, fmt.Errorf("can't split an invalid asset")
	}

	var offset int64
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
				// ordx
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
				offset = offset1[len(offset1)-1].End
				if offset < 330 {
					if len(offset2) == 0 {
						offset = 330
					} else {
						if offset2[0].Start + offset < 330 {
							return nil, nil, fmt.Errorf("no 330 plain sat, %d", offset2[0].Start + offset)
						} else {
							offset = 330
						}
					}
				}
			} else {
				if len(p.SatBindingMap) == 0 {
					// runes
					offset = 330
				} else {
					// brc20，资产需要跟随transfer铭文走，transfer铭文由Offsets定位，其数量由SatBindingMap确定
					offsets := p.Offsets[asset.Name]
					if offsets == nil {
						return nil, nil, fmt.Errorf("can't find offset for asset %s", asset.Name.String())
					}
					var requiredAmt *Decimal
					for _, off := range offsets {
						info, ok := p.SatBindingMap[off.Start]
						if !ok {
							return nil, nil, fmt.Errorf("can't find sat %d binding map", off.Start)
						}
						requiredAmt = requiredAmt.Add(&info.Amount)
						if requiredAmt.Cmp(amt) >= 0 {
							offset = off.Start + 330 // brc20 transfer 铭文的一般大小
							break
						}
					}
					if requiredAmt.Cmp(amt) != 0 {
						return nil, nil, fmt.Errorf("no accurate asset")
					}
				}
			}
		}
	} else {
		offset = value
	}

	if p.Value() < offset {
		return nil, nil, fmt.Errorf("output value too small")
	}
	if len(p.Assets) > 1 {
		return nil, nil, fmt.Errorf("only one asset can be processed in mainnet utxo")
	}

	part1, part2, err := p.Cut(offset)
	if err != nil {
		return nil, nil, err
	}

	if amt.Sign() != 0 {
		if !IsPlainAsset(name) {
			asset1, err := part1.Assets.Find(name)
			if err != nil {
				return nil, nil, err
			}
			if amt.Cmp(&asset1.Amount) != 0 {
				// 如果是非聪绑定资产，需要对结果微调下
				if asset1.BindingSat == 0 && len(part1.SatBindingMap) == 0 {
					info := AssetInfo{
						Name: *name,
						Amount: *amt,
						BindingSat: asset1.BindingSat,
					}
					part2 = part1.Clone()
					part2.OutValue.Value = 0
					part2.Assets.Subtract(&info)
					part1.Assets = TxAssets{info}
				} else {
					return nil, nil, fmt.Errorf("can't split the accurate asset")
				}
			}
		}
	}
	return part1, part2, nil
}

// 只用于计算ordx资产的偏移，其他资产直接返回0
func (p *TxOutput) GetAssetOffset(name *AssetName, amt *Decimal) (int64, error) {

	if !IsBindingSat(name) {
		return 0, fmt.Errorf("not ordx asset")
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
	if invalid, ok := p.Invalids[*assetName]; ok && invalid {
		return nil
	}
	asset, err := p.Assets.Find(assetName)
	if err != nil {
		return nil
	}
	return asset.Amount.Clone()
}

func (p *TxOutput) GetAssetV2(assetName *AssetName) (*Decimal, bool) {
	if assetName == nil || *assetName == ASSET_PLAIN_SAT {
		return NewDefaultDecimal(p.GetPlainSat()), true
	}
	invalid := p.Invalids[*assetName]
	asset, err := p.Assets.Find(assetName)
	if err != nil {
		return nil, invalid
	}
	return asset.Amount.Clone(), invalid
}

// should fill out Assets parameters.
func GenerateTxOutput(tx *wire.MsgTx, index int) *TxOutput {
	return &TxOutput{
		UtxoId:      INVALID_ID,
		OutPointStr: tx.TxHash().String() + ":" + strconv.Itoa(index),
		OutValue:    *tx.TxOut[index],
		Offsets:     make(map[AssetName]AssetOffsets),
		SatBindingMap: make(map[int64]*AssetInfo),
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

// amt的资产需要多少聪(聪网上，不足一聪的资产，不占用聪)
func GetBindingSatNum(amt *Decimal, n uint32) int64 {
	if n == 0 {
		return 0
	}
	return amt.Int64() / int64(n)
}

// amt的资产需要多少聪
func GetBindingSatNumV2(amt int64, n uint32) int64 {
	if n == 0 {
		return 0
	}
	return amt / int64(n)
}
