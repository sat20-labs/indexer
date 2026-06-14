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

type UtxoCSVRecord struct {
	Ticker string
	Height int
	UtxoId uint64
	Amount int64
}

func ReadUtxoCSV(path string) ([]*UtxoCSVRecord, error) {
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

	var result []*UtxoCSVRecord
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		height, err := strconv.Atoi(row[col["height"]])
		if err != nil {
			return nil, err
		}
		utxoId, err := strconv.ParseUint(row[col["utxo_id"]], 10, 64)
		if err != nil {
			return nil, err
		}
		amount, err := strconv.ParseInt(row[col["amount"]], 10, 64)
		if err != nil {
			return nil, err
		}
		result = append(result, &UtxoCSVRecord{
			Ticker: strings.ToLower(row[col["ticker"]]),
			Height: height,
			UtxoId: utxoId,
			Amount: amount,
		})
	}
	return result, nil
}

func ReadUtxoCSVDir(dir string) ([]*UtxoCSVRecord, error) {
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

	var result []*UtxoCSVRecord
	for _, path := range files {
		records, err := ReadUtxoCSV(path)
		if err != nil {
			return nil, err
		}
		result = append(result, records...)
	}
	return result, nil
}
