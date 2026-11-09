package bitcoind

import (
	"errors"
	"io"
	"net/http"

	"github.com/btcsuite/btcd/wire"
	"github.com/gin-gonic/gin"
	"github.com/sat20-labs/indexer/share/btclucky"
)

func (s *Service) getBTCLuckyTemplate(c *gin.Context) {
	if s.btcLucky == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": -1, "msg": "template service unavailable"})
		return
	}
	body, tag, err := s.btcLucky.TemplateBytes()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": -1, "msg": err.Error()})
		return
	}
	c.Header("ETag", tag)
	c.Header("Cache-Control", "no-cache")
	if c.GetHeader("If-None-Match") == tag {
		c.Status(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, "application/json", body)
}

func (s *Service) submitBitcoinBlock(c *gin.Context) {
	resp := btclucky.APIResponse[*btclucky.FoundBlockRecord]{Code: -1}
	if s.btcLucky == nil {
		resp.Msg = "block submission unavailable"
		c.JSON(http.StatusServiceUnavailable, resp)
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, wire.MaxBlockPayload))
	if err != nil {
		resp.Msg = err.Error()
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	resp.Data, err = s.btcLucky.SubmitBlock(raw)
	if err != nil {
		resp.Msg = err.Error()
		status := http.StatusOK
		if errors.Is(err, btclucky.ErrCoreUnavailable) {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, resp)
		return
	}
	resp.Code, resp.Msg = 0, "ok"
	c.JSON(http.StatusOK, resp)
}

func (s *Service) getBTCLuckyInfo(c *gin.Context) {
	resp := btclucky.APIResponse[btclucky.InfoResponse]{
		Code: -1,
		Msg:  "btc lucky template service is not enabled",
	}
	if s.btcLucky == nil {
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Code = 0
	resp.Msg = "ok"
	resp.Data = btclucky.InfoResponse{
		Service:     s.btcLucky.Status(),
		FoundBlocks: s.btcLucky.FoundBlocks(),
	}
	c.JSON(http.StatusOK, resp)
}
