package btclucky

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcjson"
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
	mu         sync.Mutex
	baseURL    string
	client     *http.Client
	transfer   *http.Client
	found      []FoundBlockRecord
	cached     *BlockTemplate
	tag        string
	submitURLs []string
}

func NewHTTPTemplateBackend(baseURL string, timeout time.Duration) *HTTPTemplateBackend {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	b := &HTTPTemplateBackend{
		baseURL:  strings.TrimRight(baseURL, "/"),
		client:   &http.Client{Timeout: timeout},
		transfer: &http.Client{Timeout: max(2*time.Minute, timeout)},
	}
	parsed, err := url.Parse(b.baseURL)
	if err == nil && parsed.Scheme == "https" {
		switch parsed.Host {
		case "apiprd.ordx.market":
			parsed.Host = "apiprd.sat20.org"
		case "apiprd.sat20.org":
			parsed.Host = "apiprd.ordx.market"
		default:
			return b
		}
		b.submitURLs = []string{strings.TrimRight(parsed.String(), "/") + "/v3/bitcoin/block/submit"}
	}
	return b
}

func (b *HTTPTemplateBackend) CurrentTemplate(ctx context.Context) (*BlockTemplate, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.baseURL+"/btc/lucky/template", nil)
	if err != nil {
		return nil, err
	}
	b.mu.Lock()
	cached, tag := b.cached, b.tag
	b.mu.Unlock()
	if cached != nil {
		req.Header.Set("If-None-Match", tag)
	}
	result, err := b.transfer.Do(req)
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
	tag = strings.TrimPrefix(result.Header.Get("ETag"), "W/")
	if resp.Data == nil || len(tag) < 3 || tag[0] != '"' || tag[len(tag)-1] != '"' {
		return nil, fmt.Errorf("incomplete Bitcoin template response")
	}
	resp.Data.ID = tag[1 : len(tag)-1]
	b.mu.Lock()
	b.cached, b.tag = resp.Data, tag
	b.mu.Unlock()
	return resp.Data, nil
}

// SetSubmitURLs replaces the backup submit endpoints. Template fetching stays
// on the selected provider so the coinbase service identity stays consistent.
func (b *HTTPTemplateBackend) SetSubmitURLs(urls []string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.submitURLs = append([]string(nil), urls...)
}

func (b *HTTPTemplateBackend) SubmitBlock(ctx context.Context, raw []byte) (*FoundBlockRecord, error) {
	b.mu.Lock()
	cached := b.cached
	endpoints := append([]string{b.baseURL + "/v3/bitcoin/block/submit"}, b.submitURLs...)
	b.mu.Unlock()
	if cached == nil {
		return nil, fmt.Errorf("missing Bitcoin template network")
	}
	params, err := BTCChainParams(cached.Network)
	if err != nil {
		return nil, err
	}
	_, record, err := blockRecord(params, raw)
	if err != nil {
		return nil, err
	}
	request, err := btcjson.NewRequest(btcjson.RpcVersion2, record.BlockHash, "submitblock", []interface{}{hex.EncodeToString(raw)})
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	for _, endpoint := range endpoints {
		err = b.postBlock(ctx, endpoint, body, record)
		if err == nil {
			b.rememberFound(*record)
			return record, nil
		}
		if !errors.Is(err, ErrCoreUnavailable) {
			return nil, err
		}
	}
	return nil, err
}

func (b *HTTPTemplateBackend) postBlock(ctx context.Context, endpoint string, body []byte, record *FoundBlockRecord) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	result, err := b.transfer.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCoreUnavailable, err)
	}
	defer result.Body.Close()
	// A proxy error or a truncated reply cannot establish the fate of the block.
	var reply SubmitResponse
	data, err := io.ReadAll(io.LimitReader(result.Body, 65537))
	if err != nil || len(data) > 65536 || json.Unmarshal(data, &reply) != nil || reply.JSONRPC != "2.0" || reply.ID != record.BlockHash {
		return fmt.Errorf("%w: no matching submitblock receipt (HTTP %d)", ErrCoreUnavailable, result.StatusCode)
	}
	if reply.Error != nil {
		switch reply.Error.Code {
		case btcjson.ErrRPCDeserialization, btcjson.ErrRPCVerifyRejected, btcjson.ErrRPCInvalidParams.Code:
			return reply.Error
		default:
			return fmt.Errorf("%w: %v", ErrCoreUnavailable, reply.Error)
		}
	}
	var verdict *string
	if result.StatusCode != http.StatusOK || len(reply.Result) == 0 || json.Unmarshal(reply.Result, &verdict) != nil {
		return fmt.Errorf("%w: invalid submitblock result (HTTP %d)", ErrCoreUnavailable, result.StatusCode)
	}
	record.SubmitResult = "accepted"
	if verdict != nil {
		record.SubmitResult = *verdict
		switch *verdict {
		case "duplicate":
		case "prev-blk-not-found", "inconclusive", "duplicate-inconclusive", "inconclusive-not-best-prevblk":
			return fmt.Errorf("%w: %s", ErrCoreUnavailable, *verdict)
		default:
			return fmt.Errorf("Core rejected block: %s", *verdict)
		}
	}
	// A known block can be on a side chain. Check the node that issued the receipt.
	base := strings.TrimSuffix(endpoint, "/v3/bitcoin/block/submit")
	path := base + "/btc/block/blockhash/" + strconv.FormatInt(record.BlockHeight, 10)
	var hash APIResponse[string]
	if err := b.getURL(ctx, path, &hash); err != nil || hash.Code != 0 || len(hash.Data) != 64 {
		return fmt.Errorf("%w: cannot verify block at height %d", ErrCoreUnavailable, record.BlockHeight)
	}
	if hash.Data != record.BlockHash {
		return fmt.Errorf("block %s is on a side chain at height %d", record.BlockHash, record.BlockHeight)
	}
	record.Submitted = true
	return nil
}

func (b *HTTPTemplateBackend) FoundBlocks() []FoundBlockRecord {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]FoundBlockRecord, len(b.found))
	copy(out, b.found)
	return out
}

func (b *HTTPTemplateBackend) get(path string, resp interface{}) error {
	return b.getURL(context.Background(), b.baseURL+path, resp)
}

func (b *HTTPTemplateBackend) getURL(ctx context.Context, endpoint string, resp any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	httpResp, err := b.client.Do(req)
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
