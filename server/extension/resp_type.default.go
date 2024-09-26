package extension

import (
	"github.com/sat20-labs/indexer/server/define"
)

// /default/config
type ConfigData struct {
	Version        string `json:"version"`
	MoonPayEnabled bool   `json:"moonPayEnabled"`
	StatusMessage  string `json:"statusMessage"`
}
type ConfigResp struct {
	define.BaseResp
	Data *ConfigData `json:"data"`
}

// /default/app-summary-v2
type App struct {
	Logo     string `json:"logo"`
	Title    string `json:"title"`
	Desc     string `json:"desc"`
	URL      string `json:"url"`
	ID       int    `json:"id"`
	Tag      string `json:"tag"`
	TagColor string `json:"tagColor"`
}

type AppSummaryV2Data struct {
	Apps []App `json:"apps"`
}

type AppSummaryV2Resp struct {
	define.BaseResp
	Data AppSummaryV2Data `json:"data"`
}

// /default/inscription-summary
type Minted struct {
	Title        string        `json:"title"`
	Desc         string        `json:"desc"`
	Inscriptions []Inscription `json:"inscriptions"`
}

type InscriptionSummaryData struct {
	MintedList []Minted `json:"mintedList"`
}

type InscriptionSummaryResp struct {
	define.BaseResp
	Data InscriptionSummaryData `json:"data"`
}

// /default/check-website
type CheckWebSiteReq struct {
	WebSite string `json:"website"`
}

type CheckWebSiteResp struct {
	define.BaseResp
	IsScammer bool   `json:"isScammer"`
	Warning   string `json:"warning"`
}

// /default/fee-summary
type FeeSummary struct {
	Title   string `json:"title"`
	Desc    string `json:"desc"`
	FeeRate string `json:"feeRate"`
}
type FeeSummaryList struct {
	List []*FeeSummary `json:"list"`
}

type FeeSummaryResp struct {
	define.BaseResp
	Data *FeeSummaryList `json:"data"`
}
