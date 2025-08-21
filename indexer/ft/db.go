package ft

import (
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
		for _, rng := range mint.Ordinals {
			tickinfo.MintInfo.AddMintInfo(rng, mint.Base.InscriptionId)
		}

		tickinfo.InscriptionMap[mint.Base.InscriptionId] = common.NewMintAbbrInfo(mint)
	}
}

func (s *FTIndexer) loadHolderInfoFromDB() map[uint64]*HolderInfo {
	count := 0
	startTime := time.Now()
	common.Log.Info("loadHolderInfoFromDB ...")
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
	common.Log.Infof("loadHolderInfoFromDB loaded %d records in %d ms\n", count, elapsed)

	return result
}

func (s *FTIndexer) loadUtxoMapFromDB() map[string]*map[uint64]int64 {
	count := 0
	startTime := time.Now()
	common.Log.Info("loadUtxoMapFromDB ...")
	result := make(map[string]*map[uint64]int64, 0)
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
					(*oldmap)[utxo] = amount
				} else {
					utxomap := make(map[uint64]int64, 0)
					utxomap[utxo] = amount
					result[ticker] = &utxomap
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
	common.Log.Infof("loadHolderInfoFromDB loaded %d records in %d ms\n", count, elapsed)

	return result
}

func (s *FTIndexer) loadTickListFromDB() []string {
	result := make([]string, 0)
	count := 0
	startTime := time.Now()
	common.Log.Info("loadTickListFromDB ...")
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
	common.Log.Infof("loadTickListFromDB loaded %d records in %d ms\n", count, elapsed)

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
	key := GetMintHistoryKey(strings.ToLower(ticker), inscriptionId)
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
	common.Log.Info("loadMintDataFromDB ...")
	err := s.db.BatchRead([]byte(DB_PREFIX_MINTHISTORY+strings.ToLower(tickerName)+"-"), false, func(k, v []byte) error {

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
	common.Log.Infof("loadMintDataFromDB %s loaded %d records in %d ms\n", tickerName, count, elapsed)

	return result
}

func (s *FTIndexer) getTickerFromDB(tickerName string) *common.Ticker {
	var result common.Ticker

	key := DB_PREFIX_TICKER + strings.ToLower(tickerName)
	err := db.GetValueFromDB([]byte(key), &result, s.db)
	if err == common.ErrKeyNotFound {
		common.Log.Debugf("GetTickFromDB key: %s, error: ErrKeyNotFound ", key)
		return nil
	} else if err != nil {
		common.Log.Errorf("GetTickFromDB error: %v", err)
		return nil
	}

	return &result
}
