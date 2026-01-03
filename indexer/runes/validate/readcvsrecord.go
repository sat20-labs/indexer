package validate

import (
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/common"
)

// rune,height,address,balance,available_balance,total_balance,snapshot_time
type HolderCSVRecord struct {
	Ticker             string
	Height             int
	Address            string
	Balance            string
}

func parseInt32(s string) int {
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseInt(s, 10, 32)
	return int(v)
}

func parseU64(s string) uint64 {
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

func parseI64(s string) int64 {
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}


func ReadCSVFile(path string) (map[string]*HolderCSVRecord, int, int, error) {
	if strings.HasSuffix(path, ".gz") {
		csvPath, cleanup, err := DecompressToTempCSV(path)
		if err != nil {
			panic(err)
		}
		defer cleanup()
		path = csvPath
	}
	
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	r.TrimLeadingSpace = true

	// 读取 header
	header, err := r.Read()
	if err != nil {
		return nil, 0, 0, err
	}

	col := make(map[string]int)
	for i, h := range header {
		h = strings.TrimPrefix(h, "\ufeff")
		col[h] = i
	}

	var start, end int
	start = 0xffffffff
	result := make(map[string]*HolderCSVRecord)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, 0, err
		}

		h := parseInt32(row[col["height"]])
		if h < start {
			start = h
		}
		if h > end {
			end = h
		}

		if row[col["valid"]] == "0" {
			continue
		}

		rec := &HolderCSVRecord{
			Ticker:   strings.ToLower(row[col["rune"]]),
			Height:   parseInt32(row[col["height"]]),
			Address:     row[col["address"]],
		
			Balance:     row[col["balance"]],
		}

		key := fmt.Sprintf("%s-%d", rec.Address, rec.Height)
		old, ok := result[key]
		if ok {
			common.Log.Infof("duplicated key %s", key)
			common.Log.Infof("old: %v", old)
			common.Log.Infof("new: %v", rec)
		}

		result[key] = rec
	}
	//common.Log.Infof("block %d: %d %d", height,count, invalidCount)
	return result, start, end, nil
}


func ReadCSVDir(dir string) (map[string]*HolderCSVRecord, int, int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, 0, err
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".csv") {
			files = append(files, filepath.Join(dir, name))
		}
	}

	// 保证确定性顺序
	sort.Strings(files)

	start := 0xffffffff
	end := 0
	result := make(map[string]*HolderCSVRecord, 0)
	for _, path := range files {
		records, start1, end1, err := ReadCSVFile(path)
		if err != nil {
			common.Log.Errorf("read csv file %s failed, %v", path, err)
			continue
		}
		if start1 < start {
			start = start1
		}
		if end1 > end {
			end = end1
		}
		for k, v := range records {
			old, ok := result[k]
			if ok {
				common.Log.Infof("duplicated key %s", k)
				common.Log.Infof("old: %v", old)
				common.Log.Infof("new: %v", v)
			}

			result[k] = v
		}
	}

	return result, start, end, nil
}


func CompressCSVFile(srcCSV, dstGZ string) error {
	in, err := os.Open(srcCSV)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dstGZ)
	if err != nil {
		return err
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	defer gw.Close()

	// 流式拷贝，几乎不占内存
	_, err = io.Copy(gw, in)
	return err
}

func DecompressToTempCSV(srcGZ string) (string, func(), error) {
	in, err := os.Open(srcGZ)
	if err != nil {
		return "", nil, err
	}

	gr, err := gzip.NewReader(in)
	if err != nil {
		in.Close()
		return "", nil, err
	}

	tmp, err := os.CreateTemp("", "brc20_*.csv")
	if err != nil {
		gr.Close()
		in.Close()
		return "", nil, err
	}

	_, err = io.Copy(tmp, gr)

	// 关闭顺序很重要
	gr.Close()
	in.Close()
	tmp.Close()

	if err != nil {
		os.Remove(tmp.Name())
		return "", nil, err
	}

	cleanup := func() {
		_ = os.Remove(tmp.Name())
	}

	return tmp.Name(), cleanup, nil
}


func SplitCSVFile(
	srcCSV string,
	dstDir string,
	rowsPerFile int,
	prefix string,
) error {

	if rowsPerFile <= 0 {
		return fmt.Errorf("rowsPerFile must be > 0")
	}

	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	in, err := os.Open(srcCSV)
	if err != nil {
		return err
	}
	defer in.Close()

	reader := csv.NewReader(in)
	reader.ReuseRecord = true

	// ===== 关键修复点：header 深拷贝 =====
	rawHeader, err := reader.Read()
	if err != nil {
		return fmt.Errorf("read header failed: %w", err)
	}

	header := make([]string, len(rawHeader))
	copy(header, rawHeader)
	// ===================================

	var (
		fileIdx   = 0
		rowCount  = 0
		outFile   *os.File
		outWriter *csv.Writer
	)

	closeCurrent := func() {
		if outWriter != nil {
			outWriter.Flush()
		}
		if outFile != nil {
			outFile.Close()
		}
	}

	openNewFile := func() error {
		closeCurrent()

		fileIdx++
		rowCount = 0

		name := fmt.Sprintf("%s_%05d.csv", prefix, fileIdx)
		path := filepath.Join(dstDir, name)

		f, err := os.Create(path)
		if err != nil {
			return err
		}

		w := csv.NewWriter(f)

		// 写 header（现在是稳定的）
		if err := w.Write(header); err != nil {
			f.Close()
			return err
		}

		outFile = f
		outWriter = w
		return nil
	}

	for {
		record, err := reader.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			closeCurrent()
			return err
		}

		if outWriter == nil || rowCount >= rowsPerFile {
			if err := openNewFile(); err != nil {
				return err
			}
		}

		if err := outWriter.Write(record); err != nil {
			closeCurrent()
			return err
		}

		rowCount++
	}

	closeCurrent()
	return nil
}
