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
	if len(parts) != 3 {
		return &AssetName{}
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

// 有序数组，根据名字排序
type TxAssets []AssetInfo

// TxAssetsAppend 合并两个资产列表，返回新的列表
func TxAssetsAppend(a, b *TxAssets) TxAssets {
	if a == nil {
		if b == nil {
			return nil
		}
		return b.Clone()
	}
	result := a.Clone()
	err := result.Merge(b)
	if err != nil {
		return nil
	}
	return result
}

func (p *TxAssets) Clone() TxAssets {
	if p == nil {
		return nil
	}

	newAssets := make(TxAssets, len(*p))
	for i, asset := range *p {
		newAssets[i] = *asset.Clone()
	}

	return newAssets
}

func (p *TxAssets) Sort() {
	sort.Slice(*p, func(i, j int) bool {
		if (*p)[i].Name.Protocol != (*p)[j].Name.Protocol {
			return (*p)[i].Name.Protocol < (*p)[j].Name.Protocol
		}
		if (*p)[i].Name.Type != (*p)[j].Name.Type {
			return (*p)[i].Name.Type < (*p)[j].Name.Type
		}
		return (*p)[i].Name.Ticker < (*p)[j].Name.Ticker
	})
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

func (p *TxAssets) Equal(another *TxAssets) bool {
	if p == nil && another == nil{
		return true
	}
	if len(*p) != len(*another) {
		return false
	}
	
	for i, asset := range *p {
		if !asset.Equal(&(*another)[i]) {
			return false
		}
	}
	return true
}

// 将另一个资产列表合并到当前列表中
func (p *TxAssets) Merge(another *TxAssets) error {
	if another == nil {
		return nil
	}
	cp := p.Clone()
	for _, asset := range *another {
		if err := cp.Add(&asset); err != nil {
			return err
		}
	}
	*p = cp
	return nil
}

// Subtract 从当前列表中减去另一个资产列表
func (p *TxAssets) Split(another *TxAssets) error {
	if another == nil {
		return nil
	}
	cp := p.Clone()
	for _, asset := range *another {
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

func (p *TxAssets) GetBindingSatAmout() int64 {
	amount := int64(0)
	for _, asset := range *p {
		if asset.BindingSat != 0 {
			n := GetBindingSatNum(&asset.Amount, asset.BindingSat)
			if amount < n {
				amount = n
			}
		}
	}
	return amount
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
