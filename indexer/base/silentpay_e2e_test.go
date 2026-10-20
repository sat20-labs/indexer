package base_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/fxamacker/cbor/v2"
	"github.com/gin-gonic/gin"
	"github.com/sat20-labs/indexer/common"
	baseindex "github.com/sat20-labs/indexer/indexer/base"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
	rpcbase "github.com/sat20-labs/indexer/rpcserver/base"
	rpcbitcoin "github.com/sat20-labs/indexer/rpcserver/bitcoind"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
	baseapi "github.com/sat20-labs/indexer/share/base_indexer"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

// Use the standard Base RPC Service, not a SilentPay-only router.
// The embedded legacy interface is unused by these SP-only HTTP tests.
type spE2EHTTPIndexer struct {
	baseapi.Indexer
	rpcbase.SilentPayReader
}

func spHTTP(t *testing.T, b *baseindex.BaseIndexer) *httptest.Server {
	t.Helper()
	router := gin.New()
	rpcbase.NewService(&spE2EHTTPIndexer{SilentPayReader: b}).InitRouter(router, "/test")
	(&rpcbitcoin.Service{}).InitRouter(router, "/test")
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server
}

func spRequest(t *testing.T, server *httptest.Server, method, path string, body, output any, wantStatus int) []byte {
	t.Helper()
	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil { t.Fatal(err) }
	}
	request, err := http.NewRequest(method, server.URL+"/test"+path, bytes.NewReader(data))
	if err != nil { t.Fatal(err) }
	if body != nil { request.Header.Set("Content-Type", "application/json") }
	client := &http.Client{Timeout: 10*time.Second}
	response, err := client.Do(request)
	if err != nil { t.Fatal(err) }
	defer response.Body.Close()
	data, err = io.ReadAll(io.LimitReader(response.Body, 16*1024*1024+1))
	if err != nil { t.Fatal(err) }
	if response.StatusCode != wantStatus {
		t.Fatalf("%s %s: status=%d wanted=%d body=%s", method, path, response.StatusCode, wantStatus, data)
	}
	if output != nil {
		if response.Header.Get("Content-Type") == "application/cbor" { err = cbor.Unmarshal(data, output) } else { err = json.Unmarshal(data, output) }
		if err != nil { t.Fatalf("response decoding %s: %v", path, err) }
	}
	return data
}

func liveSnapshot(t *testing.T, server *httptest.Server) baseindex.SPSnapshot {
	t.Helper()
	var meta baseindex.SPSnapshot
	spRequest(t, server, http.MethodGet, "/v1/sp/live-snapshot", nil, &meta, http.StatusOK)
	if meta.SnapshotID == "" || meta.ExpiresAt <= time.Now().Unix() { t.Fatal("invalid snapshot metadata") }
	return meta
}

func allLive(t *testing.T, server *httptest.Server, snapshot baseindex.SPSnapshot) []baseindex.SPBlock {
	t.Helper()
	cursor := ""
	blocks := make([]baseindex.SPBlock, 0)
	seen := make(map[string]bool)
	for pages := 0; pages < 20; pages++ {
		var page baseindex.SPScanPage
		spRequest(t, server, http.MethodGet, "/v1/sp/scan-data?mode=live&limit=1&snapshot_id="+snapshot.SnapshotID+"&cursor="+url.QueryEscape(cursor), nil, &page, http.StatusOK)
		if page.Anchor != snapshot.Anchor || page.SnapshotID != snapshot.SnapshotID { t.Fatal("pagination changed snapshot") }
		blocks = append(blocks, page.Blocks...)
		if page.Done { return blocks }
		if page.NextCursor == "" || seen[page.NextCursor] { t.Fatal("pagination stuck") }
		seen[page.NextCursor] = true
		cursor = page.NextCursor
	}
	t.Fatal("pagination did not finish")
	return nil
}

func findScanTx(t *testing.T, blocks []baseindex.SPBlock, index uint32) baseindex.SPTransaction {
	t.Helper()
	for _, block := range blocks {
		for _, tx := range block.Transactions { if tx.TxIndex == index { return tx } }
	}
	t.Fatalf("missing candidate index %d", index)
	return baseindex.SPTransaction{}
}

