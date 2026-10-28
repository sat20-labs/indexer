package atom

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func getUtxoBalancePrefix(utxoID uint64) []byte {
	return []byte(fmt.Sprintf("%s%d-", DB_PREFIX_UTXO_BALANCE, utxoID))
}

func getHolderAssetPrefix(addressID uint64) []byte {
	return []byte(fmt.Sprintf("%s%d-", DB_PREFIX_HOLDER_ASSET, addressID))
}

func getTickerHolderPrefix(ticker string) []byte {
	return []byte(DB_PREFIX_TICKER_HOLDER + encodeTickerName(strings.ToLower(ticker)) + "-")
}

func getTickerUtxoPrefix(ticker string) []byte {
	return []byte(DB_PREFIX_TICKER_UTXO + encodeTickerName(strings.ToLower(ticker)) + "-")
}

func getMintHistoryPrefix(ticker string) []byte {
	return []byte(DB_PREFIX_MINTHISTORY + encodeTickerName(strings.ToLower(ticker)) + "-")
}

func getAddressMintHistoryPrefix(ticker string, addressID uint64) []byte {
	return []byte(fmt.Sprintf("%s%s-%d-", DB_PREFIX_ADDRESS_MINTHISTORY, encodeTickerName(strings.ToLower(ticker)), addressID))
}

func GetAddressMintHistoryKey(ticker string, addressID uint64, id int64) string {
	return fmt.Sprintf("%s%s-%d-%s", DB_PREFIX_ADDRESS_MINTHISTORY, encodeTickerName(strings.ToLower(ticker)), addressID, common.Uint64ToString(uint64(id)))
}

func readInt64Value(database common.KVDB, key string) int64 {
	if database == nil {
		return 0
	}
	var result int64
	if err := db.GetValueFromDB([]byte(key), &result, database); err != nil {
		if err != common.ErrKeyNotFound {
			common.Log.Panicf("atom read aggregate %s failed: %v", key, err)
		}
		return 0
	}
	return result
}

func (s *Indexer) ensureHolderAggregateLoadedLocked(addressID uint64, ticker string) {
	ticker = strings.ToLower(ticker)
	if s.holderBalances[addressID] == nil {
		s.holderBalances[addressID] = make(map[string]int64)
	}
	if _, exists := s.holderBalances[addressID][ticker]; !exists {
		s.holderBalances[addressID][ticker] = readInt64Value(s.db, GetHolderAssetKey(addressID, ticker))
	}
	if s.tickerHolders[ticker] == nil {
		s.tickerHolders[ticker] = make(map[uint64]int64)
	}
	if _, exists := s.tickerHolders[ticker][addressID]; !exists {
		s.tickerHolders[ticker][addressID] = readInt64Value(s.db, GetTickerHolderKey(ticker, addressID))
	}
}

func (s *Indexer) readUtxoBalanceMapLocked(utxoID uint64) map[string]*UtxoBalance {
	items := make(map[string]*UtxoBalance)
	if s.db != nil {
		if err := s.db.Scan(common.ScanOptions{Prefix: getUtxoBalancePrefix(utxoID)}, func(_, value []byte) error {
			var balance UtxoBalance
			if err := db.DecodeBytes(value, &balance); err != nil {
				return err
			}
			items[balance.AtomicalId] = balance.Clone()
			return nil
		}); err != nil {
			common.Log.Panicf("atom load UTXO balances %d failed: %v", utxoID, err)
		}
	}
	for _, balance := range s.utxoTouched {
		if balance.UtxoId == utxoID {
			items[balance.AtomicalId] = balance.Clone()
		}
	}
	for _, balance := range s.utxoDeleted {
		if balance.UtxoId == utxoID {
			delete(items, balance.AtomicalId)
		}
	}
	return items
}

func (s *Indexer) ensureUtxoBalancesLoadedLocked(utxoID uint64) map[string]*UtxoBalance {
	if items, exists := s.utxoBalances[utxoID]; exists {
		return items
	}
	items := s.readUtxoBalanceMapLocked(utxoID)
	s.utxoBalances[utxoID] = items

	perTicker := make(map[string]int64)
	for _, balance := range items {
		ticker := strings.ToLower(balance.Ticker)
		perTicker[ticker] += balance.Amount
		s.ensureHolderAggregateLoadedLocked(balance.AddressId, ticker)
	}
	for ticker, amount := range perTicker {
		if s.tickerUtxos[ticker] == nil {
			s.tickerUtxos[ticker] = make(map[uint64]int64)
		}
		s.tickerUtxos[ticker][utxoID] = amount
	}
	return items
}

