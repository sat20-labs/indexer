package common

import (
	"fmt"
)

type BRC20Status struct {
	Version     string
	TickerCount int
}

func (p *BRC20Status) Clone() *BRC20Status {
	return &BRC20Status{
		Version:     p.Version,
		TickerCount: p.TickerCount,
	}
}

type BRC20Mint struct {
	BRC20MintInDB
	Nft *Nft
}

type BRC20MintInDB struct {
	NftId int64
	Id    int64
	Name  string
	Amt   Decimal
}

type BRC20TransferInDB struct {
	NftId int64
	Id    int64
	Name  string
	Amt   Decimal
}

type BRC20Transfer struct {
	BRC20TransferInDB
	Nft *Nft
}

type BRC20Ticker struct {
	Nft  *Nft
	Id   int64
	Name string // 只有这里保留原型
	//saʦ sats

	SelfMint bool    `json:"self_mint,omitempty"`
	Limit    Decimal `json:"limit,omitempty"`
	Max      Decimal `json:"max,omitempty"`

	Decimal uint8 `json:"decimal"`

	DeployTime         int64   `json:"deployTime,omitempty"`
	Minted             Decimal `json:"minted,omitempty"`
	MintCount          uint64  `json:"mintCount,omitempty"`
	StartInscriptionId string  `json:"startInscriptionId,omitempty"`
	EndInscriptionId   string  `json:"endInscriptionId,omitempty"`
	HolderCount        uint64  `json:"holders,omitempty"` // TODO: 要算上处理过的，哪怕最终可用余额是0也要算上
	TransactionCount   uint64  `json:"transactionCount,omitempty"`
}

type BRC20BaseContent struct {
	P      string `json:"p,omitempty"`
	Op     string `json:"op,omitempty"`
	Ticker string `json:"tick"`
}

func (s *BRC20BaseContent) Validate() error {
	if s.Op != "mint" && s.Op != "transfer" && s.Op != "deploy" {
		return fmt.Errorf("miss op")
	}
	if s.P != "brc-20" {
		return fmt.Errorf("invalid protocol: %s", s.P)
	}
	if len(s.Ticker) != 4 && len(s.Ticker) != 5 {
		return fmt.Errorf("invalid ticker: %s", s.Ticker)
	}
	return nil
}

// {"p":"brc-20","op":"deploy","tick":"doɡe","lim":"3125000000000","max":"1000000000000000","self_mint":"true"}
type BRC20DeployContent struct {
	BRC20BaseContent
	Lim      string `json:"lim"`
	Max      string `json:"max"`
	Decimal  string `json:"dec,omitempty"`
	SelfMint string `json:"self_mint,omitempty"`
}

func (s *BRC20DeployContent) Validate() error {
	err := s.BRC20BaseContent.Validate()
	if err != nil {
		return err
	}
	if s.Lim == "" {
		return fmt.Errorf("miss lim")
	}
	if s.Max == "" {
		return fmt.Errorf("miss max")
	}
	// if d.Decimal == "" {
	// 	return fmt.Errorf("miss decimal")
	// }
	// if d.SelfMint == "" {
	// 	return fmt.Errorf("miss self_mint")
	// }
	return nil
}

type BRC20AmtContent struct {
	BRC20BaseContent
	Amt string `json:"amt"`
}

func (s *BRC20AmtContent) Validate() error {
	err := s.BRC20BaseContent.Validate()
	if err != nil {
		return err
	}
	if s.Amt == "" {
		return fmt.Errorf("miss amt")
	}
	return nil
}

// {"p":"brc-20","op":"mint","tick":"wiki","amt":"1000"}
type BRC20MintContent struct {
	BRC20AmtContent
}

// {"p":"brc-20","op":"transfer","tick":"XXOK","amt":"89000000000"}
type BRC20TransferContent struct {
	BRC20AmtContent
}

const (
	BRC20_Action_InScribe_Deploy int = iota
	BRC20_Action_InScribe_Mint
	BRC20_Action_InScribe_Transfer // 铸造一个transfer铭文
	BRC20_Action_Transfer          // 转移一个transfer铭文
	BRC20_Action_Transfer_Spent    // 一个已经转移的transfer铭文被花费
	BRC20_Action_Transfer_Canceled // 销毁一个transfer铭文
)

