package wire

type BitcoinScriptsReq struct {
	Scripts []string `json:"scripts" binding:"required"`
}

type BitcoinOutpointsReq struct {
	Outpoints []string `json:"outpoints" binding:"required"`
}

type BitcoinTxIDsReq struct {
	TxIDs []string `json:"txids" binding:"required"`
}

type BitcoinUTXO struct {
	Outpoint      string `json:"outpoint"`
	Value         int64  `json:"value"`
	PkScript      string `json:"pk_script"`
	Confirmations int64  `json:"confirmations"`
}

type BitcoinScriptUTXOs struct {
	Script string         `json:"script"`
	UTXOs  []*BitcoinUTXO `json:"utxos"`
	Error  string         `json:"error,omitempty"`
}

type BitcoinUTXOsByScriptsResp struct {
	BaseResp
	Data []*BitcoinScriptUTXOs `json:"data"`
}

type BitcoinUTXOStatus struct {
	Outpoint      string `json:"outpoint"`
	Exists        bool   `json:"exists"`
	Unspent       bool   `json:"unspent"`
	Value         int64  `json:"value,omitempty"`
	PkScript      string `json:"pk_script,omitempty"`
	Confirmations int64  `json:"confirmations,omitempty"`
	BlockHash     string `json:"block_hash,omitempty"`
	Error         string `json:"error,omitempty"`
}

type BitcoinUTXOStatusResp struct {
	BaseResp
	Data []*BitcoinUTXOStatus `json:"data"`
}

type BitcoinTxStatus struct {
	TxID          string `json:"txid"`
	Exists        bool   `json:"exists"`
	InMempool     bool   `json:"in_mempool"`
	Confirmed     bool   `json:"confirmed"`
	BlockHeight   int64  `json:"block_height,omitempty"`
	BlockHash     string `json:"block_hash,omitempty"`
	BlockTime     int64  `json:"block_time,omitempty"`
	Confirmations int64  `json:"confirmations,omitempty"`
	Error         string `json:"error,omitempty"`
}

type BitcoinTxStatusResp struct {
	BaseResp
	Data []*BitcoinTxStatus `json:"data"`
}

type BitcoinRawTx struct {
	TxID  string `json:"txid"`
	RawTx string `json:"raw_tx,omitempty"`
	Error string `json:"error,omitempty"`
}

type BitcoinRawTxResp struct {
	BaseResp
	Data []*BitcoinRawTx `json:"data"`
}

type BitcoinOutspend struct {
	Outpoint   string `json:"outpoint"`
	Exists     bool   `json:"exists"`
	Spent      bool   `json:"spent"`
	SpendingTx string `json:"spending_tx,omitempty"`
	Vin        uint32 `json:"vin,omitempty"`
	Error      string `json:"error,omitempty"`
}

type BitcoinOutspendsResp struct {
	BaseResp
	Data []*BitcoinOutspend `json:"data"`
}

type BitcoinBroadcastReq struct {
	RawTx string `json:"raw_tx" binding:"required"`
}

type BitcoinBroadcastResult struct {
	Accepted bool   `json:"accepted"`
	TxID     string `json:"txid,omitempty"`
	Error    string `json:"error,omitempty"`
}

type BitcoinBroadcastResp struct {
	BaseResp
	Data *BitcoinBroadcastResult `json:"data"`
}

type BitcoinTip struct {
	Height    int64  `json:"height"`
	BlockHash string `json:"block_hash"`
	Chainwork string `json:"chainwork"`
}

type BitcoinTipResp struct {
	BaseResp
	Data *BitcoinTip `json:"data"`
}

type BitcoinBlockHeader struct {
	Height            int64  `json:"height"`
	Hash              string `json:"hash"`
	PreviousBlockHash string `json:"previous_block_hash,omitempty"`
	MerkleRoot        string `json:"merkle_root"`
	Time              int64  `json:"time"`
	MedianTime        int64  `json:"median_time"`
	Confirmations     int    `json:"confirmations"`
	Chainwork         string `json:"chainwork"`
}

type BitcoinBlockHeaderResp struct {
	BaseResp
	Data *BitcoinBlockHeader `json:"data"`
}

type BitcoinFeeRate struct {
	Slow   float64 `json:"slow"`
	Normal float64 `json:"normal"`
	Fast   float64 `json:"fast"`
	Unit   string  `json:"unit"`
}

type BitcoinFeeRateResp struct {
	BaseResp
	Data *BitcoinFeeRate `json:"data"`
}
