package btclucky

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	btcbtcjson "github.com/btcsuite/btcd/btcjson"
	btcbtcutil "github.com/btcsuite/btcd/btcutil"
	btcchaincfg "github.com/btcsuite/btcd/chaincfg"
	btcrpcclient "github.com/btcsuite/btcd/rpcclient"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func blockBytesForTest(t *testing.T, result *btcbtcjson.GetBlockTemplateResult, address string) ([]byte, *btcbtcutil.Block) {
	t.Helper()
	tpl, err := newBlockTemplate(result, "mainnet")
	if err != nil {
		t.Fatalf("newBlockTemplate: %v", err)
	}
	coinbase, header, err := tpl.work(&btcchaincfg.MainNetParams, address, 1)
	if err != nil {
		t.Fatalf("BlockTemplate.work: %v", err)
	}
	raw, err := tpl.blockBytes(coinbase, header)
	if err != nil {
		t.Fatalf("BlockTemplate.blockBytes: %v", err)
	}
	block, err := btcbtcutil.NewBlockFromBytes(raw)
	if err != nil {
		t.Fatalf("decode block bytes: %v", err)
	}
	return raw, block
}

func rpcClientForTest(t *testing.T, server *httptest.Server) *btcrpcclient.Client {
	t.Helper()
	client, err := btcrpcclient.New(&btcrpcclient.ConnConfig{
		Host:         strings.TrimPrefix(server.URL, "http://"),
		User:         "test-user",
		Pass:         "test-pass",
		HTTPPostMode: true,
		DisableTLS:   true,
	}, nil)
	if err != nil {
		t.Fatalf("create RPC client: %v", err)
	}
	t.Cleanup(client.Shutdown)
	return client
}

func testSubmitService(t *testing.T, client *btcrpcclient.Client, foundDB common.KVDB) *TemplateService {
	t.Helper()
	service, err := NewTemplateService(BTCLuckyTemplateServiceConfig{
		Network:       "mainnet",
		FoundBlocksDB: foundDB,
	})
	if err != nil {
		t.Fatalf("NewTemplateService: %v", err)
	}
	service.client = client
	return service
}

func TestCore31SubmitBlockArgs(t *testing.T) {
	tplResult := testTemplate()
	tplResult.WorkID = ""
	address := testRewardAddress(t)
	raw, block := blockBytesForTest(t, tplResult, address)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		calls++
		if request.Method != "submitblock" {
			t.Errorf("method = %q, want submitblock", request.Method)
		}
		w.Header().Set("content-type", "application/json")
		if len(request.Params) != 1 {
			fmt.Fprintf(w, `{"result":null,"error":{"code":-3,"message":"Core 31: submitblock accepts only the block hex argument"},"id":%s}`, request.ID)
			return
		}
		var gotHex string
		if err := json.Unmarshal(request.Params[0], &gotHex); err != nil {
			t.Error(err)
			return
		}
		if gotHex != fmt.Sprintf("%x", raw) {
			t.Error("submitted block bytes/hash changed")
		}
		fmt.Fprintf(w, `{"result":null,"error":null,"id":%s}`, request.ID)
	}))
	defer server.Close()

	foundDB := db.NewKVDB(t.TempDir())
	defer foundDB.Close()
	service := testSubmitService(t, rpcClientForTest(t, server), foundDB)
	record, err := service.SubmitBlock(raw)
	if err != nil {
		t.Fatalf("valid block must submit through Core 31 RPC: %v", err)
	}
	if record == nil || !record.Submitted || record.SubmitResult != "accepted" || record.BlockHash != block.Hash().String() || calls != 1 {
		t.Fatalf("unexpected submission record = %+v, calls = %d", record, calls)
	}
	records, err := loadFoundBlockRecords(foundDB)
	if err != nil {
		t.Fatalf("loadFoundBlockRecords: %v", err)
	}
	if len(records) != 1 || records[0].BlockHash != block.Hash().String() {
		t.Fatalf("accepted record persistence = %+v", records)
	}
	reloaded, err := NewTemplateService(BTCLuckyTemplateServiceConfig{Network: "mainnet", FoundBlocksDB: foundDB})
	if err != nil {
		t.Fatalf("reload NewTemplateService: %v", err)
	}
	if got := reloaded.FoundBlocks(); len(got) != 1 || got[0].BlockHash != block.Hash().String() {
		t.Fatalf("accepted record after reload = %+v", got)
	}
}

func TestSubmitBlockRejectNoDBRecord(t *testing.T) {
	address := testRewardAddress(t)
	raw, _ := blockBytesForTest(t, testTemplate(), address)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID json.RawMessage `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{"result":"rejected","error":null,"id":%s}`, request.ID)
	}))
	defer server.Close()

	foundDB := db.NewKVDB(t.TempDir())
	defer foundDB.Close()
	service := testSubmitService(t, rpcClientForTest(t, server), foundDB)
	record, err := service.SubmitBlock(raw)
	if err == nil {
		t.Fatal("Core rejection must be returned as an error")
	}
	if record == nil || record.Submitted || record.SubmitResult != "rejected" {
		t.Fatalf("unexpected rejected submission record = %+v", record)
	}
	if records, err := loadFoundBlockRecords(foundDB); err != nil {
		t.Fatalf("loadFoundBlockRecords: %v", err)
	} else if len(records) != 0 {
		t.Fatalf("rejected block persisted: %+v", records)
	}
	reloaded, err := NewTemplateService(BTCLuckyTemplateServiceConfig{Network: "mainnet", FoundBlocksDB: foundDB})
	if err != nil {
		t.Fatalf("reload NewTemplateService: %v", err)
	}
	if len(reloaded.FoundBlocks()) != 0 {
		t.Fatalf("rejected block appeared after reload: %+v", reloaded.FoundBlocks())
	}
}
