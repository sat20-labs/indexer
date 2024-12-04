package extension

import rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"

type Terms struct {
	Amount      string `json:"amount"`
	Cap         string `json:"cap"`
	HeightStart int    `json:"heightStart"`
	HeightEnd   int    `json:"heightEnd"`
	OffsetStart int    `json:"offsetStart"`
	OffsetEnd   int    `json:"offsetEnd"`
}

type RuneInfo struct {
	RuneId       string `json:"runeId"`
	Rune         string `json:"rune"`
	SpacedRune   string `json:"spacedRune"`
	Number       int    `json:"number"`
	Height       int    `json:"height"`
	TxIdx        int    `json:"txIdx"`
	Timestamp    int    `json:"timestamp"`
	Divisibility int    `json:"divisibility"`
	Symbol       string `json:"symbol"`
	Etching      string `json:"etching"`
	Premine      string `json:"premine"`
	Terms        Terms  `json:"terms"`
	Mints        string `json:"mints"`
	Burned       string `json:"burned"`
	Holders      int    `json:"holders"`
	Transactions int    `json:"transactions"`
	Mintable     bool   `json:"mintable"`
	Remaining    string `json:"remaining"`
	Start        int    `json:"start"`
	End          int    `json:"end"`
	Supply       string `json:"supply"`
	Parent       string `json:"parent,omitempty"`
}

// /runes/list
type RuneBalance struct {
	Amount       string `json:"amount"`
	RuneId       string `json:"runeId"`
	Rune         string `json:"rune"`
	SpacedRune   string `json:"spacedRune"`
	Symbol       string `json:"symbol"`
	Divisibility int    `json:"divisibility"`
}

type RuneBalanceList struct {
	rpcwire.ListResp
	List []*RuneBalance `json:"list"`
}

type RunesListResp struct {
	rpcwire.BaseResp
	Data *RuneBalanceList `json:"data"`
}

// /runes/utxos
type RuneUtxosResp struct {
	rpcwire.BaseResp
	Data *RuneBalance `json:"data"`
}

// /runes/token-summary
type AddressRunesTokenSummary struct {
	RuneInfo    *RuneInfo
	RuneBalance *RuneBalance
	RuneLogo    *Inscription
}
type AddressRunesTokenSummaryResp struct {
	rpcwire.BaseResp
	Data *AddressRunesTokenSummary `json:"data"`
}
