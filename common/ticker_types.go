package common

import "github.com/btcsuite/btcd/wire"

const (
	TICKER_STATUS_INVALID 	int = -1
	TICKER_STATUS_INIT 		int = 0
	TICKER_STATUS_NOT_START int = 1
	TICKER_STATUS_MINTING 	int = 2
	TICKER_STATUS_MINT_COMPLETED int = 3
)

const TickerSeparatedFromName = true

// amt / n == sizeof(ordinals)
type Mint struct {
	Base     *InscribeBaseContent
	Id       int64
	Name     string  
	Amt int64 `json:"amt"`

	Ordinals []*Range `json:"ordinals"`
	Desc     string   `json:"desc,omitempty"`
}

type Ticker struct {
	Base     *InscribeBaseContent
	Id       int64
	Name     string  
	Desc     string   `json:"desc,omitempty"`

	Type       string  `json:"type,omitempty"` // 默认是FT，留待以后扩展
	Limit      int64   `json:"limit,omitempty"`
	N          int     `json:"n,omitempty"`
	SelfMint   int     `json:"selfmint,omitempty"` // 0-100
	Max        int64   `json:"max,omitempty"`
	BlockStart int     `json:"blockStart,omitempty"`
	BlockEnd   int     `json:"blockEnd,omitempty"`
	Attr       SatAttr `json:"attr,omitempty"`
	Status     int     `json:"status"`
}

type RBTreeValue_Mint struct {
	InscriptionIds []string // 同一段satrange可以被多次mint，但不会被同一个ticker多次mint，所以这里肯定只有一个，因为该结构仅存在TickInfo中
}

// 仅用于TickInfo内部
type MintAbbrInfo struct {
	Id            int64
	Address       uint64
	Amount        *Decimal
	Ordinals      []*Range
	InscriptionId string
	InscriptionNum int64
	Height        int
}

// key: mint时的inscriptionId。 value: 某个资产对应的ranges
type TickAbbrInfo struct {
	N int
	MintInfo map[string][]*Range
}

func NewMintAbbrInfo(mint *Mint) *MintAbbrInfo {
	info := NewMintAbbrInfo2(mint.Base)
	info.Id = mint.Id
	info.Amount = NewDefaultDecimal(mint.Amt)
	info.Ordinals = mint.Ordinals
	return info
}

func (p *MintAbbrInfo) ToMintInfo() *MintInfo {
	return &MintInfo{
		Id: p.Id,
		//Address: p.Address,
		Amount: p.Amount.String(),
		Ordinals: p.Ordinals,
		Height: p.Height,
		InscriptionId: p.InscriptionId,
		InscriptionNum: p.InscriptionNum,
	}
}

func NewMintAbbrInfo2(base *InscribeBaseContent) *MintAbbrInfo {
	return &MintAbbrInfo{
		Address: base.InscriptionAddress,
		Amount: NewDefaultDecimal(1), 
		InscriptionId: base.InscriptionId, 
		InscriptionNum: base.Id,
		Height: int(base.BlockHeight)}
}

///////////////////////////////////////////////////
// 用于展示统一的数据信息

type TickerInfo struct {
	AssetName        `json:"name"`
	DisplayName     string `json:"displayname"`
	Id 				int64  `json:"id"`
	Divisibility 	int	   `json:"divisibility,omitempty"`
	StartBlock      int    `json:"startBlock,omitempty"`
	EndBlock        int    `json:"endBlock,omitempty"`
	SelfMint        int    `json:"selfmint,omitempty"`
	DeployHeight    int    `json:"deployHeight"`
	DeployBlocktime int64  `json:"deployBlockTime"`
	DeployTx        string `json:"deployTx"`
	Limit           string `json:"limit"`
	N               int    `json:"n"`
	TotalMinted     string `json:"totalMinted"`
	MintTimes       int64  `json:"mintTimes"`
	MaxSupply       string `json:"maxSupply,omitempty"`
	HoldersCount    int    `json:"holdersCount"`
	InscriptionId   string `json:"inscriptionId,omitempty"`
	InscriptionNum  int64  `json:"inscriptionNum,omitempty"`
	Description     string `json:"description,omitempty"`
	Rarity          string `json:"rarity,omitempty"`
	DeployAddress   string `json:"deployAddress,omitempty"`
	Content         []byte `json:"content,omitempty"`
	ContentType     string `json:"contenttype,omitempty"`
	Delegate        string `json:"delegate,omitempty"`
	Status          int    `json:"status"`
}

