package btclucky

import (
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	btcchainhash "github.com/btcsuite/btcd/chaincfg/chainhash"
)

const (
	lowPriorityYieldEvery = uint64(4096)
	lowPrioritySleepEvery = uint64(65536)
)

type Miner struct {
	mu      sync.Mutex
	cfg     BTCLuckyMinerConfig
	backend MiningJobBackend
	jobs    int
	speed   *speedMonitor
	status  MinerStatus
	quit    chan struct{}
	wg      sync.WaitGroup
	running bool
	jobGen  uint64
}

type tipHeightProvider interface {
	TipHeight() (int64, error)
}

func NewMiner(cfg BTCLuckyMinerConfig, backend MiningJobBackend) (*Miner, error) {
	cfg.Normalize()
	if backend == nil {
		return nil, fmt.Errorf("btc lucky mining backend is required")
	}
	if cfg.RewardAddr == "" {
		return nil, fmt.Errorf("btc lucky mining reward address is required")
	}
	jobs, err := ResolveJobCount(cfg.Jobs, cfg.ReserveCores)
	if err != nil {
		return nil, err
	}
	return &Miner{
		cfg:     cfg,
		backend: backend,
		jobs:    jobs,
		speed:   newSpeedMonitor(),
		status: MinerStatus{
			Enabled:          cfg.Enabled,
			Backend:          cfg.Backend,
			RewardAddress:    cfg.RewardAddr,
			MinerID:          cfg.MinerID,
			Jobs:             jobs,
			JobsMode:         cfg.Jobs,
			LowPriority:      cfg.LowPriority,
			LowPrioritySleep: cfg.LowPrioritySleep.String(),
		},
	}, nil
}

func (m *Miner) Start() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.quit = make(chan struct{})
	m.running = true
	m.status.Running = true
	m.mu.Unlock()

	if !m.backend.IsReady() {
		if err := m.backend.Start(); err != nil {
			m.setError(err)
			m.mu.Lock()
			m.running = false
			m.status.Running = false
			m.mu.Unlock()
			return err
		}
	}

	m.wg.Add(1)
	go m.controller()
	if _, ok := m.backend.(tipHeightProvider); ok {
		m.wg.Add(1)
		go m.tipMonitor()
	}
	log.Infof("BTC lucky miner started with %d jobs", m.jobs)
	return nil
}

func (m *Miner) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	close(m.quit)
	m.running = false
	m.status.Running = false
	m.mu.Unlock()

	m.wg.Wait()
	log.Infof("BTC lucky miner stopped")
}

func (m *Miner) IsMining() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

func (m *Miner) HashesPerSecond() float64 {
	return m.speed.HashesPerSecond()
}

func (m *Miner) NotifyBlockUpdated() {
	atomic.AddUint64(&m.jobGen, 1)
}

func (m *Miner) Status() MinerStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.status
	st.HashesPerSecond = m.speed.HashesPerSecond()
	if backend, ok := m.backend.(interface {
		FoundBlocks() []FoundBlockRecord
	}); ok {
		st.FoundBlocks = backend.FoundBlocks()
	}
	return st
}

func (m *Miner) controller() {
	defer m.wg.Done()

	for {
		select {
		case <-m.quit:
			return
		default:
		}

		job, err := m.backend.CurrentJob(JobRequest{
			Network:       m.cfg.Network,
			RewardAddress: m.cfg.RewardAddr,
			MinerID:       m.cfg.MinerID,
			Jobs:          m.jobs,
		})
		if err != nil {
			m.setError(err)
			if !sleepOrDone(m.quit, 10*time.Second) {
				return
			}
			continue
		}
		m.setJob(job)
		jobGen := atomic.LoadUint64(&m.jobGen)
		m.mineJob(job, jobGen)
	}
}

func (m *Miner) tipMonitor() {
	defer m.wg.Done()

	provider, ok := m.backend.(tipHeightProvider)
	if !ok {
		return
	}

	ticker := time.NewTicker(m.cfg.TipCheckInterval)
	defer ticker.Stop()

	var lastTip int64
	checkTip := func() {
		height, err := provider.TipHeight()
		if err != nil {
			m.setError(err)
			return
		}
		if height <= 0 {
			return
		}

		m.mu.Lock()
		jobHeight := m.status.BTCHeight
		m.mu.Unlock()

		if lastTip == 0 {
			lastTip = height
			if jobHeight > 0 && height > jobHeight {
				m.NotifyBlockUpdated()
			}
			return
		}
		if height != lastTip || (jobHeight > 0 && height > jobHeight) {
			lastTip = height
			m.NotifyBlockUpdated()
		}
	}

	checkTip()
	for {
		select {
		case <-m.quit:
			return
		case <-ticker.C:
			checkTip()
		}
	}
}

