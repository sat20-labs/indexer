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

func GetTickerIdKey(id int64) string {
	return fmt.Sprintf("%s%s", DB_PREFIX_ID_TO_TICKER, common.Uint64ToString(uint64(id)))
}

func GetMintHistoryKey(tickname string, id int64) string {
	return fmt.Sprintf("%s%s-%s", DB_PREFIX_MINTHISTORY, encodeTickerName(tickname), common.Uint64ToString(uint64(id)))
}

func GetTransferHistoryKey(tickname string, utxoId uint64, nftId int64) string {
	height, txIndx, _ := common.FromUtxoId(utxoId)
	return fmt.Sprintf("%s%s-%x-%x-%x", DB_PREFIX_TRANSFER_HISTORY, encodeTickerName(tickname), height, txIndx, nftId)
}

func ParseTransferHistoryKey(input string) (string, int, int64, error) {
	if !strings.HasPrefix(input, DB_PREFIX_TRANSFER_HISTORY) {
		return "", -1, -1, fmt.Errorf("invalid string format")
	}
	parts := strings.Split(input, "-")

	if len(parts) != 5 {
		return "", -1, -1, fmt.Errorf("invalid string format")
	}

	height, err := strconv.ParseInt(parts[2], 16, 32)
	if err != nil {
		return "", -1, -1, err
	}

	nftId, err := strconv.ParseInt(parts[3], 16, 64)
	if err != nil {
		return "", -1, -1, err
	}

	return decoderTickerName(parts[1]), int(height), nftId, nil
}

func GetHolderTransferHistoryKey(tickname string, holder uint64, nftId int64) string {
	return fmt.Sprintf("%s%s-%x-%x", DB_PREFIX_TRANSFER_HISTORY_HOLDER, encodeTickerName(tickname), holder, nftId)
}

func ParseHolderTransferHistoryKey(input string) (string, uint64, int64, error) {
	if !strings.HasPrefix(input, DB_PREFIX_TRANSFER_HISTORY_HOLDER) {
		return "", common.INVALID_ID, -1, fmt.Errorf("invalid string format")
	}
	parts := strings.Split(input, "-")

	if len(parts) != 4 {
		return "", common.INVALID_ID, -1, fmt.Errorf("invalid string format")
	}

	holder, err := strconv.ParseUint(parts[2], 16, 64)
	if err != nil {
		return "", common.INVALID_ID, -1, err
	}

	nftId, err := strconv.ParseInt(parts[3], 16, 64)
	if err != nil {
		return "", common.INVALID_ID, -1, err
	}

	return decoderTickerName(parts[1]), holder, nftId, nil
}

func parseTickerKey(input string) (string, error) {
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

	id, err := common.StringToUint64(parts[1])
	if err != nil {
		return "", -1, err
	}

	return decoderTickerName(parts[0]), int64(id), nil
}

func GetHolderInfoKey(addrId uint64, ticker string) string {
	return fmt.Sprintf("%s%s-%s", DB_PREFIX_HOLDER_ASSET, common.Uint64ToString(addrId), encodeTickerName(ticker))
}

func parseHolderInfoKey(input string) (uint64, string, error) {
	if !strings.HasPrefix(input, DB_PREFIX_HOLDER_ASSET) {
		return common.INVALID_ID, "", fmt.Errorf("invalid string format")
	}
	parts := strings.Split(input, "-")
	if len(parts) != 3 {
		return common.INVALID_ID, "", fmt.Errorf("invalid string format")
	}
	addrId, err := common.StringToUint64(parts[1])
	if err != nil {
		return common.INVALID_ID, "", err
	}

	return addrId, decoderTickerName(parts[2]), nil
}

func GetTickerToHolderKey(ticker string, addrId uint64) string {
	return fmt.Sprintf("%s%s-%s", DB_PREFIX_TICKER_HOLDER, encodeTickerName(ticker), common.Uint64ToString(addrId))
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

	addrId, err := common.StringToUint64(parts[1])
	if err != nil {
		return "", common.INVALID_ID, err
	}

	return decoderTickerName(parts[0]), addrId, nil
}

func newTickerInfo(name string) *BRC20TickInfo {
	return &BRC20TickInfo{
		Name: name,
		//InscriptionMap: make(map[string]*common.BRC20MintAbbrInfo, 0),
		MintAdded: make([]*common.BRC20Mint, 0),
	}
}

func GetCurseInscriptionKey(inscriptionId string) string {
	return fmt.Sprintf("%s%s", DB_PREFIX_CURSE_INSCRIPTION_ID, inscriptionId)
}

func GetUtxoToTransferKey(utxoId uint64) string {
	return fmt.Sprintf("%s%s", DB_PREFIX_UTXO_TRANSFER, common.Uint64ToString(utxoId))
}

func parseUtxoToTransferKey(input string) (uint64, error) {
	if !strings.HasPrefix(input, DB_PREFIX_UTXO_TRANSFER) {
		return common.INVALID_ID, fmt.Errorf("invalid string format")
	}
	parts := strings.Split(input, "-")
	if len(parts) != 2 {
		return common.INVALID_ID, fmt.Errorf("invalid string format")
	}
	addrId, err := common.StringToUint64(parts[1])
	if err != nil {
		return common.INVALID_ID, err
	}

	return addrId, nil
}
