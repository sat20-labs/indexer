package common

type BRC20Mint struct {
	Base *InscribeBaseContent
	Id   int64
	Name string
	Amt  Decimal `json:"amt"`

	Satoshi int64 `json:"satoshi"`
}

type BRC20Transfer struct {
	Base *InscribeBaseContent
	Id   int64
	Name string
	Amt  Decimal `json:"amt"`
}

type BRC20Ticker struct {
	Base *InscribeBaseContent
	Id   int64
	Name string

	SelfMint bool     `json:"self_mint,omitempty"`
	Limit    Decimal `json:"limit,omitempty"`
	Max      Decimal `json:"max,omitempty"`

	Decimal uint8 `json:"-"`
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

type BRC20HistoryBase struct {
	Type uint8 // inscribe-deploy/inscribe-mint/inscribe-transfer/transfer/send/receive

	TxId   string
	Idx    uint32
	Vout   uint32
	Offset uint64

	PkScriptFrom string
	PkScriptTo   string
	Satoshi      uint64

	Height uint32
}

// history
type BRC20History struct {
	BRC20HistoryBase

	Inscription *InscribeBaseContent

	// param
	Amount string

	// state
	OverallBalance      string
	TransferableBalance string
	AvailableBalance    string
}

type BRC20MintAbbrInfo struct {
	Address        uint64
	Amount         Decimal
	InscriptionId  string
	InscriptionNum int64
	Height         int
}

// key: mint时的inscriptionId。 value: 某个资产对应的数值
type BRC20TickAbbrInfo struct {
	AvailableBalance     Decimal
	TransferableBalance  Decimal
}


func NewBRC20MintAbbrInfo(mint *BRC20Mint) *BRC20MintAbbrInfo {
	info := NewBRC20MintAbbrInfo2(mint.Base)
	info.Amount = mint.Amt
	return info
}


func NewBRC20MintAbbrInfo2(base *InscribeBaseContent) *BRC20MintAbbrInfo {
	return &BRC20MintAbbrInfo{
		Address: base.InscriptionAddress,
		//Amount: 1, 
		InscriptionId: base.InscriptionId, 
		InscriptionNum: base.Id,
		Height: int(base.BlockHeight)}
}
