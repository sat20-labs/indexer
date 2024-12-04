package extension

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sat20-labs/indexer/common"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
)

func (s *Service) runes_list(c *gin.Context) {
	resp := &RunesListResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: &RuneBalanceList{},
	}

	req := AddressRangeReq{
		AddressReq: rpcwire.AddressReq{},
		RangeReq:   RangeReq{Cursor: 0, Size: 100},
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	common.Log.Debugf("address: %v, cursor: %v, size: %v", req.Address, req.Cursor, req.Size)
	c.JSON(http.StatusOK, resp)
}

func (s *Service) runes_utxoList(c *gin.Context) {
	resp := &RuneUtxosResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	address := c.Query("address")
	runeid := c.Query("runeid")
	common.Log.Debugf("address: %v, runeid: %v", address, runeid)
	c.JSON(http.StatusOK, resp)
}

func (s *Service) runes_tokenSummary(c *gin.Context) {
	resp := &AddressRunesTokenSummaryResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}
	address := c.Query("address")
	runeid := c.Query("runeid")
	common.Log.Debugf("address: %v, runeid: %v", address, runeid)
	c.JSON(http.StatusOK, resp)
}
