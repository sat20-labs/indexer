package exotic

import (
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

const (
	STATUS_KEY = "s-"

	DB_VERSION = "1.0.0"
)

type Status struct {
	Version          string
	Count            int64
}

func (p *Status) Clone() *Status {
	a := *p
	return &a
}

func initStatusFromDB(ldb common.KVDB) *Status {
	stats := &Status{}
	err := db.GetValueFromDB([]byte(STATUS_KEY), stats, ldb)
	if err == common.ErrKeyNotFound {
		common.Log.Info("initStatusFromDB no stats found in db")
		stats.Version = DB_VERSION
	} else if err != nil {
		common.Log.Panicf("initStatusFromDB failed. %v", err)
	}
	common.Log.Infof("nft stats: %v", stats)

	if stats.Version != DB_VERSION {
		common.Log.Panicf("nft data version inconsistent %s", DB_VERSION)
	}

	return stats
}

func (p *ExoticIndexer) initTickInfoFromDB(tickerName string) *TickInfo {
	tickinfo := newTickerInfo(tickerName)
	tickinfo.Ticker = p.loadTickerFromDB(tickerName)
	p.loadMintInfoFromDB(tickinfo)
	return tickinfo
}

func (p *ExoticIndexer) loadMintInfoFromDB(tickinfo *TickInfo) {
	mintList := p.loadAllMintDataFromDB(tickinfo.Name)
	for _, mint := range mintList {
		tickinfo.InscriptionMap[mint.Base.InscriptionId] = common.NewMintAbbrInfo(mint)
	}
}

func (p *ExoticIndexer) loadAllUtxoInfoFromDB() map[uint64]*HolderInfo {
	count := 0
	startTime := time.Now()
	common.Log.Debug("ExoticIndexer loadHolderInfoFromDB ...")
	result := make(map[uint64]*HolderInfo, 0)
	err := p.db.BatchRead([]byte(DB_PREFIX_TICKER_HOLDER), false, func(k, v []byte) error {

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

func (p *ExoticIndexer) loadUtxoInfoFromDB(utxoId uint64) (*HolderInfo, error) {
	var result HolderInfo
	key := GetHolderInfoKey(utxoId)
	err := db.GetValueFromDB([]byte(key), &result, p.db)
	if err == common.ErrKeyNotFound {
		//common.Log.Debugf("loadUtxoInfoFromDB key: %s, error: ErrKeyNotFound ", key)
		return nil, err
	} else if err != nil {
		common.Log.Errorf("loadUtxoInfoFromDB error: %v", err)
		return nil, err
	}
	return &result, nil
}

func (p *ExoticIndexer) loadAllTickerToUtxoMapFromDB() map[string]map[uint64]int64 {
	count := 0
	startTime := time.Now()
	common.Log.Debug("loadAllTickerToUtxoMapFromDB ...")
	result := make(map[string]map[uint64]int64, 0)
	err := p.db.BatchRead([]byte(DB_PREFIX_TICKER_UTXO), false, func(k, v []byte) error {

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
	common.Log.Infof("loadAllTickerToUtxoMapFromDB loaded %d records in %d ms", count, elapsed)

	return result
}

// 某个ticker的全量utxo （TODO 加一个address作为前缀，只加载该地址相关的utxo）
func (p *ExoticIndexer) loadTickerToUtxoMapFromDB(tickerName string) map[uint64]int64 {
	count := 0
	startTime := time.Now()
	common.Log.Debug("loadTickerToUtxoMapFromDB ...")
	result := make(map[uint64]int64, 0)
	err := p.db.BatchRead([]byte(DB_PREFIX_TICKER_UTXO+tickerName+"-"), 
		false, func(k, v []byte) error {

		key := string(k)
		_, utxoId, err := parseTickUtxoKey(key)
		if err != nil {
			common.Log.Errorln(key + " " + err.Error())
		} else {
			var amount int64

			err = db.DecodeBytes(v, &amount)
			if err == nil {
				result[utxoId] = amount
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
	common.Log.Infof("loadTickerToUtxoMapFromDB loaded %d records in %d ms", count, elapsed)

	return result
}

func (p *ExoticIndexer) loadTickListFromDB() []string {
	result := make([]string, 0)
	count := 0
	startTime := time.Now()
	common.Log.Debug("loadTickListFromDB ...")
	err := p.db.BatchRead([]byte(DB_PREFIX_TICKER), false, func(k, v []byte) error {

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

func (p *ExoticIndexer) loadMintDataFromDB(ticker, inscriptionId string) *common.Mint {
	var result common.Mint
	key := GetMintHistoryKey(ticker, inscriptionId)
	err := db.GetValueFromDB([]byte(key), &result, p.db)
	if err == common.ErrKeyNotFound {
		common.Log.Debugf("loadMintDataFromDB key: %s, error: ErrKeyNotFound ", key)
		return nil
	} else if err != nil {
		common.Log.Errorf("loadMintDataFromDB error: %v", err)
		return nil
	}

	return &result
}

func (p *ExoticIndexer) loadAllMintDataFromDB(tickerName string) map[string]*common.Mint {
	result := make(map[string]*common.Mint, 0)
	count := 0
	startTime := time.Now()
	common.Log.Debug("loadAllMintDataFromDB ...")
	err := p.db.BatchRead([]byte(DB_PREFIX_MINTHISTORY+tickerName+"-"), false, func(k, v []byte) error {

		key := string(k)

		tick, utxo, _ := ParseMintHistoryKey(key)
		if tick == tickerName {
			var mint common.Mint

			err := db.DecodeBytes(v, &mint)
			if err == nil {
				result[utxo] = &mint
			} else {
				common.Log.Errorln("loadAllMintDataFromDB DecodeBytes " + err.Error())
			}

		}
		count++

		return nil
	})

	if err != nil {
		common.Log.Panicf("Error prefetching MintHistory %s from db: %v", tickerName, err)
	}

	elapsed := time.Since(startTime).Milliseconds()
	common.Log.Infof("loadAllMintDataFromDB %s loaded %d records in %d ms", tickerName, count, elapsed)

	return result
}

func (p *ExoticIndexer) loadTickerFromDB(tickerName string) *common.Ticker {
	var result common.Ticker

	key := DB_PREFIX_TICKER + tickerName
	err := db.GetValueFromDB([]byte(key), &result, p.db)
	if err == common.ErrKeyNotFound {
		common.Log.Debugf("GetTickFromDB key: %s, error: ErrKeyNotFound ", key)
		return nil
	} else if err != nil {
		common.Log.Errorf("GetTickFromDB error: %v", err)
		return nil
	}

	return &result
}
