package common

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// //////////////////////////////////////////////////////////////
// 定义在聪网中
type AssetName struct {
	Protocol string // 必填，比如ordx, ordinals, brc20，runes，eth，等等
	Type     string // 可选，默认是ft，参考indexer的定义
	Ticker   string // 如果Type是nft类型，ticker是合集名称#铭文序号（或者聪序号）
}

func NewAssetNameFromString(name string) *AssetName {
	if name == "" {
		return &AssetName{}
	}
	parts := strings.Split(name, ":")
	if len(parts) == 0 {
		return &AssetName{}
	}
	if len(parts) == 1 {
		return &AssetName{
			Protocol: PROTOCOL_NAME_ORDX,
			Type: ASSET_TYPE_FT,
			Ticker: parts[0],
		}
	}
	if len(parts) == 2 {
		return &AssetName{
			Protocol: parts[0],
			Type: ASSET_TYPE_FT,
			Ticker: parts[1],
		}
	}
	if len(parts) == 4 {
		// runes
		return &AssetName{
			Protocol: parts[0],
			Type: parts[1],
			Ticker: parts[2]+":"+parts[3],
		}
	}
	return &AssetName{
		Protocol: parts[0],
		Type: parts[1],
		Ticker: parts[2],
	}
}

func (p *AssetName) String() string {
	return p.Protocol + ":" + p.Type + ":" + p.Ticker
}

type AssetInfo struct {
	Name       AssetName
	Amount     Decimal  // 资产数量
	BindingSat uint32 // 非0 -> 每一聪绑定的资产的数量, 0 -> 不绑定聪
}

func (p *AssetInfo) Add(another *AssetInfo) error {
	if p.Name == another.Name {
		p.Amount = *p.Amount.Add(&another.Amount)
	} else {
		return fmt.Errorf("not the same asset")
	}
	return nil
}

func (p *AssetInfo) Subtract(another *AssetInfo) error {
	if p.Name == another.Name {
		if p.Amount.Cmp(&another.Amount) < 0 {
			return fmt.Errorf("not enough asset to subtract")
		}
		p.Amount = *p.Amount.Sub(&another.Amount)
	} else {
		return fmt.Errorf("not the same asset")
	}
	return nil
}

func (p *AssetInfo) Clone() *AssetInfo {
	if p == nil {
		return nil
	}
	return &AssetInfo{
		Name:       p.Name,
		Amount:     *p.Amount.Clone(),
		BindingSat: p.BindingSat,
	}
}

func (p *AssetInfo) Equal(another *AssetInfo) bool {
	if another == nil {
		return false
	}
	return p.Name == another.Name && p.Amount.Cmp(&another.Amount) == 0 &&
		p.BindingSat == another.BindingSat
}

func (p *AssetInfo) GetBindingSatNum() int64 {
	return GetBindingSatNum(&p.Amount, p.BindingSat)
}

// 有序数组，根据名字排序
type TxAssets []AssetInfo


