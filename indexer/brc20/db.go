package brc20

import (
	"fmt"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)


func initStatusFromDB(ldb common.KVDB) *common.BRC20Status {
	stats := &common.BRC20Status{}
	err := db.GetValueFromDB([]byte(BRC20_DB_STATUS_KEY), stats, ldb)
	if err == common.ErrKeyNotFound {
		common.Log.Info("initStatusFromDB no stats found in db")
		stats.Version = BRC20_DB_VERSION
	} else if err != nil {
		common.Log.Panicf("initStatusFromDB failed. %v", err)
	}
	common.Log.Infof("nft stats: %v", stats)

	if stats.Version != BRC20_DB_VERSION {
		common.Log.Panicf("nft data version inconsistent %s", BRC20_DB_VERSION)
	}

	return stats
}


func (s *BRC20Indexer) initTickInfoFromDB(tickerName string) *BRC20TickInfo {
	tickinfo := newTickerInfo(tickerName)
	ticker := s.loadTickerFromDB(tickerName)
	tickinfo.Ticker = ticker
	s.loadMintInfoFromDB(tickinfo)
	return tickinfo
}

func (s *BRC20Indexer) loadMintInfoFromDB(tickinfo *BRC20TickInfo) map[int64]*common.BRC20Mint {
	return s.loadMintDataFromDB(tickinfo.Name)
}

func (s *BRC20Indexer) loadHoldersInTickerFromDB(name string) map[uint64]*common.Decimal {
	//common.Log.Info("BRC20Indexer loadHolderInfoInTickerFromDB ...")
	//count := 0
	//startTime := time.Now()
	holderMap := make(map[uint64]*common.Decimal, 0)
	err := s.db.BatchRead([]byte(DB_PREFIX_TICKER_HOLDER+encodeTickerName(name)), false, func(k, v []byte) error {
		// 设置前缀扫描选项

		key := string(k)
		_, addrId, err := parseTickerToHolderKey(key)
		if err != nil {
			common.Log.Panicln(key + " " + err.Error())
		} else {
			var amt common.Decimal
			err = db.DecodeBytes(v, &amt)
			if err == nil {
				holderMap[addrId] = &amt
			} else {
				common.Log.Panicln("DecodeBytes " + err.Error())
			}
		}

		return nil
	})
	if err != nil {
		common.Log.Panicf("Error prefetching HolderInfo from db: %v", err)
	}

	//elapsed := time.Since(startTime).Milliseconds()
	//common.Log.Infof("loadHolderInfoInTickerFromDB loaded %d records in %d ms\n", count, elapsed)

	return holderMap
}

// 加载holder下的所有资产信息
func (s *BRC20Indexer) loadHolderInfoFromDB(addressId uint64) *HolderInfo {
	var result HolderInfo
	result.Tickers = make(map[string]*common.BRC20TickAbbrInfo)

	common.Log.Info("BRC20Indexer loadHolderInfoFromDB ...")
	count := 0
	startTime := time.Now()
	prefix := fmt.Sprintf("%s%x-", DB_PREFIX_HOLDER_ASSET, addressId)
	err := s.db.BatchRead([]byte(prefix), false, func(k, v []byte) error {
		// 设置前缀扫描选项

		key := string(k)
		_, ticker, err := parseHolderInfoKey(key)
		if err != nil {
			common.Log.Panicln(key + " " + err.Error())
		} else {
			var info common.BRC20TickAbbrInfo
			err = db.DecodeBytes(v, &info)
			if err == nil {
				result.Tickers[ticker] = &info
			} else {
				common.Log.Panicln("DecodeBytes " + err.Error())
			}
		}

		return nil
	})
	if err != nil {
		common.Log.Panicf("Error prefetching HolderInfo from db: %v", err)
	}

	elapsed := time.Since(startTime).Milliseconds()
	common.Log.Infof("loadHolderInfoFromDB loaded %d records in %d ms\n", count, elapsed)

	return &result
}


func (s *BRC20Indexer) loadHolderInfoFromDBV2(addressId uint64) map[string]*common.Decimal  {
	result := make(map[string]*common.Decimal)

	common.Log.Info("BRC20Indexer loadHolderInfoFromDBV2 ...")
	count := 0
	startTime := time.Now()
	prefix := fmt.Sprintf("%s%x-", DB_PREFIX_HOLDER_ASSET, addressId)
	err := s.db.BatchRead([]byte(prefix), false, func(k, v []byte) error {
		// 设置前缀扫描选项

		key := string(k)
		_, ticker, err := parseHolderInfoKey(key)
		if err != nil {
			common.Log.Panicln(key + " " + err.Error())
		} else {
			var info common.BRC20TickAbbrInfo
			err = db.DecodeBytes(v, &info)
			if err == nil {
				result[ticker] = info.AssetAmt()
			} else {
				common.Log.Panicln("DecodeBytes " + err.Error())
			}
		}

		return nil
	})
	if err != nil {
		common.Log.Panicf("Error prefetching HolderInfo from db: %v", err)
	}

	elapsed := time.Since(startTime).Milliseconds()
	common.Log.Infof("loadHolderInfoFromDBV2 loaded %d records in %d ms\n", count, elapsed)

	return result
}

