package brc20

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/common"
)

func GetTickerKey(tickname string) string {
	return fmt.Sprintf("%s%s", DB_PREFIX_TICKER, strings.ToLower(tickname))
}

func GetMintHistoryKey(tickname, inscriptionId string) string {
	return fmt.Sprintf("%s%s-%s", DB_PREFIX_MINTHISTORY, strings.ToLower(tickname), inscriptionId)
}

// func GetTransferHistoryKey(tickname string, utxo string) string {
// 	return fmt.Sprintf("%s%s-%s", DB_PREFIX_TRANSFER_HISTORY, strings.ToLower(tickname), utxo)
// }

func GetTransferHistoryKey(tickname string, utxoId uint64) string {
	return fmt.Sprintf("%s%s-%d", DB_PREFIX_TRANSFER_HISTORY, strings.ToLower(tickname), utxoId)
}

func ParseTransferHistoryKey(input string) (string, string, error) {
	if !strings.HasPrefix(input, DB_PREFIX_TRANSFER_HISTORY) {
		return "", "", fmt.Errorf("invalid string format")
	}
	str := strings.TrimPrefix(input, DB_PREFIX_TRANSFER_HISTORY)
	parts := strings.Split(str, "-")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid string format")
	}

	return parts[0], parts[1], nil
}

func parseTickListKey(input string) (string, error) {
	if !strings.HasPrefix(input, DB_PREFIX_TICKER) {
		return "", fmt.Errorf("invalid string format")
	}
	return strings.TrimPrefix(input, DB_PREFIX_TICKER), nil
}

func ParseMintHistoryKey(input string) (string, string, error) {
	if !strings.HasPrefix(input, DB_PREFIX_MINTHISTORY) {
		return "", "", fmt.Errorf("invalid string format")
	}
	str := strings.TrimPrefix(input, DB_PREFIX_MINTHISTORY)
	parts := strings.Split(str, "-")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid string format")
	}

	return parts[0], parts[1], nil
}

func GetHolderInfoKey(addrId uint64, ticker string) string {
	return fmt.Sprintf("%s%d-%s", DB_PREFIX_TICKER_HOLDER, addrId, ticker)
}

func parseHolderInfoKey(input string) (uint64, string, error) {
	if !strings.HasPrefix(input, DB_PREFIX_TICKER_HOLDER) {
		return common.INVALID_ID, "", fmt.Errorf("invalid string format")
	}
	str := strings.TrimPrefix(input, DB_PREFIX_TICKER_HOLDER)
	parts := strings.Split(str, "-")
	if len(parts) != 2 {
		return common.INVALID_ID, "", fmt.Errorf("invalid string format")
	}

	addrId, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return common.INVALID_ID, "", err
	}

	return addrId, parts[1], nil
}

func newTickerInfo(name string) *BRC20TickInfo {
	return &BRC20TickInfo{
		Name:           name,
		InscriptionMap: make(map[string]*common.BRC20MintAbbrInfo, 0),
		MintAdded:      make([]*common.BRC20Mint, 0),
	}
}
