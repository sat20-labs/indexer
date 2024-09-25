package extension

import (
	"github.com/gin-gonic/gin"
)

type Service struct {
	chain string
}

func NewService(chain string) *Service {
	return &Service{
		chain: chain,
	}
}

func (s *Service) InitRouter(r *gin.Engine, basePath string) {
	g := r.Group(basePath + "/extension")
	// implement use same as office api
	g.GET("/version/detail", s.version_detail)

	g.GET("/default/config", s.walletConfig)
	g.GET("/default/inscription-summary", s.inscriptionSummary)
	g.GET("/default/app-summary-v2", s.appSummary)
	g.GET("/default/fee-summary", s.feeSummary)
	g.POST("/default/check-website", s.checkWebsite)

	// need implement
	g.GET("/address/summary", s.address_assetsSummary)
	g.GET("/address/balance", s.address_balance)
	g.GET("/address/multi-assets", s.address_AssetSummaryList)
	// implement use same as office api
	g.GET("/address/unavailable-utxo", s.address_UnavailableUtxoList)
	g.GET("/address/btc-utxo", s.address_BTCUtxoList)
	g.GET("/address/inscriptions", s.address_inscriptionList)
	g.GET("/address/search", s.address_domainInfo)
	// no need implement
	g.POST("/address/find-group-assets", s.address_findGroupAssetList)

	// need implement
	g.POST("/tx/broadcast", s.tx_broadcast)
	g.POST("/tx/decode2", s.tx_decodePsbt)

	// need implement
	g.GET("/ordinals/inscriptions", s.ordinals_inscriptionList)

	// need implement
	g.GET("/inscription/utxo", s.inscription_utxo)
	g.GET("/inscription/utxo-detail", s.inscription_utxoDetail)
	g.POST("/inscription/utxos", s.inscription_utxoList)
	g.GET("/inscription/info", s.inscription_info)

	g.GET("/runes/list", s.runes_list)
	g.GET("/runes/utxos", s.runes_utxoList)
	g.GET("/runes/token-summary", s.runes_tokenSummary)

	g.GET("/name/list", s.name_list)

	g.GET("/raresat/list", s.raresat_list)

	g.GET("/token/list", s.token_list)

	// implement use same as office api
	// g.POST("/brc20/inscribe-transfer", s.inscribeBRC20Transfer)
	// g.GET("/brc20/order-result", s.getInscribeResult)
	// g.GET("/brc20/list", s.getBRC20List)
	// g.GET("/brc20/5byte-list", s.getBRC20List5Byte)
	// g.GET("/brc20/token-summary", s.getAddressTokenSummary)
	// g.GET("/brc20/transferable-list", s.getTokenTransferableList)

	// implement use same as office api
	// g.GET("/buy-btc/channel-list", s.getBuyBtcChannelList)
	// g.POST("/buy-btc/create", s.createPaymentUrl)

	// implement use same as office api
	// g.GET("/atomicals/nft", s.getAtomicalsNFT)
	// g.GET("/atomicals/utxo", s.getAtomicalsUtxo)
	// g.GET("/arc20/balance-list", s.getArc20BalanceList)
	// g.GET("/arc20/utxos", s.getArc20Utxos)
}
