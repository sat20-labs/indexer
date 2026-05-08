package ft

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func (s *FTIndexer) initTickInfoFromDB(tickerName string) *TickInfo {
	tickinfo := newTickerInfo(tickerName)
	tickinfo.Ticker = s.getTickerFromDB(tickerName)
	s.loadMintInfoFromDB(tickinfo)
	return tickinfo
}

func (s *FTIndexer) loadMintInfoFromDB(tickinfo *TickInfo) {
	mintList := s.loadMintDataFromDB(tickinfo.Name)
	for _, mint := range mintList {
		tickinfo.InscriptionMap[mint.Base.InscriptionId] = common.NewMintAbbrInfo(mint)
	}
}

func (s *FTIndexer) loadHolderInfoFromDB() map[uint64]*HolderInfo {
	count := 0
	startTime := time.Now()
	common.Log.Debug("FTIndexer loadHolderInfoFromDB ...")
	result := make(map[uint64]*HolderInfo, 0)
	err := s.db.BatchRead([]byte(DB_PREFIX_TICKER_HOLDER), false, func(k, v []byte) error {

		key := string(k)
		utxo, err := parseHolderInfoKey(key)
		if err != nil {
			common.Log.Errorln(key + " " + err.Error())
		} else {
			var info HolderInfo
			err = db.DecodeBytes(v, &info)
			if err == nil {
				result[utxo] = &info
			} else {
				common.Log.Errorln("DecodeBytes " + err.Error())
			}

		}
		count++

		return nil
	})

	if err != nil {
		common.Log.Panicf("Error prefetching HolderInfo from db: %v", err)
	}

	elapsed := time.Since(startTime).Milliseconds()
	common.Log.Infof("loadHolderInfoFromDB loaded %d records in %d ms", count, elapsed)

	return result
}

func (s *FTIndexer) loadUtxoMapFromDB() map[string]map[uint64]int64 {
	count := 0
	startTime := time.Now()
	common.Log.Info("loadUtxoMapFromDB ...")
	result := make(map[string]map[uint64]int64, 0)
	err := s.db.BatchRead([]byte(DB_PREFIX_TICKER_UTXO), false, func(k, v []byte) error {

		key := string(k)

		ticker, utxo, err := parseTickUtxoKey(key)
		if err != nil {
			common.Log.Errorln(key + " " + err.Error())
		} else {
			var amount int64

			err = db.DecodeBytes(v, &amount)
			if err == nil {
				oldmap, ok := result[ticker]
				if ok {
					oldmap[utxo] = amount
				} else {
					utxomap := make(map[uint64]int64, 0)
					utxomap[utxo] = amount
					result[ticker] = utxomap
				}
			} else {
				common.Log.Errorln("DecodeBytes " + err.Error())
			}

		}
		count++

		return nil
	})

	if err != nil {
		common.Log.Panicf("Error prefetching HolderInfo from db: %v", err)
	}

	elapsed := time.Since(startTime).Milliseconds()
	common.Log.Infof("loadUtxoMapFromDB loaded %d records in %d ms", count, elapsed)

	return result
}

func (s *FTIndexer) loadFreezeStateFromDB() map[string]map[uint64]*common.FreezeState {
	result := make(map[string]map[uint64]*common.FreezeState)
	_ = s.db.BatchRead([]byte(DB_PREFIX_FREEZE_STATE), false, func(k, v []byte) error {
		var item common.FreezeState
		if err := db.DecodeBytes(v, &item); err != nil {
			return nil
		}
		tickerMap, ok := result[item.Ticker]
		if !ok {
			tickerMap = make(map[uint64]*common.FreezeState)
			result[item.Ticker] = tickerMap
		}
		state := item
		tickerMap[item.AddressId] = &state
		return nil
	})
	return result
}

func (s *FTIndexer) loadTickListFromDB() []string {
	result := make([]string, 0)
	count := 0
	startTime := time.Now()
	common.Log.Debug("loadTickListFromDB ...")
	err := s.db.BatchRead([]byte(DB_PREFIX_TICKER), false, func(k, v []byte) error {

		key := string(k)
		tickname, err := parseTickListKey(key)
		if err == nil {
			result = append(result, tickname)
		}
		count++

		return nil
	})
	if err != nil {
		common.Log.Panicf("Error prefetching ticklist from db: %v", err)
	}

	elapsed := time.Since(startTime).Milliseconds()
	common.Log.Infof("loadTickListFromDB loaded %d records in %d ms", count, elapsed)

	return result
}

