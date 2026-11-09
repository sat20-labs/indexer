package btclucky

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	btcchainhash "github.com/btcsuite/btcd/chaincfg/chainhash"
)

const (
	lowPriorityYieldEvery = uint64(4096)
	lowPrioritySleepEvery = uint64(65536)
)

type Miner struct {
	mu      sync.Mutex
	cfg     BTCLuckyMinerConfig
	backend MiningBackend
	jobs    int
	speed   *speedMonitor
	status  MinerStatus
	quit    chan struct{}
	wg      sync.WaitGroup
	running bool
	params  *chaincfg.Params
}

func NewMiner(cfg BTCLuckyMinerConfig, backend MiningBackend) (*Miner, error) {
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
	params, err := BTCChainParams(cfg.Network)
	if err != nil {
		return nil, err
	}
	if _, err := btcPayToAddrScript(cfg.RewardAddr, params); err != nil {
		return nil, err
	}
	return &Miner{
		params:  params,
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
	m.wg.Add(1)
	go m.controller()
	m.mu.Unlock()
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

func (m *Miner) Status() MinerStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.status
	if m.running {
		st.HashesPerSecond = m.speed.HashesPerSecond()
	}
	if backend, ok := m.backend.(interface {
		FoundBlocks() []FoundBlockRecord
	}); ok {
		st.FoundBlocks = backend.FoundBlocks()
	}
	return st
}

func (m *Miner) controller() {
	defer m.wg.Done()
	updates := make(chan *BlockTemplate)
	refresh := make(chan struct{}, 1)
	var monitor sync.WaitGroup
	monitor.Add(1)
	go func() { defer monitor.Done(); m.watchTemplates(updates, refresh) }()
	defer monitor.Wait()
	var tpl *BlockTemplate
	for {
		if tpl == nil {
			select {
			case <-m.quit:
				return
			case tpl = <-updates:
			}
		}
		m.setTemplate(tpl)
		next, candidate, err := m.mineTemplate(tpl, updates)
		if err != nil {
			m.setError(err)
		}
		if len(candidate) > 0 {
			m.submit(candidate)
			select {
			case refresh <- struct{}{}:
			default:
			}
		}
		tpl = next
	}
}

// One outstanding GET per miner. A slow template fetch must never delay
// publishing a block that another goroutine has already solved.
func (m *Miner) watchTemplates(updates chan<- *BlockTemplate, refresh <-chan struct{}) {
	ticker := time.NewTicker(m.cfg.TipCheckInterval)
	defer ticker.Stop()
	var previous string
	var fetchFailed bool
	lastLog := time.Now()
	for {
		t, err := m.backend.CurrentTemplate()
		if err != nil || fetchFailed {
			m.setError(err)
		}
		fetchFailed = err != nil
		if err == nil && t != nil && t.ID != previous {
			select {
			case updates <- t:
				previous = t.ID
			case <-m.quit:
				return
			}
		}
		if time.Since(lastLog) >= 30*time.Second {
			st := m.Status()
			log.Infof("BTC lucky miner: workers=%d hashrate=%.2f H/s height=%d template=%s error=%q", st.Jobs, st.HashesPerSecond, st.BTCHeight, st.TemplateID, st.LastError)
			lastLog = time.Now()
		}
		select {
		case <-m.quit:
			return
		case <-refresh:
		case <-ticker.C:
		}
	}
}

func (m *Miner) submit(raw []byte) {
	for {
		record, err := m.backend.SubmitBlock(raw)
		if err == nil && record != nil {
			m.setSubmit(record.SubmitResult)
			log.Infof("BTC lucky block submitted hash=%s result=%s", record.BlockHash, record.SubmitResult)
			return
		}
		if err == nil {
			err = fmt.Errorf("missing Bitcoin submission result")
		}
		m.setSubmit(err.Error())
		if !errors.Is(err, ErrCoreUnavailable) || !sleepOrDone(m.quit, time.Second) {
			return
		}
	}
}

// Continue hashing the downloaded template during an indexer outage. Stop
// workers only after a replacement is in hand, on success, or on shutdown.
func (m *Miner) mineTemplate(t *BlockTemplate, updates <-chan *BlockTemplate) (*BlockTemplate, []byte, error) {
	params, err := BTCChainParams(t.Network)
	if err != nil || params.Net != m.params.Net {
		return nil, nil, fmt.Errorf("unexpected template network %q", t.Network)
	}
	bits, err := strconv.ParseUint(t.Bits, 16, 32)
	if err != nil {
		return nil, nil, err
	}
	difficulty := blockchain.CompactToBig(uint32(bits))
	if difficulty.Sign() <= 0 || difficulty.Cmp(params.PowLimit) > 0 {
		return nil, nil, fmt.Errorf("invalid Bitcoin bits %q", t.Bits)
	}
	target, err := btcchainhash.NewHashFromStr(fmt.Sprintf("%064x", difficulty))
	if err != nil || (t.Target != "" && target.String() != t.Target) {
		return nil, nil, fmt.Errorf("inconsistent Bitcoin target %q", t.Target)
	}
	var seed [8]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return nil, nil, err
	}
	start := binary.LittleEndian.Uint64(seed[:])
	type result struct {
		raw []byte
		err error
	}
	found := make(chan result, 1)
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < m.jobs; i++ {
		wg.Add(1)
		go func(extra uint64) {
			defer wg.Done()
			raw, err := m.mineRange(t, target, extra, stop)
			if err != nil || raw != nil {
				select {
				case found <- result{raw, err}:
				default:
				}
			}
		}(start + uint64(i))
	}
	var next *BlockTemplate
	var hit result
	select {
	case <-m.quit:
	case hit = <-found:
	case next = <-updates:
	}
	close(stop)
	wg.Wait()
	// A worker can find a valid block while the replacement GET is in flight.
	if hit.raw == nil && hit.err == nil {
		select {
		case hit = <-found:
		default:
		}
	}
	return next, hit.raw, hit.err
}

