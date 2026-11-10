package btclucky

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	btcbtcjson "github.com/btcsuite/btcd/btcjson"
	btcbtcutil "github.com/btcsuite/btcd/btcutil"
	btcchaincfg "github.com/btcsuite/btcd/chaincfg"
	btcrpcclient "github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
)

const foundBlockDBPrefix = "btclucky:found:"

type TemplateService struct {
	mu         sync.Mutex
	cfg        BTCLuckyTemplateServiceConfig
	params     *btcchaincfg.Params
	client     *btcrpcclient.Client
	running    bool
	cacheMu    sync.Mutex
	body       []byte
	etag       string
	parent     string
	bits       string
	longPollID string
	checked    time.Time
	probeErr   error
	refreshed  time.Time
	found      []FoundBlockRecord
	status     TemplateServiceStatus
}

func NewTemplateService(cfg BTCLuckyTemplateServiceConfig) (*TemplateService, error) {
	cfg.Normalize()
	params, err := BTCChainParams(cfg.Network)
	if err != nil {
		return nil, err
	}
	svc := &TemplateService{
		cfg:    cfg,
		params: params,
		status: TemplateServiceStatus{
			CoinbaseMaturity: cfg.CoinbaseMaturity,
			Enabled:          cfg.Enabled,
			Backend:          cfg.Backend,
			BTCNetwork:       cfg.Network,
		},
	}
	if err := svc.loadFoundBlocks(); err != nil {
		log.Warnf("failed to read BTC lucky found block history: %v", err)
	}
	return svc, nil
}

func (s *TemplateService) Name() string {
	return BTCLuckyBackendLocalTemplate
}

func (s *TemplateService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}
	client, err := btcrpcclient.New(&btcrpcclient.ConnConfig{
		Host:         s.cfg.RPCConnect,
		User:         s.cfg.RPCUser,
		Pass:         s.cfg.RPCPass,
		HTTPPostMode: true,
		DisableTLS:   s.cfg.RPCDisableTLS,
	}, nil)
	if err != nil {
		s.status.LastError = err.Error()
		return err
	}
	s.client = client
	s.running = true
	s.status.Running = true
	s.status.BTCRPCConnected = true
	return nil
}

func (s *TemplateService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		s.client.Shutdown()
		s.client = nil
	}
	s.running = false
	s.status.Running = false
	s.status.BTCRPCConnected = false
}

func (s *TemplateService) IsReady() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running && s.client != nil
}

// TemplateBytes publishes one immutable response. A burst of GETs shares
// the same Core probe, including the first request after a restart.
func (s *TemplateService) TemplateBytes() (payload []byte, tag string, err error) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return nil, "", fmt.Errorf("Bitcoin Core is unavailable")
	}
	now := time.Now()
	if now.Sub(s.checked) < time.Second {
		if s.probeErr != nil {
			return nil, "", s.probeErr
		}
		return s.body, s.etag, nil
	}
	// All concurrent miners share the outcome, including a failed Core probe.
	defer func() {
		s.checked, s.probeErr = time.Now(), err
		if err != nil {
			s.setTemplateError(err)
		}
	}()
	if s.body != nil {
		tip, err := client.GetBestBlockHash()
		if err != nil {
			return nil, "", err
		}
		if tip.String() == s.parent && now.Sub(s.refreshed) < 30*time.Second {
			return s.body, s.etag, nil
		}
	}
	tpl, err := client.GetBlockTemplate(&btcbtcjson.TemplateRequest{Mode: "template", Rules: []string{"segwit"}})
	if err != nil {
		return nil, "", err
	}
	// Core updates curtime on every GBT. A time-only update leaves the work
	// usable; changes to testnet difficulty or to the transaction set do not
	// escape this check. LongPollID is supplied by Bitcoin Core.
	if s.body != nil && tpl.LongPollID != "" && tpl.LongPollID == s.longPollID && tpl.PreviousHash == s.parent && tpl.Bits == s.bits {
		s.refreshed = now
		return s.body, s.etag, nil
	}
	data, err := newBlockTemplate(tpl, s.cfg.Network)
	if err != nil {
		return nil, "", err
	}
	body, err := json.Marshal(APIResponse[*BlockTemplate]{Msg: "ok", Data: data})
	if err != nil {
		return nil, "", err
	}
	s.body = body
	s.etag = fmt.Sprintf(`"%x"`, sha256.Sum256(body))
	s.parent, s.bits, s.longPollID = tpl.PreviousHash, tpl.Bits, tpl.LongPollID
	s.refreshed = now
	s.mu.Lock()
	s.status.LastTemplateTime = now
	s.status.BTCHeight = tpl.Height
	s.status.TemplateCacheSize = 1
	s.status.LastError = ""
	s.mu.Unlock()
	return s.body, s.etag, nil
}

// SubmitBlock uses exactly the submitted witness serialization. It does not
// consult the current template, so a restart cannot invalidate a solved block.
func (s *TemplateService) SubmitBlock(raw []byte) (*FoundBlockRecord, error) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	record, err := submitCoreBlock(client, s.params, raw)
	if err != nil {
		s.setSubmitResult("", err)
		return record, err
	}
	s.setSubmitResult(record.SubmitResult, nil)
	s.cacheMu.Lock()
	s.checked = time.Time{}
	s.cacheMu.Unlock()
	if record.lucky && record.SubmitResult == "accepted" {
		s.rememberFound(*record)
	}
	return record, nil
}

