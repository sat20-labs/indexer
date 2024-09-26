package extension

// import (
// 	serverCommon "github.com/sat20-labs/indexer/server/define"
// )

// // /brc20/list
// // /brc20/5byte-list
// type TokenBalance struct {
// 	AvailableBalance       string `json:"availableBalance"`
// 	OverallBalance         string `json:"overallBalance"`
// 	Ticker                 string `json:"ticker"`
// 	TransferableBalance    string `json:"transferableBalance"`
// 	AvailableBalanceSafe   string `json:"availableBalanceSafe"`
// 	AvailableBalanceUnSafe string `json:"availableBalanceUnSafe"`
// }
// type Brc20List struct {
// 	Total int             `json:"total"`
// 	List  []*TokenBalance `json:"list"`
// }

// type Brc20ListResp struct {
// 	serverCommon.BaseResp
// 	Data *Brc20List `json:"data"`
// }

// // /brc20/inscribe-transfer
// type Brc20InscribeTransferReq struct {
// 	Address     string `json:"address"`
// 	Tick        string `json:"tick"`
// 	Amount      string `json:"amount"`
// 	FeeRate     string `json:"fee_rate"`
// 	OutputValue uint64 `json:"output_value"`
// }

// type Brc20InscribeTransferData struct {
// 	OrderId          string `json:"order_id"`
// 	PayAddress       string `json:"pay_address"`
// 	TotalFee         uint64 `json:"total_fee"`
// 	MinerFee         uint64 `json:"miner_fee"`
// 	OriginServiceFee uint64 `json:"origin_service_fee"`
// 	ServiceFee       uint64 `json:"service_fee"`
// 	OutputValue      uint64 `json:"output_value"`
// }

// type Brc20InscribeTransferResp struct {
// 	serverCommon.BaseResp
// 	Data *Brc20InscribeTransferData `json:"data"`
// }

// // /brc20/order-result
// type TokenTransfer struct {
// 	Ticker            string `json:"ticker"`
// 	Amount            string `json:"amount"`
// 	InscriptionId     string `json:"inscriptionId"`
// 	InscriptionNumber uint64 `json:"inscriptionNumber"`
// 	Timestamp         uint64 `json:"timestamp"`
// }

// type Brc20OrderResp struct {
// 	serverCommon.BaseResp
// 	Data *TokenTransfer `json:"data"`
// }

// // /brc20/5byte-list
// type TokenInfo struct {
// 	TotalSupply   string `json:"totalSupply"`
// 	TotalMinted   string `json:"totalMinted"`
// 	Decimal       int    `json:"decimal"`
// 	Holder        string `json:"holder"`
// 	InscriptionId string `json:"inscriptionId"`
// }

// // /brc20/token-summary
// type AddressTokenSummary struct {
// 	TokenInfo        *TokenInfo       `json:"tokenInfo"`
// 	TokenBalance     *TokenBalance    `json:"tokenBalance"`
// 	HistoryList      []*TokenTransfer `json:"historyList"`
// 	TransferableList []*TokenTransfer `json:"transferableList"`
// }

// type Brc20AddressTokenSummaryResp struct {
// 	serverCommon.BaseResp
// 	Data *AddressTokenSummary `json:"data"`
// }

// // /brc20/transferable-list
// type Brc20TransferableList struct {
// 	serverCommon.ListResp
// 	List []*TokenTransfer `json:"list"`
// }

// type Brc20TransferableListResp struct {
// 	serverCommon.BaseResp
// 	Data *AddressTokenSummary `json:"data"`
// }
