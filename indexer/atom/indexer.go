package atom

import (
	"sort"
	"strings"
	"sync"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/base"
)

type Indexer struct {
	db             common.KVDB
	baseIndexer    *base.BaseIndexer
	chaincfgParam  *chaincfg.Params
	heights        ActivationHeights
	status         *Status
	mutex          sync.RWMutex
	tickerMap      map[string]*Ticker
	tickerById     map[int64]string
	utxoBalances   map[uint64]map[string]*UtxoBalance
	holderBalances map[uint64]map[string]int64
	tickerHolders  map[string]map[uint64]int64
	tickerUtxos    map[string]map[uint64]int64
	mintHistory    map[string][]*MintInfo

	tickerTouched map[string]*Ticker
	tickerIdAdded map[int64]string
	utxoTouched   map[string]*UtxoBalance
	utxoDeleted   map[string]*UtxoBalance
	holderTouched map[string]int64
	mintsAdded    []*MintInfo
	actionsAdded  []*ActionHistory
}

func NewIndexer(db common.KVDB, param *chaincfg.Params) *Indexer {
	heights := ActivationHeights{
		Activation:     AtomicalsActivationMainnet,
		Dmint:          AtomicalsActivationDmintMainnet,
		Commitz:        AtomicalsActivationCommitzMainnet,
		Density:        AtomicalsActivationDensityMainnet,
		Rollover:       AtomicalsActivationRolloverMainnet,
		CustomColoring: AtomicalsActivationColoringMainnet,
	}
	if param.Net != wire.MainNet {
		heights = ActivationHeights{
			Activation:     AtomicalsActivationTestnet4,
			Dmint:          AtomicalsActivationTestnet4,
			Commitz:        AtomicalsActivationTestnet4,
			Density:        AtomicalsActivationTestnet4,
			Rollover:       AtomicalsActivationTestnet4,
			CustomColoring: AtomicalsActivationTestnet4,
		}
	}
	return &Indexer{
		db:             db,
		chaincfgParam:  param,
		heights:        heights,
		status:         &Status{Version: DB_VERSION},
		tickerMap:      make(map[string]*Ticker),
		tickerById:     make(map[int64]string),
		utxoBalances:   make(map[uint64]map[string]*UtxoBalance),
		holderBalances: make(map[uint64]map[string]int64),
		tickerHolders:  make(map[string]map[uint64]int64),
		tickerUtxos:    make(map[string]map[uint64]int64),
		mintHistory:    make(map[string][]*MintInfo),
		tickerTouched:  make(map[string]*Ticker),
		tickerIdAdded:  make(map[int64]string),
		utxoTouched:    make(map[string]*UtxoBalance),
		utxoDeleted:    make(map[string]*UtxoBalance),
		holderTouched:  make(map[string]int64),
	}
}

func (s *Indexer) Init(baseIndexer *base.BaseIndexer) {
	s.baseIndexer = baseIndexer
	s.status = s.loadStatusFromDB()
	s.loadTickersFromDB()
	s.loadUtxoBalancesFromDB()
	s.loadMintHistoryFromDB()
}

func (s *Indexer) Clone(baseIndexer *base.BaseIndexer) *Indexer {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	clone := NewIndexer(s.db, s.chaincfgParam)
	clone.baseIndexer = baseIndexer
	clone.status = s.status.Clone()
	for k, v := range s.tickerTouched {
		clone.tickerTouched[k] = v.Clone()
	}
	for k, v := range s.tickerIdAdded {
		clone.tickerIdAdded[k] = v
	}
	for k, v := range s.utxoTouched {
		clone.utxoTouched[k] = v.Clone()
	}
	for k, v := range s.utxoDeleted {
		clone.utxoDeleted[k] = v.Clone()
	}
	for k, v := range s.holderTouched {
		clone.holderTouched[k] = v
	}
	for _, v := range s.mintsAdded {
		clone.mintsAdded = append(clone.mintsAdded, v.Clone())
	}
	for _, v := range s.actionsAdded {
		n := *v
		clone.actionsAdded = append(clone.actionsAdded, &n)
	}
	return clone
}

func (s *Indexer) Subtract(backup *Indexer) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Only remove pending data that is still identical to the flushed backup.
	// A key may be updated again after prepareDBBuffer; that newer value must
	// remain pending for the next UpdateDB, otherwise persisted helper indexes
	// such as ticker holders can drift from the UTXO balance state.
	for k, v := range backup.tickerTouched {
		if current := s.tickerTouched[k]; current == nil || *current == *v {
			delete(s.tickerTouched, k)
		}
	}
	for k, v := range backup.tickerIdAdded {
		if s.tickerIdAdded[k] == v {
			delete(s.tickerIdAdded, k)
		}
	}
	for k, v := range backup.utxoTouched {
		if current := s.utxoTouched[k]; current == nil || *current == *v {
			delete(s.utxoTouched, k)
		}
	}
	for k, v := range backup.utxoDeleted {
		if current := s.utxoDeleted[k]; current == nil || *current == *v {
			delete(s.utxoDeleted, k)
		}
	}
	for k, v := range backup.holderTouched {
		if s.holderTouched[k] == v {
			delete(s.holderTouched, k)
		}
	}
	s.mintsAdded = filterFlushedMints(s.mintsAdded, backup.mintsAdded)
	s.actionsAdded = filterFlushedActions(s.actionsAdded, backup.actionsAdded)
}

func filterFlushedMints(current, flushed []*MintInfo) []*MintInfo {
	if len(current) == 0 || len(flushed) == 0 {
		return current
	}
	flushedIds := make(map[int64]bool, len(flushed))
	for _, item := range flushed {
		flushedIds[item.Id] = true
	}
	result := make([]*MintInfo, 0, len(current))
	for _, item := range current {
		if !flushedIds[item.Id] {
			result = append(result, item)
		}
	}
	return result
}

func filterFlushedActions(current, flushed []*ActionHistory) []*ActionHistory {
	if len(current) == 0 || len(flushed) == 0 {
		return current
	}
	flushedIds := make(map[int64]bool, len(flushed))
	for _, item := range flushed {
		flushedIds[item.Id] = true
	}
	result := make([]*ActionHistory, 0, len(current))
	for _, item := range current {
		if !flushedIds[item.Id] {
			result = append(result, item)
		}
	}
	return result
}

func (s *Indexer) GetDBVersion() string {
	return s.getDBVersion()
}

func (s *Indexer) GetTickersWithRange(start, limit int) ([]string, int) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	ids := make([]int, 0, len(s.tickerById))
	for id := range s.tickerById {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	total := len(ids)
	if limit <= 0 || start+limit > total {
		limit = total - start
	}
	if start < 0 || start >= total || limit <= 0 {
		return nil, total
	}
	result := make([]string, 0, limit)
	for _, id := range ids[start : start+limit] {
		name := s.tickerById[int64(id)]
		assetName := common.TickerName{Protocol: common.PROTOCOL_NAME_ATOM, Type: common.ASSET_TYPE_FT, Ticker: name}
		result = append(result, assetName.String())
	}
	return result, total
}

func (s *Indexer) GetTicker(name string) *Ticker {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.getTickerLocked(strings.ToLower(name)).Clone()
}

func (s *Indexer) TickExisted(name string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.getTickerLocked(strings.ToLower(name)) != nil
}

func (s *Indexer) getTickerLocked(name string) *Ticker {
	t, ok := s.tickerMap[strings.ToLower(name)]
	if ok {
		return t
	}
	return nil
}
