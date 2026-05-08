package ft

import (
	"fmt"
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
	return fmt.Sprintf("%s%s-%d", DB_PREFIX_TICKER_UTXO, ticker, utxo)
}

func GetUnbindHistoryKey(ticker string, addressId, utxoId uint64) string {
	return fmt.Sprintf("%s%s-%d-%d", DB_PREFIX_UNBIND_TICKER, ticker, addressId, utxoId)
}

func GetFreezeHistoryKey(ticker string, addressId uint64, action, txid string) string {
	return fmt.Sprintf("%s%s-%d-%s-%s", DB_PREFIX_FREEZE_HISTORY, ticker, addressId, action, txid)
}

func GetFreezeStateKey(ticker string, addressId uint64) string {
	return fmt.Sprintf("%s%s-%d", DB_PREFIX_FREEZE_STATE, ticker, addressId)
}

func GetImageKey(ticker, utxo string) string {
	return DB_PREFIX_IMAGE + ticker + "-" + utxo
}
