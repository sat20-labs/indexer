package runes

import (
	"fmt"

	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"github.com/sat20-labs/indexer/indexer/runes/table"
	"lukechampine.com/uint128"
)

type runeHolderTotals struct {
	amount uint128.Uint128
	count  uint64
}

// collectHolderTotals scans the holder table and overlays pending writes once.
// A per-rune GetBalances would rescan the entire pending map for every rune.
func (s *Indexer) collectHolderTotals() (map[string]runeHolderTotals, map[uint64]bool, error) {
	totals := make(map[string]runeHolderTotals)
	holders := make(map[uint64]bool)
	err := s.runeIdAddressToBalanceTbl.Cache.ForEach([]byte(store.RUNEID_ADDRESS_TO_BALANCE), func(key []byte, value *pb.RuneIdAddressToBalance) error {
		row, err := table.RuneIdAddressToBalanceFromString(string(key))
		if err != nil {
			return fmt.Errorf("parse holder key %q: %w", key, err)
		}
		if value.Balance == nil || value.Balance.Value == nil || row.AddressId != value.AddressId {
			return fmt.Errorf("invalid holder balance %q", key)
		}
		id := row.RuneId.String()
		total := totals[id]
		amount := uint128.Uint128{Hi: value.Balance.Value.Hi, Lo: value.Balance.Value.Lo}
		next := total.amount.AddWrap(amount)
		if next.Cmp(total.amount) < 0 {
			return fmt.Errorf("holder amount overflows uint128 for rune %s", id)
		}
		total.amount = next
		total.count++
		totals[id] = total
		holders[value.AddressId] = true
		return nil
	})
	return totals, holders, err
}
