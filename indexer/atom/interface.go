package atom

import (
	"sort"
	"strings"

	"github.com/sat20-labs/indexer/common"
)

func (s *Indexer) GetTickerInfo(name string) *common.TickerInfo {
	ticker := s.GetTicker(name)
	if ticker == nil {
		return nil
	}
	return &common.TickerInfo{
		AssetName: common.AssetName{
			Protocol: common.PROTOCOL_NAME_ATOM,
			Type:     common.ASSET_TYPE_FT,
			Ticker:   strings.ToLower(ticker.Name),
		},
		DisplayName:     ticker.DisplayName,
		Id:              ticker.Id,
		Divisibility:    0,
		StartBlock:      int(ticker.MintHeight),
		DeployHeight:    ticker.DeployHeight,
		DeployBlocktime: ticker.DeployTime,
		DeployTx:        ticker.DeployTx,
		Limit:           common.NewDefaultDecimal(ticker.MintAmount).String(),
		N:               1,
		TotalMinted:     common.NewDefaultDecimal(ticker.MintedAmount).String(),
		MintTimes:       ticker.MintedTimes,
		MaxSupply:       common.NewDefaultDecimal(ticker.MaxSupply).String(),
		HoldersCount:    ticker.HolderCount,
		InscriptionId:   ticker.AtomicalId,
		Status:          common.TICKER_STATUS_INIT,
	}
}

func (s *Indexer) GetMintAmount(name string) (*common.Decimal, int64) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	ticker := s.getTickerLocked(strings.ToLower(name))
	if ticker == nil {
		return nil, 0
	}
	return common.NewDefaultDecimal(ticker.MintedAmount), ticker.MintedTimes
}

func (s *Indexer) GetHoldersWithTick(name string) map[uint64]*common.Decimal {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	result := make(map[uint64]*common.Decimal)
	for addressId, amount := range s.tickerHolders[strings.ToLower(name)] {
		if amount > 0 {
			result[addressId] = common.NewDefaultDecimal(amount)
		}
	}
	return result
}

func (s *Indexer) GetMintHistory(name string, start, limit int) []*common.MintInfo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	items := s.mintHistory[strings.ToLower(name)]
	total := len(items)
	if start < 0 {
		start = 0
	}
	if start >= total {
		return nil
	}
	if limit <= 0 || start+limit > total {
		limit = total - start
	}
	result := make([]*common.MintInfo, 0, limit)
	for _, item := range items[start : start+limit] {
		result = append(result, item.ToCommon(""))
	}
	return result
}

func (s *Indexer) GetMintHistoryWithAddress(addressId uint64, name string, start, limit int) ([]*common.MintInfo, int) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	filtered := make([]*MintInfo, 0)
	for _, item := range s.mintHistory[strings.ToLower(name)] {
		if item.AddressId == addressId {
			filtered = append(filtered, item)
		}
	}
	total := len(filtered)
	if start < 0 {
		start = 0
	}
	if start >= total {
		return nil, total
	}
	if limit <= 0 || start+limit > total {
		limit = total - start
	}
	result := make([]*common.MintInfo, 0, limit)
	for _, item := range filtered[start : start+limit] {
		result = append(result, item.ToCommon(""))
	}
	return result, total
}

func (s *Indexer) GetAddressAssets(addressId uint64) map[string]int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	result := make(map[string]int64)
	for ticker, amount := range s.holderBalances[addressId] {
		if amount > 0 {
			result[ticker] = amount
		}
	}
	return result
}

func (s *Indexer) GetUtxoAssets(utxoId uint64) map[string]int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	result := make(map[string]int64)
	for _, balance := range s.utxoBalances[utxoId] {
		result[balance.Ticker] += balance.Amount
	}
	return result
}

func (s *Indexer) GetAssetsWithUtxo(utxoId uint64) map[string]common.AssetOffsets {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	result := make(map[string]common.AssetOffsets)
	for _, balance := range s.utxoBalances[utxoId] {
		if balance.Amount <= 0 {
			continue
		}
		offsets := result[balance.Ticker]
		start := offsets.Size()
		result[balance.Ticker] = append(offsets, &common.OffsetRange{
			Start: start,
			End:   start + balance.Amount,
		})
	}
	return result
}

func (s *Indexer) GetAssetSummaryByAddress(utxos map[uint64]int64) map[string]int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	result := make(map[string]int64)
	for utxoId := range utxos {
		for _, balance := range s.utxoBalances[utxoId] {
			result[balance.Ticker] += balance.Amount
		}
	}
	return result
}

func (s *Indexer) GetUtxoBalancesWithTick(ticker string) map[uint64]int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	result := make(map[uint64]int64)
	for utxoId, amount := range s.tickerUtxos[strings.ToLower(ticker)] {
		if amount > 0 {
			result[utxoId] = amount
		}
	}
	return result
}

func (s *Indexer) HasAssetInUtxo(utxoId uint64) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.utxoBalances[utxoId]) > 0
}

func (s *Indexer) CheckSelf() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for ticker, holders := range s.tickerHolders {
		var holderTotal int64
		for _, amount := range holders {
			holderTotal += amount
		}
		var utxoTotal int64
		for _, amount := range s.tickerUtxos[ticker] {
			utxoTotal += amount
		}
		if holderTotal != utxoTotal {
			common.Log.Errorf("atom ticker %s holder total %d != utxo total %d", ticker, holderTotal, utxoTotal)
			return false
		}
	}
	return true
}

func SortAssetNames(items []string) {
	sort.Strings(items)
}
