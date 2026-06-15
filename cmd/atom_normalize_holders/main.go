package main

import (
	"encoding/csv"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

type holderRow struct {
	ticker  string
	height  string
	address string
	amount  int64
}

func main() {
	inPath := flag.String("in", "", "input holder CSV")
	outPath := flag.String("out", "", "output holder CSV")
	flag.Parse()

	if *inPath == "" {
		fatalf("-in is required")
	}
	if *outPath == "" {
		*outPath = *inPath
	}

	rows, err := readRows(*inPath)
	if err != nil {
		fatalf("%v", err)
	}
	rows = normalizeRows(rows)
	if err := writeRows(*outPath, rows); err != nil {
		fatalf("%v", err)
	}
}

func readRows(path string) ([]holderRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("%s is empty", path)
	}

	rows := make([]holderRow, 0, len(records)-1)
	for i, record := range records[1:] {
		if len(record) != 4 {
			return nil, fmt.Errorf("%s row %d: expected 4 fields, got %d", path, i+2, len(record))
		}
		amount, err := strconv.ParseInt(record[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%s row %d: invalid amount %q", path, i+2, record[3])
		}
		rows = append(rows, holderRow{
			ticker:  record[0],
			height:  record[1],
			address: record[2],
			amount:  amount,
		})
	}
	return rows, nil
}

func normalizeRows(rows []holderRow) []holderRow {
	type key struct {
		ticker  string
		height  string
		address string
	}
	grouped := make(map[key]int64, len(rows))
	for _, row := range rows {
		row.address = normalizeAddress(row.address)
		grouped[key{ticker: row.ticker, height: row.height, address: row.address}] += row.amount
	}

	normalized := make([]holderRow, 0, len(grouped))
	for key, amount := range grouped {
		normalized = append(normalized, holderRow{
			ticker:  key.ticker,
			height:  key.height,
			address: key.address,
			amount:  amount,
		})
	}
	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].amount != normalized[j].amount {
			return normalized[i].amount > normalized[j].amount
		}
		if normalized[i].ticker != normalized[j].ticker {
			return normalized[i].ticker < normalized[j].ticker
		}
		if normalized[i].height != normalized[j].height {
			return normalized[i].height < normalized[j].height
		}
		return normalized[i].address < normalized[j].address
	})
	return normalized
}

func normalizeAddress(address string) string {
	pkScript, err := hex.DecodeString(address)
	if err != nil || len(pkScript) == 0 {
		return address
	}
	_, addresses, _, err := txscript.ExtractPkScriptAddrs(pkScript, &chaincfg.MainNetParams)
	if err != nil || len(addresses) == 0 {
		return address
	}
	return addresses[0].EncodeAddress()
}

func writeRows(path string, rows []holderRow) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(f)
	if err := writer.Write([]string{"ticker", "height", "address", "amount"}); err != nil {
		_ = f.Close()
		return err
	}
	for _, row := range rows {
		if err := writer.Write([]string{row.ticker, row.height, row.address, strconv.FormatInt(row.amount, 10)}); err != nil {
			_ = f.Close()
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
