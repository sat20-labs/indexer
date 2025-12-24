package base

import (
	"net/http"
	"strconv"

	ordxCommon "github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/exotic"
	"github.com/sat20-labs/indexer/rpcserver/wire"

	"github.com/gin-gonic/gin"
)

// @Summary Health Check
// @Description Check the health status of the service
// @Tags ordx
// @Produce json
// @Success 200 {object} wire.HealthStatusResp "Successful response"
// @Router /health [get]
func (s *Service) getHealth(c *gin.Context) {
	rsp := &wire.HealthStatusResp{
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
// @Success 200 {object} wire.SatInfoResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /sat/{sat} [get]
func (s *Service) getSatInfo(c *gin.Context) {
	resp := &wire.SatInfoResp{
		BaseResp: wire.BaseResp{
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


// @Summary Retrieves the supported attributes of a sat
// @Description Retrieves the supported attributes of a sat
// @Tags ordx
// @Produce json
// @Security Bearer
// @Success 200 {array} wire.SatributesResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /info/satributes [get]
func (s *Service) getSatributes(c *gin.Context) {
	resp := &wire.SatributesResp{
		BaseResp: wire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: exotic.SatributeList,
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Retrieves available UTXOs
// @Description Get UTXOs in a address and its value is greater than the specific value. If value=0, get all UTXOs
// @Tags ordx
// @Produce json
// @Param address path string true "address"
// @Param value path int64 true "value"
// @Security Bearer
// @Success 200 {array} wire.PlainUtxosResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /utxo/address/{address}/{value} [post]
func (s *Service) getPlainUtxos(c *gin.Context) {
	resp := &wire.PlainUtxosResp{
		BaseResp: wire.BaseResp{
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
	resp := &wire.AllUtxosResp{
		BaseResp: wire.BaseResp{
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
