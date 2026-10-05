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
	for addressID, amount := range s.tickerHoldersLocked(name) {
		if amount > 0 {
			result[addressID] = common.NewDefaultDecimal(amount)
		}
	}
	return result
}

func paginateMintHistory(items []*MintInfo, start, limit int) []*MintInfo {
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
	return items[start : start+limit]
}

func (s *Indexer) GetMintHistory(name string, start, limit int) []*common.MintInfo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	items := paginateMintHistory(s.mintHistoryLocked(name, nil), start, limit)
	result := make([]*common.MintInfo, 0, len(items))
	for _, item := range items {
		result = append(result, item.ToCommon(""))
	}
	return result
}

func (s *Indexer) GetMintHistoryWithAddress(addressID uint64, name string, start, limit int) ([]*common.MintInfo, int) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	items := s.mintHistoryLocked(name, &addressID)
	total := len(items)
	page := paginateMintHistory(items, start, limit)
	result := make([]*common.MintInfo, 0, len(page))
	for _, item := range page {
		result = append(result, item.ToCommon(""))
	}
	return result, total
}

func (s *Indexer) GetAddressAssets(addressID uint64) map[string]int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.holderAssetsLocked(addressID)
}

func (s *Indexer) GetUtxoAssets(utxoID uint64) map[string]int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	result := make(map[string]int64)
	for _, balance := range s.allUtxoBalancesLocked(utxoID) {
		result[balance.Ticker] += balance.Amount
	}
	return result
}

// GetUtxoBalances returns the atomical-level balances for one confirmed UTXO.
// It does not populate the processing cache when called from RPC paths.
func (s *Indexer) GetUtxoBalances(utxoID uint64) []*UtxoBalance {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.allUtxoBalancesLocked(utxoID)
}

func (s *Indexer) GetAssetsWithUtxo(utxoID uint64) map[string]common.AssetOffsets {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	result := make(map[string]common.AssetOffsets)
	for _, balance := range s.allUtxoBalancesLocked(utxoID) {
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
	for utxoID := range utxos {
		for _, balance := range s.allUtxoBalancesLocked(utxoID) {
			result[balance.Ticker] += balance.Amount
		}
	}
	return result
}

func (s *Indexer) GetUtxoBalancesWithTick(ticker string) map[uint64]int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.tickerUtxosLocked(ticker)
}

func (s *Indexer) HasAssetInUtxo(utxoID uint64) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.readUtxoBalanceMapLocked(utxoID)) > 0
}

func (s *Indexer) CheckSelf() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for ticker, info := range s.tickerMap {
		holders := s.tickerHoldersLocked(ticker)
		var holderTotal int64
		for _, amount := range holders {
			holderTotal += amount
		}
		utxos := s.tickerUtxosLocked(ticker)
		var utxoTotal int64
		for _, amount := range utxos {
			utxoTotal += amount
		}
		if holderTotal != utxoTotal {
			common.Log.Errorf("atom ticker %s holder total %d != utxo total %d", ticker, holderTotal, utxoTotal)
			return false
		}
		if info != nil && info.HolderCount != len(holders) {
			common.Log.Errorf("atom ticker %s holder count %d != %d", ticker, info.HolderCount, len(holders))
			return false
		}
	}
	return true
}

func SortAssetNames(items []string) {
	sort.Strings(items)
}
