package validate

import "testing"

func TestReadAtomUtxo860000CSV(t *testing.T) {
	testUtxoCSV(t, "./utxos/atom-utxos-860000.csv", "atom", 860000, 12899, 19330214, 27780330264199168, 1000)
}

func TestReadAtomUtxo900000CSV(t *testing.T) {
	testUtxoCSV(t, "./utxos/atom-utxos-900000.csv", "atom", 900000, 13156, 19325316, 27780330264199168, 1000)
}

func TestReadQuarkUtxo860000CSV(t *testing.T) {
	testUtxoCSV(t, "./utxos/quark-utxos-860000.csv", "quark", 860000, 154128, 9696961667, 28265729728184320, 20000)
}

func TestReadQuarkUtxo900000CSV(t *testing.T) {
	testUtxoCSV(t, "./utxos/quark-utxos-900000.csv", "quark", 900000, 136522, 9372835953, 28269509542674432, 20000)
}

func TestReadAllUtxo860000CSV(t *testing.T) {
	records, err := ReadUtxoCSV("./utxos/all-utxos-860000.csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 832478 {
		t.Fatalf("utxo count mismatch: got %d", len(records))
	}
	var actualTotal int64
	for _, record := range records {
		actualTotal += record.Amount
		if record.Height != 860000 {
			t.Fatalf("unexpected height %d", record.Height)
		}
	}
	if actualTotal != 42655520242 {
		t.Fatalf("utxo total mismatch: got %d", actualTotal)
	}
	if records[0].Ticker != "0" || records[0].UtxoId != 27876742945046528 || records[0].Amount != 1000 {
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
