package exotic

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func encodeHolderBalance(amount int64) ([]byte, error) {
	if amount < 0 {
		return nil, fmt.Errorf("negative exotic holder balance %d", amount)
	}
	return common.Uint64ToBytes(uint64(amount)), nil
}

func decodeHolderBalance(value []byte) (int64, error) {
	if len(value) != 8 {
		return 0, fmt.Errorf("invalid exotic holder balance length %d", len(value))
	}
	n := common.BytesToUint64(value)
	if n > uint64(^uint64(0)>>1) {
		return 0, fmt.Errorf("exotic holder balance overflows int64: %d", n)
	}
	return int64(n), nil
}

func (p *ExoticIndexer) loadTickerHolderAmount(ticker string, addressID uint64) int64 {
	key := string(GetTickerHolderKey(ticker, addressID))
	if amount, ok := p.holderBalanceTouched[key]; ok {
		return amount
	}
	value, err := p.db.Read([]byte(key))
	if err == common.ErrKeyNotFound {
		return 0
	}
	if err != nil {
		common.Log.Panicf("load exotic holder balance %s/%d: %v", ticker, addressID, err)
	}
	amount, err := decodeHolderBalance(value)
	if err != nil {
		common.Log.Panicf("decode exotic holder balance %s/%d: %v", ticker, addressID, err)
	}
	return amount
}

func (p *ExoticIndexer) adjustTickerHolderAmount(ticker string, addressID uint64, delta int64) {
	if delta == 0 {
		return
	}
	ticker = strings.ToLower(ticker)
	key := string(GetTickerHolderKey(ticker, addressID))
	amount := p.loadTickerHolderAmount(ticker, addressID) + delta
	if amount < 0 {
		common.Log.Panicf("exotic holder balance %s/%d became negative: %d", ticker, addressID, amount)
	}
	p.holderBalanceTouched[key] = amount
}

func (p *ExoticIndexer) getTickerHolderAmounts(ticker string) map[uint64]int64 {
	ticker = strings.ToLower(ticker)
	prefix := GetTickerHolderPrefix(ticker)
	result := make(map[uint64]int64)
	if err := p.db.Scan(common.ScanOptions{Prefix: prefix}, func(k, v []byte) error {
		addressID, err := ParseTickerHolderKey(ticker, k)
		if err != nil {
			return err
		}
		amount, err := decodeHolderBalance(v)
		if err != nil {
			return err
		}
		if amount > 0 {
			result[addressID] = amount
		}
		return nil
	}); err != nil {
		common.Log.Panicf("scan exotic holders for %s: %v", ticker, err)
	}

	for key, amount := range p.holderBalanceTouched {
		keyBytes := []byte(key)
		if !bytes.HasPrefix(keyBytes, prefix) {
			continue
		}
		addressID, err := ParseTickerHolderKey(ticker, keyBytes)
		if err != nil {
			common.Log.Panicf("parse pending exotic holder key: %v", err)
		}
		if amount == 0 {
			delete(result, addressID)
		} else {
			result[addressID] = amount
		}
	}
	return result
}

func (p *ExoticIndexer) getTickerUtxos(ticker string) map[uint64]int64 {
	ticker = strings.ToLower(ticker)
	result := make(map[uint64]int64)
	prefix := []byte(DB_PREFIX_TICKER_UTXO + ticker + "-")
	if err := p.db.Scan(common.ScanOptions{Prefix: prefix}, func(k, v []byte) error {
		_, utxoID, err := parseTickUtxoKey(string(k))
		if err != nil {
			return err
		}
		var amount int64
		if err := db.DecodeBytes(v, &amount); err != nil {
			return err
		}
		result[utxoID] = amount
		return nil
	}); err != nil {
		common.Log.Panicf("scan exotic ticker UTXOs for %s: %v", ticker, err)
	}
	for utxoID, amount := range p.utxoMap[ticker] {
		result[utxoID] = amount
	}
	for utxoID := range p.utxoDeleted[ticker] {
		delete(result, utxoID)
	}
	return result
}

func (p *ExoticIndexer) holderInfoForRead(utxoID uint64) *HolderInfo {
	if info := p.holderInfo[utxoID]; info != nil {
		return info
	}
	for _, deleted := range p.utxoDeleted {
		if deleted[utxoID] {
			return nil
		}
	}
	info, err := p.loadUtxoInfoFromDB(utxoID)
	if err != nil {
		return nil
	}
	return info
}
