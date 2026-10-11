package base

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	baseindex "github.com/sat20-labs/indexer/indexer/base"
	baseapi "github.com/sat20-labs/indexer/share/base_indexer"
)

type spRecoveryTestReader struct {
	calls int
}

// Exercise the standard Service router using an SP-capable test Indexer.
type spRecoveryTestIndexer struct {
	baseapi.Indexer
	SilentPayReader
}

func (r *spRecoveryTestReader) SilentPayStatus(context.Context) (baseindex.SPStatus, error) {
	r.calls++
	if r.calls == 1 {
		panic(baseindex.ErrSPCorrupt)
	}
	return baseindex.SPStatus{DataReady: true}, nil
}
func (*spRecoveryTestReader) SilentPayCreateSnapshot(context.Context) (baseindex.SPSnapshot, error) {
	return baseindex.SPSnapshot{}, baseindex.ErrSPNotReady
}
func (*spRecoveryTestReader) SilentPayScan(context.Context, baseindex.SPScanRequest) (baseindex.SPScanPage, error) {
	return baseindex.SPScanPage{}, baseindex.ErrSPNotReady
}
func (*spRecoveryTestReader) SilentPayLookup(context.Context, []baseindex.SPPosition) ([]baseindex.SPPositionResult, error) {
	return nil, baseindex.ErrSPNotReady
}
func (*spRecoveryTestReader) SilentPayOutpoints(context.Context, string, []baseindex.SPPosition) (baseindex.SPAnchor, []baseindex.SPPositionResult, error) {
	return baseindex.SPAnchor{}, nil, baseindex.ErrSPNotReady
}

// The HTTP layer keeps Gin's standard Recovery behavior: a request-scoped
// index read panic returns 500, but it must not crash or poison the server.
// This does not change BaseIndexer.UpdateDB's fatal persistence errors.
func TestSilentPayHTTPRecoveryKeepsIndexerRunning(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	reader := &spRecoveryTestReader{}
	router := gin.Default()
	NewService(&spRecoveryTestIndexer{SilentPayReader: reader}).InitRouter(router, "/test")

	url := "/test/v1/sp/status"
	bad := httptest.NewRecorder()
	router.ServeHTTP(bad, httptest.NewRequest(http.MethodGet, url, nil))
	if bad.Code != http.StatusInternalServerError {
		t.Fatalf("recovered request status=%d, want 500", bad.Code)
	}

	good := httptest.NewRecorder()
	router.ServeHTTP(good, httptest.NewRequest(http.MethodGet, url, nil))
	if good.Code != http.StatusOK || reader.calls != 2 {
		t.Fatalf("server did not continue after request panic: status=%d calls=%d", good.Code, reader.calls)
	}
	var status baseindex.SPStatus
	if err := json.Unmarshal(good.Body.Bytes(), &status); err != nil || !status.DataReady {
		t.Fatalf("subsequent SP response invalid: %+v, err=%v", status, err)
	}
}
