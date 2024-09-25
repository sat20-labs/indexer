package ft

import (
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/ordx/common"
)

func (s *FTIndexer) initTickInfoFromDB(tickerName string) *TickInfo {
	tickinfo := newTickerInfo(tickerName)
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
	err := s.db.View(func(txn *badger.Txn) error {
		// 设置前缀扫描选项
		prefixBytes := []byte(DB_PREFIX_TICKER_HOLDER)
		prefixOptions := badger.DefaultIteratorOptions
		prefixOptions.Prefix = prefixBytes

		// 使用前缀扫描选项创建迭代器
		it := txn.NewIterator(prefixOptions)
		defer it.Close()

		// 遍历匹配前缀的key
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			key := string(item.Key())

			utxo, err := parseHolderInfoKey(key)
			if err != nil {
				common.Log.Errorln(key + " " + err.Error())
			} else {
				var info HolderInfo
				value, err := item.ValueCopy(nil)
				if err != nil {
					common.Log.Errorln("ValueCopy " + key + " " + err.Error())
				} else {
					err = common.DecodeBytes(value, &info)
					if err == nil {
						result[utxo] = &info
					} else {
						common.Log.Errorln("DecodeBytes " + err.Error())
					}
				}
			}
			count++
		}
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
	err := s.db.View(func(txn *badger.Txn) error {
		// 设置前缀扫描选项
		prefixBytes := []byte(DB_PREFIX_TICKER_UTXO)
		prefixOptions := badger.DefaultIteratorOptions
		prefixOptions.Prefix = prefixBytes

		// 使用前缀扫描选项创建迭代器
		it := txn.NewIterator(prefixOptions)
		defer it.Close()

		// 遍历匹配前缀的key
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			key := string(item.Key())

			ticker, utxo, err := parseTickUtxoKey(key)
			if err != nil {
				common.Log.Errorln(key + " " + err.Error())
			} else {
				var amount int64
				value, err := item.ValueCopy(nil)
				if err != nil {
					common.Log.Errorln("ValueCopy " + key + " " + err.Error())
				} else {
					err = common.DecodeBytes(value, &amount)
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
			}
			count++
		}
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
	err := s.db.View(func(txn *badger.Txn) error {
		prefixBytes := []byte(DB_PREFIX_TICKER)
		prefixOptions := badger.DefaultIteratorOptions
		prefixOptions.Prefix = prefixBytes
		it := txn.NewIterator(prefixOptions)
		defer it.Close()
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			key := string(item.Key())
			tickname, err := parseTickListKey(key)
			if err == nil {
				result = append(result, tickname)
			}
			count++
		}

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
	err := s.db.View(func(txn *badger.Txn) error {
		key := GetMintHistoryKey(strings.ToLower(ticker), inscriptionId)
		err := common.GetValueFromDB([]byte(key), txn, &result)
		if err == badger.ErrKeyNotFound {
			common.Log.Debugf("GetMintFromDB key: %s, error: ErrKeyNotFound ", key)
			return err
		} else if err != nil {
			common.Log.Debugf("GetMintFromDB error: %v", err)
			return err
		}
		return nil
	})
	if err != nil {
		common.Log.Debugf("GetMintFromDB error: %v", err)
		return nil
	}

	return &result
}

func (s *FTIndexer) loadMintDataFromDB(tickerName string) map[string]*common.Mint {
	result := make(map[string]*common.Mint, 0)
	count := 0
	startTime := time.Now()
	common.Log.Info("loadMintDataFromDB ...")
	err := s.db.View(func(txn *badger.Txn) error {
		prefixBytes := []byte(DB_PREFIX_MINTHISTORY + strings.ToLower(tickerName) + "-")
		prefixOptions := badger.DefaultIteratorOptions
		prefixOptions.Prefix = prefixBytes
		it := txn.NewIterator(prefixOptions)
		defer it.Close()
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			key := string(item.Key())

			tick, utxo, _ := ParseMintHistoryKey(key)
			if tick == tickerName {
				var mint common.Mint
				value, err := item.ValueCopy(nil)
				if err != nil {
					common.Log.Errorln("loadMintDataFromDB ValueCopy " + key + " " + err.Error())
				} else {
					err = common.DecodeBytes(value, &mint)
					if err == nil {
						result[utxo] = &mint
					} else {
						common.Log.Errorln("loadMintDataFromDB DecodeBytes " + err.Error())
					}
				}
			}
			count++
		}

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
	err := s.db.View(func(txn *badger.Txn) error {
		key := DB_PREFIX_TICKER + strings.ToLower(tickerName)
		err := common.GetValueFromDB([]byte(key), txn, &result)
		if err == badger.ErrKeyNotFound {
			common.Log.Debugf("GetTickFromDB key: %s, error: ErrKeyNotFound ", key)
			return err
		} else if err != nil {
			common.Log.Debugf("GetTickFromDB error: %v", err)
			return err
		}
		return nil
	})
	if err != nil {
		common.Log.Debugf("GetTickFromDB error: %v", err)
		return nil
	}
	return &result
}
