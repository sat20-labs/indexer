package ordx

import (
	"net/http"

	"github.com/gin-gonic/gin"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
)

func (s *Handle) getNonce(c *gin.Context) {
	resp := &rpcwire.GetNonceResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	var req rpcwire.GetNonceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}


	result, err := s.model.GetNonce(&req)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Nonce = result
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Handle) getkv(c *gin.Context) {
	resp := &rpcwire.GetValueResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}

	key := c.Param("key")

	result, err := s.model.GetKV(key)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Value = result
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

	result, err := s.model.PutKVs(&req)
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

	result, err := s.model.DelKVs(&req)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	} else {
		resp.Deleted = result
	}

	c.JSON(http.StatusOK, resp)
}
