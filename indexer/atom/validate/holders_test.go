package validate

import "testing"

func TestReadAtomHolderCSV(t *testing.T) {
	testHolderCSV(t, "./holders/atom-holders-950000.csv", "atom", 950000, 8087, 19114209, "bc1pu62x0qzqn758srcmm0ctlxgum55a06am3njj3jgatkmyu9plmypsshzp45", 1151992)
}

func TestReadQuarkHolder950000CSV(t *testing.T) {
	testHolderCSV(t, "./holders/quark-holders-950000.csv", "quark", 950000, 56650, 9160533072, "bc1qt87mprj2fl6kw4leypz2vw0valfqndgcdutwgf", 114409682)
}

func testHolderCSV(t *testing.T, path, ticker string, height, count int, total int64, topAddress string, topAmount int64) {
	t.Helper()

	records, err := ReadHolderCSV(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != count {
		t.Fatalf("holder count mismatch: got %d", len(records))
	}
	var actualTotal int64
	for _, record := range records {
		actualTotal += record.Amount
		if record.Ticker != ticker {
			t.Fatalf("unexpected ticker %q", record.Ticker)
		}
		if record.Height != height {
			t.Fatalf("unexpected height %d", record.Height)
		}
	}
	if actualTotal != total {
		t.Fatalf("holder total mismatch: got %d", actualTotal)
	}
	if records[0].Address != topAddress || records[0].Amount != topAmount {
		t.Fatalf("unexpected top holder: %s %d", records[0].Address, records[0].Amount)
	}
}