type BRC20ActionHistory struct {
	Height   int
	Action   int
	NftId    int64 // transfer nft
	Ticker   string
	Amount   Decimal

	FromUtxoId uint64
	FromAddr   uint64
	ToUtxoId   uint64
	ToAddr     uint64
}

type BRC20MintAbbrInfo struct {
	Id             int64
	Address        uint64
	Amount         Decimal
	InscriptionId  string
	InscriptionNum int64
	Height         int
}

type TransferNFT struct {
	NftId     int64
	Id        int64  // transfer id
	UtxoId    uint64 // 铸造时的utxoId
	Amount    Decimal
	IsInvalid bool
}

func (p *TransferNFT) Clone() *TransferNFT {
	return &TransferNFT{
		NftId: p.NftId,
		Id: p.Id,
		UtxoId: p.UtxoId,
		Amount: *p.Amount.Clone(),
		IsInvalid: p.IsInvalid,
	}
}

// key: mint时的inscriptionId。 value: 某个资产对应的数值
type BRC20TickAbbrInfo struct {
	AvailableBalance    *Decimal
	TransferableBalance *Decimal
	TransferableData    map[uint64]*TransferNFT // key:utxoId
}

func (p *BRC20TickAbbrInfo) AssetAmt() *Decimal {
	return DecimalAdd(p.AvailableBalance, p.TransferableBalance)
}

func (p *BRC20TickAbbrInfo) Equal(that *BRC20TickAbbrInfo) bool {
	if p == nil && that == nil {
		return true
	}
	if p == nil {
		return false
	}
	if that == nil {
		return false
	}
	if p.AvailableBalance.Cmp(that.AvailableBalance) != 0 {
		return false
	}
	if p.TransferableBalance.Cmp(that.TransferableBalance) != 0 {
		return false
	}
	if len(p.TransferableData) != len(that.TransferableData) {
		return false
	}
	for utxoId := range that.TransferableData {
		_, ok := p.TransferableData[utxoId]
		if !ok {
			return false
		}
	}
	return true
}

func NewBRC20TickAbbrInfo(availableAmt, transferableAmt *Decimal) *BRC20TickAbbrInfo {
	return &BRC20TickAbbrInfo{
		AvailableBalance:    availableAmt.Clone(),
		TransferableBalance: transferableAmt.Clone(),
		TransferableData:    make(map[uint64]*TransferNFT),
		// InvalidTransferableData: make(map[uint64]*TransferNFT),
	}
}

func NewBRC20MintAbbrInfo(mint *BRC20Mint) *BRC20MintAbbrInfo {
	info := NewBRC20MintAbbrInfo2(mint.Nft.Base)
	info.Id = mint.Id
	info.Amount = mint.Amt
	return info
}

func NewBRC20MintAbbrInfo2(base *InscribeBaseContent) *BRC20MintAbbrInfo {
	return &BRC20MintAbbrInfo{
		Address: base.InscriptionAddress,
		//Amount: 1,
		InscriptionId:  base.InscriptionId,
		InscriptionNum: base.Id,
		Height:         int(base.BlockHeight)}
}

func (p *BRC20MintAbbrInfo) ToMintInfo() *MintInfo {
	return &MintInfo{
		Id: p.Id,
		//Address: p.Address,
		Amount:         p.Amount.ToFormatString(),
		Ordinals:       nil,
		Height:         p.Height,
		InscriptionId:  p.InscriptionId,
		InscriptionNum: p.InscriptionNum,
	}
}

func (p *BRC20MintAbbrInfo) ToMintAbbrInfo() *MintAbbrInfo {
	return &MintAbbrInfo{
		Id:             p.Id,
		Address:        p.Address,
		Amount:         p.Amount.Clone(),
		Ordinals:       nil,
		Height:         p.Height,
		InscriptionId:  p.InscriptionId,
		InscriptionNum: p.InscriptionNum,
	}
}

type BRC20TransferInfo struct {
	NftId   int64    `json:"nftId"`
	Name    string   `json:"name"`
	Amt     *Decimal `json:"amt"`
	Invalid bool     `json:"invalid"`
}