func TestSilentPayE2EReceiveLiveRecoverAndSpend(t *testing.T) {
	chain := newSPTestChain(t)
	scan, spend := e2eKey(31), e2eKey(32)
	payment, keys := e2ePayment(t, chain.chain[1].Transactions[0], scan, spend, 2)
	second, _ := e2ePayment(t, chain.chain[2].Transactions[0], scan, spend, 1)
	creation := chain.appendBlock(t, payment, second)
	b, _ := e2eBase(t, chain)
	e2eSync(t, b, true)
	server := spHTTP(t, b)
	var status baseindex.SPStatus
	spRequest(t, server, http.MethodGet, "/v1/sp/status", nil, &status, http.StatusOK)
	if !status.DataReady || status.BaseAnchor.Height != 102 { t.Fatalf("status: %+v", status) }
	chain.mu.RLock()
	statusRawTxCalls := chain.rawTxCalls
	chain.mu.RUnlock()
	if statusRawTxCalls != 0 { t.Fatalf("status performed %d unnecessary blocking bitcoind transaction probes", statusRawTxCalls) }
	before := liveSnapshot(t, server)
	initial := allLive(t, server, before)
	if len(initial) != 2 { t.Fatalf("limit=1 did not enumerate both transactions: %+v", initial) }
	candidate := findScanTx(t, initial, 1)
	if candidate.TxID != "" || len(candidate.P2TROutputs) != 2 || candidate.P2TROutputs[1].Vout != 2 { t.Fatal("invalid public scan payload") }
	if len(e2eRecover(t, candidate, e2eKey(33), spend)) != 0 { t.Fatal("wrong scan key identified funds") }

	// Spend k=0 in a noncandidate transaction (no P2TR output).
	chain.appendBlock(t, e2eSpendSP(t, payment, 0, keys[0]))
	e2eSync(t, b, true)
	after := liveSnapshot(t, server)
	oldTx := findScanTx(t, allLive(t, server, before), 1)
	if !bytes.Equal(oldTx.UnspentBitmap, []byte{3}) { t.Fatal("old Live snapshot changed") }
	newTx := findScanTx(t, allLive(t, server, after), 1)
	if !bytes.Equal(newTx.UnspentBitmap, []byte{2}) || len(newTx.P2TROutputs) != 2 { t.Fatal("spent k=0 removed from scan data") }
	found := e2eRecover(t, newTx, scan, spend)
	if len(found) != 2 || found[2] == nil { t.Fatal("recovery missed k=1 after k=0 spend") }

	position := baseindex.SPPosition{Height: 102, BlockHash: creation.BlockHash().String(), TxIndex: 1, Vout: 0, ExpectedTxID: payment.TxHash().String()}
	var states struct { Anchor baseindex.SPAnchor `json:"anchor"`; Data []baseindex.SPPositionResult `json:"data"` }
	spRequest(t, server, http.MethodPost, "/v1/sp/outpoints/query", map[string]any{"snapshot_id": before.SnapshotID, "positions": []baseindex.SPPosition{position}}, &states, http.StatusOK)
	if states.Anchor != before.Anchor || len(states.Data) != 1 || states.Data[0].State != "unspent" { t.Fatalf("historical status drift: %+v", states) }
	spRequest(t, server, http.MethodPost, "/v1/sp/outpoints/query", map[string]any{"snapshot_id": after.SnapshotID, "positions": []baseindex.SPPosition{position}}, &states, http.StatusOK)
	if states.Anchor != after.Anchor || states.Data[0].State != "spent" { t.Fatal("new status not updated") }

	var lookup struct { Data []baseindex.SPPositionResult `json:"data"` }
	spRequest(t, server, http.MethodPost, "/v1/sp/txids-by-position", []baseindex.SPPosition{position}, &lookup, http.StatusOK)
	if len(lookup.Data) != 1 || lookup.Data[0].TxID != payment.TxHash().String() { t.Fatal("position lookup failed") }
	var raw rpcwire.BitcoinRawTxResp
	spRequest(t, server, http.MethodPost, "/v3/bitcoin/rawtx/batch", map[string]any{"txids": []string{lookup.Data[0].TxID, (chainhash.Hash{}).String()}}, &raw, http.StatusOK)
	if raw.Code != 0 || len(raw.Data) != 2 || raw.Data[0].RawTx == "" || raw.Data[1].Error == "" { t.Fatal("rawtx per-item errors lost") }
	verified, err := bitcoin_rpc.DecodeMsgTx(raw.Data[0].RawTx)
	if err != nil || verified.TxHash() != payment.TxHash() || verified.TxOut[2].Value != 20000 { t.Fatal("matched transaction verification failed") }
	// The spending key comes from the public scan payload and wallet secrets,
	// not the sender fixture. Execute validates the actual Taproot signature.
	lastSpend := e2eSpendSP(t, verified, 2, found[2])
	chain.appendBlock(t, lastSpend)
	e2eSync(t, b, true)
	final := liveSnapshot(t, server)
	remaining := allLive(t, server, final)
	if len(remaining) != 1 || remaining[0].Transactions[0].TxIndex != 2 { t.Fatal("last spend failed to remove Live transaction") }
	var archive baseindex.SPScanPage
	spRequest(t, server, http.MethodGet, "/v1/sp/scan-data?mode=archive&from_height=102&to_height=104", nil, &archive, http.StatusOK)
	if !archive.Done || len(archive.Blocks) != 3 || len(archive.Blocks[0].Transactions) != 2 { t.Fatal("Archive lost fully spent history or empty blocks") }
	if !b.CheckSelf() { t.Fatal("Base CheckSelf rejected valid indexed fixture") }
	spRequest(t, server, http.MethodGet, "/v1/sp/scan-data?mode=live&limit=501&snapshot_id="+final.SnapshotID, nil, nil, http.StatusBadRequest)
	spRequest(t, server, http.MethodPost, "/v1/sp/outpoints/query", map[string]any{"positions": []baseindex.SPPosition{position}}, nil, http.StatusBadRequest)
	b.CloseSilentPayViews()
	spRequest(t, server, http.MethodGet, "/v1/sp/scan-data?mode=live&snapshot_id="+final.SnapshotID, nil, nil, http.StatusConflict)
}

