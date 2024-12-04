package extension

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

func (s *Service) walletConfig(c *gin.Context) {
	resp := &ConfigResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: &ConfigData{
			Version:        "0.1.3",
			MoonPayEnabled: true,
			StatusMessage:  "",
		},
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Service) appSummary(c *gin.Context) {
	resp := &AppSummaryV2Resp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: AppSummaryV2Data{
			Apps: []App{
				{
					Logo:     "https://ordx.market/logo.png",
					Title:    "Sat20 Marketplace",
					Desc:     "Trade NFT, FT and domain names.",
					URL:      "https://ordx.market",
					ID:       1,
					Tag:      "Marketplace",
					TagColor: "rgba(34,200,249,0.7)",
				},
				{
					Logo:     "https://app.sat20.org/logo.png",
					Title:    "Sat20 explorer",
					Desc:     "Sat20 explorer for sat20 protocol data.",
					URL:      "https://app.sat20.org/#/explorer",
					ID:       2,
					Tag:      "Technology",
					TagColor: "rgba(249,192,34,0.8)",
				},
				{
					Logo:     "https://app.sat20.org/logo.png",
					Title:    "Sat20 Inscribe",
					Desc:     "Ultra Low Inscribing Cost!",
					URL:      "https://ordx.market/inscribe",
					ID:       3,
					Tag:      "Technology",
					TagColor: "rgba(34,249,128,0.6)",
				},
				{
					Logo:     "https://app.sat20.org/logo.png",
					Title:    "Sat20 Tools",
					Desc:     "Sat20 spilt, Merge & Transfer Tools!",
					URL:      "https://ordx.market/tools",
					ID:       4,
					Tag:      "Technology",
					TagColor: "rgba(249,192,34,0.8)",
				},
				// {
				// 	Logo:     "https://static.unisat.io/res/images/app-oxalus.jpg",
				// 	Title:    "OXALUS",
				// 	Desc:     "The NFT Social Commerce Platform! Where Social meets Digital Collectors, connect NFT creators and collectors in one hub!",
				// 	URL:      "https://oxalus.io",
				// 	ID:       2,
				// 	Tag:      "Social",
				// 	TagColor: "rgba(34,200,249,0.7)",
				// },
				// {
				// 	Logo:     "https://static.unisat.io/res/images/app-teleordinal.jpg",
				// 	Title:    "TeleOrdinal",
				// 	Desc:     "Decentralized P2P Ordinals marketplace.",
				// 	URL:      "https://app.teleordinal.xyz",
				// 	ID:       4,
				// 	Tag:      "Marketplace",
				// 	TagColor: "rgba(249,192,34,0.8)",
				// },
				// {
				// 	Logo:     "https://static.unisat.io/res/images/app-unisat.png",
				// 	Title:    "UniSat Inscribe",
				// 	Desc:     "Ultra Low Inscribing Cost!",
				// 	URL:      "https://unisat.io/inscribe",
				// 	ID:       1,
				// 	Tag:      "Inscription Service",
				// 	TagColor: "rgba(34,249,128,0.6)",
				// },
				// {
				// 	Logo:     "https://static.unisat.io/res/images/app-unisat.png",
				// 	Title:    "UniSat Marketplace",
				// 	Desc:     "Trade BRC-20, domain names, and collections securely and seamlessly.",
				// 	URL:      "https://unisat.io/market",
				// 	ID:       3,
				// 	Tag:      "Marketplace",
				// 	TagColor: "rgba(249,192,34,0.8)",
				// },
				// {
				// 	Logo:     "https://ordinals.market/favicon.png",
				// 	Title:    "Ordinals Marketplace",
				// 	Desc:     "Unrivalled Speed, Unmatched Data in Bitcoin's Premier Ordinals Marketplace.",
				// 	URL:      "https://ordinals.market",
				// 	ID:       5,
				// 	Tag:      "Marketplace",
				// 	TagColor: "rgba(249,192,34,0.8)",
				// },
				// {
				// 	Logo:     "https://static.unisat.io/res/images/app-magiceden.svg",
				// 	Title:    "Magic Eden",
				// 	Desc:     "Welcome to the world of Ordinals, brought to you by the first audited, secure platform",
				// 	URL:      "https://magiceden.io/ordinals",
				// 	ID:       6,
				// 	Tag:      "Marketplace",
				// 	TagColor: "rgba(249,192,34,0.8)",
				// },
				// {
				// 	Logo:     "https://static.unisat.io/res/images/app-openordex.png",
				// 	Title:    "OpenOrdex",
				// 	Desc:     "An open source zero-fee trustless Bitcoin NFT marketplace based on partially signed bitcoin transactions",
				// 	URL:      "https://openordex.org/",
				// 	ID:       7,
				// 	Tag:      "Marketplace",
				// 	TagColor: "rgba(249,192,34,0.8)",
				// },
			},
		},
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Service) inscriptionSummary(c *gin.Context) {
	resp := &InscriptionSummaryResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: InscriptionSummaryData{
			MintedList: []Minted{},
		},
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Service) feeSummary(c *gin.Context) {
	resp := &FeeSummaryResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: &FeeSummaryList{
			List: []*FeeSummary{
				{
					Title:   "Slow",
					Desc:    "About 1 hours",
					FeeRate: "20",
				},
				{
					Title:   "Normal",
					Desc:    "About 30 minutes",
					FeeRate: "50",
				},
				{
					Title:   "Fast",
					Desc:    "About 10 minutes",
					FeeRate: "100",
				},
			},
		},
	}

	ret, err := bitcoin_rpc.ShareBitconRpc.EstimateSmartFeeWithMode(1, "ECONOMICAL")
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	// BTC/kb -> sat/vb
	resp.Data.List[0].FeeRate = strconv.FormatFloat((ret.FeeRate * 100000), 'f', 2, 64)

	ret, err = bitcoin_rpc.ShareBitconRpc.EstimateSmartFeeWithMode(1, "ECONOMICAL")
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data.List[1].FeeRate = strconv.FormatFloat((ret.FeeRate * 100000), 'f', 2, 64)

	ret, err = bitcoin_rpc.ShareBitconRpc.EstimateSmartFeeWithMode(1, "CONSERVATIVE")
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data.List[2].FeeRate = strconv.FormatFloat((ret.FeeRate * 100000), 'f', 2, 64)
	c.JSON(http.StatusOK, resp)
}

func (s *Service) checkWebsite(c *gin.Context) {
	resp := &CheckWebSiteResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		IsScammer: false,
		Warning:   "",
	}

	var req AddressFindGroupAssetsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	c.JSON(http.StatusOK, resp)
}
