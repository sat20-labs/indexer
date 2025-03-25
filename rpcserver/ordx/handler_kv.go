package ordx

import (
	"net/http"

	"github.com/gin-gonic/gin"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
)

// include plain sats
func (s *Handle) getkv(c *gin.Context) {
	resp := &rpcwire.GetValueResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	key := c.Param("key")

	result, err := s.model.GetKV(key)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Data = []*rpcwire.KeyValue{result}
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Handle) putKVs(c *gin.Context) {
	resp := &rpcwire.PutKValueResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	var req rpcwire.PutKValueReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	result, err := s.model.PutKVs(req.KValues)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Succeeded = result
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Handle) delKVs(c *gin.Context) {
	resp := &rpcwire.DelKValueResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	var req rpcwire.DelKValueReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	result, err := s.model.DelKVs(req.Keys)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Deleted = result
	}

	c.JSON(http.StatusOK, resp)
}
