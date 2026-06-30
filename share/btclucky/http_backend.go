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

type bestHeightResponse struct {
	Code int            `json:"code"`
	Msg  string         `json:"msg"`
	Data map[string]int `json:"data"`
}

type HTTPTemplateBackend struct {
	mu      sync.Mutex
	baseURL string
	client  *http.Client
	found   []FoundBlockRecord
	lastJob time.Time
	lastErr string
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

func (b *HTTPTemplateBackend) Name() string {
	return BTCLuckyBackendHTTPTemplate
}

func (b *HTTPTemplateBackend) Start() error {
	if b.baseURL == "" {
		return fmt.Errorf("btc lucky http-template backend missing base url")
	}
	return nil
}

func (b *HTTPTemplateBackend) Stop() {}

func (b *HTTPTemplateBackend) IsReady() bool {
	return b.baseURL != ""
}

func (b *HTTPTemplateBackend) CurrentJob(req JobRequest) (*CompactMiningJob, error) {
	var resp APIResponse[*CompactMiningJob]
	if err := b.post("/btc/lucky/job", req, &resp); err != nil {
		b.setError(err)
		return nil, err
	}
	if resp.Code != 0 {
		err := errors.New(resp.Msg)
		b.setError(err)
		return nil, err
	}
	if resp.Data == nil {
		err := fmt.Errorf("btc lucky indexer returned empty job")
		b.setError(err)
		return nil, err
	}
	b.mu.Lock()
	b.lastJob = time.Now()
	b.lastErr = ""
	b.mu.Unlock()
	return resp.Data, nil
}

func (b *HTTPTemplateBackend) SubmitSolution(solution *MiningSolution) (*FoundBlockRecord, error) {
	var resp APIResponse[*FoundBlockRecord]
	if err := b.post("/btc/lucky/submit", solution, &resp); err != nil {
		b.setError(err)
		return nil, err
	}
	if resp.Data != nil {
		b.rememberFound(*resp.Data)
	}
	if resp.Code != 0 {
		err := errors.New(resp.Msg)
		b.setError(err)
		return resp.Data, err
	}
	if resp.Data == nil {
		err := fmt.Errorf("btc lucky indexer returned empty submit result")
		b.setError(err)
		return nil, err
	}
	return resp.Data, nil
}

func (b *HTTPTemplateBackend) TipHeight() (int64, error) {
	if b.baseURL == "" {
		return 0, fmt.Errorf("btc lucky http-template backend missing base url")
	}
	var resp bestHeightResponse
	if err := b.get("/bestheight", &resp); err != nil {
		b.setError(err)
		return 0, err
	}
	if resp.Code != 0 {
		err := errors.New(resp.Msg)
		b.setError(err)
		return 0, err
	}
	height, ok := resp.Data["height"]
	if !ok {
		err := fmt.Errorf("btc lucky indexer bestheight response missing height")
		b.setError(err)
		return 0, err
	}
	b.mu.Lock()
	b.lastErr = ""
	b.mu.Unlock()
	return int64(height), nil
}

func (b *HTTPTemplateBackend) Status() TemplateServiceStatus {
	b.mu.Lock()
	defer b.mu.Unlock()
	return TemplateServiceStatus{
		Enabled:          true,
		Running:          true,
		Backend:          BTCLuckyBackendHTTPTemplate,
		LastTemplateTime: b.lastJob,
		LastError:        b.lastErr,
	}
}

func (b *HTTPTemplateBackend) FoundBlocks() []FoundBlockRecord {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]FoundBlockRecord, len(b.found))
	copy(out, b.found)
	return out
}

func (b *HTTPTemplateBackend) post(path string, req interface{}, resp interface{}) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	httpResp, err := b.client.Post(b.baseURL+path, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf("btc lucky indexer http status %d", httpResp.StatusCode)
	}
	return json.NewDecoder(httpResp.Body).Decode(resp)
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

func (b *HTTPTemplateBackend) setError(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err != nil {
		b.lastErr = err.Error()
	}
}

func (b *HTTPTemplateBackend) rememberFound(record FoundBlockRecord) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lastErr = ""
	b.found = append(b.found, record)
	if len(b.found) > 32 {
		b.found = b.found[len(b.found)-32:]
	}
}