func balancesToSortedSlice(items map[string]*UtxoBalance) []*UtxoBalance {
	ids := make([]string, 0, len(items))
	for atomicalID := range items {
		ids = append(ids, atomicalID)
	}
	sortAtomicalIds(ids)
	result := make([]*UtxoBalance, 0, len(ids))
	for _, atomicalID := range ids {
		result = append(result, items[atomicalID].Clone())
	}
	return result
}

func (s *Indexer) allUtxoBalancesLocked(utxoID uint64) []*UtxoBalance {
	return balancesToSortedSlice(s.readUtxoBalanceMapLocked(utxoID))
}

// forEachCurrentUtxoBalanceLocked reads durable balances with pending writes
// overlaid, without filling the processing cache. The caller holds s.mutex;
// visit must not mutate balances or the pending maps.
func (s *Indexer) forEachCurrentUtxoBalanceLocked(visit func(*UtxoBalance)) error {
	if s.db != nil {
		if err := s.db.Scan(common.ScanOptions{Prefix: []byte(DB_PREFIX_UTXO_BALANCE)}, func(key, value []byte) error {
			if _, deleted := s.utxoDeleted[string(key)]; deleted {
				return nil
			}
			if _, replaced := s.utxoTouched[string(key)]; replaced {
				return nil
			}
			var balance UtxoBalance
			if err := db.DecodeBytes(value, &balance); err != nil {
				return err
			}
			visit(&balance)
			return nil
		}); err != nil {
			return err
		}
	}
	for key, balance := range s.utxoTouched {
		if _, deleted := s.utxoDeleted[key]; !deleted {
			visit(balance)
		}
	}
	return nil
}

func (s *Indexer) holderAssetsLocked(addressID uint64) map[string]int64 {
	result := make(map[string]int64)
	if s.db != nil {
		prefix := getHolderAssetPrefix(addressID)
		if err := s.db.Scan(common.ScanOptions{Prefix: prefix}, func(key, value []byte) error {
			var amount int64
			if err := db.DecodeBytes(value, &amount); err != nil {
				return err
			}
			encodedTicker := strings.TrimPrefix(string(key), string(prefix))
			tickerBytes, err := decodeTickerName(encodedTicker)
			if err != nil {
				return err
			}
			if amount > 0 {
				result[tickerBytes] = amount
			}
			return nil
		}); err != nil {
			common.Log.Panicf("atom scan holder assets %d failed: %v", addressID, err)
		}
	}
	for key, amount := range s.holderTouched {
		prefix := fmt.Sprintf("%s%d-", DB_PREFIX_HOLDER_ASSET, addressID)
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		encodedTicker := strings.TrimPrefix(key, prefix)
		ticker, err := decodeTickerName(encodedTicker)
		if err != nil {
			common.Log.Panicf("atom parse pending holder key %s failed: %v", key, err)
		}
		if amount == 0 {
			delete(result, ticker)
		} else {
			result[ticker] = amount
		}
	}
	return result
}

func decodeTickerName(encoded string) (string, error) {
	value, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return strings.ToLower(string(value)), nil
}

func (s *Indexer) tickerHoldersLocked(ticker string) map[uint64]int64 {
	ticker = strings.ToLower(ticker)
	result := make(map[uint64]int64)
	prefix := getTickerHolderPrefix(ticker)
	if s.db != nil {
		if err := s.db.Scan(common.ScanOptions{Prefix: prefix}, func(key, value []byte) error {
			var amount int64
			if err := db.DecodeBytes(value, &amount); err != nil {
				return err
			}
			addressID, err := strconv.ParseUint(strings.TrimPrefix(string(key), string(prefix)), 10, 64)
			if err != nil {
				return err
			}
			if amount > 0 {
				result[addressID] = amount
			}
			return nil
		}); err != nil {
			common.Log.Panicf("atom scan ticker holders %s failed: %v", ticker, err)
		}
	}
	for key, amount := range s.holderTouched {
		if !strings.HasPrefix(key, string(prefix)) {
			continue
		}
		addressID, err := strconv.ParseUint(strings.TrimPrefix(key, string(prefix)), 10, 64)
		if err != nil {
			common.Log.Panicf("atom parse pending ticker holder key %s failed: %v", key, err)
		}
		if amount == 0 {
			delete(result, addressID)
		} else {
			result[addressID] = amount
		}
	}
	return result
}

