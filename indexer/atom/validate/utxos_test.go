package validate

import "testing"

func TestReadAtomUtxo950000CSV(t *testing.T) {
	testUtxoCSV(t, "./utxos/atom-utxos-950000.csv", "atom", 950000, 12932, 19114209, 27780330264199168, 1000)
}

func TestReadQuarkUtxo950000CSV(t *testing.T) {
	testUtxoCSV(t, "./utxos/quark-utxos-950000.csv", "quark", 950000, 158791, 9160533072, 28269509542674432, 20000)
}

func TestReadAllUtxo950000CSV(t *testing.T) {
	records, err := ReadUtxoCSV("./utxos/all-utxos-950000.csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 773049 {
		t.Fatalf("utxo count mismatch: got %d", len(records))
	}
	var actualTotal int64
	for _, record := range records {
		actualTotal += record.Amount
		if record.Height != 950000 {
			t.Fatalf("unexpected height %d", record.Height)
		}
	}
	if actualTotal != 38741982969 {
		t.Fatalf("utxo total mismatch: got %d", actualTotal)
	}
	if records[0].Ticker != "0" || records[0].UtxoId != 27946356145127424 || records[0].Amount != 1000 {
		t.Fatalf("unexpected first utxo: %+v", records[0])
	}
}

func testUtxoCSV(t *testing.T, path, ticker string, height, count int, total int64, firstUtxoId uint64, firstAmount int64) {
	t.Helper()

	records, err := ReadUtxoCSV(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != count {
		t.Fatalf("utxo count mismatch: got %d", len(records))
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
		t.Fatalf("utxo total mismatch: got %d", actualTotal)
	}
	if records[0].UtxoId != firstUtxoId || records[0].Amount != firstAmount {
		t.Fatalf("unexpected first utxo: %d %d", records[0].UtxoId, records[0].Amount)
	}
}