func TestSilentPayE2EReorgReloadRestoresLive(t *testing.T) {
	chain := newSPTestChain(t)
	payment, keys := e2ePayment(t, chain.chain[1].Transactions[0], e2eKey(31), e2eKey(32), 1)
	chain.appendBlock(t, payment)
	b, _ := e2eBase(t, chain)
	e2eSync(t, b, true)
	snapshot, err := b.SilentPayCreateSnapshot(context.Background())
	if err != nil { t.Fatal(err) }
	chain.appendBlock(t, e2eSpendSP(t, payment, 0, keys[0]))
	e2eSync(t, b, false) // old branch is inside Base's uncommitted history window
	chain.truncate(102)
	chain.appendBlock(t)
	chain.appendBlock(t)
	if fork := b.SyncToChainTip(make(chan struct{})); fork != 103 { t.Fatalf("expected reorg at 103, got %d", fork) }
	b.CloseSilentPayViews()
	b.Init() // same operation used by the existing manager after DB reopen
	b.SetBlockCallback(func(*common.Block, []*common.Range) {})
	e2eSync(t, b, true)
	if _, err := b.SilentPayScan(context.Background(), baseindex.SPScanRequest{Mode: "live", SnapshotID: snapshot.SnapshotID, ToHeight: -1}); !errors.Is(err, baseindex.ErrSPSnapshotExpired) { t.Fatalf("old snapshot survived reload: %v", err) }
	fresh, err := b.SilentPayCreateSnapshot(context.Background())
	if err != nil { t.Fatal(err) }
	page, err := b.SilentPayScan(context.Background(), baseindex.SPScanRequest{Mode: "live", SnapshotID: fresh.SnapshotID, ToHeight: -1})
	if err != nil || len(page.Blocks) != 1 || !bytes.Equal(page.Blocks[0].Transactions[0].UnspentBitmap, []byte{1}) { t.Fatalf("reorg lost original live output: %+v %v", page, err) }
	if !b.CheckSelf() { t.Fatal("reorg replay CheckSelf failed") }
}

func e2eSPDiskKey(prefix string, height, index uint32) []byte {
	key := make([]byte, len(prefix)+8)
	copy(key, prefix)
	binary.BigEndian.PutUint32(key[len(prefix):], height)
	binary.BigEndian.PutUint32(key[len(prefix)+4:], index)
	return key
}

