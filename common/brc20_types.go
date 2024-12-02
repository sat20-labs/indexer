package common



type Brc20Mint struct {
	Base     *InscribeBaseContent
	Id       int64
	Name     string  
	Amt      *Decimal `json:"amt"`

	Satoshi int64 `json:"satoshi"`
}

type Brc20Transfer struct {
	Base     *InscribeBaseContent
	Id       int64
	Name     string  
	Amt 	 *Decimal `json:"amt"`
}

type Brc20Ticker struct {
	Base     *InscribeBaseContent
	Id       int64
	Name     string  

	SelfMint   bool     `json:"self_mint,omitempty"`
	Limit      *Decimal   `json:"limit,omitempty"`
	Max        *Decimal   `json:"max,omitempty"`

	Decimal   uint8  `json:"-"`
}


type Brc20BaseContent struct {
	OrdxBaseContent
	Ticker string `json:"tick"`
}

//{"p":"brc-20","op":"deploy","tick":"doɡe","lim":"3125000000000","max":"1000000000000000","self_mint":"true"}
type Brc20DeployContent struct {
	Brc20BaseContent
	Lim      string `json:"lim"`
	Max      string `json:"max"`
	Decimal  string `json:"dec,omitempty"`
	SelfMint string `json:"self_mint,omitempty"`
}

// {"p":"brc-20","op":"mint","tick":"wiki","amt":"1000"}
type Brc20MintContent struct {
	Brc20BaseContent
	Amt    string `json:"amt"`
}

//{"p":"brc-20","op":"transfer","tick":"XXOK","amt":"89000000000"}
type Brc20TransferContent struct {
	Brc20BaseContent
	Amt    string `json:"amt"`
}