func (s *BRC20Indexer) loadTickAbbrInfoFromDB(addressId uint64, ticker string) *common.BRC20TickAbbrInfo {
	var result common.BRC20TickAbbrInfo

	key := GetHolderInfoKey(addressId, ticker)
	err := db.GetValueFromDB([]byte(key), &result, s.db)
	if err == common.ErrKeyNotFound {
		common.Log.Debugf("GetMintFromDB key: %s, error: ErrKeyNotFound ", key)
		return nil
	} else if err != nil {
		common.Log.Debugf("GetMintFromDB error: %v", err)
		return nil
	}

	return &result
}

func (s *BRC20Indexer) loadTickListFromDB() []string {
	result := make([]string, 0)
	count := 0
	startTime := time.Now()
	common.Log.Info("BRC20Indexer loadTickListFromDB ...")
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

func (s *BRC20Indexer) loadMintFromDB(ticker string, id int64) *common.BRC20Mint {
	var result common.BRC20Mint

	key := GetMintHistoryKey(ticker, id)
	err := db.GetValueFromDB([]byte(key), &result, s.db)
	if err == common.ErrKeyNotFound {
		common.Log.Debugf("GetMintFromDB key: %s, error: ErrKeyNotFound ", key)
		return nil
	} else if err != nil {
		common.Log.Debugf("GetMintFromDB error: %v", err)
		return nil
	}
	// return nil

	return &result
}

func (s *BRC20Indexer) loadMintDataFromDB(tickerName string) map[int64]*common.BRC20Mint {
	result := make(map[int64]*common.BRC20Mint, 0)
	count := 0
	startTime := time.Now()
	common.Log.Debug("BRC20Indexer loadMintDataFromDB ...")
	err := s.db.BatchRead([]byte(DB_PREFIX_MINTHISTORY+encodeTickerName(tickerName)+"-"),
		false, func(k, v []byte) error {

			key := string(k)

			tick, nftId, _ := ParseMintHistoryKey(key)
			if tick == tickerName {
				var mint common.BRC20MintInDB
				err := db.DecodeBytes(v, &mint)
				if err == nil {
					nft := s.nftIndexer.GetNftWithId(mint.NftId)
					result[nftId] = &common.BRC20Mint{
						BRC20MintInDB: mint,
						Nft:           nft,
					}
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
	common.Log.Debugf("loadMintDataFromDB %s loaded %d records in %d ms\n", tickerName, count, elapsed)

	return result
}

func (s *BRC20Indexer) loadTransferHistoryFromDB(tickerName string) []*common.BRC20ActionHistory {
	result := make([]*common.BRC20ActionHistory, 0)
	count := 0
	startTime := time.Now()
	common.Log.Debug("BRC20Indexer loadTransferHistoryFromDB ...")
	err := s.db.BatchRead([]byte(DB_PREFIX_TRANSFER_HISTORY+encodeTickerName(tickerName)+"-"),
		false, func(k, v []byte) error {

			key := string(k)

			tick, _, _ := ParseTransferHistoryKey(key)
			if tick == tickerName {
				var history common.BRC20ActionHistory

				err := db.DecodeBytes(v, &history)
				if err == nil {
					result = append(result, &history)
				} else {
					common.Log.Errorln("loadTransferHistoryFromDB DecodeBytes " + err.Error())
				}

			}
			count++

			return nil
		})

	if err != nil {
		common.Log.Panicf("loadTransferHistoryFromDB %s from db: %v", tickerName, err)
	}

	elapsed := time.Since(startTime).Milliseconds()
	common.Log.Debugf("loadTransferHistoryFromDB %s loaded %d records in %d ms\n", tickerName, count, elapsed)

	return result
}

func (s *BRC20Indexer) loadTickerFromDB(tickerName string) *common.BRC20Ticker {
	var result common.BRC20Ticker

	key := DB_PREFIX_TICKER + encodeTickerName(tickerName)
	err := db.GetValueFromDB([]byte(key), &result, s.db)
	if err == common.ErrKeyNotFound {
		common.Log.Debugf("GetTickFromDB key: %s, error: ErrKeyNotFound ", key)
		return nil
	} else if err != nil {
		common.Log.Debugf("GetTickFromDB error: %v", err)
		return nil
	}

	return &result
}

func (s *BRC20Indexer) loadTransferFromDB(utxoId uint64) *TransferNftInfo {
	var result TransferNftInfo

	key := GetUtxoToTransferKey(utxoId)
	err := db.GetValueFromDB([]byte(key), &result, s.db)
	if err == common.ErrKeyNotFound {
		common.Log.Debugf("GetTickFromDB key: %s, error: ErrKeyNotFound ", key)
		return nil
	} else if err != nil {
		common.Log.Debugf("GetTickFromDB error: %v", err)
		return nil
	}

	return &result
}
