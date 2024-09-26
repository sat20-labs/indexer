package base

import (
	"net/http"
	"strconv"

	ordxCommon "github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/exotic"
	"github.com/sat20-labs/indexer/server/define"

	"github.com/gin-gonic/gin"
)

// @Summary Health Check
// @Description Check the health status of the service
// @Tags ordx
// @Produce json
// @Success 200 {object} HealthStatusResp "Successful response"
// @Router /health [get]
func (s *Service) getHealth(c *gin.Context) {
	rsp := &HealthStatusResp{
		Status:    "ok",
		Version:   ordxCommon.ORDX_INDEXER_VERSION,
		BaseDBVer: s.model.indexer.GetBaseDBVer(),
		OrdxDBVer: s.model.indexer.GetOrdxDBVer(),
	}

	tip := s.model.indexer.GetChainTip()
	sync := s.model.indexer.GetSyncHeight()
	code := 200
	if tip != sync && tip != sync+1 {
		code = 201
		rsp.Status = "syncing"
	}

	c.JSON(code, rsp)
}

// @Summary Retrieves information about a sat
// @Description Retrieves information about a sat based on the given sat ID
// @Tags ordx
// @Produce json
// @Security Bearer
// @Param sat path int true "Sat ID"
// @Success 200 {object} SatInfoResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /sat/{sat} [get]
func (s *Service) getSatInfo(c *gin.Context) {
	resp := &SatInfoResp{
		BaseResp: define.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}
	satNumber, err := strconv.ParseInt(c.Param("sat"), 10, 64)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = s.model.GetSatInfo(satNumber)
	c.JSON(http.StatusOK, resp)
}

// @Summary find specific sats in address
// @Description find specific sats in address
// @Tags ordx
// @Produce json
// @Security Bearer
// @Param address body string true "address"
// @Param sats body []number true "sats"
// @Success 200 {object} SpecificSatResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /sat/FindSatsInAddress/ [post]
func (s *Service) findSatsInAddress(c *gin.Context) {
	resp := &SpecificSatResp{
		BaseResp: define.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}
	var req SpecificSatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	if req.Address == "" {
		resp.Code = -1
		resp.Msg = "invalid address"
		c.JSON(http.StatusOK, resp)
		return
	}
	if len(req.Sats) <= 0 {
		resp.Code = -1
		resp.Msg = "invalid sats"
		c.JSON(http.StatusOK, resp)
		return
	}

	result, err := s.model.findSatsInAddress(&req)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = result
	c.JSON(http.StatusOK, resp)
}

// 不开放
func (s *Service) findSat(c *gin.Context) {
	resp := &SpecificSatResp{
		BaseResp: define.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	sat, err := strconv.ParseInt(c.Param("sat"), 10, 64)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	result, err := s.model.findSat(sat)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = []*define.SpecificSat{result}
	c.JSON(http.StatusOK, resp)
}

// @Summary Retrieves the supported attributes of a sat
// @Description Retrieves the supported attributes of a sat
// @Tags ordx
// @Produce json
// @Security Bearer
// @Success 200 {array} SatributesResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /info/satributes [get]
func (s *Service) getSatributes(c *gin.Context) {
	resp := &SatributesResp{
		BaseResp: define.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: exotic.SatributeList,
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Retrieves all sat ranges and attributes in a given utxo
// @Description Retrieves all sat ranges and attributes in a given utxo
// @Tags ordx.exotic
// @Produce json
// @Param utxo path string true "utxo"
// @Security Bearer
// @Success 200 {array} SatributeRange "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /exotic/utxo/{utxo} [get]
func (s *Service) getExoticWithUtxo(c *gin.Context) {
	resp := &SatRangeResp{
		BaseResp: define.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	utxo := c.Param("utxo")
	satRanges, err := s.model.GetSatRangeInUtxo(utxo)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = satRanges
	c.JSON(http.StatusOK, resp)
}

// @Summary Retrieves UTXOs which have exotic sat for a given address
// @Description Retrieves UTXOs which have exotic sat for a given address
// @Tags ordx.exotic
// @Produce json
// @Param address path string true "Address"
// @Security Bearer
// @Success 200 {array} SatRangeUtxoResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /exotic/address/{address} [get]
func (s *Service) getExoticUtxos(c *gin.Context) {
	resp := &SatRangeUtxoResp{
		BaseResp: define.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	address := c.Param("address")
	satributeSatList, err := s.model.GetExoticUtxos(address)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = satributeSatList
	c.JSON(http.StatusOK, resp)
}

// @Summary Retrieves available UTXOs
// @Description Get UTXOs in a address and its value is greater than the specific value. If value=0, get all UTXOs
// @Tags ordx
// @Produce json
// @Param address path string true "address"
// @Param value path int64 true "value"
// @Security Bearer
// @Success 200 {array} PlainUtxo "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /utxo/address/{address}/{value} [post]
func (s *Service) getPlainUtxos(c *gin.Context) {
	resp := &PlainUtxosResp{
		BaseResp: define.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Total: 0,
		Data:  nil,
	}

	value, err := strconv.ParseInt(c.Param("value"), 10, 64)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	address := c.Param("address")
	start, err := strconv.Atoi(c.DefaultQuery("start", "0"))
	if err != nil {
		start = 0
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil {
		limit = 0
	}
	availableUtxoList, total, err := s.model.getPlainUtxos(address, value, start, limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Total = total
	resp.Data = availableUtxoList
	c.JSON(http.StatusOK, resp)
}

func (s *Service) getAllUtxos(c *gin.Context) {
	resp := &AllUtxosResp{
		BaseResp: define.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Total:      0,
		PlainUtxos: nil,
		OtherUtxos: nil,
	}

	address := c.Param("address")
	start, err := strconv.Atoi(c.DefaultQuery("start", "0"))
	if err != nil {
		start = 0
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil {
		limit = 0
	}
	PlainUtxos, OtherUtxos, total, err := s.model.getAllUtxos(address, start, limit)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Total = total
	resp.PlainUtxos = PlainUtxos
	resp.OtherUtxos = OtherUtxos
	c.JSON(http.StatusOK, resp)
}

// @Summary getExoticUtxosWithType
// @Description Get UTXOs which is the specific exotic type in a address
// @Tags ordx.exotic
// @Produce json
// @Param address path string true "address"
// @Param type path string true "type"
// @Security Bearer
// @Success 200 {array} SpecificExoticUtxo "List of SpecificExoticUtxo"
// @Failure 401 "Invalid API Key"
// @Router /exotic/address/{address}/{type} [get]
func (s *Service) getExoticUtxosWithType(c *gin.Context) {
	resp := &SpecificExoticUtxoResp{
		BaseResp: define.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	address := c.Param("address")
	typ := c.Param("type")

	satritbuteSatList, err := s.model.GetExoticUtxosWithType(address, typ, 1)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = satritbuteSatList
	c.JSON(http.StatusOK, resp)
}