func (p *TxAssets) Clone() TxAssets {
	if p == nil || len(*p) == 0 {
		return nil
	}

	newAssets := make(TxAssets, len(*p))
	for i, asset := range *p {
		newAssets[i] = *asset.Clone()
	}

	return newAssets
}
// Binary search to find the index of an AssetName
func (p *TxAssets) findIndex(name *AssetName) (int, bool) {
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

func (p *TxAssets) Equal(another TxAssets) bool {
	if p == nil && another == nil{
		return true
	}
	if len(*p) != len(another) {
		return false
	}
	
	for i, asset := range *p {
		if !asset.Equal(&(another)[i]) {
			return false
		}
	}
	return true
}

// 将另一个资产列表合并到当前列表中
func (p *TxAssets) Merge(another TxAssets) error {
	if another == nil {
		return nil
	}
	cp := p.Clone()
	for _, asset := range another {
		if err := cp.Add(&asset); err != nil {
			return err
		}
	}
	*p = cp
	return nil
}

// Subtract 从当前列表中减去另一个资产列表
func (p *TxAssets) Split(another TxAssets) error {
	if another == nil {
		return nil
	}
	cp := p.Clone()
	for _, asset := range another {
		if err := cp.Subtract(&asset); err != nil {
			return err
		}
	}
	*p = cp
	return nil
}

// Add 将另一个资产列表合并到当前列表中
func (p *TxAssets) Add(asset *AssetInfo) error {
	if asset == nil {
		return nil
	}
	index, found := p.findIndex(&asset.Name)
	if found {
		(*p)[index].Amount = *(*p)[index].Amount.Add(asset.Amount.Clone())
	} else {
		*p = append(*p, AssetInfo{}) // Extend slice
		copy((*p)[index+1:], (*p)[index:])
		(*p)[index] = *asset.Clone()
	}
	return nil
}

// Subtract 从当前列表中减去另一个资产列表
func (p *TxAssets) Subtract(asset *AssetInfo) error {
	if asset == nil {
		return nil
	}
	if asset.Amount.IsZero() {
		return nil
	}

	index, found := p.findIndex(&asset.Name)
	if !found {
		return errors.New("asset not found")
	}
	if (*p)[index].Amount.Cmp(&asset.Amount) < 0 {
		return errors.New("insufficient asset amount")
	}
	(*p)[index].Amount = *(*p)[index].Amount.Sub(&asset.Amount)
	if (*p)[index].Amount.IsZero() {
		*p = append((*p)[:index], (*p)[index+1:]...)
	}
	return nil
}

// PickUp 从资产列表中提取指定名称和数量的资产，原资产不改变
func (p *TxAssets) PickUp(asset *AssetName, amt *Decimal) (*AssetInfo, error) {
	if asset == nil {
		return nil, fmt.Errorf("need a specific asset")
	}
	index, found := p.findIndex(asset)
	if !found {
		return nil, errors.New("asset not found")
	}
	if (*p)[index].Amount.Cmp(amt) < 0 {
		return nil, errors.New("insufficient asset amount")
	}
	
	picked := AssetInfo{Name: *asset, Amount: *amt, BindingSat: (*p)[index].BindingSat}
	return &picked, nil
}

func (p *TxAssets) Find(asset *AssetName) (*AssetInfo, error) {
	index, found := p.findIndex(asset)
	if !found {
		return nil, errors.New("asset not found")
	}
	return &(*p)[index], nil
}

// 这里必须假定每个聪都是单独染色，这个计算只适合聪网，聪网确保每一个聪只代表一种资产，跟btc主网不同
// 不足一聪的，不占聪，只适合聪网。主网的每一聪，都是完整的资产绑定，不可能缺少一部分资产。
func (p *TxAssets) GetBindingSatAmout() int64 {
	if p == nil {
		return 0
	}
	amount := int64(0)
	for _, asset := range *p {
		if asset.BindingSat != 0 {
			amount += GetBindingSatNum(&asset.Amount, asset.BindingSat)
		}
	}
	return amount
}

// 是否存在一些n>0并且没有对应聪的资产
func (p *TxAssets) GetUnboundAssetCount() int {
	if p == nil {
		return 0
	}
	c := 0
	for _, asset := range *p {
		if asset.BindingSat != 0 &&
		asset.Amount.Int64()%int64(asset.BindingSat) != 0 {
			c++
		}
	}
	return c
}

func (p *TxAssets) IsZero() bool {
	if len(*p) == 0 {
		return true
	}
	for _, asset := range *p {
		if !asset.Amount.IsZero() {
			return false
		}
	}
	return true
}

// 应对同时Add多个资产数据的方案
type TxAssetsBuilder struct {
    m map[AssetName]*AssetInfo
}

func NewTxAssetsBuilder(capHint int) *TxAssetsBuilder {
    return &TxAssetsBuilder{
        m: make(map[AssetName]*AssetInfo, capHint),
    }
}

// 不分配内存，直接使用asset
func (b *TxAssetsBuilder) Add(asset *AssetInfo) {
    if asset == nil {
        return
    }

    if exist, ok := b.m[asset.Name]; ok {
        exist.Amount.AddInPlace(&asset.Amount)
        return
    }

    // 关键点：
    // 直接接管 asset，不 Clone，不复制 big.Int
    b.m[asset.Name] = asset
}

// 分配内存版本
func (b *TxAssetsBuilder) AddClone(asset *AssetInfo) {
    if asset == nil {
        return
    }

    if exist, ok := b.m[asset.Name]; ok {
        exist.Amount.AddInPlace(&asset.Amount)
        return
    }

    cloned := *asset
    cloned.Amount = *asset.Amount.Clone()
    b.m[asset.Name] = &cloned
}

func (b *TxAssetsBuilder) Build() TxAssets {
    if len(b.m) == 0 {
        return nil
    }

    res := make(TxAssets, 0, len(b.m))
    for _, asset := range b.m {
        res = append(res, *asset)
    }

    sort.Slice(res, func(i, j int) bool {
        ai := res[i].Name
        aj := res[j].Name

        if ai.Protocol != aj.Protocol {
            return ai.Protocol < aj.Protocol
        }
        if ai.Type != aj.Type {
            return ai.Type < aj.Type
        }
        return ai.Ticker < aj.Ticker
    })

    return res
}

