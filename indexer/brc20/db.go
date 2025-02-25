package brc20

import (
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func (s *BRC20Indexer) initTickInfoFromDB(tickerName string) *BRC20TickInfo {
	tickinfo := newTickerInfo(tickerName)
	s.loadMintInfoFromDB(tickinfo)
	return tickinfo
}

func (s *BRC20Indexer) loadMintInfoFromDB(tickinfo *BRC20TickInfo) {
	mintList := s.loadMintDataFromDB(tickinfo.Name)
	for _, mint := range mintList {
		// for _, rng := range mint.Ordinals {
		// 	tickinfo.MintInfo.AddMintInfo(rng, mint.Base.InscriptionId)
		// }

		tickinfo.InscriptionMap[mint.Nft.Base.InscriptionId] = common.NewBRC20MintAbbrInfo(mint)
	}
}

func (s *BRC20Indexer) loadHolderInfoFromDB() error {
	count := 0
	startTime := time.Now()
	common.Log.Info("BRC20Indexer loadHolderInfoFromDB ...")
	holderMap := make(map[uint64]*HolderInfo, 0)
	tickerToHolderMap := make(map[string]map[uint64]bool)
	transferNftMap := make(map[uint64]*TransferNftInfo)
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
			if item.IsDeletedOrExpired() {
				continue
			}
			key := string(item.Key())

			addrId, ticker, err := parseHolderInfoKey(key)
			if err != nil {
				common.Log.Errorln(key + " " + err.Error())
			} else {
				var info common.BRC20TickAbbrInfo
				value, err := item.ValueCopy(nil)
				if err != nil {
					common.Log.Errorln("ValueCopy " + key + " " + err.Error())
				} else {
					err = db.DecodeBytes(value, &info)
					if err == nil {
						holder, ok := holderMap[addrId]
						if ok {
							holder.Tickers[ticker] = &info
						} else {
							holder = &HolderInfo{
								AddressId: addrId,
								Tickers:   make(map[string]*common.BRC20TickAbbrInfo),
							}
							holder.Tickers[ticker] = &info
							holderMap[addrId] = holder
						}

						holders, ok := tickerToHolderMap[ticker]
						if ok {
							holders[addrId] = true
						} else {
							holders = make(map[uint64]bool)
							holders[addrId] = true
						}
						tickerToHolderMap[ticker] = holders

						for _, nft := range info.TransferableData {
							transferNftMap[nft.UtxoId] = &TransferNftInfo{
								AddressId:   addrId,
								Ticker:      ticker,
								TransferNft: nft,
							}
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

	s.holderMap = holderMap
	s.tickerToHolderMap = tickerToHolderMap
	s.transferNftMap = transferNftMap

	return nil
}

func (s *BRC20Indexer) loadTickListFromDB() []string {
	result := make([]string, 0)
	count := 0
	startTime := time.Now()
	common.Log.Info("BRC20Indexer loadTickListFromDB ...")
	err := s.db.View(func(txn *badger.Txn) error {
		prefixBytes := []byte(DB_PREFIX_TICKER)
		prefixOptions := badger.DefaultIteratorOptions
		prefixOptions.Prefix = prefixBytes
		it := txn.NewIterator(prefixOptions)
		defer it.Close()
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			if item.IsDeletedOrExpired() {
				continue
			}
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

func (s *BRC20Indexer) getTickListFromDB() []string {
	return s.loadTickListFromDB()
}

// key: inscriptionId
func (s *BRC20Indexer) getMintListFromDB(tickname string) map[string]*common.BRC20Mint {
	return s.loadMintDataFromDB(tickname)
}

func (s *BRC20Indexer) getMintFromDB(ticker, inscriptionId string) *common.BRC20Mint {
	var result common.BRC20Mint
	err := s.db.View(func(txn *badger.Txn) error {
		key := GetMintHistoryKey(strings.ToLower(ticker), inscriptionId)
		err := db.GetValueFromDB([]byte(key), txn, &result)
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

func (s *BRC20Indexer) loadMintDataFromDB(tickerName string) map[string]*common.BRC20Mint {
	result := make(map[string]*common.BRC20Mint, 0)
	count := 0
	startTime := time.Now()
	common.Log.Info("BRC20Indexer loadMintDataFromDB ...")
	err := s.db.View(func(txn *badger.Txn) error {
		prefixBytes := []byte(DB_PREFIX_MINTHISTORY + strings.ToLower(tickerName) + "-")
		prefixOptions := badger.DefaultIteratorOptions
		prefixOptions.Prefix = prefixBytes
		it := txn.NewIterator(prefixOptions)
		defer it.Close()
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			if item.IsDeletedOrExpired() {
				continue
			}
			key := string(item.Key())

			tick, inscriptionId, _ := ParseMintHistoryKey(key)
			if tick == tickerName {
				var mint common.BRC20Mint
				value, err := item.ValueCopy(nil)
				if err != nil {
					common.Log.Errorln("loadMintDataFromDB ValueCopy " + key + " " + err.Error())
				} else {
					err = db.DecodeBytes(value, &mint)
					if err == nil {
						result[inscriptionId] = &mint
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

func (s *BRC20Indexer) loadTransferHistoryFromDB(tickerName string) []*common.BRC20TransferHistory {
	result := make([]*common.BRC20TransferHistory, 0)
	count := 0
	startTime := time.Now()
	common.Log.Info("BRC20Indexer loadTransferHistoryFromDB ...")
	err := s.db.View(func(txn *badger.Txn) error {
		prefixBytes := []byte(DB_PREFIX_TRANSFER_HISTORY + strings.ToLower(tickerName) + "-")
		prefixOptions := badger.DefaultIteratorOptions
		prefixOptions.Prefix = prefixBytes
		it := txn.NewIterator(prefixOptions)
		defer it.Close()
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			if item.IsDeletedOrExpired() {
				continue
			}
			key := string(item.Key())

			tick, _, _ := ParseTransferHistoryKey(key)
			if tick == tickerName {
				var history common.BRC20TransferHistory
				value, err := item.ValueCopy(nil)
				if err != nil {
					common.Log.Errorln("loadTransferHistoryFromDB ValueCopy " + key + " " + err.Error())
				} else {
					err = db.DecodeBytes(value, &history)
					if err == nil {
						result = append(result, &history)
					} else {
						common.Log.Errorln("loadTransferHistoryFromDB DecodeBytes " + err.Error())
					}
				}
			}
			count++
		}

		return nil
	})

	if err != nil {
		common.Log.Panicf("loadTransferHistoryFromDB %s from db: %v", tickerName, err)
	}

	elapsed := time.Since(startTime).Milliseconds()
	common.Log.Infof("loadTransferHistoryFromDB %s loaded %d records in %d ms\n", tickerName, count, elapsed)

	return result
}

func (s *BRC20Indexer) getTickerFromDB(tickerName string) *common.BRC20Ticker {
	var result common.BRC20Ticker
	err := s.db.View(func(txn *badger.Txn) error {
		key := DB_PREFIX_TICKER + strings.ToLower(tickerName)
		err := db.GetValueFromDB([]byte(key), txn, &result)
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
