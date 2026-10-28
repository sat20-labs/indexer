package btclucky

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTemplateServiceCoinbaseMaturity(t *testing.T) {
	for _, maturity := range []uint64{0, 10, 100} {
		svc, err := NewTemplateService(BTCLuckyTemplateServiceConfig{Network: "mainnet", CoinbaseMaturity: maturity})
		if err != nil {
			t.Fatal(err)
		}
		want := maturity
		if want == 0 {
			want = 100
		}
		if got := svc.Status().CoinbaseMaturity; got != want {
			t.Fatalf("maturity=%d want=%d", got, want)
		}
	}
}

func TestHTTPInfoDefaultsCoinbaseMaturity(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       uint64
		fail       bool
	}{
		{name: "missing field", body: `{"code":0,"data":{"service":{}}}`, want: 100},
		{name: "missing service", body: `{"code":0,"data":{}}`, want: 100},
		{name: "zero", body: `{"code":0,"data":{"service":{"coinbaseMaturity":0}}}`, want: 100},
		{name: "private chain", body: `{"code":0,"data":{"service":{"coinbaseMaturity":10}}}`, want: 10},
		{name: "public chain", body: `{"code":0,"data":{"service":{"coinbaseMaturity":100}}}`, want: 100},
		{name: "API error", body: `{"code":-1,"msg":"not ready"}`, fail: true},
		{name: "empty info", body: `{"code":0,"data":null}`, fail: true},
		{name: "invalid maturity", body: `{"code":0,"data":{"service":{"coinbaseMaturity":-1}}}`, fail: true},
		{name: "invalid JSON", body: `{`, fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/btc/lucky/info" {
					t.Errorf("unexpected path %s", r.URL.Path)
				}
				w.Write([]byte(tc.body))
			}))
			defer server.Close()
			info, err := NewHTTPTemplateBackend(server.URL, time.Second).Info()
			if tc.fail {
				if err == nil {
					t.Fatal("invalid response accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if info.Service.CoinbaseMaturity != tc.want {
				t.Fatalf("maturity=%d want=%d", info.Service.CoinbaseMaturity, tc.want)
			}
		})
	}
	server := httptest.NewServer(http.NotFoundHandler())
	if _, err := NewHTTPTemplateBackend(server.URL, time.Second).Info(); err == nil {
		t.Fatal("HTTP failure must not become a default maturity")
	}
	server.Close()
	if _, err := NewHTTPTemplateBackend(server.URL, time.Second).Info(); err == nil {
		t.Fatal("transport failure must not become a default maturity")
	}
}
