package wire


type SendRawTxReq struct {
	SignedTxHex string  `json:"signedTxHex" binding:"required"`
	Maxfeerate  float32 `json:"maxfeerate,omitempty"`
}

type SendRawTxResp struct {
	BaseResp
	Data string `json:"data"  example:"ae74538baa914f3799081ba78429d5d84f36a0127438e9f721dff584ac17b346"`
}

type SendRawTxsReq struct {
	SignedTxHex []string `json:"signedTxsHex" binding:"required"`
	Maxfeerate  float32 `json:"maxfeerate,omitempty"`
}

type SendRawTxsResp struct {
	BaseResp
	Data []string `json:"data"`
}

type TestRawTxReq struct {
	SignedTxs []string  `json:"signedTxs" binding:"required"`
}

type TxTestResult struct {
	TxId   string `json:"txid"`
	Allowed bool   `json:"allowed"`
	RejectReason string `json:"reject-reason"`
}

type TestRawTxResp struct {
	BaseResp
	Data []*TxTestResult `json:"data"`
}

type RawBlockResp struct {
	BaseResp
	Data string `json:"data" example:""`
}

type TxResp struct {
	BaseResp
	Data any `json:"data"`
}

type Vin struct {
	Utxo     string `json:"utxo"`
	Address  string `json:"address"`
	Value    int64  `json:"value"`
	Sequence uint32 `json:"sequence"`
}

type Vout struct {
	Address string `json:"address"`
	Value   int64  `json:"value"`
}

type TxInfo struct {
	TxID          string `json:"txid"`
	Version       uint32 `json:"version"`
	Confirmations uint64 `json:"confirmations"`
	BlockHeight   int64  `json:"block_height"`
	BlockTime     int64  `json:"block_time"`
	Vins          []Vin  `json:"vin"`
	Vouts         []Vout `json:"vout"`
}


type TxSimpleInfoResp struct {
	BaseResp
	Data *TxSimpleInfo `json:"data"`
}

type TxSimpleInfo struct {
	TxID          string `json:"txid"`
	Version       uint32 `json:"version"`
	Confirmations uint64 `json:"confirmations"`
	BlockHeight   int64  `json:"block_height"`
	BlockTime     int64  `json:"block_time"`
}

type BlockHashResp struct {
	BaseResp
	Data string `json:"data" example:""`
}

type BestBlockhashResp struct {
	BaseResp
	Data string `json:"data"`
}

type BestBlockHeightResp struct {
	BaseResp
	Data int64 `json:"data"`
}