func (s *FTIndexer) getTickListFromDB() []string {
	return s.loadTickListFromDB()
}

// key: utxo
func (s *FTIndexer) getMintListFromDB(tickname string) map[string]*common.Mint {
	return s.loadMintDataFromDB(tickname)
}

func (s *FTIndexer) getMintFromDB(ticker, inscriptionId string) *common.Mint {
	var result common.Mint
	key := GetMintHistoryKey(ticker, inscriptionId)
	err := db.GetValueFromDB([]byte(key), &result, s.db)
	if err == common.ErrKeyNotFound {
		common.Log.Debugf("GetMintFromDB key: %s, error: ErrKeyNotFound ", key)
		return nil
	} else if err != nil {
		common.Log.Errorf("GetMintFromDB error: %v", err)
		return nil
	}

	return &result
}

func (s *FTIndexer) loadMintDataFromDB(tickerName string) map[string]*common.Mint {
	result := make(map[string]*common.Mint, 0)
	count := 0
	startTime := time.Now()
	common.Log.Debug("loadMintDataFromDB ...")
	err := s.db.BatchRead([]byte(DB_PREFIX_MINTHISTORY+tickerName+"-"), false, func(k, v []byte) error {

		key := string(k)

		tick, utxo, _ := ParseMintHistoryKey(key)
		if tick == tickerName {
			var mint common.Mint

			err := db.DecodeBytes(v, &mint)
			if err == nil {
				result[utxo] = &mint
			} else {
				common.Log.Errorln("loadMintDataFromDB DecodeBytes " + err.Error())
			}

		}
		count++

		return nil
	})

	if err != nil {
		common.Log.Panicf("Error prefetching MintHistory %s from db: %v", tickerName, err)
	}

	elapsed := time.Since(startTime).Milliseconds()
	common.Log.Infof("loadMintDataFromDB %s loaded %d records in %d ms", tickerName, count, elapsed)

	return result
}

func (s *FTIndexer) getTickerFromDB(tickerName string) *common.Ticker {
	var result common.Ticker

	key := DB_PREFIX_TICKER + tickerName
	err := db.GetValueFromDB([]byte(key), &result, s.db)
	if err == common.ErrKeyNotFound {
		common.Log.Debugf("getTickerFromDB key: %s, error: ErrKeyNotFound ", key)
		return nil
	} else if err != nil {
		common.Log.Errorf("getTickerFromDB error: %v", err)
		return nil
	}

	return &result
}

func (s *FTIndexer) getUnbindHistoryFromDB(ticker string, addressId *uint64) []*common.UnbindHistory {
	result := make([]*common.UnbindHistory, 0)
	prefix := DB_PREFIX_UNBIND_TICKER + strings.ToLower(ticker) + "-"
	if addressId != nil {
		prefix += strconv.FormatUint(*addressId, 10) + "-"
	}

	_ = s.db.BatchRead([]byte(prefix), false, func(k, v []byte) error {
		var item common.UnbindHistory
		if err := db.DecodeBytes(v, &item); err == nil {
			result = append(result, &item)
		}
		return nil
	})

	sort.Slice(result, func(i, j int) bool {
		if result[i].AddressId != result[j].AddressId {
			return result[i].AddressId < result[j].AddressId
		}
		return result[i].UtxoId < result[j].UtxoId
	})

	return result
}

func (s *FTIndexer) getFreezeHistoryFromDB(ticker string, addressId *uint64) []*common.FreezeHistory {
	result := make([]*common.FreezeHistory, 0)
	prefix := DB_PREFIX_FREEZE_HISTORY + strings.ToLower(ticker) + "-"
	if addressId != nil {
		prefix += strconv.FormatUint(*addressId, 10) + "-"
	}

	_ = s.db.BatchRead([]byte(prefix), false, func(k, v []byte) error {
		var item common.FreezeHistory
		if err := db.DecodeBytes(v, &item); err == nil {
			result = append(result, &item)
		}
		return nil
	})

	sort.Slice(result, func(i, j int) bool {
		if result[i].AddressId != result[j].AddressId {
			return result[i].AddressId < result[j].AddressId
		}
		if result[i].ConfirmHeight != result[j].ConfirmHeight {
			return result[i].ConfirmHeight < result[j].ConfirmHeight
		}
		if result[i].FreezeHeight != result[j].FreezeHeight {
			return result[i].FreezeHeight < result[j].FreezeHeight
		}
		if result[i].Action != result[j].Action {
			return result[i].Action < result[j].Action
		}
		return result[i].TxId < result[j].TxId
	})

	return result
}
