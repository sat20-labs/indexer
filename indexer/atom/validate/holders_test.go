package validate

import "testing"

func TestReadAtomHolderCSV(t *testing.T) {
	records, err := ReadHolderCSV("./holders/atom-holders-900000.csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 8213 {
		t.Fatalf("holder count mismatch: got %d", len(records))
	}
	var total int64
	for _, record := range records {
		total += record.Amount
		if record.Ticker != "atom" {
			t.Fatalf("unexpected ticker %q", record.Ticker)
		}
		if record.Height != 900000 {
			t.Fatalf("unexpected height %d", record.Height)
		}
	}
	if total != 19325316 {
		t.Fatalf("holder total mismatch: got %d", total)
	}
	if records[0].Address != "bc1pu62x0qzqn758srcmm0ctlxgum55a06am3njj3jgatkmyu9plmypsshzp45" || records[0].Amount != 1151992 {
		t.Fatalf("unexpected top holder: %s %d", records[0].Address, records[0].Amount)
	}
}
