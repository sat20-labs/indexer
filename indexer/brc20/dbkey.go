package brc20

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/common"
)

// brc20 的ticker允许的字符比较多，存在分隔符-的可能性，比如主网上的42-2
// 构建数据库key时，需要对ticker编码

func encodeTickerName(tickerName string) string {
	return base64.StdEncoding.EncodeToString([]byte(tickerName))
}

func decoderTickerName(tickerName string) string {
	b, err := base64.StdEncoding.DecodeString(tickerName)
	if err != nil {
		common.Log.Panicf("invalid ticker name %s", tickerName)
	}
	return string(b)
}

func GetTickerKey(tickname string) string {
	return fmt.Sprintf("%s%s", DB_PREFIX_TICKER, encodeTickerName(tickname))
}

func GetMintHistoryKey(tickname string, id int64) string {
	return fmt.Sprintf("%s%s-%x", DB_PREFIX_MINTHISTORY, encodeTickerName(tickname), id)
}

// func GetTransferHistoryKey(tickname string, utxo string) string {
// 	return fmt.Sprintf("%s%s-%s", DB_PREFIX_TRANSFER_HISTORY, encodeTickerName(strings.ToLower(tickname)), utxo)
// }

func GetTransferHistoryKey(tickname string, utxoId uint64) string {
	return fmt.Sprintf("%s%s-%x", DB_PREFIX_TRANSFER_HISTORY, encodeTickerName(tickname), utxoId)
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

	return decoderTickerName(parts[0]), parts[1], nil
}

func parseTickListKey(input string) (string, error) {
	if !strings.HasPrefix(input, DB_PREFIX_TICKER) {
		return "", fmt.Errorf("invalid string format")
	}
	return decoderTickerName(strings.TrimPrefix(input, DB_PREFIX_TICKER)), nil
}

func ParseMintHistoryKey(input string) (string, int64, error) {
	if !strings.HasPrefix(input, DB_PREFIX_MINTHISTORY) {
		return "", -1, fmt.Errorf("invalid string format")
	}
	str := strings.TrimPrefix(input, DB_PREFIX_MINTHISTORY)
	parts := strings.Split(str, "-") // ticker name 可能有-，比如：42-c

	if len(parts) != 2 {
		return "", -1, fmt.Errorf("invalid string format")
	}

	id, err := strconv.ParseInt(parts[1], 16, 64)
	if err != nil {
		return "", -1, err
	}

	return decoderTickerName(parts[0]), id, nil
}

func GetHolderInfoKey(addrId uint64, ticker string) string {
	return fmt.Sprintf("%s%x-%s", DB_PREFIX_HOLDER_ASSET, addrId, encodeTickerName(ticker))
}

func parseHolderInfoKey(input string) (uint64, string, error) {
	if !strings.HasPrefix(input, DB_PREFIX_HOLDER_ASSET) {
		return common.INVALID_ID, "", fmt.Errorf("invalid string format")
	}
	parts := strings.Split(input, "-")
	if len(parts) != 3 {
		return common.INVALID_ID, "", fmt.Errorf("invalid string format")
	}
	addrId, err := strconv.ParseUint(parts[1], 16, 64)
	if err != nil {
		return common.INVALID_ID, "", err
	}

	return addrId, decoderTickerName(parts[2]), nil
}

func GetTickerToHolderKey(ticker string, addrId uint64) string {
	return fmt.Sprintf("%s%s-%x", DB_PREFIX_TICKER_HOLDER, encodeTickerName(ticker), addrId)
}

func parseTickerToHolderKey(input string) (string, uint64, error) {
	if !strings.HasPrefix(input, DB_PREFIX_TICKER_HOLDER) {
		return "", common.INVALID_ID, fmt.Errorf("invalid string format")
	}
	str := strings.TrimPrefix(input, DB_PREFIX_TICKER_HOLDER)
	parts := strings.Split(str, "-")
	if len(parts) != 2 {
		return "", common.INVALID_ID, fmt.Errorf("invalid string format")
	}

	addrId, err := strconv.ParseUint(parts[1], 16, 64)
	if err != nil {
		return "", common.INVALID_ID, err
	}

	return decoderTickerName(parts[0]), addrId, nil
}

func newTickerInfo(name string) *BRC20TickInfo {
	return &BRC20TickInfo{
		Name:           name,
		//InscriptionMap: make(map[string]*common.BRC20MintAbbrInfo, 0),
		MintAdded:      make([]*common.BRC20Mint, 0),
	}
}


func GetCurseInscriptionKey(inscriptionId string) string {
	return fmt.Sprintf("%s%s", DB_PREFIX_CURSE_INSCRIPTION_ID, inscriptionId)
}

func GetUtxoToTransferKey(utxoId uint64) string {
	return fmt.Sprintf("%s%x", DB_PREFIX_UTXO_TRANSFER, utxoId)
}

func parseUtxoToTransferKey(input string) (uint64, error) {
	if !strings.HasPrefix(input, DB_PREFIX_UTXO_TRANSFER) {
		return common.INVALID_ID, fmt.Errorf("invalid string format")
	}
	parts := strings.Split(input, "-")
	if len(parts) != 2 {
		return common.INVALID_ID, fmt.Errorf("invalid string format")
	}
	addrId, err := strconv.ParseUint(parts[1], 16, 64)
	if err != nil {
		return common.INVALID_ID, err
	}

	return addrId, nil
}