package runes

import (
	"strings"

	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
)

func (s *Indexer) getMintFromDB(runeName, inscriptionId string) *common.Mint {
	var result common.Mint
	err := s.db.View(func(txn *badger.Txn) error {
		key := GetMintHistoryKey(strings.ToLower(runeName), inscriptionId)
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

func (s *Indexer) getTickerFromDB(runeName string) *common.Ticker {
	var result common.Ticker
	err := s.db.View(func(txn *badger.Txn) error {
		key := DB_PREFIX_RUNE + strings.ToLower(runeName)
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
