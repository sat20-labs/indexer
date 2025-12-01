package exotic

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/common"
)

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

func parseHolderInfoKey(input string) (uint64, error) {
	if !strings.HasPrefix(input, DB_PREFIX_TICKER_HOLDER) {
		return common.INVALID_ID, fmt.Errorf("invalid string format")
	}
	str := strings.TrimPrefix(input, DB_PREFIX_TICKER_HOLDER)
	parts := strings.Split(str, "-")
	if len(parts) != 1 {
		return common.INVALID_ID, errors.New("invalid string format")
	}

	return strconv.ParseUint(parts[0], 10, 64)
}

func parseTickUtxoKey(input string) (string, uint64, error) {
	if !strings.HasPrefix(input, DB_PREFIX_TICKER_UTXO) {
		return "", common.INVALID_ID, fmt.Errorf("invalid string format")
	}
	str := strings.TrimPrefix(input, DB_PREFIX_TICKER_UTXO)
	parts := strings.Split(str, "-")
	if len(parts) != 2 {
		return "", common.INVALID_ID, errors.New("invalid string format")
	}

	utxoId, err := strconv.ParseUint(parts[1], 10, 64)

	return parts[0], utxoId, err
}

func newTickerInfo(name string) *TickInfo {
	return &TickInfo{
		Name:           strings.ToLower(name),
		UtxoMap:        make(map[uint64]common.AssetOffsets),
		InscriptionMap: make(map[string]*common.MintAbbrInfo, 0),
		MintAdded:      make([]*common.Mint, 0),
	}
}