func (m *Miner) mineRange(t *BlockTemplate, target *btcchainhash.Hash, extra uint64, stop <-chan struct{}) ([]byte, error) {
	var count uint64
	var best btcchainhash.Hash
	lastReport := time.Now()
	flush := func() {
		if count > 0 {
			m.speed.Add(count)
			m.updateBestShare(best.String())
			count = 0
		}
	}
	defer flush()
	for ; ; extra += uint64(m.jobs) {
		cb, header, err := t.work(m.params, m.cfg.RewardAddr, extra)
		if err != nil {
			return nil, err
		}
		var data miningHeader
		if err := header.Serialize(bytes.NewBuffer(data[:0])); err != nil {
			return nil, err
		}
		for nonce := uint32(0); ; nonce++ {
			if nonce%256 == 0 {
				select {
				case <-stop:
					return nil, nil
				case <-m.quit:
					return nil, nil
				default:
				}
				if time.Since(lastReport) >= time.Second {
					flush()
					lastReport = time.Now()
				}
			}
			hash := data.hashNonce(nonce)
			if count == 0 || lessWorkHash(&hash, &best) {
				best = hash
			}
			count++
			if !lessWorkHash(target, &hash) {
				header.Nonce = nonce
				return t.blockBytes(cb, header)
			}
			m.lowPriorityPause(uint64(nonce), stop)
			if nonce == ^uint32(0) {
				break
			}
		}
	}
}

// Bitcoin stores a work hash with its least significant byte first.
func lessWorkHash(a, b *btcchainhash.Hash) bool {
	for i := len(a) - 1; i >= 0; i-- {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

func (m *Miner) lowPriorityPause(nonce uint64, stop <-chan struct{}) {
	if nonce%lowPriorityYieldEvery == 0 {
		if runtime.GOOS == "js" && !m.cfg.LowPriority {
			// Go scheduling alone cannot return control to the browser event loop.
			time.Sleep(time.Millisecond)
		} else {
			runtime.Gosched()
		}
	}
	if m.cfg.LowPriority && nonce%lowPrioritySleepEvery == 0 {
		sleepOrDone(stop, m.cfg.LowPrioritySleep)
	}
}

func (m *Miner) setTemplate(t *BlockTemplate) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.TemplateID = t.ID
	m.status.CurrentTarget = t.Target
	m.status.BTCHeight = t.Height
	m.status.LastTemplateTime = time.Now()
	m.status.LastError = ""
}

func (m *Miner) setError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err == nil {
		m.status.LastError = ""
	} else {
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
