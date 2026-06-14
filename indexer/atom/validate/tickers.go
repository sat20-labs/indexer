package validate

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type TickerCSVRecord struct {
	Ticker       string
	Height       int
	AtomicalId   string
	UtxoCount    int
	UtxoAmount   int64
	MintedTimes  int64
	MintedAmount int64
	MaxMints     int64
}

func ReadTickerCSV(path string) ([]*TickerCSVRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true

	header, err := r.Read()
	if err != nil {
		return nil, err
	}
	col := make(map[string]int, len(header))
	for i, h := range header {
		col[strings.TrimPrefix(h, "\ufeff")] = i
	}

	var result []*TickerCSVRecord
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		record := &TickerCSVRecord{
			Ticker:     strings.ToLower(row[col["ticker"]]),
			AtomicalId: row[col["atomical_id"]],
		}
		if record.Height, err = atoi(row[col["height"]]); err != nil {
			return nil, err
		}
		if record.UtxoCount, err = atoi(row[col["utxo_count"]]); err != nil {
			return nil, err
		}
		if record.UtxoAmount, err = parseInt(row[col["utxo_amount"]]); err != nil {
			return nil, err
		}
		if record.MintedTimes, err = parseInt(row[col["minted_times"]]); err != nil {
			return nil, err
		}
		if record.MintedAmount, err = parseInt(row[col["minted_amount"]]); err != nil {
			return nil, err
		}
		if record.MaxMints, err = parseInt(row[col["max_mints"]]); err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, nil
}

func ReadTickerCSVDir(dir string) ([]*TickerCSVRecord, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(strings.ToLower(name), ".csv") {
			files = append(files, filepath.Join(dir, name))
		}
	}
	sort.Strings(files)

	var result []*TickerCSVRecord
	for _, path := range files {
		records, err := ReadTickerCSV(path)
		if err != nil {
			return nil, err
		}
		result = append(result, records...)
	}
	return result, nil
}

func atoi(raw string) (int, error) {
	return strconv.Atoi(raw)
}

func parseInt(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}
