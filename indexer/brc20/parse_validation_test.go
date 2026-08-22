//go:build validation

package brc20

import (
	"fmt"
	"testing"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/brc20/validate"
)

func TestParseValidateHolderFileData(t *testing.T) {
	validateHolderData, err := validate.ReadBRC20HolderCSV("./validate/holders/holders_931177.csv")
	if err != nil {
		common.Log.Panicf("ReadBRC20HolderCSV failed, %v", err)
	}

	var startHeight, endHeight int
	startHeight = 0xffffffff

	tickers := make(map[string]bool)
	heightToHolderRecords := make(map[int]map[string]map[string]*validate.BRC20HolderCSVRecord)
	for _, record := range validateHolderData {
		tickerToHolders, ok := heightToHolderRecords[record.LastHeight]
		if !ok {
			tickerToHolders = make(map[string]map[string]*validate.BRC20HolderCSVRecord)
			heightToHolderRecords[record.LastHeight] = tickerToHolders
		}
		holders, ok := tickerToHolders[record.Token]
		if !ok {
			holders = make(map[string]*validate.BRC20HolderCSVRecord)
			tickerToHolders[record.Token] = holders
		}
		holders[record.Address] = record
		tickers[record.Token] = true

		if record.LastHeight > endHeight {
			endHeight = record.LastHeight
		}
		if record.LastHeight < startHeight {
			startHeight = record.LastHeight
		}
	}

	// 931177
	fmt.Printf("len %d, height %d-%d, %v\n", len(heightToHolderRecords), startHeight, endHeight, tickers)
}

func TestParseValidateHolderDir(t *testing.T) {
	var err error
	validateHolderData, err := validate.ReadBRC20HolderCSVDir("./validate/holders")
	if err != nil {
		common.Log.Panicf("ReadBRC20HolderCSVDir failed, %v", err)
	}

	var startHeight, endHeight int
	startHeight = 0xffffffff

	tickers := make(map[string]bool)
	heightToHolderRecords := make(map[int]map[string]map[string]*validate.BRC20HolderCSVRecord)
	for _, record := range validateHolderData {
		tickerToHolders, ok := heightToHolderRecords[record.LastHeight]
		if !ok {
			tickerToHolders = make(map[string]map[string]*validate.BRC20HolderCSVRecord)
			heightToHolderRecords[record.LastHeight] = tickerToHolders
		}
		holders, ok := tickerToHolders[record.Token]
		if !ok {
			holders = make(map[string]*validate.BRC20HolderCSVRecord)
			tickerToHolders[record.Token] = holders
		}
		holders[record.Address] = record
		tickers[record.Token] = true

		if record.LastHeight > endHeight {
			endHeight = record.LastHeight
		}
		if record.LastHeight < startHeight {
			startHeight = record.LastHeight
		}
	}

	// 928228-928300
	fmt.Printf("len %d, height %d-%d, %v\n", len(heightToHolderRecords), startHeight, endHeight, tickers)
}

func TestParseSplitFile(t *testing.T) {
	err := validate.SplitCSVFile("./validate/ordi.csv", "./validate/ordi", 40000, "ordi")
	if err != nil {
		t.Fatal(err)
	}
}

func TestFilterFile(t *testing.T) {
	err := validate.FilterCSVFile("./validate/dior_records.csv", "./validate/dior_records_2", 902394)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseValidateDir_history(t *testing.T) {
	var err error
	validateData, start, end, err := validate.ReadBRC20CSVDir("./validate/ordi")
	if err != nil {
		common.Log.Panicf("ReadBRC20CSVDir failed, %v", err)
	}

	fmt.Printf("len %d, height %d %d", len(validateData), start, end)
}

func TestParseValidateData_history(t *testing.T) {
	validateHolderData, start, end, err := validate.ReadBRC20CSV("./validate/pizza_records.csv")
	if err != nil {
		common.Log.Panicf("ReadBRC20CSV failed, %v", err)
	}

	fmt.Printf("len %d, height %d %d", len(validateHolderData), start, end)
}

func TestCompareValidateFile(t *testing.T) {

	validateData1, _, _, err := validate.ReadBRC20CSVDir("./validate/ordi")
	if err != nil {
		common.Log.Panicf("ReadBRC20CSVDir failed, %v", err)
	}

	validateData2, _, _, err := validate.ReadBRC20CSV("./validate/ordi_records.csv")
	if err != nil {
		common.Log.Panicf("ReadBRC20CSVDir failed, %v", err)
	}

	// validateData2, err := validate.ReadBRC20CSV("./validate/ordi.csv")
	// if err != nil {
	// 	common.Log.Panicf("ReadBRC20CSVDir failed, %v", err)
	// }

	diff1 := findDiffInMap(validateData1, validateData2)
	fmt.Printf("diff1 %d\n", len(diff1))
	for _, d := range diff1 {
		fmt.Printf("%v\n", validateData1[d])
	}

	diff2 := findDiffInMap(validateData2, validateData1)
	fmt.Printf("diff2 %d\n", len(diff2))
	for _, d := range diff2 {
		fmt.Printf("%v\n", validateData2[d])
	}

}

func TestParseValidateData_tickers(t *testing.T) {
	tickerAll, err := validate.ReadBRC20TickersCSV("./validate/ticker_all.csv")
	if err != nil {
		common.Log.Panicf("ReadBRC20TickersCSV failed, %v", err)
	}

	fmt.Printf("tickers %d", len(tickerAll))
}
