package bitcoind

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	bitcoindrpc "github.com/OLProtocol/go-bitcoind"
	"github.com/gin-gonic/gin"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

type bitcoinEvidenceRPCStub struct {
	txid    string
	block   string
	script  string
	unspent bool
}

func (s *bitcoinEvidenceRPCStub) TestTx([]string) ([]bitcoindrpc.TransactionTestResult, error) {
	return nil, nil
}
func (s *bitcoinEvidenceRPCStub) SendTx(string) (string, error) { return `"` + s.txid + `"`, nil }
func (s *bitcoinEvidenceRPCStub) GetTx(string) (*bitcoindrpc.RawTransaction, error) {
	return &bitcoindrpc.RawTransaction{
		Txid: s.txid, BlockHash: s.block, Blocktime: 1_700_000_000, Confirmations: 3,
		Vout: []bitcoindrpc.Vout{{Value: 0.00001234, N: 0, ScriptPubKey: bitcoindrpc.ScriptPubKey{Hex: s.script}}},
	}, nil
}
func (s *bitcoinEvidenceRPCStub) GetRawTx(string) (string, error) { return "02000000000000000000", nil }
func (s *bitcoinEvidenceRPCStub) GetTxOut(string, uint32, bool) (*bitcoindrpc.UTransactionOut, error) {
	if !s.unspent {
		return nil, nil
	}
	return &bitcoindrpc.UTransactionOut{
		Bestblock: s.block, Confirmations: 3, Value: 0.00001234,
		ScriptPubKey: bitcoindrpc.ScriptPubKey{Hex: s.script},
	}, nil
}
func (s *bitcoinEvidenceRPCStub) GetBlockCount() (uint64, error)      { return 12, nil }
func (s *bitcoinEvidenceRPCStub) GetBestBlockHash() (string, error)   { return s.block, nil }
func (s *bitcoinEvidenceRPCStub) GetBlockHash(uint64) (string, error) { return s.block, nil }
func (s *bitcoinEvidenceRPCStub) GetRawBlock(string) (string, error)  { return "block", nil }
func (s *bitcoinEvidenceRPCStub) GetBlockHeader(string) (*bitcoindrpc.BlockHeader, error) {
	return &bitcoindrpc.BlockHeader{
		Hash: s.block, Height: 12, Confirmations: 3, Merkleroot: "merkle", Time: 1_700_000_000,
		Mediantime: 1_699_999_000, Chainwork: "work", Previousblockhash: "previous",
	}, nil
}
func (s *bitcoinEvidenceRPCStub) GetMemPoolEntry(string) (*bitcoindrpc.MemPoolEntry, error) {
	return &bitcoindrpc.MemPoolEntry{}, nil
}
func (s *bitcoinEvidenceRPCStub) GetMemPool() ([]string, error) { return nil, nil }
func (s *bitcoinEvidenceRPCStub) EstimateSmartFeeWithMode(blocks int, _ string) (*bitcoindrpc.EstimateSmartFeeResult, error) {
	return &bitcoindrpc.EstimateSmartFeeResult{FeeRate: float64(7-blocks) / 100000}, nil
}

func TestParseEvidenceOutpoint(t *testing.T) {
	txid := "0000000000000000000000000000000000000000000000000000000000000001"
	gotTxID, gotVout, err := parseEvidenceOutpoint(txid + ":7")
	if err != nil {
		t.Fatal(err)
	}
	if gotTxID != txid || gotVout != 7 {
		t.Fatalf("got %s:%d", gotTxID, gotVout)
	}
	for _, invalid := range []string{"", txid, "bad:0", txid + ":-1", txid + ":4294967296"} {
		if _, _, err := parseEvidenceOutpoint(invalid); err == nil {
			t.Fatalf("accepted invalid outpoint %q", invalid)
		}
	}
}

func TestBitcoinValueSatsRoundsRPCFloat(t *testing.T) {
	if got := bitcoinValueSats(0.00000001); got != 1 {
		t.Fatalf("got %d", got)
	}
	if got := bitcoinValueSats(1.23456789); got != 123456789 {
		t.Fatalf("got %d", got)
	}
}

func TestValidateEvidenceBatch(t *testing.T) {
	if validateEvidenceBatch(1) != nil || validateEvidenceBatch(maxBitcoinEvidenceBatch) != nil {
		t.Fatal("valid batch rejected")
	}
	if validateEvidenceBatch(0) == nil || validateEvidenceBatch(maxBitcoinEvidenceBatch+1) == nil {
		t.Fatal("invalid batch accepted")
	}
}

func TestBitcoinEvidenceBackendGuard(t *testing.T) {
	previous := bitcoin_rpc.ShareBitconRpc
	bitcoin_rpc.ShareBitconRpc = nil
	t.Cleanup(func() { bitcoin_rpc.ShareBitconRpc = previous })
	if requireBitcoinEvidenceBackend() == nil {
		t.Fatal("missing Bitcoin evidence backend accepted")
	}
	status := getBitcoinUTXOStatus("invalid")
	if status.Error == "" {
		t.Fatal("UTXO status did not report the missing backend")
	}
}

func TestBitcoinEvidenceHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	txid := "0000000000000000000000000000000000000000000000000000000000000001"
	block := "0000000000000000000000000000000000000000000000000000000000000002"
	stub := &bitcoinEvidenceRPCStub{txid: txid, block: block, script: "5120" + string(bytes.Repeat([]byte{'0'}, 64)), unspent: true}
	previous := bitcoin_rpc.ShareBitconRpc
	bitcoin_rpc.ShareBitconRpc = stub
	t.Cleanup(func() { bitcoin_rpc.ShareBitconRpc = previous })

	router := gin.New()
	(&Service{}).InitRouter(router, "/btc/testnet")

	status := rpcwire.BitcoinUTXOStatusResp{}
	postBitcoinEvidenceJSON(t, router, "/btc/testnet/v3/bitcoin/utxos/status", map[string]interface{}{"outpoints": []string{txid + ":0"}}, &status)
	if status.Code != 0 || len(status.Data) != 1 || !status.Data[0].Exists || !status.Data[0].Unspent || status.Data[0].Value != 1234 {
		t.Fatalf("UTXO status response=%+v", status)
	}

	txStatus := rpcwire.BitcoinTxStatusResp{}
	postBitcoinEvidenceJSON(t, router, "/btc/testnet/v3/bitcoin/tx/status/batch", map[string]interface{}{"txids": []string{txid}}, &txStatus)
	if txStatus.Code != 0 || len(txStatus.Data) != 1 || !txStatus.Data[0].Confirmed || txStatus.Data[0].BlockHeight != 12 {
		t.Fatalf("transaction status response=%+v", txStatus)
	}

	raw := rpcwire.BitcoinRawTxResp{}
	postBitcoinEvidenceJSON(t, router, "/btc/testnet/v3/bitcoin/rawtx/batch", map[string]interface{}{"txids": []string{txid}}, &raw)
	if raw.Code != 0 || len(raw.Data) != 1 || raw.Data[0].RawTx == "" {
		t.Fatalf("raw transaction response=%+v", raw)
	}

	stub.unspent = false
	outspend := rpcwire.BitcoinOutspendsResp{}
	postBitcoinEvidenceJSON(t, router, "/btc/testnet/v3/bitcoin/outspends/batch", map[string]interface{}{"outpoints": []string{txid + ":0"}}, &outspend)
	if outspend.Code != 0 || len(outspend.Data) != 1 || !outspend.Data[0].Spent || outspend.Data[0].SpendingTx != "" {
		t.Fatalf("outspend response=%+v", outspend)
	}

	broadcast := rpcwire.BitcoinBroadcastResp{}
	postBitcoinEvidenceJSON(t, router, "/btc/testnet/v3/bitcoin/tx/broadcast", map[string]interface{}{"raw_tx": "02000000000000000000"}, &broadcast)
	if broadcast.Code != 0 || broadcast.Data == nil || !broadcast.Data.Accepted || broadcast.Data.TxID != txid {
		t.Fatalf("broadcast response=%+v", broadcast)
	}

	tip := rpcwire.BitcoinTipResp{}
	getBitcoinEvidenceJSON(t, router, "/btc/testnet/v3/bitcoin/tip", &tip)
	if tip.Code != 0 || tip.Data == nil || tip.Data.Height != 12 || tip.Data.BlockHash != block {
		t.Fatalf("tip response=%+v", tip)
	}
	header := rpcwire.BitcoinBlockHeaderResp{}
	getBitcoinEvidenceJSON(t, router, "/btc/testnet/v3/bitcoin/block-header/12", &header)
	if header.Code != 0 || header.Data == nil || header.Data.Height != 12 || header.Data.Hash != block {
		t.Fatalf("header response=%+v", header)
	}
	fee := rpcwire.BitcoinFeeRateResp{}
	getBitcoinEvidenceJSON(t, router, "/btc/testnet/v3/bitcoin/fee-rate", &fee)
	if fee.Code != 0 || fee.Data == nil || fee.Data.Slow != 1 || fee.Data.Normal != 4 || fee.Data.Fast != 6 || fee.Data.Unit != "sat/vB" {
		t.Fatalf("fee response=%+v", fee)
	}
}

func postBitcoinEvidenceJSON(t *testing.T, router http.Handler, path string, request interface{}, response interface{}) {
	t.Helper()
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	httpRequest := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	httpRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, httpRequest)
	decodeBitcoinEvidenceResponse(t, recorder, response)
}

func getBitcoinEvidenceJSON(t *testing.T, router http.Handler, path string, response interface{}) {
	t.Helper()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	decodeBitcoinEvidenceResponse(t, recorder, response)
}

func decodeBitcoinEvidenceResponse(t *testing.T, recorder *httptest.ResponseRecorder, response interface{}) {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), response); err != nil {
		t.Fatalf("decode response %s: %v", recorder.Body.String(), err)
	}
}
