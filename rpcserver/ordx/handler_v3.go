package ordx

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
)

// include plain sats
func (s *Handle) getAssetSummaryV3(c *gin.Context) {
	resp := &rpcwire.AssetSummaryRespV3{
		BaseResp: rpcwire.BaseResp{
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

	result, err := s.model.GetAssetSummaryV3(address, int(start), limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = result
	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getUtxosWithTickerV3(c *gin.Context) {
	resp := &rpcwire.UtxosWithAssetRespV3{
		BaseResp: rpcwire.BaseResp{
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

	result, total, err := s.model.GetUtxosWithAssetNameV3(address, ticker, int(start), limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.ListResp = rpcwire.ListResp{
		Total: uint64(total),
		Start: start,
	}
	resp.Data = result

	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getUtxoInfoV3(c *gin.Context) {
	resp := &rpcwire.TxOutputRespV3{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	utxo := c.Param("utxo")
	result, err := s.model.GetUtxoInfoV3(utxo)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = result
	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getUtxoInfoListV3(c *gin.Context) {
	resp := &rpcwire.TxOutputListRespV3{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	var req rpcwire.UtxosReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	result, err := s.model.GetUtxoInfoListV3(&req)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = result
	}

	c.JSON(http.StatusOK, resp)
}

// @Summary Get Holder List v3
// @Description Get a list of holders for a specific ticker
// @Tags ordx.tick
// @Produce json
// @Param ticker path string true "Ticker name"
// @Query start query int false "Start index for pagination"
// @Query limit query int false "Limit for pagination"
// @Security Bearer
// @Success 200 {object} HolderListRespV3 "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /v3/tick/holders/{ticker} [get]
func (s *Handle) getHolderListV3(c *gin.Context) {
	resp := &HolderListRespV3{
		BaseResp: rpcwire.BaseResp{
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
	holderlist, err := s.model.GetHolderListV3(tickerName, start, limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = &HolderListDataV3{
		ListResp: rpcwire.ListResp{
			Total: uint64(len(holderlist)),
			Start: int64(start),
		},
		Detail: holderlist,
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Get Mint History v3
// @Description Get the mint history for a specific ticker
// @Tags ordx.tick
// @Produce json
// @Param ticker path string true "Ticker name"
// @Query start query int false "Start index for pagination"
// @Query limit query int false "Limit for pagination"
// @Security Bearer
// @Success 200 {object} MintHistoryRespV3 "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /v3/tick/history/{ticker} [get]
func (s *Handle) getMintHistoryV3(c *gin.Context) {
	resp := &MintHistoryRespV3{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: &MintHistoryDataV3{
			ListResp: rpcwire.ListResp{
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
	mintHistory, err := s.model.GetMintHistoryV3(tickerName, start, limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = &MintHistoryDataV3{
		ListResp: rpcwire.ListResp{
			Total: uint64(mintHistory.Total),
			Start: int64(start),
		},
		Detail: mintHistory,
	}
	c.JSON(http.StatusOK, resp)
}
