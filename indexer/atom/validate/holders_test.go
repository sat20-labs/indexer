package validate

import "testing"

func TestReadAtomHolderCSV(t *testing.T) {
	testHolderCSV(t, "./holders/atom-holders-900000.csv", "atom", 900000, 8213, 19325316, "bc1pu62x0qzqn758srcmm0ctlxgum55a06am3njj3jgatkmyu9plmypsshzp45", 1151992)
}

func TestReadAtomHolder860000CSV(t *testing.T) {
	testHolderCSV(t, "./holders/atom-holders-860000.csv", "atom", 860000, 7680, 19330214, "bc1p3eze9y3krkxk848t0ph4d0y4mml22ht3z7g5snr8npdecrfkmuzsm433rk", 572773)
}

func TestReadQuarkHolder860000CSV(t *testing.T) {
	testHolderCSV(t, "./holders/quark-holders-860000.csv", "quark", 860000, 21993, 9696961667, "bc1p3dfge99g2ulp8ry7almgr64fmwqzfxqclgh5mca39rzl2lw3zvasavfyzu", 285969399)
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