func (s *Indexer) tickerUtxosLocked(ticker string) map[uint64]int64 {
	ticker = strings.ToLower(ticker)
	type entry struct {
		utxoID uint64
		amount int64
	}
	entries := make(map[string]entry)
	prefix := getTickerUtxoPrefix(ticker)
	if s.db != nil {
		if err := s.db.Scan(common.ScanOptions{Prefix: prefix}, func(key, value []byte) error {
			parts := strings.Split(strings.TrimPrefix(string(key), string(prefix)), "-")
			if len(parts) < 2 {
				return fmt.Errorf("invalid atom ticker UTXO key %s", key)
			}
			utxoID, err := strconv.ParseUint(parts[0], 10, 64)
			if err != nil {
				return err
			}
			var amount int64
			if err := db.DecodeBytes(value, &amount); err != nil {
				return err
			}
			entries[string(key)] = entry{utxoID: utxoID, amount: amount}
			return nil
		}); err != nil {
			common.Log.Panicf("atom scan ticker UTXOs %s failed: %v", ticker, err)
		}
	}
	for key, balance := range s.utxoDeleted {
		if strings.EqualFold(balance.Ticker, ticker) {
			delete(entries, GetTickerUtxoKey(ticker, balance.UtxoId, balance.AtomicalId))
		}
		_ = key
	}
	for _, balance := range s.utxoTouched {
		if strings.EqualFold(balance.Ticker, ticker) {
			key := GetTickerUtxoKey(ticker, balance.UtxoId, balance.AtomicalId)
			entries[key] = entry{utxoID: balance.UtxoId, amount: balance.Amount}
		}
	}
	result := make(map[uint64]int64)
	for _, item := range entries {
		result[item.utxoID] += item.amount
	}
	return result
}

// mintHistoryPageLocked is used only by service queries, with s.mutex held.
// Durable keys already order mint IDs; only pending IDs need sorting. Decode
// values only for the requested page, and count keys when an exact total is needed.
func (s *Indexer) mintHistoryPageLocked(ticker string, addressID *uint64, start, limit int, countTotal bool) ([]*MintInfo, int) {
	ticker = strings.ToLower(ticker)
	if start < 0 {
		start = 0
	}
	prefix := getMintHistoryPrefix(ticker)
	if addressID != nil {
		prefix = getAddressMintHistoryPrefix(ticker, *addressID)
	}
	pending := make(map[int64]*MintInfo)
	for _, mint := range s.mintsAdded {
		if !strings.EqualFold(mint.Ticker, ticker) {
			continue
		}
		if addressID != nil && mint.AddressId != *addressID {
			continue
		}
		pending[mint.Id] = mint
	}
	ids := make([]int64, 0, len(pending))
	for id := range pending {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var pageIDs []int64
	total := 0
	accept := func(id int64) {
		if total >= start && (limit <= 0 || len(pageIDs) < limit) {
			pageIDs = append(pageIDs, id)
		}
		total++
	}
	pageComplete := func() bool {
		return !countTotal && limit > 0 && len(pageIDs) == limit
	}
	next := 0
	if s.db != nil {
		if err := s.db.Scan(common.ScanOptions{Prefix: prefix, KeysOnly: true}, func(key, _ []byte) error {
			suffix := string(key[len(prefix):])
			if len(suffix) != 16 {
				return fmt.Errorf("invalid mint history key %q", key)
			}
			id, err := strconv.ParseInt(suffix, 16, 64)
			if err != nil || common.Uint64ToString(uint64(id)) != suffix {
				return fmt.Errorf("invalid mint history ID in key %q", key)
			}
			for next < len(ids) && ids[next] <= id {
				accept(ids[next])
				next++
				if pageComplete() {
					return common.ErrStopScan
				}
			}
			// Pending values replace durable values with the same ID.
			if _, exists := pending[id]; !exists {
				accept(id)
			}
			if pageComplete() {
				return common.ErrStopScan
			}
			return nil
		}); err != nil {
			common.Log.Panicf("atom scan mint history %s failed: %v", ticker, err)
		}
	}
	for next < len(ids) && !pageComplete() {
		accept(ids[next])
		next++
	}
	items := make([]*MintInfo, 0, len(pageIDs))
	for _, id := range pageIDs {
		if mint, exists := pending[id]; exists {
			items = append(items, mint.Clone())
			continue
		}
		key := string(prefix) + common.Uint64ToString(uint64(id))
		var mint MintInfo
		if err := db.GetValueFromDB([]byte(key), &mint, s.db); err != nil {
			common.Log.Panicf("atom read mint history %s failed: %v", key, err)
		}
		if mint.Id != id {
			common.Log.Panicf("atom mint history ID differs from key %s: %d", key, mint.Id)
		}
		items = append(items, mint.Clone())
	}
	return items, total
}
