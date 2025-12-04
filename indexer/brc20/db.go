package brc20

import (
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func (s *BRC20Indexer) initTickInfoFromDB(tickerName string) *BRC20TickInfo {
	tickinfo := newTickerInfo(tickerName)
	ticker := s.getTickerFromDB(tickerName)
	tickinfo.Ticker = ticker
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
	err := s.db.BatchRead([]byte(DB_PREFIX_TICKER_HOLDER), false, func(k, v []byte) error {
		// 设置前缀扫描选项

		key := string(k)
		addrId, err := parseHolderInfoKey(key)
		if err != nil {
			common.Log.Panicln(key + " " + err.Error())
		} else {
			var holdInfo HolderInfo
			err = db.DecodeBytes(v, &holdInfo)
			if err == nil {
				// if addrId != info.AddressId {
				// 	common.Log.Panicln("key addrId and value addrId not equal")
				// }
				holderMap[addrId] = &holdInfo
				for ticker, tickAbbrInfo := range holdInfo.Tickers {
					holders, ok := tickerToHolderMap[ticker]
					if ok {
						holders[addrId] = true
					} else {
						holders = make(map[uint64]bool)
						holders[addrId] = true
					}
					tickerToHolderMap[ticker] = holders

					for utxoId, transferNft := range tickAbbrInfo.TransferableData {
						transferNftMap[utxoId] = &TransferNftInfo{
							AddressId:   addrId,
							Ticker:      ticker,
							UtxoId:      utxoId,
							TransferNft: transferNft,
						}
					}
				}
			} else {
				common.Log.Panicln("DecodeBytes " + err.Error())
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

func (s *BRC20Indexer) getTickListFromDB() []string {
	return s.loadTickListFromDB()
}

// key: inscriptionId
func (s *BRC20Indexer) getMintListFromDB(tickname string) map[string]*common.BRC20Mint {
	return s.loadMintDataFromDB(tickname)
}

func (s *BRC20Indexer) getMintFromDB(ticker, inscriptionId string) *common.BRC20Mint {
	var result common.BRC20Mint

	key := GetMintHistoryKey(strings.ToLower(ticker), inscriptionId)
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

func (s *BRC20Indexer) loadMintDataFromDB(tickerName string) map[string]*common.BRC20Mint {
	result := make(map[string]*common.BRC20Mint, 0)
	count := 0
	startTime := time.Now()
	common.Log.Debug("BRC20Indexer loadMintDataFromDB ...")
	err := s.db.BatchRead([]byte(DB_PREFIX_MINTHISTORY+strings.ToLower(tickerName)+"-"),
		false, func(k, v []byte) error {

			key := string(k)

			tick, inscriptionId, _ := ParseMintHistoryKey(key)
			if tick == tickerName {
				var mint common.BRC20Mint

				err := db.DecodeBytes(v, &mint)
				if err == nil {
					result[inscriptionId] = &mint
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

func (s *BRC20Indexer) loadTransferHistoryFromDB(tickerName string) []*common.BRC20TransferHistory {
	result := make([]*common.BRC20TransferHistory, 0)
	count := 0
	startTime := time.Now()
	common.Log.Debug("BRC20Indexer loadTransferHistoryFromDB ...")
	err := s.db.BatchRead([]byte(DB_PREFIX_TRANSFER_HISTORY+strings.ToLower(tickerName)+"-"),
		false, func(k, v []byte) error {

			key := string(k)

			tick, _, _ := ParseTransferHistoryKey(key)
			if tick == tickerName {
				var history common.BRC20TransferHistory

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

func (s *BRC20Indexer) getTickerFromDB(tickerName string) *common.BRC20Ticker {
	var result common.BRC20Ticker

	key := DB_PREFIX_TICKER + strings.ToLower(tickerName)
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