type MintInfo struct {
	Id             int64  `json:"id"`  // ticker内的铸造序号，非全局
	Address        string `json:"mintaddress"`
	Amount         string `json:"amount"`
	Ordinals       []*Range `json:"ordinals,omitempty"`
	Height         int    `json:"height"`
	InscriptionId  string `json:"inscriptionId,omitempty"`  // 铭文id，或者符文的铸造输出utxo
	InscriptionNum int64  `json:"inscriptionNumber,omitempty"`
}

type MintHistory struct {
	AssetName        `json:"Name"`
	Total    int           `json:"Total,omitempty"`
	Start    int           `json:"Start,omitempty"`
	Limit    int           `json:"Limit,omitempty"`
	Items    []*MintInfo   `json:"Items,omitempty"`
}

type OffsetToAmount struct {
	Offset int64	`json:"Offset"`
	Amount string 	`json:"Amount"`
}

type DisplayAsset struct {
	AssetName             `json:"Name"`
	Amount  string        `json:"Amount"`
	Precision int         `json:"Precision"`
	BindingSat int        `json:"BindingSat"`

	// 以下仅用在主网上，聪网不涉及
	Offsets []*OffsetRange `json:"Offsets,omitempty"`
	OffsetToAmts []*OffsetToAmount `json:"OffsetToAmts,omitempty"` // brc20 transfer nft, offset->decimal
	Invalid     bool      `json:"invalid,omitempty"` // 表示该Utxo的资产数据只能看，不能用。用于brc20: inscribe-transfer用过后
}

func (p *DisplayAsset) ToAssetInfo() *AssetInfo {
	amount, err := NewDecimalFromString(p.Amount, p.Precision)
	if err != nil {
		// should not happen
		return nil
	}
	return &AssetInfo{
		Name: p.AssetName,
		Amount: *amount,
		BindingSat: uint32(p.BindingSat),
	}
}

type AssetsInUtxo struct {
	UtxoId      uint64          `json:"UtxoId"`
	OutPoint    string     		`json:"Outpoint"` // tx:vout
	Value       int64      		`json:"Value"`
	PkScript    []byte      	`json:"PkScript"`
	Assets  	[]*DisplayAsset `json:"Assets"`
}

func (p *AssetsInUtxo) ToTxAssets() TxAssets {
	assets := make(TxAssets, 0, len(p.Assets))
	for _, asset := range p.Assets {
		assets = append(assets, *asset.ToAssetInfo())
	}
	return assets
}

func (p *AssetsInUtxo) ToTxOutput() *TxOutput {
	if p == nil {
		return nil
	}

	offsets := make(map[AssetName]AssetOffsets)
	satBindingMap := make(map[int64]*AssetInfo)
	invalids := make(map[AssetName]bool)

	var assets TxAssets
	if len(p.Assets) != 0 {
		assets = make(TxAssets, 0, len(p.Assets))
		for _, v := range p.Assets {
			assetInfo := v.ToAssetInfo()
			assets = append(assets, *assetInfo)
			if len(v.Offsets) != 0 {
				offsets[v.AssetName] = v.Offsets
			}
			for _, offset := range v.OffsetToAmts {
				piece := assetInfo.Clone()
				amt, err := NewDecimalFromString(offset.Amount, assetInfo.Amount.Precision)
				if err != nil {
					continue
				}
				piece.Amount = *amt
				satBindingMap[offset.Offset] = piece
			}
			if v.Invalid {
				invalids[v.AssetName] = v.Invalid
			}
		}
	}
	return &TxOutput{
		UtxoId: p.UtxoId,
		OutPointStr: p.OutPoint,
		OutValue: wire.TxOut{
			Value: p.Value,
			PkScript: p.PkScript,
		},
		Assets: assets,
		Offsets: offsets,
		SatBindingMap: satBindingMap,
		Invalids: invalids,
	}
}
