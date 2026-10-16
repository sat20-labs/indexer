package btclucky

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	btcbtcutil "github.com/btcsuite/btcd/btcutil"
	btcchaincfg "github.com/btcsuite/btcd/chaincfg"
	btcrpcclient "github.com/btcsuite/btcd/rpcclient"
)

// Core 31 no longer accepts the historical BIP22 workid options object.
// Exercise the real btcd RPC serialization used by SubmitSolution, not a mock client.
func TestTemplateSubmitSolutionUsesCore31SubmitBlockArguments(t *testing.T) {
	template := testTemplate()
	template.WorkID = ""
	address := testRewardAddress(t)
	work, err := assembleBTCWork(template, &btcchaincfg.MainNetParams, address, "core31-test", 1, 0, template.CurTime)
	if err != nil {
		t.Fatal(err)
	}
	var expected bytes.Buffer
	if err := work.block.Serialize(&expected); err != nil {
		t.Fatal(err)
	}
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
			t.Errorf("method=%s", request.Method)
		}
		w.Header().Set("content-type", "application/json")
		if len(request.Params) != 1 {
			fmt.Fprintf(w, `{"result":null,"error":{"code":-3,"message":"Core 31: Position 2 (dummy) must not be an options object"},"id":%s}`, request.ID)
			return
		}
		var blockHex string
		if err := json.Unmarshal(request.Params[0], &blockHex); err != nil {
			t.Error(err)
			return
		}
		parsed, err := btcbtcutil.NewBlockFromBytes(expected.Bytes())
		if err != nil {
			t.Error(err)
			return
		}
		if fmt.Sprintf("%x", expected.Bytes()) != blockHex || parsed.Hash().String() != work.blockHash.String() {
			t.Error("submitted block bytes/hash changed")
		}
		fmt.Fprintf(w, `{"result":null,"error":null,"id":%s}`, request.ID)
	}))
	defer server.Close()
	client, err := btcrpcclient.New(&btcrpcclient.ConnConfig{Host: strings.TrimPrefix(server.URL, "http://"), User: "test-user", Pass: "test-pass", HTTPPostMode: true, DisableTLS: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Shutdown()
	service := &TemplateService{client: client, jobs: map[string]*cachedBTCJob{"original-job": {
		job: &CompactMiningJob{JobID: "original-job", TemplateID: "original-template", RewardAddress: address, MinerID: "core31-test"}, template: template, params: &btcchaincfg.MainNetParams,
	}}}
	record, err := service.SubmitSolution(&MiningSolution{JobID: "original-job", TemplateID: "original-template", RewardAddress: address, ExtraNonce: 1, NTime: template.CurTime, HeaderHash: work.blockHash.String()})
	if err != nil {
		t.Fatalf("valid solution must submit through Core 31 RPC: %v", err)
	}
	if record == nil || !record.Submitted || record.SubmitResult != "accepted" || record.BlockHash != work.blockHash.String() || calls != 1 {
		t.Fatalf("unexpected submission record=%+v calls=%d", record, calls)
	}
}
