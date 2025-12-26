package exotic

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/base"
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
		Name:           name,
		UtxoMap:        make(map[uint64]common.AssetOffsets),
		InscriptionMap: make(map[string]*common.MintAbbrInfo, 0),
		MintAdded:      make([]*common.Mint, 0),
	}
}

func  PrintHoldersWithMap(holders map[uint64]int64, baseIndexer *base.BaseIndexer) {
	var total int64
	type pair struct {
		addressId uint64
		amt       int64
	}
	mid := make([]*pair, 0)
	for addressId, amt := range holders {
		//common.Log.Infof("%x: %s", addressId, amt.String())
		total += amt
		mid = append(mid, &pair{
			addressId: addressId,
			amt:       amt,
		})
	}
	sort.Slice(mid, func(i, j int) bool {
		return mid[i].amt > mid[j].amt
	})
	limit := 10 //len(mid) // 40
	for i, item := range mid {
		if i > limit {
			break
		}
		if item.amt == 0 {
			continue
		}
		address, err := baseIndexer.GetAddressByID(item.addressId)
		if err != nil {
			common.Log.Panicf("printHoldersWithMap GetAddressByID %x failed, %v", item.addressId, err)
			address = "-\t"
		}
		fmt.Printf("\"%s\": %d,\n", address, item.amt)
	}
	common.Log.Infof("total in holders: %d", total)
}

func (p *ExoticIndexer) printHolders(name string) {
	holdermap := p.GetHolderAndAmountWithTick(name)
	common.Log.Infof("holders from holder DB")
	PrintHoldersWithMap(holdermap, p.baseIndexer)
}