func TestSilentPayE2ECheckSelfDetectsCorruptionWithoutRepair(t *testing.T) {
	chain := newSPTestChain(t)
	payment, _ := e2ePayment(t, chain.chain[1].Transactions[0], e2eKey(31), e2eKey(32), 2)
	chain.appendBlock(t, payment)
	b, kv := e2eBase(t, chain)
	e2eSync(t, b, true)
	if !b.CheckSelf() { t.Fatal("baseline invalid") }
	archiveKey, liveKey := e2eSPDiskKey("sp/a/", 102, 1), e2eSPDiskKey("sp/l/", 102, 1)
	original, err := kv.Read(archiveKey)
	if err != nil { t.Fatal(err) }
	live, err := kv.Read(liveKey)
	if err != nil { t.Fatal(err) }
	tests := []struct { name string; mutate func() }{
		{"missing_archive_and_live", func() { if err := kv.Delete(archiveKey); err != nil { t.Fatal(err) }; if err := kv.Delete(liveKey); err != nil { t.Fatal(err) } }},
		{"wrong_live_bit", func() { if err := kv.Write(liveKey, []byte{2}); err != nil { t.Fatal(err) } }},
		{"malformed_tweak", func() { bad := append([]byte(nil), original...); bad[33] = 4; if err := kv.Write(archiveKey, bad); err != nil { t.Fatal(err) } }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.mutate()
			beforeA, errA := kv.Read(archiveKey)
			beforeL, errL := kv.Read(liveKey)
			if test.name == "missing_archive_and_live" {
				expectSilentPayPanic(t, func() {
					_, _ = b.SilentPayScan(context.Background(), baseindex.SPScanRequest{Mode: "archive", FromHeight: 102, ToHeight: 102})
				})
			}
			expectSilentPayPanic(t, func() { b.CheckSelf() })
			afterA, afterErrA := kv.Read(archiveKey)
			afterL, afterErrL := kv.Read(liveKey)
			if !bytes.Equal(beforeA, afterA) || !bytes.Equal(beforeL, afterL) || (errA == nil) != (afterErrA == nil) || (errL == nil) != (afterErrL == nil) { t.Fatal("CheckSelf modified damaged records") }
			if err := kv.Write(archiveKey, original); err != nil { t.Fatal(err) }
			if err := kv.Write(liveKey, live); err != nil { t.Fatal(err) }
		})
	}
	chain.mu.Lock(); chain.failRaw = true; chain.mu.Unlock()
	if !b.CheckSelf() { t.Fatal("DB-only self-check unexpectedly read missing Bitcoin RPC evidence") }
	chain.mu.Lock(); chain.failRaw = false; chain.mu.Unlock()
	if !b.CheckSelf() { t.Fatal("restored fixture failed self-check") }
}

func TestSilentPayE2EDatabaseReopen(t *testing.T) {
	chain := newSPTestChain(t)
	payment, _ := e2ePayment(t, chain.chain[1].Transactions[0], e2eKey(31), e2eKey(32), 1)
	chain.appendBlock(t, payment)
	oldRPC, oldChain := bitcoin_rpc.ShareBitconRpc, common.CHAIN
	bitcoin_rpc.ShareBitconRpc, common.CHAIN = chain, "testnet"
	defer func() { bitcoin_rpc.ShareBitconRpc, common.CHAIN = oldRPC, oldChain }()
	path := filepath.Join(t.TempDir(), "base")
	var previousID string
	for run := 0; run < 2; run++ {
		kv := indexdb.NewKVDBWithCache(path, 1)
		if kv == nil { t.Fatal("open database") }
		b := baseindex.NewBaseIndexer(kv, &chaincfg.RegressionNetParams, 0, 1000)
		b.Init()
		b.SetBlockCallback(func(*common.Block, []*common.Range) {})
		if run == 0 { e2eSync(t, b, true) }
		meta, err := b.SilentPayCreateSnapshot(context.Background())
		if err != nil { b.CloseSilentPayViews(); kv.Close(); t.Fatal(err) }
		if run == 1 {
			if _, err := b.SilentPayScan(context.Background(), baseindex.SPScanRequest{Mode: "live", SnapshotID: previousID, ToHeight: -1}); !errors.Is(err, baseindex.ErrSPSnapshotExpired) { t.Fatal("restart accepted previous snapshot token") }
			page, err := b.SilentPayScan(context.Background(), baseindex.SPScanRequest{Mode: "live", SnapshotID: meta.SnapshotID, ToHeight: -1})
			if err != nil || len(page.Blocks) != 1 { t.Fatalf("persistent Live missing: %+v %v", page, err) }
			var recovered map[uint32]*btcec.PrivateKey = e2eRecover(t, page.Blocks[0].Transactions[0], e2eKey(31), e2eKey(32))
			if recovered[0] == nil { t.Fatal("reopened wallet scan failed") }
			e2eSpendSP(t, payment, 0, recovered[0])
		}
		previousID = meta.SnapshotID
		b.CloseSilentPayViews()
		if err := kv.Close(); err != nil { t.Fatal(err) }
	}
}

