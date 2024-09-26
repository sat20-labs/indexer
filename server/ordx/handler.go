package ordx

import (
	"net/http"
	"strconv"

	ordx "github.com/sat20-labs/ordx/common"
	serverOrdx "github.com/sat20-labs/ordx/server/define"
	"github.com/sat20-labs/ordx/share/base_indexer"
	"github.com/gin-gonic/gin"
)

const QueryParamDefaultLimit = "100"

type Handle struct {
	model *Model
}

func NewHandle(indexer base_indexer.Indexer) *Handle {
	return &Handle{
		model: NewModel(indexer),
	}
}

// @Summary Get the current btc height
// @Description the current btc height
// @Tags ordx
// @Produce json
// @Security Bearer
// @Success 200 {object} BestHeightResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /bestheight [get]
func (s *Handle) getBestHeight(c *gin.Context) {
	resp := &BestHeightResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: map[string]int{"height": s.model.GetSyncHeight()},
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Get the height block info
// @Description the height block info
// @Tags ordx
// @Produce json
// @Security Bearer
// @Success 200 {object} BestHeightResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /height [get]
func (s *Handle) getBlockInfo(c *gin.Context) {
	resp := &BlockInfoData{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	height, err := strconv.ParseInt(c.Param("height"), 10, 32)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	result, err := s.model.GetBlockInfo(int(height))
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = result
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Handle) isDeployAllowed(c *gin.Context) {
	resp := &serverOrdx.BaseResp{}

	ticker := c.Param("ticker")
	if ticker == "" {
		resp.Code = -1
		resp.Msg = "no ticker"
		c.JSON(http.StatusOK, resp)
		return
	}
	address := c.Param("address")
	if ticker == "" {
		resp.Code = -1
		resp.Msg = "no address"
		c.JSON(http.StatusOK, resp)
		return
	}
	_, err := s.model.IsDeployAllowed(address, ticker)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Code = 0
		resp.Msg = ""
	}

	c.JSON(http.StatusOK, resp)
}

// @Summary Get status list for all tickers
// @Description Get status list for all tickers
// @Tags ordx
// @Produce json
// @Query start query int false "Start index for pagination"
// @Query limit query int false "Limit for pagination"
// @Security Bearer
// @Success 200 {object} StatusListResp
// @Failure 401 "Invalid API Key"
// @Router /tick/status [get]
func (s *Handle) getTickerStatusList(c *gin.Context) {
	resp := &StatusListResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: &StatusListData{
			ListResp: serverOrdx.ListResp{
				Total: 0,
				Start: 0,
			},
			Height: uint64(0),
			Detail: nil,
		},
	}

	height := s.model.GetSyncHeight()
	ticklist, err := s.model.GetTickerStatusList()
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = &StatusListData{
			ListResp: serverOrdx.ListResp{
				Total: uint64(len(ticklist)),
				Start: 0,
			},
			Height: uint64(height),
			Detail: ticklist,
		}
	}

	c.JSON(http.StatusOK, resp)
}

// @Summary Get a ticker's status
// @Description Get the status of a specific ticker
// @Tags ordx.tick
// @Produce json
// @Param tickerName path string true "Ticker name"
// @Security Bearer
// @Success 200 {object} StatusResp
// @Failure 401 "Invalid API Key"
// @Router /tick/info/{ticker} [get]
func (s *Handle) getTickerStatus(c *gin.Context) {
	resp := &StatusResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	tickerName := c.Param("ticker")
	tickerStatus, err := s.model.GetTickerStatus(tickerName)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = tickerStatus
	c.JSON(http.StatusOK, resp)
}

