package runes

import (
	"fmt"
	"strings"
)

func GetTickerKey(tickname string) string {
	return fmt.Sprintf("%s%s", DB_PREFIX_RUNE, strings.ToLower(tickname))
}

func GetMintHistoryKey(tickname, inscriptionId string) string {
	return fmt.Sprintf("%s%s-%s", DB_PREFIX_MINT_HISTORY, strings.ToLower(tickname), inscriptionId)
}

func GetHolderInfoKey(utxo uint64) string {
	return fmt.Sprintf("%s%d", DB_PREFIX_RUNE_HOLDER, utxo)
}
