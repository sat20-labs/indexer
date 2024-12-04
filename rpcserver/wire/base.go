package wire

type AddressReq struct {
	Address string `form:"address" binding:"required"`
}

type AddressTickerReq struct {
	AddressReq
	Ticker string `form:"ticker" binding:"required"`
}


type BaseResp struct {
	Code int    `json:"code" example:"0"`
	Msg  string `json:"msg" example:"ok"`
}

type ListResp struct {
	Start int64  `json:"start" example:"0"`
	Total uint64 `json:"total" example:"9992"`
}

