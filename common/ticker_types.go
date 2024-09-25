package common



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
	SelfMint   int     `json:"selfmint,omitempty"` // 0-100
	Max        int64   `json:"max,omitempty"`
	BlockStart int     `json:"blockStart,omitempty"`
	BlockEnd   int     `json:"blockEnd,omitempty"`
	Attr       SatAttr `json:"attr,omitempty"`
}

type RBTreeValue_Mint struct {
	InscriptionIds []string // 同一段satrange可以被多次mint，但不会被同一个ticker多次mint，所以这里肯定只有一个，因为该结构仅存在TickInfo中
}

// 仅用于TickInfo内部
type MintAbbrInfo struct {
	Address       uint64
	Amount        int64
	InscriptionId string
	InscriptionNum int64
	Height        int
}

// key: mint时的inscriptionId。 value: 某个资产对应的ranges
type TickAbbrInfo struct {
	MintInfo map[string][]*Range
}

func NewMintAbbrInfo(mint *Mint) *MintAbbrInfo {
	info := NewMintAbbrInfo2(mint.Base)
	info.Amount = mint.Amt
	return info
}

func NewMintAbbrInfo2(base *InscribeBaseContent) *MintAbbrInfo {
	return &MintAbbrInfo{
		Address: base.InscriptionAddress,
		Amount: 1, 
		InscriptionId: base.InscriptionId, 
		InscriptionNum: base.Id,
		Height: int(base.BlockHeight)}
}
