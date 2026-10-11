package base

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/gin-gonic/gin"
	baseindex "github.com/sat20-labs/indexer/indexer/base"
	"github.com/sat20-labs/indexer/rpcserver/wire"
)

// SilentPayReader is an optional capability of the existing Base RPC indexer.
// IndexerMgr supplies it under the normal RPC admission barrier. Keeping it
// separate from the large shared Indexer interface avoids changing other
// modules and tests that only use the legacy Base RPC methods.
type SilentPayReader interface {
	SilentPayStatus(context.Context) (baseindex.SPStatus, error)
	SilentPayCreateSnapshot(context.Context) (baseindex.SPSnapshot, error)
	SilentPayScan(context.Context, baseindex.SPScanRequest) (baseindex.SPScanPage, error)
	SilentPayLookup(context.Context, []baseindex.SPPosition) ([]baseindex.SPPositionResult, error)
	SilentPayOutpoints(context.Context, string, []baseindex.SPPosition) (baseindex.SPAnchor, []baseindex.SPPositionResult, error)
}

// spRequest is a short handler helper, not a second service or a Gin
// middleware. Keep the existing four concurrent requests and 30s timeout;
// normal Base RPCs retain their existing behavior.
func (s *Service) spRequest(c *gin.Context) (SilentPayReader, context.Context, func(), bool) {
	c.Header("Cache-Control", "no-store")
	reader, ok := s.model.indexer.(SilentPayReader)
	if !ok {
		s.spFail(c, baseindex.ErrSPNotReady)
		return nil, nil, nil, false
	}
	select {
	case s.spRequests <- struct{}{}:
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		return reader, ctx, func() {
			cancel()
			<-s.spRequests
		}, true
	default:
		s.spFail(c, baseindex.ErrSPSnapshotLimit)
		return nil, nil, nil, false
	}
}

func (s *Service) spFail(c *gin.Context, err error) {
	status, message := http.StatusInternalServerError, "silentpay_read_failed"
	switch {
	case errors.Is(err, baseindex.ErrSPInvalidRequest):
		status, message = http.StatusBadRequest, baseindex.ErrSPInvalidRequest.Error()
	case errors.Is(err, baseindex.ErrSPSnapshotExpired):
		status, message = http.StatusConflict, baseindex.ErrSPSnapshotExpired.Error()
	case errors.Is(err, baseindex.ErrSPSnapshotLimit):
		status, message = http.StatusTooManyRequests, baseindex.ErrSPSnapshotLimit.Error()
	case errors.Is(err, baseindex.ErrSPNotReady):
		status, message = http.StatusServiceUnavailable, "silentpay_not_ready"
	case errors.Is(err, baseindex.ErrSPCorrupt):
		status, message = http.StatusInternalServerError, baseindex.ErrSPCorrupt.Error()
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		status, message = http.StatusRequestTimeout, "silentpay_request_cancelled"
	}
	c.JSON(status, &wire.BaseResp{Code: -1, Msg: message})
}

func (s *Service) getSPStatus(c *gin.Context) {
	reader, ctx, release, ok := s.spRequest(c)
	if !ok {
		return
	}
	defer release()
	data, err := reader.SilentPayStatus(ctx)
	if err != nil {
		s.spFail(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func (s *Service) getSPLiveSnapshot(c *gin.Context) {
	reader, ctx, release, ok := s.spRequest(c)
	if !ok {
		return
	}
	defer release()
	data, err := reader.SilentPayCreateSnapshot(ctx)
	if err != nil {
		s.spFail(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func spQueryInt(c *gin.Context, key string, fallback int) (int, error) {
	value, present := c.GetQuery(key)
	if !present {
		return fallback, nil
	}
	n, err := strconv.ParseUint(value, 10, 32)
	if err != nil || n > 0x7fffffff {
		return 0, baseindex.ErrSPInvalidRequest
	}
	return int(n), nil
}

func (s *Service) getSPScanData(c *gin.Context) {
	reader, ctx, release, ok := s.spRequest(c)
	if !ok {
		return
	}
	defer release()
	from, err := spQueryInt(c, "from_height", 0)
	if err != nil {
		s.spFail(c, err)
		return
	}
	to, err := spQueryInt(c, "to_height", -1)
	if err != nil {
		s.spFail(c, err)
		return
	}
	limit, err := spQueryInt(c, "limit", 100)
	if err != nil || limit == 0 || limit > 500 || len(c.Query("cursor")) > 2048 || len(c.Query("snapshot_id")) > 64 || len(c.Query("anchor_hash")) > 64 {
		s.spFail(c, baseindex.ErrSPInvalidRequest)
		return
	}
	format := c.DefaultQuery("format", "cbor")
	if format != "json" && format != "cbor" {
		s.spFail(c, baseindex.ErrSPInvalidRequest)
		return
	}
	page, err := reader.SilentPayScan(ctx, baseindex.SPScanRequest{
		Mode: c.Query("mode"), SnapshotID: c.Query("snapshot_id"),
		FromHeight: from, ToHeight: to, AnchorHash: c.Query("anchor_hash"),
		Cursor: c.Query("cursor"), Limit: limit,
	})
	if err != nil {
		s.spFail(c, err)
		return
	}
	// Encode before emitting bytes: a failed read must never appear complete.
	// CBOR keeps key material binary; JSON debugging encodes bytes as base64.
	var payload []byte
	media := "application/cbor"
	if format == "json" {
		media = "application/json"
		payload, err = json.Marshal(page)
	} else {
		payload, err = cbor.Marshal(page)
	}
	if err != nil {
		s.spFail(c, fmt.Errorf("%w: scan response encoding: %v", baseindex.ErrSPCorrupt, err))
		return
	}
	if len(payload) > 16*1024*1024 {
		// A client-requested page exceeding the HTTP limit is not index
		// corruption and must not crash the service.
		s.spFail(c, baseindex.ErrSPInvalidRequest)
		return
	}
	c.Data(http.StatusOK, media, payload)
}

func spBindPositions(c *gin.Context, target any) error {
	if !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return baseindex.ErrSPInvalidRequest
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 256*1024)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return baseindex.ErrSPInvalidRequest
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return baseindex.ErrSPInvalidRequest
	}
	return nil
}

func (s *Service) getSPTxIDsByPosition(c *gin.Context) {
	reader, ctx, release, ok := s.spRequest(c)
	if !ok {
		return
	}
	defer release()
	var positions []baseindex.SPPosition
	if err := spBindPositions(c, &positions); err != nil || len(positions) == 0 || len(positions) > 500 {
		s.spFail(c, baseindex.ErrSPInvalidRequest)
		return
	}
	data, err := reader.SilentPayLookup(ctx, positions)
	if err != nil {
		s.spFail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": data})
}

func (s *Service) getSPOutpoints(c *gin.Context) {
	reader, ctx, release, ok := s.spRequest(c)
	if !ok {
		return
	}
	defer release()
	var request struct {
		SnapshotID string `json:"snapshot_id"`
		Positions []baseindex.SPPosition `json:"positions"`
	}
	if err := spBindPositions(c, &request); err != nil || request.SnapshotID == "" || len(request.SnapshotID) > 64 || len(request.Positions) == 0 || len(request.Positions) > 500 {
		s.spFail(c, baseindex.ErrSPInvalidRequest)
		return
	}
	anchor, data, err := reader.SilentPayOutpoints(ctx, request.SnapshotID, request.Positions)
	if err != nil {
		s.spFail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "anchor": anchor, "data": data})
}
