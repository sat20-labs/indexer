package bitcoind

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sat20-labs/indexer/share/btclucky"
)

func (s *Service) getBTCLuckyJob(c *gin.Context) {
	resp := btclucky.APIResponse[*btclucky.CompactMiningJob]{
		Code: -1,
		Msg:  "btc lucky template service is not enabled",
	}
	if s.btcLucky == nil || !s.btcLucky.IsReady() {
		c.JSON(http.StatusOK, resp)
		return
	}

	var req btclucky.JobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	job, err := s.btcLucky.CurrentJob(req)
	if err != nil {
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Code = 0
	resp.Msg = "ok"
	resp.Data = job
	c.JSON(http.StatusOK, resp)
}

func (s *Service) submitBTCLuckySolution(c *gin.Context) {
	resp := btclucky.APIResponse[*btclucky.FoundBlockRecord]{
		Code: -1,
		Msg:  "btc lucky template service is not enabled",
	}
	if s.btcLucky == nil || !s.btcLucky.IsReady() {
		c.JSON(http.StatusOK, resp)
		return
	}

	var req btclucky.MiningSolution
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	found, err := s.btcLucky.SubmitSolution(&req)
	if found != nil {
		resp.Data = found
	}
	if err != nil {
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Code = 0
	resp.Msg = "ok"
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
