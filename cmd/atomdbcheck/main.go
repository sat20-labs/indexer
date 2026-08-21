package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/atom"
	"github.com/sat20-labs/indexer/indexer/db"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: atomdbcheck <atom-db-path>\n")
		os.Exit(2)
	}

	kv := db.NewKVDB(os.Args[1])
	if kv == nil {
		fmt.Fprintf(os.Stderr, "open db failed: %s\n", os.Args[1])
		os.Exit(1)
	}
	defer kv.Close()

	var status atom.Status
	err := db.GetValueFromDB([]byte(atom.DB_STATUS_KEY), &status, kv)
	if err == common.ErrKeyNotFound {
		fmt.Println("status|missing")
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "read status failed: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Printf("status|version=%s|height=%d|tickers=%d|mints=%d|actions=%d\n",
			status.Version, status.Height, status.TickerCount, status.MintCount, status.ActionCount)
	}

	prefixes := []struct {
		name   string
		prefix string
	}{
		{"ticker", atom.DB_PREFIX_TICKER},
		{"id_to_ticker", atom.DB_PREFIX_ID_TO_TICKER},
		{"utxo_balance", atom.DB_PREFIX_UTXO_BALANCE},
		{"ticker_utxo", atom.DB_PREFIX_TICKER_UTXO},
		{"holder_asset", atom.DB_PREFIX_HOLDER_ASSET},
		{"ticker_holder", atom.DB_PREFIX_TICKER_HOLDER},
		{"mint_history", atom.DB_PREFIX_MINTHISTORY},
		{"action", atom.DB_PREFIX_ACTION},
	}
	for _, item := range prefixes {
		count := 0
		if err := kv.BatchRead([]byte(item.prefix), false, func(_, _ []byte) error {
			count++
			return nil
		}); err != nil {
			fmt.Fprintf(os.Stderr, "scan %s failed: %v\n", item.name, err)
			os.Exit(1)
		}
		fmt.Printf("prefix|%s|%d\n", item.name, count)
	}

	if err := kv.BatchRead([]byte(atom.DB_PREFIX_TICKER), false, func(_, v []byte) error {
		var ticker atom.Ticker
		if err := db.DecodeBytes(v, &ticker); err != nil {
			return err
		}
		fmt.Printf("ticker|name=%s|id=%d|atomical=%s|deploy_height=%d|deploy_tx=%s|mint_amount=%d|max_mints=%d|minted_times=%d|minted_amount=%d|holders=%d|mint_bitworkc=%s|mint_bitworkr=%s|bitworkc=%s|bitworkr=%s|mint_height=%d|mint_mode=%s\n",
			ticker.Name, ticker.Id, ticker.AtomicalId, ticker.DeployHeight, ticker.DeployTx, ticker.MintAmount, ticker.MaxMints, ticker.MintedTimes, ticker.MintedAmount, ticker.HolderCount, ticker.MintBitworkc, ticker.MintBitworkr, ticker.Bitworkc, ticker.Bitworkr, ticker.MintHeight, ticker.MintMode)
		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "scan tickers failed: %v\n", err)
		os.Exit(1)
	}

	if err := kv.BatchRead([]byte(atom.DB_PREFIX_MINTHISTORY), false, func(_, v []byte) error {
		var mint atom.MintInfo
		if err := db.DecodeBytes(v, &mint); err != nil {
			return err
		}
		fmt.Printf("mint|ticker=%s|id=%d|atomical=%s|height=%d|tx_index=%d|txid=%s|amount=%d|utxo_id=%d|outpoint=%s|address_id=%d\n",
			mint.Ticker, mint.Id, mint.AtomicalId, mint.Height, mint.TxIndex, mint.TxId, mint.Amount, mint.UtxoId, mint.Outpoint, mint.AddressId)
		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "scan mint history failed: %v\n", err)
		os.Exit(1)
	}

	if err := kv.BatchRead([]byte(atom.DB_PREFIX_UTXO_BALANCE), false, func(k, v []byte) error {
		var balance atom.UtxoBalance
		if err := db.DecodeBytes(v, &balance); err != nil {
			return err
		}
		fmt.Printf("utxo|key=%s|ticker=%s|atomical=%s|amount=%d|utxo_id=%d|outpoint=%s|address_id=%d\n",
			strings.TrimSpace(string(k)), balance.Ticker, balance.AtomicalId, balance.Amount, balance.UtxoId, balance.Outpoint, balance.AddressId)
		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "scan utxo balances failed: %v\n", err)
		os.Exit(1)
	}
}