func blockRecord(params *btcchaincfg.Params, raw []byte) (*wire.MsgBlock, *FoundBlockRecord, error) {
	if len(raw) > wire.MaxBlockPayload {
		return nil, nil, fmt.Errorf("oversized Bitcoin block")
	}
	r := bytes.NewReader(raw)
	var block wire.MsgBlock
	if err := block.Deserialize(r); err != nil {
		return nil, nil, err
	}
	if r.Len() != 0 || len(block.Transactions) == 0 || !blockchain.IsCoinBaseTx(block.Transactions[0]) || len(block.Transactions[0].TxOut) == 0 {
		return nil, nil, fmt.Errorf("invalid Bitcoin block body")
	}
	cb := block.Transactions[0]
	height, _ := blockchain.ExtractCoinbaseHeight(btcbtcutil.NewTx(cb))
	record := &FoundBlockRecord{
		BlockHash:    block.BlockHash().String(),
		BlockHeight:  int64(height),
		CoinbaseTxID: cb.TxHash().String(),
		Amount:       cb.TxOut[0].Value,
		CreatedAt:    time.Now(),
	}
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(cb.TxOut[0].PkScript, params)
	if err == nil && len(addrs) == 1 {
		record.RewardAddress = addrs[0].EncodeAddress()
	}
	record.lucky = bytes.Contains(cb.TxIn[0].SignatureScript, []byte(coinbaseTag))
	return &block, record, nil
}

func submitCoreBlock(client *btcrpcclient.Client, params *btcchaincfg.Params, raw []byte) (*FoundBlockRecord, error) {
	block, record, err := blockRecord(params, raw)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, ErrCoreUnavailable
	}
	arg, _ := json.Marshal(hex.EncodeToString(raw))
	result, err := client.RawRequest("submitblock", []json.RawMessage{arg})
	if err != nil {
		var rpcErr *btcbtcjson.RPCError
		if errors.As(err, &rpcErr) && rpcErr.Code != btcbtcjson.ErrRPCInWarmup {
			record.SubmitResult = err.Error()
			return record, err
		}
		return nil, fmt.Errorf("%w: %v", ErrCoreUnavailable, err)
	}
	if string(result) != "null" {
		if err = json.Unmarshal(result, &record.SubmitResult); err != nil {
			return nil, err
		}
		if record.SubmitResult == "duplicate" {
			hash := block.BlockHash()
			header, lookupErr := client.GetBlockHeaderVerbose(&hash)
			if lookupErr != nil {
				return nil, fmt.Errorf("%w: %v", ErrCoreUnavailable, lookupErr)
			}
			if header.Confirmations > 0 {
				record.Submitted = true
				return record, nil
			}
		}
		err = fmt.Errorf("Core rejected block: %s", record.SubmitResult)
		return record, err
	}
	record.Submitted = true
	record.SubmitResult = "accepted"
	return record, nil
}

var ErrCoreUnavailable = errors.New("Bitcoin Core unavailable")

func (s *TemplateService) Status() TemplateServiceStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.status
	st.Enabled = s.cfg.Enabled
	st.Backend = s.cfg.Backend
	st.BTCNetwork = s.cfg.Network
	st.Running = s.running
	st.BTCRPCConnected = s.client != nil && s.running
	return st
}

func (s *TemplateService) FoundBlocks() []FoundBlockRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]FoundBlockRecord, len(s.found))
	copy(out, s.found)
	return out
}

func (s *TemplateService) setTemplateError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.status.LastError = err.Error()
	}
}

func (s *TemplateService) setSubmitResult(result string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.LastSubmitTime = time.Now()
	s.status.LastSubmitResult = result
	if err != nil {
		s.status.LastError = err.Error()
	} else {
		s.status.LastError = ""
	}
}

func (s *TemplateService) rememberFound(record FoundBlockRecord) {
	s.mu.Lock()
	s.found = append(s.found, record)
	if len(s.found) > 32 {
		s.found = s.found[len(s.found)-32:]
	}
	db := s.cfg.FoundBlocksDB
	s.mu.Unlock()

	log.Infof("BTC lucky found block metadata block_hash=%s height=%d coinbase_txid=%s vout=%d amount=%d reward_address=%s submitted=%v result=%q",
		record.BlockHash, record.BlockHeight, record.CoinbaseTxID, record.Vout, record.Amount,
		record.RewardAddress, record.Submitted, record.SubmitResult)

	if err := appendFoundBlockRecord(db, record); err != nil {
		log.Warnf("failed to persist BTC lucky found block record: %v", err)
	}
}

func (s *TemplateService) loadFoundBlocks() error {
	records, err := loadFoundBlockRecords(s.cfg.FoundBlocksDB)
	if err != nil {
		return err
	}
	if len(records) > 32 {
		records = records[len(records)-32:]
	}
	s.found = records
	return nil
}

func appendFoundBlockRecord(localDB common.KVDB, record FoundBlockRecord) error {
	if localDB == nil {
		return nil
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s%020d:%s", foundBlockDBPrefix, record.CreatedAt.UnixNano(), record.BlockHash)
	return localDB.Write([]byte(key), data)
}

func loadFoundBlockRecords(localDB common.KVDB) ([]FoundBlockRecord, error) {
	if localDB == nil {
		return nil, nil
	}
	var records []FoundBlockRecord
	err := localDB.BatchRead([]byte(foundBlockDBPrefix), false, func(k, v []byte) error {
		var record FoundBlockRecord
		if err := json.Unmarshal(v, &record); err != nil {
			return err
		}
		records = append(records, record)
		return nil
	})
	if err != nil && err != common.ErrKeyNotFound {
		return nil, err
	}
	return records, nil
}