// @Summary Get Holder List
// @Description Get a list of holders for a specific ticker
// @Tags ordx.tick
// @Produce json
// @Param ticker path string true "Ticker name"
// @Query start query int false "Start index for pagination"
// @Query limit query int false "Limit for pagination"
// @Security Bearer
// @Success 200 {object} HolderListResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /tick/holders/{ticker} [get]
func (s *Handle) getHolderList(c *gin.Context) {
	resp := &HolderListResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	tickerName := c.Param("ticker")
	start, err := strconv.Atoi(c.DefaultQuery("start", "0"))
	if err != nil {
		start = 0
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", QueryParamDefaultLimit))
	if err != nil {
		limit = 100
	}
	holderlist, err := s.model.GetHolderList(tickerName, start, limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = &HolderListData{
		ListResp: serverOrdx.ListResp{
			Total: uint64(len(holderlist)),
			Start: int64(start),
		},
		Detail: holderlist,
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Get Mint History
// @Description Get the mint history for a specific ticker
// @Tags ordx.tick
// @Produce json
// @Param ticker path string true "Ticker name"
// @Query start query int false "Start index for pagination"
// @Query limit query int false "Limit for pagination"
// @Security Bearer
// @Success 200 {object} MintHistoryResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /tick/history/{ticker} [get]
func (s *Handle) getMintHistory(c *gin.Context) {
	resp := &MintHistoryResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: &MintHistoryData{
			ListResp: serverOrdx.ListResp{
				Total: 0,
				Start: 0,
			},
			Detail: nil,
		},
	}
	tickerName := c.Param("ticker")
	start, err := strconv.Atoi(c.DefaultQuery("start", "0"))
	if err != nil {
		start = 0
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", QueryParamDefaultLimit))
	if err != nil {
		limit = 100
	}
	mintHistory, err := s.model.GetMintHistory(tickerName, start, limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = &MintHistoryData{
		ListResp: serverOrdx.ListResp{
			Total: uint64(mintHistory.Total),
			Start: int64(start),
		},
		Detail: mintHistory,
	}
	c.JSON(http.StatusOK, resp)
}

// inner api
func (s *Handle) getSplittedInscriptionList(c *gin.Context) {
	resp := &InscriptionIdListResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}
	tickerName := c.Param("ticker")
	if tickerName == "" {
		resp.Code = -1
		resp.Msg = "invalid ticker name"
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = s.model.GetSplittedInscriptionList(tickerName)
	c.JSON(http.StatusOK, resp)
}

// @Summary Get Mint Detail
// @Description Get detailed information about a mint based on the inscription ID
// @Tags ordx.mint
// @Produce json
// @Param inscriptionid path string true "Inscription ID"
// @Security Bearer
// @Success 200 {object} MintDetailInfoResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /mint/details/{inscriptionid} [get]
func (s *Handle) getMintDetailInfo(c *gin.Context) {
	resp := &MintDetailInfoResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	inscriptionId := c.Param("inscriptionid")
	if len(inscriptionId) < 32 {
		resp.Code = -1
		resp.Msg = "invalid inscription id"
		c.JSON(http.StatusOK, resp)
		return
	}

	mintDetail, err := s.model.GetMintDetailInfo(inscriptionId)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = mintDetail
	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getMintPermission(c *gin.Context) {
	resp := &MintPermissionResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	address := c.Param("address")
	if address == "" {
		resp.Code = -1
		resp.Msg = "invalid address"
		c.JSON(http.StatusOK, resp)
		return
	}

	ticker := c.Param("ticker")
	if ticker == "" {
		resp.Code = -1
		resp.Msg = "invalid ticker"
		c.JSON(http.StatusOK, resp)
		return
	}

	mintDetail, err := s.model.GetMintPermissionInfo(ticker, address)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = mintDetail
	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getFeeInfo(c *gin.Context) {
	resp := &FeeResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	address := c.Param("address")
	if address == "" {
		resp.Code = -1
		resp.Msg = "invalid address"
		c.JSON(http.StatusOK, resp)
		return
	}

	mintDetail, err := s.model.GetFeeInfo(address)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = mintDetail
	c.JSON(http.StatusOK, resp)
}

// @Summary Get Balance Summary List
// @Description Get a summary list of balances for a specific address
// @Tags ordx.address
// @Produce json
// @Param address path string true "Address"
// @Query start query int false "Start index for pagination"
// @Query limit query int false "Limit for pagination"
// @Security Bearer
// @Success 200 {object} BalanceSummaryListResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /address/summary/{address} [get]
func (s *Handle) getBalanceSummaryList(c *gin.Context) {
	resp := &BalanceSummaryListResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	address := c.Param("address")
	start, err := strconv.ParseInt(c.DefaultQuery("start", "0"), 10, 64)
	if err != nil {
		start = 0
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", QueryParamDefaultLimit))
	if err != nil {
		limit = 100
	}

	balanceSummaryList, err := s.model.GetBalanceSummaryList(address, int(start), limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = &BalanceSummaryListData{
		ListResp: serverOrdx.ListResp{
			Total: uint64(len(balanceSummaryList)),
			Start: start,
		},
		Detail: balanceSummaryList,
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Get Utxo List
// @Description Get a list of UTXOs for a specific address and ticker
// @Tags ordx.address
// @Produce json
// @Param address path string true "Address"
// @Param ticker path string true "Ticker symbol"
// @Query start query int false "Start index for pagination"
// @Query limit query int false "Limit for pagination"
// @Security Bearer
// @Success 200 {object} UtxoListResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /address/utxolist/{address}/{ticker} [get]
func (s *Handle) getUtxoList(c *gin.Context) {
	resp := &UtxoListResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	address := c.Param("address")
	ticker := c.Param("ticker")
	start, err := strconv.ParseInt(c.DefaultQuery("start", "0"), 10, 64)
	if err != nil {
		start = 0
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", QueryParamDefaultLimit))
	if err != nil {
		limit = 100
	}

	tickerUtxoInfoList, total, err := s.model.GetUtxoList(address, ticker, int(start), limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = &UtxoListData{
		ListResp: serverOrdx.ListResp{
			Total: uint64(total),
			Start: start,
		},
		Detail: tickerUtxoInfoList,
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getUtxoList2(c *gin.Context) {
	resp := &UtxoListResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	address := c.Param("address")
	ticker := c.Param("ticker")
	start, err := strconv.Atoi(c.DefaultQuery("start", "0"))
	if err != nil {
		start = 0
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", QueryParamDefaultLimit))
	if err != nil {
		limit = 100
	}

	tickerUtxoInfoList, total, err := s.model.GetUtxoList2(address, ticker, start, limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = &UtxoListData{
		ListResp: serverOrdx.ListResp{
			Total: uint64(total),
			Start: int64(start),
		},
		Detail: tickerUtxoInfoList,
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getUtxoList3(c *gin.Context) {
	resp := &UtxoListResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	address := c.Param("address")
	start, err := strconv.Atoi(c.DefaultQuery("start", "0"))
	if err != nil {
		start = 0
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", QueryParamDefaultLimit))
	if err != nil {
		limit = 100
	}

	tickerUtxoInfoList, total, err := s.model.GetUtxoList3(address, start, limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = &UtxoListData{
		ListResp: serverOrdx.ListResp{
			Total: uint64(total),
			Start: int64(start),
		},
		Detail: tickerUtxoInfoList,
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Get mint history for a specific address
// @Description Get the mint history for a specific address with pagination
// @Tags ordx.address
// @Produce json
// @Param tickerName path string true "Name of the ticker"
// @Param address path string true "Address to get the mint history for"
// @Query start query int false "Start index for pagination" default(0)
// @Query limit query int false "Number of items to fetch" default(100)
// @Security Bearer
// @Success 200 {object} MintHistoryResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /address/history/{address}/{:ticker} [get]
func (s *Handle) getAddressMintHistory(c *gin.Context) {
	resp := &MintHistoryResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: &MintHistoryData{
			ListResp: serverOrdx.ListResp{
				Total: 0,
				Start: 0,
			},
			Detail: nil,
		},
	}

	start, err := strconv.Atoi(c.DefaultQuery("start", "0"))
	if err != nil {
		start = 0
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", QueryParamDefaultLimit))
	if err != nil {
		limit = 100
	}
	address := c.Param("address")
	ticker := c.Param("ticker")
	mintHistory, err := s.model.GetAddressMintHistory(ticker, address, start, limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = &MintHistoryData{
		ListResp: serverOrdx.ListResp{
			Total: uint64(mintHistory.Total),
			Start: int64(start),
		},
		Detail: mintHistory,
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Get asset details in a UTXO
// @Description Get asset details in a UTXO
// @Tags ordx.utxo
// @Produce json
// @Param utxo path string true "UTXO"
// @Security Bearer
// @Success 200 {object} AssetsResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /address/assets/{utxo} [get]
func (s *Handle) getAssetDetailInfo(c *gin.Context) {
	resp := &AssetsResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	utxo := c.Param("utxo")
	utxoAssets, err := s.model.GetDetailAssetWithUtxo(utxo)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = &AssetsData{
		ListResp: serverOrdx.ListResp{
			Total: 1,
			Start: 0,
		},
		Detail: utxoAssets,
	}
	c.JSON(http.StatusOK, resp)
}


func (s *Handle) getAssetOffset(c *gin.Context) {
	resp := &AssetOffsetResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	utxo := c.Param("utxo")
	utxoAssets, err := s.model.GetAssetOffsetWithUtxo(utxo)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = &AssetOffsetData{
		ListResp: serverOrdx.ListResp{
			Total: uint64(len(utxoAssets)),
			Start: 0,
		},
		AssetOffset: utxoAssets,
	}
	c.JSON(http.StatusOK, resp)
}


// @Summary Get assets with abbreviated info in the UTXO
// @Description Get assets with abbreviated info in the UTXO
// @Tags ordx.utxo
// @Produce json
// @Param utxo path string true "UTXO value"
// @Security Bearer
// @Success 200 {array} AssetAbbrInfo
// @Failure 401 "Invalid API Key"
// @Router /getAssetByUtxo/{utxo} [get]
func (s *Handle) getAbbrAssetsWithUtxo(c *gin.Context) {
	resp := &AssetListResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}
	utxo := c.Param("utxo")
	assetList, err := s.model.GetAbbrAssetsWithUtxo(utxo)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = assetList
	c.JSON(http.StatusOK, resp)
}

// @Summary Get seed of sats in the UTXO
// @Description Get seed of sats in the UTXO, according to ticker and sat's attributes
// @Tags ordx.utxo
// @Produce json
// @Param utxo path string true "UTXO value"
// @Security Bearer
// @Success 200 {array} Seed
// @Failure 401 "Invalid API Key"
// @Router /utxo/seed/{utxo} [get]
func (s *Handle) getSeedWithUtxo(c *gin.Context) {
	resp := &SeedsResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}
	utxo := c.Param("utxo")
	seeds, err := s.model.GetSeedsWithUtxo(utxo)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = seeds
	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getSatRangeWithUtxo(c *gin.Context) {
	resp := &UtxoInfoResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}
	utxo := c.Param("utxo")
	ret, err := s.model.GetSatRangeWithUtxo(utxo)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = ret
	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getAssetsWithUtxos(c *gin.Context) {
	resp := &AbbrAssetsWithUtxosResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	var req UtxosReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	result, err := s.model.GetAssetsWithUtxos(&req)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = result
	}

	c.JSON(http.StatusOK, resp)
}

// @Summary Get asset details in a range
// @Description Get asset details in a range
// @Tags ordx.range
// @Produce json
// @Param start path string true "start"
// @Param size path string true "size"
// @Security Bearer
// @Success 200 {object} AssetsResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /range/{start}/{size} [get]
func (s *Handle) getAssetDetailInfoWithRange(c *gin.Context) {
	resp := &AssetsResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	start, err := strconv.ParseInt(c.Param("start"), 10, 64)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	size, err := strconv.ParseInt(c.Param("size"), 10, 64)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	req := RangesReq{Ranges: []*ordx.Range{{Start: start, Size: size}}}
	assets, err := s.model.GetDetailAssetWithRanges(&req)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = &AssetsData{
		ListResp: serverOrdx.ListResp{
			Total: 1,
			Start: 0,
		},
		Detail: assets,
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Get asset details in a range
// @Description Get asset details in a range
// @Tags ordx.range
// @Produce json
// @Security Bearer
// @Success 200 {object} AssetsResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /ranges [post]
func (s *Handle) getAssetDetailInfoWithRanges(c *gin.Context) {
	resp := &AssetsResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	var req RangesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	assets, err := s.model.GetDetailAssetWithRanges(&req)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = &AssetsData{
		ListResp: serverOrdx.ListResp{
			Total: 1,
			Start: 0,
		},
		Detail: assets,
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Get name service status
// @Description Get name service status
// @Tags ordx
// @Produce json
// @Query start query int false "Start index for pagination"
// @Query limit query int false "Limit for pagination"
// @Security Bearer
// @Success 200 {object} NSStatusResp
// @Failure 401 "Invalid API Key"
// @Router /ns/status [get]
func (s *Handle) getNSStatus(c *gin.Context) {
	resp := &NSStatusResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	start, err := strconv.Atoi(c.DefaultQuery("start", "0"))
	if err != nil {
		start = 0
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", QueryParamDefaultLimit))
	if err != nil {
		limit = 0
	}

	result, err := s.model.GetNSStatusList(start, limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = result
	}

	c.JSON(http.StatusOK, resp)
}

// @Summary Get name's properties
// @Description Get name's properties
// @Tags ordx
// @Produce json
// @Security Bearer
// @Success 200 {object} NamePropertiesResp
// @Failure 401 "Invalid API Key"
// @Router /ns/name [get]
func (s *Handle) getNameInfo(c *gin.Context) {
	resp := &NamePropertiesResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	name := c.Param("name")
	result, err := s.model.GetNameInfo(name)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = result
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getNameValues(c *gin.Context) {
	resp := &NamePropertiesResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	name := c.Param("name")
	prefix := c.Param("prefix")
	start, err := strconv.Atoi(c.DefaultQuery("start", "0"))
	if err != nil {
		start = 0
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", QueryParamDefaultLimit))
	if err != nil {
		limit = 0
	}
	result, err := s.model.GetNameValues(name, prefix, start, limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = result
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getNameRouting(c *gin.Context) {
	resp := &NameRoutingResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	name := c.Param("name")
	result, err := s.model.GetNameRouting(name)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = result
	}

	c.JSON(http.StatusOK, resp)
}

// @Summary Get all names in an address
// @Description Get all names in an address
// @Tags ordx
// @Produce json
// @Security Bearer
// @Success 200 {object} NamesWithAddressResp
// @Failure 401 "Invalid API Key"
// @Router /ns/address [get]
func (s *Handle) getNamesWithAddress(c *gin.Context) {
	resp := &NamesWithAddressResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	address := c.Param("address")
	sub := c.Param("sub")
	start, err := strconv.Atoi(c.DefaultQuery("start", "0"))
	if err != nil {
		start = 0
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", QueryParamDefaultLimit))
	if err != nil {
		limit = 0
	}
	result, err := s.model.GetNamesWithAddress(address, sub, start, limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = result
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getNamesWithFilters(c *gin.Context) {
	resp := &NamesWithAddressResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	address := c.Param("address")
	sub := c.Param("sub")
	filters := c.Param("filters")
	start, err := strconv.Atoi(c.DefaultQuery("start", "0"))
	if err != nil {
		start = 0
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", QueryParamDefaultLimit))
	if err != nil {
		limit = 0
	}
	result, err := s.model.GetNamesWithFilters(address, sub, filters, start, limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = result
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getNamesWithSat(c *gin.Context) {
	resp := &NamesWithAddressResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	sat := c.Param("sat")
	iSat, err := strconv.ParseInt(sat, 10, 64)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	result, err := s.model.GetNamesWithSat(iSat)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = result
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getNameWithInscriptionId(c *gin.Context) {
	resp := &NamePropertiesResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	inscriptionId := c.Param("id")
	result, err := s.model.GetNameWithInscriptionId(inscriptionId)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = result
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Handle) checkNames(c *gin.Context) {
	resp := &NameCheckResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	var req NameCheckReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	result, err := s.model.GetNameCheckResult(&req)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = result
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Handle) addCollection(c *gin.Context) {
	resp := &AddCollectionResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	var req AddCollectionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	err := s.model.AddCollection(&req)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getNftStatus(c *gin.Context) {
	resp := &NftStatusResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	start, err := strconv.Atoi(c.DefaultQuery("start", "0"))
	if err != nil {
		start = 0
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", QueryParamDefaultLimit))
	if err != nil {
		limit = 0
	}

	result, err := s.model.GetNftStatusList(start, limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = result
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getNftInfo(c *gin.Context) {
	resp := &NftInfoResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	idstr := c.Param("id")
	id, err := strconv.ParseInt(idstr, 10, 64)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	result, err := s.model.GetNftInfo(id)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = result
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getNftsWithAddress(c *gin.Context) {
	resp := &NftsWithAddressResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	address := c.Param("address")
	start, err := strconv.Atoi(c.DefaultQuery("start", "0"))
	if err != nil {
		start = 0
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", QueryParamDefaultLimit))
	if err != nil {
		limit = 0
	}
	result, total, err := s.model.GetNftsWithAddress(address, start, limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = result
		resp.Data.Total = uint64(total)
		resp.Data.Start = int64(start)
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getNftsWithSat(c *gin.Context) {
	resp := &NftsWithAddressResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	sat := c.Param("sat")
	iSat, err := strconv.ParseInt(sat, 10, 64)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	result, err := s.model.GetNftsWithSat(iSat)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = result
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getNftWithInscriptionId(c *gin.Context) {
	resp := &NftInfoResp{
		BaseResp: serverOrdx.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	inscriptionId := c.Param("id")

	result, err := s.model.GetNftInfoWithInscriptionId(inscriptionId)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = result
	}

	c.JSON(http.StatusOK, resp)
}
