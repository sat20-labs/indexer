package brc20

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

type TransferNftHistory struct {
	UtxoId []uint64
}

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
	s.loadMintDataFromDB(tickerName)
	return tickinfo
}

func (s *BRC20Indexer) loadHoldersInTickerFromDB(name string) map[uint64]*common.Decimal {
	//common.Log.Info("BRC20Indexer loadHolderInfoInTickerFromDB ...")
	//count := 0
	//startTime := time.Now()
	holderMap := make(map[uint64]*common.Decimal, 0)
	err := s.db.BatchRead([]byte(DB_PREFIX_TICKER_HOLDER+encodeTickerName(name)+"-"), false, func(k, v []byte) error {
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
	result := NewHolderInfo()

	common.Log.Debug("BRC20Indexer loadHolderInfoFromDB ...")
	count := 0
	startTime := time.Now()
	prefix := fmt.Sprintf("%s%s-", DB_PREFIX_HOLDER_ASSET, common.Uint64ToString(addressId))
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
				if info.AssetAmt().Sign() != 0 || len(info.TransferableData) != 0 {
					result.Tickers[ticker] = &info
				}
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
	common.Log.Infof("loadHolderInfoFromDB loaded %d records in %d ms", count, elapsed)

	return result
}

func (s *BRC20Indexer) loadHolderInfoFromDBV2(addressId uint64) map[string]*common.Decimal {
	result := make(map[string]*common.Decimal)

	common.Log.Debug("BRC20Indexer loadHolderInfoFromDBV2 ...")
	count := 0
	startTime := time.Now()
	prefix := fmt.Sprintf("%s%s-", DB_PREFIX_HOLDER_ASSET, common.Uint64ToString(addressId))
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
				amt := info.AssetAmt()
				if amt.Sign() != 0 {
					result[ticker] = info.AssetAmt()
				}
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
	common.Log.Infof("loadHolderInfoFromDBV2 loaded %d records in %d ms", count, elapsed)

	return result
}

func (s *BRC20Indexer) checkHolderAssetFromDB(addressId uint64) bool {

	hasAsset := false
	prefix := fmt.Sprintf("%s%s-", DB_PREFIX_HOLDER_ASSET, common.Uint64ToString(addressId))
	s.db.BatchRead([]byte(prefix), false, func(k, v []byte) error {
		// 设置前缀扫描选项
		var info common.BRC20TickAbbrInfo
		err := db.DecodeBytes(v, &info)
		if err == nil {
			if info.AssetAmt().Sign() != 0 {
				hasAsset = true
				return fmt.Errorf("found")
			}
		}
		return nil
	})

	return hasAsset
}

func (s *BRC20Indexer) checkHolderExistingFromDB(addressId uint64) bool {

	existing := false
	prefix := fmt.Sprintf("%s%s-", DB_PREFIX_HOLDER_ASSET, common.Uint64ToString(addressId))
	s.db.BatchRead([]byte(prefix), false, func(k, v []byte) error {
		// 设置前缀扫描选项
		var info common.BRC20TickAbbrInfo
		err := db.DecodeBytes(v, &info)
		if err == nil {
			existing = true
			return fmt.Errorf("found")
		}
		return nil
	})

	return existing
}

func (s *BRC20Indexer) loadTickAbbrInfoFromDB(addressId uint64, ticker string) *common.BRC20TickAbbrInfo {
	var result common.BRC20TickAbbrInfo

	key := GetHolderInfoKey(addressId, ticker)
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

func (s *BRC20Indexer) loadTickListFromDB() []string {
	count := 0
	startTime := time.Now()
	common.Log.Debug("BRC20Indexer loadTickListFromDB ...")

	type pair struct {
		id   int64
		name string
	}

	tickers := make([]*pair, 0)
	err := s.db.BatchRead([]byte(DB_PREFIX_TICKER), false, func(k, v []byte) error {

		//key := string(k)
		//tickname, err := parseTickListKey(key)
		//if err == nil {
		var ticker common.BRC20Ticker
		err := db.DecodeBytes(v, &ticker)
		if err == nil {
			tickers = append(tickers, &pair{
				id:   ticker.Id,
				name: strings.ToLower(ticker.Name),
			})
		} else {
			common.Log.Panicln("DecodeBytes " + err.Error())
		}

		count++

		return nil
	})
	if err != nil {
		common.Log.Panicf("Error prefetching ticklist from db: %v", err)
	}

	sort.Slice(tickers, func(i, j int) bool {
		return tickers[i].id < tickers[j].id
	})

	result := make([]string, len(tickers))
	for i, t := range tickers {
		result[i] = t.name
	}

	elapsed := time.Since(startTime).Milliseconds()
	common.Log.Infof("loadTickListFromDB loaded %d records in %d ms", count, elapsed)

	return result
}


func (s *BRC20Indexer) loadTickListFromDBv2() map[int64]string {
	count := 0
	startTime := time.Now()
	common.Log.Debug("BRC20Indexer loadTickListFromDB ...")


	tickers := make(map[int64]string )
	err := s.db.BatchRead([]byte(DB_PREFIX_TICKER), false, func(k, v []byte) error {

		//key := string(k)
		//tickname, err := parseTickListKey(key)
		//if err == nil {
		var ticker common.BRC20Ticker
		err := db.DecodeBytes(v, &ticker)
		if err == nil {
			tickers[ticker.Id] = strings.ToLower(ticker.Name)
		} else {
			common.Log.Panicln("DecodeBytes " + err.Error())
		}

		count++

		return nil
	})
	if err != nil {
		common.Log.Panicf("Error prefetching ticklist from db: %v", err)
	}

	elapsed := time.Since(startTime).Milliseconds()
	common.Log.Infof("loadTickListFromDB loaded %d records in %d ms", count, elapsed)

	return tickers
}


func (s *BRC20Indexer) loadTickListFromDBv3(start, limit int) []string {
	if start >= s.status.TickerCount {
		return nil
	}
	if limit < 0 {
		limit = s.status.TickerCount
	}
	
	if start + limit > s.status.TickerCount {
		limit = s.status.TickerCount -  start
	}

	prefix := []byte(DB_PREFIX_ID_TO_TICKER)
	seekKey := GetTickerIdKey(int64(start))
	
	tickers := make([]string, 0, limit)
	s.db.BatchReadV2(prefix, []byte(seekKey), false, func(k, v []byte) error {

		//key := string(k)
		//tickname, err := parseTickListKey(key)
		//if err == nil {
		var name string
		err := db.DecodeBytes(v, &name)
		if err == nil {
			tickers = append(tickers, name)
		} else {
			common.Log.Panicln("DecodeBytes " + err.Error())
		}
		if len(tickers) == limit {
			return fmt.Errorf("reach limit")
		}

		return nil
	})

	return tickers
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
	common.Log.Debugf("loadMintDataFromDB %s loaded %d records in %d ms", tickerName, count, elapsed)

	return result
}

// 按时间排序的历史数据
func (s *BRC20Indexer) loadTransferHistoryFromDB(tickerName string) []*common.BRC20ActionHistory {
	result := make([]*common.BRC20ActionHistory, 0)
	count := 0
	startTime := time.Now()
	common.Log.Debug("BRC20Indexer loadTransferHistoryFromDB ...")
	err := s.db.BatchRead([]byte(DB_PREFIX_TRANSFER_HISTORY+encodeTickerName(tickerName)+"-"),
		false, func(k, v []byte) error {

			key := string(k)

			tick, _, _, _ := ParseTransferHistoryKey(key)
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
	common.Log.Debugf("loadTransferHistoryFromDB %s loaded %d records in %d ms", tickerName, count, elapsed)

	return result
}

func (s *BRC20Indexer) loadTransferHistoryWithHeightFromDB(tickerName string, height int) []*common.BRC20ActionHistory {
	result := make([]*common.BRC20ActionHistory, 0)
	count := 0
	startTime := time.Now()
	common.Log.Debug("BRC20Indexer loadTransferHistoryWithHeightFromDB ...")
	prefix := fmt.Sprintf("%s%s-%x-", DB_PREFIX_TRANSFER_HISTORY, encodeTickerName(tickerName), height)
	err := s.db.BatchRead([]byte(prefix), false, func(k, v []byte) error {
		var history common.BRC20ActionHistory
		err := db.DecodeBytes(v, &history)
		if err == nil {
			result = append(result, &history)
		} else {
			common.Log.Errorln("loadTransferHistoryWithHeightFromDB DecodeBytes " + err.Error())
		}
		count++

		return nil
	})

	if err != nil {
		common.Log.Panicf("loadTransferHistoryWithHeightFromDB %s from db: %v", tickerName, err)
	}

	elapsed := time.Since(startTime).Milliseconds()
	common.Log.Debugf("loadTransferHistoryWithHeightFromDB %s loaded %d records in %d ms", tickerName, count, elapsed)

	return result
}

// 无效，原因是height的顺序跟key的顺序不一致
func (s *BRC20Indexer) loadLatestTransferHistoryFromDB(tickerName string, limit int) []*common.BRC20ActionHistory {
	result := make([]*common.BRC20ActionHistory, 0)
	count := 0
	startTime := time.Now()
	common.Log.Debug("BRC20Indexer loadLatestTransferHistoryFromDB ...")
	prefix := fmt.Sprintf("%s%s-", DB_PREFIX_TRANSFER_HISTORY, encodeTickerName(tickerName))
	s.db.BatchRead([]byte(prefix), true, func(k, v []byte) error {
		var history common.BRC20ActionHistory
		err := db.DecodeBytes(v, &history)
		if err == nil {
			result = append(result, &history)
		} else {
			common.Log.Errorln("loadLatestTransferHistoryFromDB DecodeBytes " + err.Error())
		}
		count++
		if count == limit {
			return fmt.Errorf("reach limit")
		}

		return nil
	})

	elapsed := time.Since(startTime).Milliseconds()
	common.Log.Debugf("loadLatestTransferHistoryFromDB %s loaded %d records in %d ms", tickerName, count, elapsed)

	return result
}

func (s *BRC20Indexer) loadTransferHistoryWithHolderFromDB(tickerName string, holder uint64) []*common.BRC20ActionHistory {
	startTime := time.Now()
	common.Log.Debug("BRC20Indexer loadTransferHistoryWithHolderFromDB ...")
	prefix := fmt.Sprintf("%s%s-%x-", DB_PREFIX_TRANSFER_HISTORY_HOLDER, encodeTickerName(tickerName), holder)
	historyKeys := make([]string, 0)
	s.db.BatchRead([]byte(prefix), false, func(k, v []byte) error {
		_, _, nftId, err := ParseHolderTransferHistoryKey(string(k))
		if err != nil {
			return nil
		}

		var history TransferNftHistory
		err = db.DecodeBytes(v, &history)
		if err == nil {
			for _, utxoId := range history.UtxoId {
				key := GetTransferHistoryKey(tickerName, utxoId, nftId)
				historyKeys = append(historyKeys, key)
			}
		} else {
			common.Log.Errorln("loadTransferHistoryWithHolderFromDB DecodeBytes " + err.Error())
		}

		return nil
	})

	sort.Slice(historyKeys, func(i, j int) bool {
		return historyKeys[i] < historyKeys[j]
	})
	
	result := make([]*common.BRC20ActionHistory, 0)
	for _, key := range historyKeys {
		var history common.BRC20ActionHistory
		err := db.GetValueFromDB([]byte(key), &history, s.db)
		if err == common.ErrKeyNotFound {
			common.Log.Debugf("BRC20ActionHistory key: %s, error: ErrKeyNotFound ", key)
			continue
		} else if err != nil {
			common.Log.Errorf("BRC20ActionHistory error: %v", err)
			continue
		}
		result = append(result, &history)
	}

	elapsed := time.Since(startTime).Milliseconds()
	common.Log.Debugf("loadTransferHistoryWithHolderFromDB %s loaded %d records in %d ms", tickerName, len(result), elapsed)

	return result
}

func (s *BRC20Indexer) loadTickerFromDB(tickerName string) *common.BRC20Ticker {
	var result common.BRC20Ticker

	key := GetTickerKey(tickerName)
	err := db.GetValueFromDB([]byte(key), &result, s.db)
	if err == common.ErrKeyNotFound {
		common.Log.Debugf("loadTickerFromDB key: %s, error: ErrKeyNotFound ", key)
		return nil
	} else if err != nil {
		common.Log.Errorf("loadTickerFromDB error: %v", err)
		return nil
	}

	return &result
}

func (s *BRC20Indexer) loadTransferFromDB(utxoId uint64) *TransferNftInfo {
	var result TransferNftInfo

	key := GetUtxoToTransferKey(utxoId)
	err := db.GetValueFromDB([]byte(key), &result, s.db)
	if err == common.ErrKeyNotFound {
		common.Log.Debugf("loadTransferFromDB key: %s, error: ErrKeyNotFound ", key)
		return nil
	} else if err != nil {
		common.Log.Errorf("loadTransferFromDB error: %v", err)
		return nil
	}

	return &result
}
