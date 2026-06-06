package atom

import (
	"os"
	"runtime"
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func (s *Indexer) setDBVersion() {
	if err := db.SetRawValueToDB([]byte(DB_VER_KEY), []byte(DB_VERSION), s.db); err != nil {
		common.Log.Panicf("atom set db version failed: %v", err)
	}
}

func (s *Indexer) getDBVersion() string {
	value, err := db.GetRawValueFromDB([]byte(DB_VER_KEY), s.db)
	if err != nil {
		return ""
	}
	return string(value)
}

func (s *Indexer) loadStatusFromDB() *Status {
	status := &Status{Version: DB_VERSION}
	err := db.GetValueFromDB([]byte(DB_STATUS_KEY), status, s.db)
	if err == common.ErrKeyNotFound {
		return status
	}
	if err != nil {
		common.Log.Panicf("atom load status failed: %v", err)
	}
	if status.Version != DB_VERSION {
		common.Log.Panicf("atom db version inconsistent %s != %s", status.Version, DB_VERSION)
	}
	return status
}

func (s *Indexer) loadTickersFromDB() {
	err := s.db.BatchRead([]byte(DB_PREFIX_TICKER), false, func(k, v []byte) error {
		var ticker Ticker
		if err := db.DecodeBytes(v, &ticker); err != nil {
			return err
		}
		name := strings.ToLower(ticker.Name)
		s.tickerMap[name] = &ticker
		s.tickerById[ticker.Id] = name
		return nil
	})
	if err != nil {
		common.Log.Panicf("atom load tickers failed: %v", err)
	}
}

func (s *Indexer) loadUtxoBalancesFromDB() {
	err := s.db.BatchRead([]byte(DB_PREFIX_UTXO_BALANCE), false, func(k, v []byte) error {
		var balance UtxoBalance
		if err := db.DecodeBytes(v, &balance); err != nil {
			return err
		}
		s.addLoadedUtxoBalanceInMemory(&balance)
		return nil
	})
	if err != nil {
		common.Log.Panicf("atom load utxo balances failed: %v", err)
	}
}

func (s *Indexer) loadMintHistoryFromDB() {
	err := s.db.BatchRead([]byte(DB_PREFIX_MINTHISTORY), false, func(k, v []byte) error {
		var mint MintInfo
		if err := db.DecodeBytes(v, &mint); err != nil {
			return err
		}
		ticker := strings.ToLower(mint.Ticker)
		s.mintHistory[ticker] = append(s.mintHistory[ticker], &mint)
		return nil
	})
	if err != nil {
		common.Log.Panicf("atom load mint history failed: %v", err)
	}
	for ticker := range s.mintHistory {
		sortMintHistory(s.mintHistory[ticker])
	}
}

func sortMintHistory(items []*MintInfo) {
	for i := 1; i < len(items); i++ {
		item := items[i]
		j := i - 1
		for ; j >= 0 && items[j].Id > item.Id; j-- {
			items[j+1] = items[j]
		}
		items[j+1] = item
	}
}

func (s *Indexer) UpdateDB() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.setDBVersion()
	wb := s.db.NewWriteBatch()
	defer wb.Close()
	if err := db.SetDB([]byte(DB_STATUS_KEY), s.status, wb); err != nil {
		common.Log.Panicf("atom write status failed: %v", err)
	}
	for name, ticker := range s.tickerTouched {
		if err := db.SetDB([]byte(GetTickerKey(name)), ticker, wb); err != nil {
			common.Log.Panicf("atom write ticker failed: %v", err)
		}
	}
	for id, name := range s.tickerIdAdded {
		if err := db.SetDB([]byte(GetTickerIdKey(id)), name, wb); err != nil {
			common.Log.Panicf("atom write ticker id failed: %v", err)
		}
	}
	for key, balance := range s.utxoTouched {
		if err := db.SetDB([]byte(key), balance, wb); err != nil {
			common.Log.Panicf("atom write utxo balance failed: %v", err)
		}
		if err := db.SetDB([]byte(GetTickerUtxoKey(balance.Ticker, balance.UtxoId, balance.AtomicalId)), balance.Amount, wb); err != nil {
			common.Log.Panicf("atom write ticker utxo failed: %v", err)
		}
	}
	for key, balance := range s.utxoDeleted {
		if err := wb.Delete([]byte(key)); err != nil {
			common.Log.Panicf("atom delete utxo balance failed: %v", err)
		}
		if err := wb.Delete([]byte(GetTickerUtxoKey(balance.Ticker, balance.UtxoId, balance.AtomicalId))); err != nil {
			common.Log.Panicf("atom delete ticker utxo failed: %v", err)
		}
	}
	for key, amount := range s.holderTouched {
		if amount == 0 {
			if err := wb.Delete([]byte(key)); err != nil {
				common.Log.Panicf("atom delete holder failed: %v", err)
			}
			continue
		}
		if err := db.SetDB([]byte(key), amount, wb); err != nil {
			common.Log.Panicf("atom write holder failed: %v", err)
		}
	}
	for _, mint := range s.mintsAdded {
		if err := db.SetDB([]byte(GetMintHistoryKey(mint.Ticker, mint.Id)), mint, wb); err != nil {
			common.Log.Panicf("atom write mint history failed: %v", err)
		}
	}
	for _, action := range s.actionsAdded {
		if err := db.SetDB([]byte(GetActionKey(action.Height, action.TxIndex, action.Id)), action, wb); err != nil {
			common.Log.Panicf("atom write action failed: %v", err)
		}
	}
	if err := wb.Flush(); err != nil {
		common.Log.Panicf("atom flush failed: %v", err)
	}
	s.logDebugMemoryLocked()

	s.tickerTouched = make(map[string]*Ticker)
	s.tickerIdAdded = make(map[int64]string)
	s.utxoTouched = make(map[string]*UtxoBalance)
	s.utxoDeleted = make(map[string]*UtxoBalance)
	s.holderTouched = make(map[string]int64)
	s.mintsAdded = nil
	s.actionsAdded = nil
}

func (s *Indexer) logDebugMemoryLocked() {
	if os.Getenv("ATOM_DEBUG_MEMORY") == "" {
		return
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	common.Log.Infof(
		"AtomIndexer.Memory height=%d alloc_mb=%d sys_mb=%d num_gc=%d tickers=%d utxos=%d holders=%d ticker_utxos=%d mint_history_tickers=%d pending_tickers=%d pending_ticker_ids=%d pending_utxos=%d pending_deleted=%d pending_holders=%d pending_mints=%d pending_actions=%d",
		s.status.Height,
		m.Alloc/1024/1024,
		m.Sys/1024/1024,
		m.NumGC,
		len(s.tickerMap),
		len(s.utxoBalances),
		len(s.holderBalances),
		len(s.tickerUtxos),
		len(s.mintHistory),
		len(s.tickerTouched),
		len(s.tickerIdAdded),
		len(s.utxoTouched),
		len(s.utxoDeleted),
		len(s.holderTouched),
		len(s.mintsAdded),
		len(s.actionsAdded),
	)
}
