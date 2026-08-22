package exotic

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/sat20-labs/indexer/common"
)

func GetTickerKey(tickname string) string {
	return fmt.Sprintf("%s%s", DB_PREFIX_TICKER, tickname)
}

func GetMintHistoryKey(tickname, inscriptionId string) string {
	return fmt.Sprintf("%s%s-%s", DB_PREFIX_MINTHISTORY, tickname, inscriptionId)
}

func GetHolderInfoKey(utxo uint64) string {
	return fmt.Sprintf("%s%d", DB_PREFIX_TICKER_HOLDER, utxo)
}

func GetTickerUtxoKey(ticker string, utxo uint64) string {
	return fmt.Sprintf("%s%s-%d", DB_PREFIX_TICKER_UTXO, strings.ToLower(ticker), utxo)
}

func GetTickerHolderPrefix(ticker string) []byte {
	prefix := []byte(DB_PREFIX_HOLDER_BALANCE + strings.ToLower(ticker))
	return append(prefix, 0)
}

func GetTickerHolderKey(ticker string, addressID uint64) []byte {
	key := GetTickerHolderPrefix(ticker)
	return append(key, common.Uint64ToBytes(addressID)...)
}

func ParseTickerHolderKey(ticker string, key []byte) (uint64, error) {
	prefix := GetTickerHolderPrefix(ticker)
	if !bytes.HasPrefix(key, prefix) || len(key) != len(prefix)+8 {
		return common.INVALID_ID, fmt.Errorf("invalid exotic holder key %x", key)
	}
	return common.BytesToUint64(key[len(prefix):]), nil
}

func GetImageKey(ticker, utxo string) string {
	return DB_PREFIX_IMAGE + ticker + "-" + utxo
}
