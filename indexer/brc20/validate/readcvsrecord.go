package validate

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
)


type BRC20CSVRecord struct {
	Ticker             string
	Type               string
	Valid              bool

	TxID               string
	Idx                int
	Vout               int
	Offset             int64

	InscriptionNumber  int64
	InscriptionID      string

	From               string
	To                 string

	Value            int64
	Fee                int64

	Amount             string
	OverallBalance     string
	TransferBalance    string
	AvailableBalance   string

	Height              int
	TxIdx               int
	BlockHash           string
	BlockTime           int64

	H                   int
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


func ReadBRC20CSV(path string) (map[string]*BRC20CSVRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	r.TrimLeadingSpace = true

	// 读取 header
	header, err := r.Read()
	if err != nil {
		return nil, err
	}

	col := make(map[string]int)
	for i, h := range header {
		h = strings.TrimPrefix(h, "\ufeff")
		col[h] = i
	}

	height := 790577
	invalidCount := 0
	count := 0
	result := make(map[string]*BRC20CSVRecord)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		h := parseInt32(row[col["height"]])

		if row[col["valid"]] == "0" {
			if h == height {
				invalidCount++
			}
			continue
		}

		rec := &BRC20CSVRecord{
			Ticker:   strings.ToLower(row[col["ticker"]]),
			Type:     row[col["type"]],
			Valid:    row[col["valid"]] == "1",

			TxID:     row[col["txid"]],
			Idx:      parseInt32(row[col["idx"]]),
			Vout:     parseInt32(row[col["vout"]]),
			Offset:   parseI64(row[col["offset"]]),

			InscriptionNumber: parseI64(row[col["inscriptionNumber"]]),
			InscriptionID:     row[col["inscriptionId"]],

			From:    row[col["from"]],
			To:      row[col["to"]],

			Value:   parseI64(row[col["satoshi"]]),
			Fee:     parseI64(row[col["fee"]]),

			Amount:           row[col["amount"]],
			OverallBalance:   row[col["overallBalance"]],
			TransferBalance:  row[col["transferBalance"]],
			AvailableBalance: row[col["availableBalance"]],

			Height:    h,
			TxIdx:     parseInt32(row[col["txidx"]]),
			BlockHash: row[col["blockhash"]],
			BlockTime: parseI64(row[col["blocktime"]]),

			H: parseInt32(row[col["h"]]),
		}

		if rec.Height == height {
			count++
		}

		utxoId := common.ToUtxoId(rec.Height, rec.TxIdx, rec.Vout)
		key := fmt.Sprintf("%d-%x-%d", rec.InscriptionNumber, utxoId, rec.Offset)

		old, ok := result[key]
		if ok {
			common.Log.Infof("duplicated key %s", key)
			common.Log.Infof("old: %v", old)
			common.Log.Infof("new: %v", rec)
		}

		result[key] = rec
	}
	common.Log.Infof("block %d: %d %d", height,count, invalidCount)
	return result, nil
}

// height 必须相同
func InsertByInscriptionNumber(records []*BRC20CSVRecord, newRec *BRC20CSVRecord) []*BRC20CSVRecord {
	// 二分查找插入位置
	idx := sort.Search(len(records), func(i int) bool {
		if records[i].TxIdx == newRec.TxIdx {
			if records[i].Vout == newRec.Vout {
				if newRec.Value == 0 && newRec.Offset == 0 {
					// 取消的transfer，vout超出tx的txOut数量，offset=0，value=0
					return true
				}
				if records[i].Offset == newRec.Offset {
					// 后面加入的放后面
					return true
				}
				return records[i].Offset > newRec.Offset
			}
			return records[i].Vout > newRec.Vout
		}
		return records[i].TxIdx > newRec.TxIdx
	})

	// 扩容一个位置
	records = append(records, nil)

	// 向后挪动
	copy(records[idx+1:], records[idx:])

	// 插入
	records[idx] = newRec

	return records
}

var ActionToInt = map[string]int {
	"inscribe-deploy": 0,
	"inscribe-mint": 1,
	"inscribe-transfer": 2,
	"transfer": 3,
	"transfer-cancel": 4,
}


type BRC20HolderCSVRecord struct {
	Address                  string
	OverallBalance           string
	TransferableBalance      string
	AvailableBalance         string
	AvailableBalanceSafe     string
	AvailableBalanceUnsafe   string
	LastHeight               int
	UpdatedAt                time.Time
	Token                    string
}

const holderTimeLayout = "2006/01/02 15:04"

func ReadBRC20HolderCSV(path string) ([]*BRC20HolderCSVRecord, error) {

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	r.TrimLeadingSpace = true

	// 读取 header
	header, err := r.Read()
	if err != nil {
		return nil, err
	}

	col := make(map[string]int)
	for i, h := range header {
		h = strings.TrimPrefix(h, "\ufeff")
		col[h] = i
	}

	result := make([]*BRC20HolderCSVRecord, 0)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		rec := &BRC20HolderCSVRecord{
			Address:                row[col["address"]],
			OverallBalance:         row[col["overall_balance"]],
			TransferableBalance:    row[col["transferable_balance"]],
			AvailableBalance:       row[col["available_balance"]],
			AvailableBalanceSafe:   row[col["available_balance_safe"]],
			AvailableBalanceUnsafe: row[col["available_balance_unsafe"]],
			LastHeight:             parseInt32(row[col["last_height"]]),
			Token:                  strings.ToLower(row[col["token"]]),
		}

		// 时间字段单独解析
		if s := row[col["updated_at"]]; s != "" {
			t, err := time.ParseInLocation(
				holderTimeLayout,
				s,
				time.Local,
			)
			if err != nil {
				//return nil, err
			}
			rec.UpdatedAt = t
		}

		result = append(result, rec)

	}

	return result, nil
}


func ReadBRC20HolderCSVDir(dir string) ([]*BRC20HolderCSVRecord, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
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

	result := make([]*BRC20HolderCSVRecord, 0)
	for _, path := range files {
		records, err := ReadBRC20HolderCSV(path)
		if err != nil {
			common.Log.Errorf("read csv file %s failed, %v", path, err)
			continue
		}
		result = append(result, records...)
	}

	return result, nil
}
