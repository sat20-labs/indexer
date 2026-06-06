package atom

import (
	"encoding/base64"
	"fmt"

	"github.com/sat20-labs/indexer/common"
)

func encodeTickerName(ticker string) string {
	return base64.StdEncoding.EncodeToString([]byte(ticker))
}

func GetTickerKey(ticker string) string {
	return DB_PREFIX_TICKER + encodeTickerName(ticker)
}

func GetTickerIdKey(id int64) string {
	return DB_PREFIX_ID_TO_TICKER + common.Uint64ToString(uint64(id))
}

func GetUtxoBalanceKey(utxoId uint64, atomicalId string) string {
	return fmt.Sprintf("%s%d-%s", DB_PREFIX_UTXO_BALANCE, utxoId, atomicalId)
}

func GetTickerUtxoKey(ticker string, utxoId uint64, atomicalId string) string {
	return fmt.Sprintf("%s%s-%d-%s", DB_PREFIX_TICKER_UTXO, encodeTickerName(ticker), utxoId, atomicalId)
}

func GetHolderAssetKey(addressId uint64, ticker string) string {
	return fmt.Sprintf("%s%d-%s", DB_PREFIX_HOLDER_ASSET, addressId, encodeTickerName(ticker))
}

func GetTickerHolderKey(ticker string, addressId uint64) string {
	return fmt.Sprintf("%s%s-%d", DB_PREFIX_TICKER_HOLDER, encodeTickerName(ticker), addressId)
}

func GetMintHistoryKey(ticker string, id int64) string {
	return fmt.Sprintf("%s%s-%s", DB_PREFIX_MINTHISTORY, encodeTickerName(ticker), common.Uint64ToString(uint64(id)))
}

func GetActionKey(height, txIndex int, id int64) string {
	return fmt.Sprintf("%s%08x-%08x-%s", DB_PREFIX_ACTION, height, txIndex, common.Uint64ToString(uint64(id)))
}
