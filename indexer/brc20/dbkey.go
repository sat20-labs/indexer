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

func GetImageKey(ticker, utxo string) string {
	return DB_PREFIX_IMAGE + strings.ToLower(ticker) + "-" + utxo
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

func GetHolderInfoKey(addrId uint64) string {
	return fmt.Sprintf("%s%d", DB_PREFIX_TICKER_HOLDER, addrId)
}

func parseHolderInfoKey(input string) (uint64, error) {
	if !strings.HasPrefix(input, DB_PREFIX_TICKER_HOLDER) {
		return common.INVALID_ID, fmt.Errorf("invalid string format")
	}
	str := strings.TrimPrefix(input, DB_PREFIX_TICKER_HOLDER)
	parts := strings.Split(str, "-")
	if len(parts) != 1 {
		return common.INVALID_ID, fmt.Errorf("invalid string format")
	}

	addrId, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return common.INVALID_ID, err
	}

	return addrId, nil
}

func newTickerInfo(name string) *BRC20TickInfo {
	return &BRC20TickInfo{
		Name:           name,
		InscriptionMap: make(map[string]*common.BRC20MintAbbrInfo, 0),
		MintAdded:      make([]*common.BRC20Mint, 0),
	}
}

