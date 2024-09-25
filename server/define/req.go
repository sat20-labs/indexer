package define

type AddressReq struct {
	Address string `form:"address" binding:"required"`
}

type AddressTickerReq struct {
	AddressReq
	Ticker string `form:"ticker" binding:"required"`
}
