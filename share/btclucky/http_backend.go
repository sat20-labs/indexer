package btclucky

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
)

type APIResponse[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

type InfoResponse struct {
	Service     TemplateServiceStatus `json:"service"`
	FoundBlocks []FoundBlockRecord    `json:"foundBlocks,omitempty"`
}

func (b *HTTPTemplateBackend) Info() (*InfoResponse, error) {
	var resp APIResponse[*InfoResponse]
	if err := b.get("/btc/lucky/info", &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, errors.New(resp.Msg)
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("btc lucky indexer returned empty info")
	}
	if resp.Data.Service.CoinbaseMaturity == 0 {
		resp.Data.Service.CoinbaseMaturity = defaultCoinbaseMaturity
	}
	return resp.Data, nil
}

type HTTPTemplateBackend struct {
	mu      sync.Mutex
	baseURL string
	client  *http.Client
	found   []FoundBlockRecord
	cached  *BlockTemplate
	tag     string
}

func NewHTTPTemplateBackend(baseURL string, timeout time.Duration) *HTTPTemplateBackend {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &HTTPTemplateBackend{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
	}
}

func (b *HTTPTemplateBackend) CurrentTemplate() (*BlockTemplate, error) {
	req, err := http.NewRequest(http.MethodGet, b.baseURL+"/btc/lucky/template", nil)
	if err != nil {
		return nil, err
	}
	b.mu.Lock()
	cached, tag := b.cached, b.tag
	b.mu.Unlock()
	if cached != nil {
		req.Header.Set("If-None-Match", tag)
	}
	result, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()
	if result.StatusCode == http.StatusNotModified && cached != nil {
		return cached, nil
	}
	if result.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("template HTTP %d", result.StatusCode)
	}
	var resp APIResponse[*BlockTemplate]
	if err = json.NewDecoder(result.Body).Decode(&resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, errors.New(resp.Msg)
	}
	tag = result.Header.Get("ETag")
	if resp.Data == nil || len(tag) < 3 || tag[0] != '"' || tag[len(tag)-1] != '"' {
		return nil, fmt.Errorf("incomplete Bitcoin template response")
	}
	resp.Data.ID = tag[1 : len(tag)-1]
	b.mu.Lock()
	b.cached, b.tag = resp.Data, tag
	b.mu.Unlock()
	return resp.Data, nil
}

func (b *HTTPTemplateBackend) SubmitBlock(raw []byte) (*FoundBlockRecord, error) {
	result, err := b.client.Post(b.baseURL+"/v3/bitcoin/block/submit", "application/octet-stream", bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCoreUnavailable, err)
	}
	defer result.Body.Close()
	if result.StatusCode >= 500 {
		return nil, fmt.Errorf("%w: HTTP %d", ErrCoreUnavailable, result.StatusCode)
	}
	if result.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("block submit HTTP %d", result.StatusCode)
	}
	var resp APIResponse[*FoundBlockRecord]
	if err := json.NewDecoder(result.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCoreUnavailable, err)
	}
	if resp.Data != nil {
		b.rememberFound(*resp.Data)
	}
	if resp.Code != 0 {
		return resp.Data, errors.New(resp.Msg)
	}
	if resp.Data == nil || !resp.Data.Submitted {
		return nil, fmt.Errorf("missing block submission acknowledgement")
	}
	return resp.Data, nil
}

func (b *HTTPTemplateBackend) FoundBlocks() []FoundBlockRecord {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]FoundBlockRecord, len(b.found))
	copy(out, b.found)
	return out
}

func (b *HTTPTemplateBackend) get(path string, resp interface{}) error {
	httpResp, err := b.client.Get(b.baseURL + path)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf("btc lucky indexer http status %d", httpResp.StatusCode)
	}
	return json.NewDecoder(httpResp.Body).Decode(resp)
}

func (b *HTTPTemplateBackend) rememberFound(record FoundBlockRecord) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.found = append(b.found, record)
	if len(b.found) > 32 {
		b.found = b.found[len(b.found)-32:]
	}
}

// CoreSubmitBackend fetches templates from the indexer and sends solved
// blocks directly to the operator's Bitcoin Core. The caller owns client.
type CoreSubmitBackend struct {
	*HTTPTemplateBackend
	client *rpcclient.Client
	params *chaincfg.Params
}

func NewCoreSubmitBackend(httpBackend *HTTPTemplateBackend, client *rpcclient.Client, network string) (*CoreSubmitBackend, error) {
	if httpBackend == nil || client == nil {
		return nil, fmt.Errorf("template backend and Core RPC client are required")
	}
	params, err := BTCChainParams(network)
	if err != nil {
		return nil, err
	}
	return &CoreSubmitBackend{HTTPTemplateBackend: httpBackend, client: client, params: params}, nil
}

func (b *CoreSubmitBackend) SubmitBlock(raw []byte) (*FoundBlockRecord, error) {
	record, err := submitCoreBlock(b.client, b.params, raw)
	if record != nil {
		b.rememberFound(*record)
	}
	return record, err
}
