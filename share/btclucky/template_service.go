package btclucky

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	btcbtcjson "github.com/btcsuite/btcd/btcjson"
	btcbtcutil "github.com/btcsuite/btcd/btcutil"
	btcchaincfg "github.com/btcsuite/btcd/chaincfg"
	btcrpcclient "github.com/btcsuite/btcd/rpcclient"
	"github.com/sat20-labs/indexer/common"
)

const foundBlockDBPrefix = "btclucky:found:"

type cachedBTCJob struct {
	job      *CompactMiningJob
	template *btcbtcjson.GetBlockTemplateResult
	params   *btcchaincfg.Params
	created  time.Time
}

type TemplateService struct {
	mu      sync.Mutex
	cfg     BTCLuckyTemplateServiceConfig
	params  *btcchaincfg.Params
	client  *btcrpcclient.Client
	running bool
	lastTpl *btcbtcjson.GetBlockTemplateResult
	jobs    map[string]*cachedBTCJob
	found   []FoundBlockRecord
	status  TemplateServiceStatus
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
		jobs:   make(map[string]*cachedBTCJob),
		status: TemplateServiceStatus{
			Enabled:    cfg.Enabled,
			Backend:    cfg.Backend,
			BTCNetwork: cfg.Network,
		},
	}
	if err := svc.loadFoundBlocks(); err != nil {
		return nil, err
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

func (s *TemplateService) CurrentJob(req JobRequest) (*CompactMiningJob, error) {
	if req.RewardAddress == "" {
		return nil, fmt.Errorf("missing btc lucky reward address")
	}
	if req.Jobs < 1 {
		req.Jobs = 1
	}

	s.mu.Lock()
	client := s.client
	running := s.running
	s.mu.Unlock()
	if !running || client == nil {
		return nil, fmt.Errorf("btc template service is not running")
	}

	now := time.Now()
	template, err := s.currentTemplate(client)
	if err != nil {
		s.setTemplateError(err)
		return nil, err
	}
	if _, err := btcPayToAddrScript(req.RewardAddress, s.params); err != nil {
		s.setTemplateError(err)
		return nil, err
	}

	templateID := templateIDFromTemplate(template)
	jobID := fmt.Sprintf("%s:%x", templateID, now.UnixNano())
	target := template.Target
	if target == "" {
		t, err := targetFromTemplate(template)
		if err != nil {
			s.setTemplateError(err)
			return nil, err
		}
		target = fmt.Sprintf("%064x", t)
	}
	workerRanges := makeWorkerRanges(req.Jobs)
	for i := range workerRanges {
		work, err := assembleBTCWork(template, s.params, req.RewardAddress,
			req.MinerID, workerRanges[i].ExtraNonceStart, 0, template.CurTime)
		if err != nil {
			s.setTemplateError(err)
			return nil, err
		}
		workerRanges[i].MerkleRoot = work.header.MerkleRoot.String()
	}
	job := &CompactMiningJob{
		JobID:             jobID,
		TemplateID:        templateID,
		Network:           s.cfg.Network,
		Height:            template.Height,
		PreviousBlockHash: template.PreviousHash,
		Version:           template.Version,
		Bits:              template.Bits,
		Target:            target,
		CurTime:           template.CurTime,
		MinTime:           template.MinTime,
		RewardAddress:     req.RewardAddress,
		MinerID:           req.MinerID,
		WorkerRanges:      workerRanges,
	}

	s.mu.Lock()
	s.jobs[jobID] = &cachedBTCJob{job: job, template: template, params: s.params, created: now}
	s.pruneJobsLocked()
	s.status.BTCHeight = template.Height
	s.status.TemplateCacheSize = 1
	s.status.ActiveJobs = len(s.jobs)
	s.status.LastError = ""
	s.mu.Unlock()

	return job, nil
}

func (s *TemplateService) currentTemplate(client *btcrpcclient.Client) (*btcbtcjson.GetBlockTemplateResult, error) {
	s.mu.Lock()
	template := s.lastTpl
	s.mu.Unlock()
	if template != nil {
		return template, nil
	}

	return s.refreshTemplate(client)
}

func (s *TemplateService) RefreshTemplate() error {
	s.mu.Lock()
	client := s.client
	running := s.running
	s.mu.Unlock()
	if !running || client == nil {
		return fmt.Errorf("btc template service is not running")
	}
	_, err := s.refreshTemplate(client)
	return err
}

func (s *TemplateService) refreshTemplate(client *btcrpcclient.Client) (*btcbtcjson.GetBlockTemplateResult, error) {
	template, err := client.GetBlockTemplate(&btcbtcjson.TemplateRequest{
		Mode:  "template",
		Rules: []string{"segwit"},
	})
	if err != nil {
		s.setTemplateError(err)
		return nil, err
	}

	now := time.Now()

	s.mu.Lock()
	s.lastTpl = template
	s.status.LastTemplateTime = now
	s.status.BTCHeight = template.Height
	s.status.TemplateCacheSize = 1
	s.pruneJobsLocked()
	s.status.ActiveJobs = len(s.jobs)
	s.status.LastError = ""
	s.mu.Unlock()

	return template, nil
}

func (s *TemplateService) SubmitSolution(solution *MiningSolution) (*FoundBlockRecord, error) {
	s.mu.Lock()
	cached := s.jobs[solution.JobID]
	client := s.client
	s.mu.Unlock()
	if cached == nil {
		return nil, fmt.Errorf("unknown btc lucky mining job %s", solution.JobID)
	}
	if solution.TemplateID != cached.job.TemplateID {
		return nil, fmt.Errorf("btc lucky mining solution template mismatch")
	}
	if solution.RewardAddress != cached.job.RewardAddress {
		return nil, fmt.Errorf("btc lucky mining solution reward address mismatch")
	}

	work, err := assembleBTCWork(cached.template, cached.params, solution.RewardAddress,
		cached.job.MinerID, solution.ExtraNonce, solution.Nonce, solution.NTime)
	if err != nil {
		s.setSubmitResult("", err)
		return nil, err
	}
	if work.blockHash.String() != solution.HeaderHash {
		err := fmt.Errorf("solution hash mismatch: got %s want %s", work.blockHash, solution.HeaderHash)
		s.setSubmitResult("", err)
		return nil, err
	}
	target, err := targetFromTemplate(cached.template)
	if err != nil {
		s.setSubmitResult("", err)
		return nil, err
	}
	if hashToBig(&work.blockHash).Cmp(target) > 0 {
		err := fmt.Errorf("solution does not satisfy target")
		s.setSubmitResult("", err)
		return nil, err
	}

	record := FoundBlockRecord{
		BlockHash:     work.blockHash.String(),
		BlockHeight:   cached.template.Height,
		CoinbaseTxID:  work.coinbase.TxHash().String(),
		Vout:          0,
		Amount:        work.coinbase.TxOut[0].Value,
		RewardAddress: solution.RewardAddress,
		JobID:         solution.JobID,
		TemplateID:    solution.TemplateID,
		CreatedAt:     time.Now(),
	}
	if client == nil {
		err := fmt.Errorf("btc rpc client is not connected")
		record.SubmitResult = err.Error()
		s.setSubmitResult(record.SubmitResult, err)
		s.rememberFound(record)
		return &record, err
	}
	var buf bytes.Buffer
	if err := work.block.Serialize(&buf); err != nil {
		record.SubmitResult = err.Error()
		s.setSubmitResult(record.SubmitResult, err)
		s.rememberFound(record)
		return &record, err
	}
	block, err := btcbtcutil.NewBlockFromBytes(buf.Bytes())
	if err != nil {
		record.SubmitResult = err.Error()
		s.setSubmitResult(record.SubmitResult, err)
		s.rememberFound(record)
		return &record, err
	}
	err = client.SubmitBlock(block, &btcbtcjson.SubmitBlockOptions{
		WorkID: cached.template.WorkID,
	})
	if err != nil {
		record.SubmitResult = err.Error()
		s.setSubmitResult(record.SubmitResult, err)
		s.rememberFound(record)
		return &record, err
	}
	record.Submitted = true
	record.SubmitResult = "accepted"

	s.setSubmitResult(record.SubmitResult, nil)
	s.rememberFound(record)
	return &record, nil
}

func (s *TemplateService) Status() TemplateServiceStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.status
	st.Enabled = s.cfg.Enabled
	st.Backend = s.cfg.Backend
	st.BTCNetwork = s.cfg.Network
	st.Running = s.running
	st.BTCRPCConnected = s.client != nil && s.running
	st.ActiveJobs = len(s.jobs)
	if s.lastTpl != nil {
		st.BTCHeight = s.lastTpl.Height
		st.TemplateCacheSize = 1
	}
	return st
}

func (s *TemplateService) FoundBlocks() []FoundBlockRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]FoundBlockRecord, len(s.found))
	copy(out, s.found)
	return out
}

func (s *TemplateService) pruneJobsLocked() {
	limit := s.cfg.CacheLimit
	if limit <= 0 || len(s.jobs) <= limit {
		return
	}

	for len(s.jobs) > limit {
		var oldestID string
		var oldest time.Time
		for id, job := range s.jobs {
			if oldestID == "" || job.created.Before(oldest) {
				oldestID = id
				oldest = job.created
			}
		}
		delete(s.jobs, oldestID)
	}
}

func templateIDFromTemplate(template *btcbtcjson.GetBlockTemplateResult) string {
	if template == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d:%s", template.PreviousHash, template.Height, template.WorkID)
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

	log.Infof("BTC lucky found block metadata block_hash=%s height=%d coinbase_txid=%s vout=%d amount=%d reward_address=%s job_id=%s template_id=%s submitted=%v result=%q",
		record.BlockHash, record.BlockHeight, record.CoinbaseTxID, record.Vout, record.Amount,
		record.RewardAddress, record.JobID, record.TemplateID, record.Submitted, record.SubmitResult)

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
