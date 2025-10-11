package common

type BRC20Mint struct {
	Nft  *Nft
	Id   int64
	Name string
	Amt  Decimal `json:"amt"`

	Satoshi int64 `json:"satoshi"`
}

type BRC20Transfer struct {
	Nft *Nft
	// UtxoId uint64
	Name string
	Amt  Decimal `json:"amt"`
}

type BRC20Ticker struct {
	Nft  *Nft
	Id   int64
	Name string
	//saʦ sats

	SelfMint bool    `json:"self_mint,omitempty"`
	Limit    Decimal `json:"limit,omitempty"`
	Max      Decimal `json:"max,omitempty"`

	Decimal uint8 `json:"-"`

	DeployTime         int64   `json:"deployTime,omitempty"`
	Minted             Decimal `json:"minted,omitempty"`
	StartInscriptionId string  `json:"startInscriptionId,omitempty"`
	EndInscriptionId   string  `json:"endInscriptionId,omitempty"`
	HolderCount        uint64  `json:"holders,omitempty"`
	TransactionCount   uint64  `json:"transactionCount,omitempty"`
}

type BRC20BaseContent struct {
	OrdxBaseContent
	Ticker string `json:"tick"`
}

// {"p":"brc-20","op":"deploy","tick":"doɡe","lim":"3125000000000","max":"1000000000000000","self_mint":"true"}
type BRC20DeployContent struct {
	BRC20BaseContent
	Lim      string `json:"lim"`
	Max      string `json:"max"`
	Decimal  string `json:"dec,omitempty"`
	SelfMint string `json:"self_mint,omitempty"`
}

// {"p":"brc-20","op":"mint","tick":"wiki","amt":"1000"}
type BRC20MintContent struct {
	BRC20BaseContent
	Amt string `json:"amt"`
}

// {"p":"brc-20","op":"transfer","tick":"XXOK","amt":"89000000000"}
type BRC20TransferContent struct {
	BRC20BaseContent
	Amt string `json:"amt"`
}

type BRC20TransferHistory struct {
	Height int
	// Utxo   string // transferring utxo
	UtxoId uint64
	NftId  int64 // transfer nft

	FromAddr uint64
	ToAddr   uint64

	Ticker string
	Amount string
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
	NftId  int64
	UtxoId uint64
	Amount Decimal
}

// key: mint时的inscriptionId。 value: 某个资产对应的数值
type BRC20TickAbbrInfo struct {
	Balance Decimal
	// AvailableBalance Decimal
	// TransferableBalance Decimal
	TransferableData map[uint64]*TransferNFT // key:utxoId
	// InvalidTransferableData map[uint64]*TransferNFT // key:utxoId
}

func NewBRC20TickAbbrInfo(amt Decimal) *BRC20TickAbbrInfo {
	// balance := amt.Clone()
	// balance.SetValue(0)
	return &BRC20TickAbbrInfo{
		Balance: amt,
		// AvailableBalance:   amt,
		// TransferableBalance: *balance,
		TransferableData: make(map[uint64]*TransferNFT),
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

type BRC20TransferInfo struct {
	InscriptionId string `json:"inscriptionId"`
	Name          string `json:"name"`
	Amt           string `json:"amt"`
}
