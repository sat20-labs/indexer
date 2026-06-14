package atom

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

func (s *Indexer) WriteCompareSnapshot(path string) error {
	if path == "" {
		return nil
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var builder strings.Builder
	fmt.Fprintf(
		&builder,
		"status|version=%s|height=%d|tickers=%d|mints=%d|actions=%d\n",
		s.status.Version,
		s.status.Height,
		s.status.TickerCount,
		s.status.MintCount,
		s.status.ActionCount,
	)

	tickers := make([]string, 0, len(s.tickerMap))
	for ticker := range s.tickerMap {
		tickers = append(tickers, ticker)
	}
	sort.Strings(tickers)
	for _, ticker := range tickers {
		info := s.tickerMap[ticker]
		fmt.Fprintf(&builder, "ticker|name=%s|atomical=%s\n", ticker, info.AtomicalId)
		fmt.Fprintf(
			&builder,
			"ticker_detail|name=%s|atomical=%s|subtype=%s|mint_mode=%s|mint_amount=%d|max_mints=%d|max_mints_global=%d|max_supply=%d|minted_times=%d|minted_amount=%d|deploy_height=%d|mint_height=%d\n",
			ticker,
			info.AtomicalId,
			info.Subtype,
			info.MintMode,
			info.MintAmount,
			info.MaxMints,
			info.MaxMintsGlobal,
			info.MaxSupply,
			info.MintedTimes,
			info.MintedAmount,
			info.DeployHeight,
			info.MintHeight,
		)
	}

	type utxoRow struct {
		ticker   string
		outpoint string
		utxoId   uint64
		amount   int64
		atomical string
		sortKey  string
	}
	rows := make([]utxoRow, 0)
	for _, balances := range s.utxoBalances {
		for _, balance := range balances {
			if balance.Amount <= 0 {
				continue
			}
			row := utxoRow{
				ticker:   balance.Ticker,
				outpoint: balance.Outpoint,
				utxoId:   balance.UtxoId,
				amount:   balance.Amount,
				atomical: balance.AtomicalId,
			}
			row.sortKey = row.ticker + "|" + row.outpoint + "|" + row.atomical
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].sortKey < rows[j].sortKey
	})
	for _, row := range rows {
		fmt.Fprintf(
			&builder,
			"utxo|ticker=%s|outpoint=%s|utxo_id=%d|amount=%d|atomical=%s\n",
			row.ticker,
			row.outpoint,
			row.utxoId,
			row.amount,
			row.atomical,
		)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(builder.String()), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
