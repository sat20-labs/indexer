package base_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/sat20-labs/indexer/common"
	baseindex "github.com/sat20-labs/indexer/indexer/base"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
)

// Emulate the on-disk result of indexing Bitcoin before the SP release:
// Base is complete but there are no keys in the SP namespace. The test uses
// real Base block records and the existing independent Bitcoin block fixture.
func TestSilentPayLegacyBaseReloadSyncAndCheckSelf(t *testing.T) {
	chain := newSPTestChain(t)
	b, kv := e2eBase(t, chain)
	e2eSync(t, b, true)
	if !b.CheckSelf() {
		t.Fatal("original Base fixture failed before simulating legacy data")
	}
	b.CloseSilentPayViews()

	views, ok := kv.(common.SnapshotDB)
	if !ok {
		t.Fatal("test Base DB does not support read views")
	}
	view, err := views.NewReadView()
	if err != nil {
		t.Fatal(err)
	}
	var keys [][]byte
	err = view.Scan(common.ScanOptions{
		Prefix: []byte("sp/"), KeysOnly: true,
	}, func(key, _ []byte) error {
		keys = append(keys, append([]byte(nil), key...))
		return nil
	})
	if closeErr := view.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if err != nil || len(keys) == 0 {
		t.Fatalf("fixture does not contain complete SP history: %v, %d", err, len(keys))
	}
	for _, key := range keys {
		if err := kv.Delete(key); err != nil {
			t.Fatal(err)
		}
	}

	b.Init()
	b.SetBlockCallback(func(*common.Block, []*common.Range) {})
	if _, err := b.SilentPayStatus(context.Background()); !errors.Is(err, baseindex.ErrSPNotInitialized) {
		t.Fatalf("legacy database did not disable SP: %v", err)
	}
	server := spHTTP(t, b)
	requests := []struct {
		method string
		path string
		body any
	}{
		{http.MethodGet, "/v1/sp/status", nil},
		{http.MethodGet, "/v1/sp/live-snapshot", nil},
		{http.MethodGet, "/v1/sp/scan-data?mode=archive", nil},
		{http.MethodPost, "/v1/sp/txids-by-position", []baseindex.SPPosition{{Height: 1, TxIndex: 1}}},
		{http.MethodPost, "/v1/sp/outpoints/query", map[string]any{
			"snapshot_id": "unused", "positions": []baseindex.SPPosition{{Height: 1, TxIndex: 1}},
		}},
	}
	for _, request := range requests {
		var body rpcwire.BaseResp
		spRequest(t, server, request.method, request.path, request.body, &body, http.StatusServiceUnavailable)
		if body.Code != -1 || body.Msg != baseindex.ErrSPNotInitialized.Error() {
			t.Fatalf("%s %s was not marked uninitialized: %+v", request.method, request.path, body)
		}
	}

	// A new real Bitcoin block must still advance Base and remain fully
	// auditable. No missing SP history is manufactured on subsequent commits.
	chain.appendBlock(t)
	e2eSync(t, b, true)
	if !b.CheckSelf() {
		t.Fatal("legacy Base stopped passing CheckSelf after advancing without SP")
	}
	if _, err := kv.Read([]byte("sp/schema")); !errors.Is(err, common.ErrKeyNotFound) {
		t.Fatalf("legacy sync unexpectedly wrote a partial SP schema: %v", err)
	}
	b.Init()
	if _, err := b.SilentPayStatus(context.Background()); !errors.Is(err, baseindex.ErrSPNotInitialized) {
		t.Fatalf("legacy SP was implicitly activated after another restart: %v", err)
	}
}