func TestSilentPayE2ESameBlockSpendAndFullySpentHistoryAudit(t *testing.T) {
	chain := newSPTestChain(t)
	payment, keys := e2ePayment(t, chain.chain[1].Transactions[0], e2eKey(31), e2eKey(32), 2)
	transactions := []*wire.MsgTx{payment, e2eSpendSP(t, payment, 0, keys[0]), e2eSpendSP(t, payment, 2, keys[1])}
	creation := chain.appendBlock(t, transactions...)
	b, kv := e2eBase(t, chain)
	e2eSync(t, b, true)
	meta, err := b.SilentPayCreateSnapshot(context.Background())
	if err != nil { t.Fatal(err) }
	page, err := b.SilentPayScan(context.Background(), baseindex.SPScanRequest{Mode: "live", SnapshotID: meta.SnapshotID, ToHeight: -1})
	if err != nil || !page.Done || len(page.Blocks) != 0 { t.Fatalf("same-block fully spent transaction remained Live: %+v %v", page, err) }
	if !b.CheckSelf() { t.Fatal("same-block spending failed full self-check") }
	// DB-only audit cannot distinguish a fully spent SP candidate omitted
	// from both the Archive and directory from an ordinary BTC transaction.
	// This is an explicitly accepted limitation of avoiding raw Bitcoin RPC.
	archiveKey := e2eSPDiskKey("sp/a/", 102, 1)
	if err := kv.Delete(archiveKey); err != nil { t.Fatal(err) }
	blockKey := make([]byte, len("sp/b/")+4)
	copy(blockKey, "sp/b/")
	binary.BigEndian.PutUint32(blockKey[len("sp/b/"):], 102)
	hash := creation.BlockHash()
	emptyDirectory := append(append([]byte{1}, hash[:]...), 0)
	if err := kv.Write(blockKey, emptyDirectory); err != nil { t.Fatal(err) }
	if !b.CheckSelf() { t.Fatal("DB-only audit incorrectly claimed independent knowledge of omitted candidate") }
	if _, err := kv.Read(archiveKey); !errors.Is(err, common.ErrKeyNotFound) { t.Fatalf("audit modified disk data: %v", err) }
}

func expectSilentPayPanic(t *testing.T, run func()) {
	t.Helper()
	panicked := false
	func() {
		defer func() { panicked = recover() != nil }()
		run()
	}()
	if !panicked { t.Fatal("expected a fatal indexer panic") }
}


// Exercise the same Base.Clone(true) -> Base.Subtract(frozen) -> frozen.UpdateDB
// -> SetSyncStats -> Base.UpdateDB ordering used by IndexerMgr's buffered
// commit path. SP must retain both the earlier Archive row and later Live bit.
func TestSilentPayE2EBaseBufferedCloneSubtractCommit(t *testing.T) {
	chain := newSPTestChain(t)
	payment, keys := e2ePayment(t, chain.chain[1].Transactions[0], e2eKey(31), e2eKey(32), 2)
	chain.appendBlock(t, payment) // height 102
	b, _ := e2eBase(t, chain)
	e2eSync(t, b, false)
	frozen := b.Clone(true)

	chain.appendBlock(t, e2eSpendSP(t, payment, 0, keys[0])) // height 103
	e2eSync(t, b, false)
	b.Subtract(frozen)
	frozen.UpdateDB()
	b.SetSyncStats(frozen.GetSyncStats())
	b.UpdateDB()
	// Fixed-height build shutdown can issue a harmless duplicate flush.
	b.UpdateDB()

	snapshot, err := b.SilentPayCreateSnapshot(context.Background())
	if err != nil { t.Fatal(err) }
	live, err := b.SilentPayScan(context.Background(), baseindex.SPScanRequest{
		Mode: "live", SnapshotID: snapshot.SnapshotID, ToHeight: -1,
	})
	if err != nil || !live.Done || len(live.Blocks) != 1 || len(live.Blocks[0].Transactions) != 1 {
		t.Fatalf("buffered commit lost Live candidate: %+v, %v", live, err)
	}
	tx := live.Blocks[0].Transactions[0]
	if live.Blocks[0].Height != 102 || tx.TxIndex != 1 || !bytes.Equal(tx.UnspentBitmap, []byte{2}) || len(tx.P2TROutputs) != 2 {
		t.Fatalf("buffered Base commit produced incorrect Live state: %+v", tx)
	}
	archive, err := b.SilentPayScan(context.Background(), baseindex.SPScanRequest{
		Mode: "archive", FromHeight: 102, ToHeight: 103,
	})
	if err != nil || !archive.Done || len(archive.Blocks) != 2 || len(archive.Blocks[0].Transactions) != 1 {
		t.Fatalf("buffered commit lost historical Archive: %+v, %v", archive, err)
	}
	if !b.CheckSelf() { t.Fatal("buffered commit Base CheckSelf failed") }
}
