package atom

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

type checkpointSnapshot struct {
	height       int
	tickers      int64
	detail       map[string]map[string]string
	utxos        map[string][]int64
	holderCounts map[string]int
	holders      map[string]map[string]int64
}

func TestMainnetCheckpointAgainstOfficialSnapshot(t *testing.T) {
	path := os.Getenv("ATOM_CHECKPOINT_SNAPSHOT")
	if path == "" {
		t.Skip("set ATOM_CHECKPOINT_SNAPSHOT to validate a mainnet checkpoint against an official snapshot")
	}

	snapshot := readCheckpointSnapshot(t, path)
	checkpoint := mainnetCheckpoint[snapshot.height]
	if checkpoint == nil {
		t.Fatalf("no checkpoint for snapshot height %d", snapshot.height)
	}

	if checkpoint.TickerCount != 0 && snapshot.tickers != checkpoint.TickerCount {
		t.Fatalf("ticker count mismatch at %d: snapshot=%d checkpoint=%d", snapshot.height, snapshot.tickers, checkpoint.TickerCount)
	}
	if checkpoint.AssetUtxoCount != 0 {
		count := 0
		for _, rows := range snapshot.utxos {
			count += len(rows)
		}
		if count != checkpoint.AssetUtxoCount {
			t.Fatalf("asset utxo count mismatch at %d: snapshot=%d checkpoint=%d", snapshot.height, count, checkpoint.AssetUtxoCount)
		}
	}
	for _, name := range checkpoint.RejectedTicker {
		name = strings.ToLower(name)
		if snapshot.detail[name] != nil {
			t.Fatalf("rejected ticker %s exists at %d", name, snapshot.height)
		}
	}
	for name, expected := range checkpoint.Tickers {
		name = strings.ToLower(name)
		if expected.DeployHeight != 0 && snapshot.height < expected.DeployHeight {
			continue
		}
		actual := snapshot.detail[name]
		if actual == nil {
			t.Fatalf("missing ticker %s at %d", name, snapshot.height)
		}
		if expected.AtomicalId != "" && actual["atomical"] != expected.AtomicalId {
			t.Fatalf("ticker %s atomical mismatch at %d: snapshot=%s checkpoint=%s", name, snapshot.height, actual["atomical"], expected.AtomicalId)
		}
		// Official debug snapshots do not consistently expose deploy_height for
		// every ticker type. Presence and atomical id are snapshot-validated;
		// deploy heights should be checked from the atomical tx height.
		checkInt(t, snapshot.height, name, "minted_times", actual, expected.MintedTimes)
		checkInt(t, snapshot.height, name, "minted_amount", actual, expected.MintedAmount)
		checkInt(t, snapshot.height, name, "max_mints", actual, expected.MaxMints)
		if expected.HolderCount != 0 {
			count, ok := snapshot.holderCounts[name]
			if !ok {
				t.Fatalf("ticker %s holder count missing in snapshot at %d", name, snapshot.height)
			}
			if count != expected.HolderCount {
				t.Fatalf("ticker %s holder count mismatch at %d: snapshot=%d checkpoint=%d", name, snapshot.height, count, expected.HolderCount)
			}
		}
		for address, amount := range expected.Holders {
			actualAmount, ok := snapshot.holders[name][address]
			if !ok {
				t.Fatalf("ticker %s holder %s missing in snapshot at %d", name, address, snapshot.height)
			}
			if actualAmount != amount {
				t.Fatalf("ticker %s holder %s amount mismatch at %d: snapshot=%d checkpoint=%d", name, address, snapshot.height, actualAmount, amount)
			}
		}
		if expected.UtxoCount != 0 || expected.UtxoAmount != 0 {
			rows := snapshot.utxos[name]
			var amount int64
			for _, value := range rows {
				amount += value
			}
			if expected.UtxoCount != 0 && len(rows) != expected.UtxoCount {
				t.Fatalf("ticker %s utxo count mismatch at %d: snapshot=%d checkpoint=%d", name, snapshot.height, len(rows), expected.UtxoCount)
			}
			if expected.UtxoAmount != 0 && amount != expected.UtxoAmount {
				t.Fatalf("ticker %s utxo amount mismatch at %d: snapshot=%d checkpoint=%d", name, snapshot.height, amount, expected.UtxoAmount)
			}
		}
	}
}

func readCheckpointSnapshot(t *testing.T, path string) *checkpointSnapshot {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	result := &checkpointSnapshot{
		detail:       make(map[string]map[string]string),
		utxos:        make(map[string][]int64),
		holderCounts: make(map[string]int),
		holders:      make(map[string]map[string]int64),
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		fields := make(map[string]string)
		for _, part := range parts[1:] {
			key, value, ok := strings.Cut(part, "=")
			if ok {
				fields[key] = value
			}
		}
		switch parts[0] {
		case "status":
			result.height = parseRequiredInt(t, line, fields["height"])
			result.tickers = int64(parseRequiredInt(t, line, fields["tickers"]))
		case "ticker":
			name := strings.ToLower(fields["name"])
			if result.detail[name] == nil {
				result.detail[name] = make(map[string]string)
			}
			result.detail[name]["atomical"] = fields["atomical"]
		case "ticker_detail":
			name := strings.ToLower(fields["name"])
			if result.detail[name] == nil {
				result.detail[name] = make(map[string]string)
			}
			for key, value := range fields {
				result.detail[name][key] = value
			}
		case "utxo":
			name := strings.ToLower(fields["ticker"])
			result.utxos[name] = append(result.utxos[name], int64(parseRequiredInt(t, line, fields["amount"])))
		case "holder_count":
			name := strings.ToLower(fields["ticker"])
			result.holderCounts[name] = parseRequiredInt(t, line, fields["count"])
		case "holder":
			name := strings.ToLower(fields["ticker"])
			if result.holders[name] == nil {
				result.holders[name] = make(map[string]int64)
			}
			result.holders[name][fields["address"]] = int64(parseRequiredInt(t, line, fields["amount"]))
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if result.height == 0 {
		t.Fatalf("snapshot %s has no status height", path)
	}
	return result
}

func checkInt(t *testing.T, height int, ticker, field string, actual map[string]string, expected int64) {
	t.Helper()
	if expected == 0 {
		return
	}
	value := int64(parseRequiredInt(t, ticker+" "+field, actual[field]))
	if value != expected {
		t.Fatalf("ticker %s %s mismatch at %d: snapshot=%d checkpoint=%d", ticker, field, height, value, expected)
	}
}

func parseRequiredInt(t *testing.T, context, raw string) int {
	t.Helper()
	value, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("invalid integer in %s: %q", context, raw)
	}
	return value
}
