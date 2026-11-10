package bitcoind

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/btcsuite/btcd/btcjson"
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
	var request btcjson.Request
	response := btclucky.SubmitResponse{JSONRPC: "2.0"}
	fail := func(status int, code btcjson.RPCErrorCode, err error) {
		response.Error = btcjson.NewRPCError(code, err.Error())
		c.JSON(status, response)
	}
	payload, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 2*wire.MaxBlockPayload+1024))
	if err != nil {
		fail(http.StatusBadRequest, btcjson.ErrRPCInvalidRequest.Code, err)
		return
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		fail(http.StatusOK, btcjson.ErrRPCParse.Code, err)
		return
	}
	if request.Jsonrpc != btcjson.RpcVersion2 || !btcjson.IsValidIDType(request.ID) {
		fail(http.StatusOK, btcjson.ErrRPCInvalidRequest.Code, errors.New("invalid JSON-RPC 2.0 request"))
		return
	}
	response.ID = request.ID
	if request.Method != "submitblock" {
		fail(http.StatusOK, btcjson.ErrRPCMethodNotFound.Code, errors.New("method not found"))
		return
	}
	if len(request.Params) < 1 || len(request.Params) > 2 {
		fail(http.StatusOK, btcjson.ErrRPCInvalidParams.Code, errors.New("submitblock requires block hex"))
		return
	}
	var blockHex string
	if err := json.Unmarshal(request.Params[0], &blockHex); err != nil || blockHex == "" {
		fail(http.StatusOK, btcjson.ErrRPCInvalidParams.Code, errors.New("block data must be a hex string"))
		return
	}
	raw, err := hex.DecodeString(blockHex)
	if err != nil {
		fail(http.StatusOK, btcjson.ErrRPCDeserialization, err)
		return
	}
	if s.btcLucky == nil {
		fail(http.StatusServiceUnavailable, btcjson.ErrRPCInWarmup, errors.New("block submission unavailable"))
		return
	}
	record, err := s.btcLucky.SubmitBlock(raw)
	if err != nil {
		if errors.Is(err, btclucky.ErrCoreUnavailable) {
			fail(http.StatusServiceUnavailable, btcjson.ErrRPCInWarmup, err)
			return
		}
		var rpcError *btcjson.RPCError
		if errors.As(err, &rpcError) {
			fail(http.StatusOK, rpcError.Code, rpcError)
			return
		}
		if record == nil {
			fail(http.StatusOK, btcjson.ErrRPCDeserialization, err)
			return
		}
	}
	response.Result = json.RawMessage("null")
	if record != nil && record.SubmitResult != "accepted" {
		response.Result, _ = json.Marshal(record.SubmitResult)
	}
	c.JSON(http.StatusOK, response)
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
