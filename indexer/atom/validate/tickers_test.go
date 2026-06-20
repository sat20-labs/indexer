package validate

import "testing"

func TestReadTicker950000CSV(t *testing.T) {
	records, err := ReadTickerCSV("./tickers/tickers-950000.csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 706 {
		t.Fatalf("ticker count mismatch: got %d", len(records))
	}

	var utxoCount int
	var utxoAmount int64
	for _, record := range records {
		if record.Height != 950000 {
			t.Fatalf("unexpected height %d", record.Height)
		}
		utxoCount += record.UtxoCount
		utxoAmount += record.UtxoAmount
	}
	if utxoCount != 773049 || utxoAmount != 38741982969 {
		t.Fatalf("ticker totals mismatch: count=%d amount=%d", utxoCount, utxoAmount)
	}

	first := records[0]
	if first.Ticker != "0" ||
		first.AtomicalId != "74354f5abd480379ff5346c2258bb87510c75859f9623734792000d9ff9cef81i0" ||
		first.UtxoCount != 18 ||
		first.UtxoAmount != 20000 ||
		first.MintedTimes != 21 ||
		first.MintedAmount != 21000 ||
		first.MaxMints != 21 {
		t.Fatalf("unexpected first ticker: %+v", first)
	}
}