func (m *Miner) mineJob(job *CompactMiningJob, jobGen uint64) {
	var workerWg sync.WaitGroup
	for _, r := range job.WorkerRanges {
		workerRange := r
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			m.mineRange(job, workerRange, jobGen)
		}()
	}

	done := make(chan struct{})
	go func() {
		workerWg.Wait()
		close(done)
	}()

	select {
	case <-m.quit:
		workerWg.Wait()
	case <-done:
	}
}

func (m *Miner) mineRange(job *CompactMiningJob, r WorkerRange, jobGen uint64) {
	target, ok := new(big.Int).SetString(job.Target, 16)
	if !ok {
		m.setError(fmt.Errorf("invalid btc lucky target %q", job.Target))
		return
	}

	for extraNonce := r.ExtraNonceStart; extraNonce <= r.ExtraNonceEnd; extraNonce++ {
		for nonce := uint32(0); ; nonce++ {
			select {
			case <-m.quit:
				return
			default:
			}
			if atomic.LoadUint64(&m.jobGen) != jobGen {
				return
			}

			hash, err := m.currentWork(job, r, nonce)
			if err != nil {
				m.setError(err)
				return
			}
			m.speed.Add(1)
			m.lowPriorityPause(uint64(nonce))
			m.updateBestShare(hash.String())

			if hashToBig(hash).Cmp(target) <= 0 {
				solution := &MiningSolution{
					JobID:         job.JobID,
					TemplateID:    job.TemplateID,
					Network:       job.Network,
					RewardAddress: job.RewardAddress,
					WorkerID:      r.WorkerID,
					ExtraNonce:    extraNonce,
					NTime:         job.CurTime,
					Nonce:         nonce,
					HeaderHash:    hash.String(),
				}
				record, err := m.backend.SubmitSolution(solution)
				if err != nil {
					if record != nil {
						log.Infof("BTC lucky miner found block submit failed block_hash=%s height=%d coinbase_txid=%s vout=%d amount=%d reward_address=%s job_id=%s template_id=%s worker_id=%d submitted=%v result=%q error=%q",
							record.BlockHash, record.BlockHeight, record.CoinbaseTxID, record.Vout, record.Amount,
							record.RewardAddress, record.JobID, record.TemplateID, solution.WorkerID,
							record.Submitted, record.SubmitResult, err.Error())
					} else {
						log.Infof("BTC lucky miner found candidate submit failed header_hash=%s reward_address=%s job_id=%s template_id=%s worker_id=%d extra_nonce=%d nonce=%d ntime=%d error=%q",
							solution.HeaderHash, solution.RewardAddress, solution.JobID, solution.TemplateID,
							solution.WorkerID, solution.ExtraNonce, solution.Nonce, solution.NTime, err.Error())
					}
					m.setSubmit(err.Error())
					return
				}
				log.Infof("BTC lucky miner found block submitted block_hash=%s height=%d coinbase_txid=%s vout=%d amount=%d reward_address=%s job_id=%s template_id=%s worker_id=%d submitted=%v result=%q",
					record.BlockHash, record.BlockHeight, record.CoinbaseTxID, record.Vout, record.Amount,
					record.RewardAddress, record.JobID, record.TemplateID, solution.WorkerID,
					record.Submitted, record.SubmitResult)
				m.setSubmit(record.SubmitResult)
				return
			}

			if nonce == ^uint32(0) {
				break
			}
		}
		if extraNonce == ^uint64(0) {
			return
		}
	}
}

func (m *Miner) lowPriorityPause(nonce uint64) {
	if nonce%lowPriorityYieldEvery == 0 {
		runtime.Gosched()
	}
	if m.cfg.LowPriority && nonce%lowPrioritySleepEvery == 0 {
		time.Sleep(m.cfg.LowPrioritySleep)
	}
}

func (m *Miner) currentWork(job *CompactMiningJob, r WorkerRange, nonce uint32) (*btcchainhash.Hash, error) {
	hash, err := hashCompactJobHeader(job, r, nonce)
	if err != nil {
		return nil, err
	}
	return &hash, nil
}

func (m *Miner) setJob(job *CompactMiningJob) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.JobID = job.JobID
	m.status.CurrentTarget = job.Target
	m.status.BTCHeight = job.Height
	m.status.LastJobTime = time.Now()
	m.status.LastError = ""
}

func (m *Miner) setError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		m.status.LastError = err.Error()
	}
}

func (m *Miner) updateBestShare(hash string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.status.BestShare == "" || hash < m.status.BestShare {
		m.status.BestShare = hash
	}
}

func (m *Miner) setSubmit(result string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.LastSubmitTime = time.Now()
	m.status.LastSubmitResult = result
}

func sleepOrDone(done <-chan struct{}, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-done:
		return false
	case <-timer.C:
		return true
	}
}
