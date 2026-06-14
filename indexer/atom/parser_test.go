package atom

import (
	"strings"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/fxamacker/cbor/v2"
	"github.com/sat20-labs/indexer/common"
)

func atomWitness(t *testing.T, op []byte, args map[string]any) []byte {
	t.Helper()
	return atomWitnessPayload(t, op, map[string]any{"args": args})
}

func atomWitnessPayload(t *testing.T, op []byte, value map[string]any) []byte {
	t.Helper()
	payload, err := cbor.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	script := append([]byte{0x20}, make([]byte, 32)...)
	script = append(script, txscript.OP_IF, 0x04, 'a', 't', 'o', 'm')
	script = append(script, op...)
	if len(payload) > 0xff {
		script = append(script, txscript.OP_PUSHDATA2, byte(len(payload)), byte(len(payload)>>8))
	} else if len(payload) >= txscript.OP_PUSHDATA1 {
		script = append(script, txscript.OP_PUSHDATA1, byte(len(payload)))
	} else {
		script = append(script, byte(len(payload)))
	}
	script = append(script, payload...)
	script = append(script, txscript.OP_ENDIF)
	return script
}

func TestParseOperationRequiresTaprootPubkeyPrefix(t *testing.T) {
	payload, err := cbor.Marshal(map[string]any{"args": map[string]any{"mint_ticker": "electron"}})
	if err != nil {
		t.Fatal(err)
	}
	script := []byte{txscript.OP_0, txscript.OP_IF, 0x04, 'a', 't', 'o', 'm', 0x03, 'd', 'm', 't', byte(len(payload))}
	script = append(script, payload...)
	script = append(script, txscript.OP_ENDIF)
	tx := &common.Transaction{
		Inputs: []*common.TxInput{testTxInput(1, strings.Repeat("a", 64), 0, 1000, 27001, script)},
	}
	if op := ParseOperation(tx, false); op != nil {
		t.Fatalf("expected unofficial witness script to be ignored, got %#v", op)
	}
}

func testTxInput(utxoId uint64, txid string, vout int, value int64, height int, witness []byte) *common.TxInput {
	output := common.NewTxOutputV2(value)
	output.UtxoId = utxoId
	output.OutPointStr = txid + ":0"
	if vout != 0 {
		output.OutPointStr = txid + ":1"
	}
	output.OutHeight = height
	return &common.TxInput{
		TxOutputV2: *output,
		Witness:    [][]byte{witness},
	}
}

func testTxOutput(utxoId, addressId uint64, txid string, vout int, value int64) *common.TxOutputV2 {
	output := common.NewTxOutputV2(value)
	output.UtxoId = utxoId
	output.AddressId = addressId
	output.OutPointStr = txid + ":0"
	if vout != 0 {
		output.OutPointStr = txid + ":1"
	}
	output.TxOutIndex = vout
	return output
}

func TestParseDirectFtOperation(t *testing.T) {
	commitTx := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tx := &common.Transaction{
		TxId:   "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Inputs: []*common.TxInput{testTxInput(1, commitTx, 0, 1000, 27001, atomWitness(t, []byte{0x02, 'f', 't'}, map[string]any{"request_ticker": "atomt", "bitworkc": "aaaa"}))},
	}
	op := ParseOperation(tx, false)
	if op == nil || op.Op != OpDirectFT {
		t.Fatalf("expected ft operation, got %#v", op)
	}
	if got := stringArg(op.Payload.Args, "request_ticker"); got != "atomt" {
		t.Fatalf("ticker mismatch: %s", got)
	}
}

func TestParseFtOperationWithBinaryMetadata(t *testing.T) {
	commitTx := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tx := &common.Transaction{
		TxId: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Inputs: []*common.TxInput{testTxInput(1, commitTx, 0, 1000, 27001,
			atomWitnessPayload(t, []byte{0x02, 'f', 't'}, map[string]any{
				"args": map[string]any{"request_ticker": "atomt", "bitworkc": "aaaa"},
				"image.png": map[string]any{
					"ct": "image/png",
					"d":  []byte{0x89, 0x50, 0x4e, 0x47},
				},
			}))},
	}
	op := ParseOperation(tx, false)
	if op == nil || op.Op != OpDirectFT {
		t.Fatalf("expected ft operation with binary metadata, got %#v", op)
	}
}

func TestParseMintRejectsBinaryMetaBeforeDensity(t *testing.T) {
	commitTx := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tx := &common.Transaction{
		TxId: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Inputs: []*common.TxInput{testTxInput(1, commitTx, 0, 1000, 27001,
			atomWitnessPayload(t, []byte{0x03, 'd', 'f', 't'}, map[string]any{
				"args": map[string]any{
					"request_ticker": "rekt",
					"bitworkc":       "aaaa",
					"mint_height":    uint64(27010),
					"mint_amount":    uint64(777),
					"max_mints":      uint64(1337),
				},
				"meta": map[string]any{
					"rekt.json": map[string]any{
						"$ct": "application/json",
						"$b":  []byte("{}"),
					},
				},
			}))},
	}
	if op := ParseOperation(tx, false); op != nil {
		t.Fatalf("expected binary meta mint payload to be rejected before density, got %#v", op)
	}
	if op := ParseOperation(tx, true); op != nil {
		t.Fatalf("expected binary meta mint payload to be rejected after density, got %#v", op)
	}
}

func TestBitworkHelpersMatchOfficialVectors(t *testing.T) {
	tests := []struct {
		base   string
		target int64
		want   string
	}{
		{"", 64, "0000"},
		{"a", 65, "a000.1"},
		{"abcd", 80, "abcd0"},
		{"abcd", 83, "abcd0.3"},
		{"0123456789abcdef", 129, "01234567.1"},
		{"abcdefe", 33000, "abcdefe0000000000000000000000000.8"},
	}
	for _, test := range tests {
		got, ok := deriveBitworkPrefix(test.base, test.target)
		if !ok || got != test.want {
			t.Fatalf("deriveBitworkPrefix(%q,%d)=%q,%v want %q", test.base, test.target, got, ok, test.want)
		}
	}
	got, ok := calculateExpectedBitwork("888888888888", 49995, 3333, 1, 64)
	if !ok || got != "8888.15" {
		t.Fatalf("expected bitwork mismatch: %q %v", got, ok)
	}
	if isPerpetualBitworkMatch("8888888888888888888888888888888888888888888888888888888888888888", "888888888888", 49995, 3333, 1, 64, false) {
		t.Fatalf("expected rollover-disabled perpetual bitwork to fail")
	}
	if !isPerpetualBitworkMatch("8888888888888888888888888888888888888888888888888888888888888888", "888888888888", 49995, 3333, 1, 64, true) {
		t.Fatalf("expected rollover-enabled perpetual bitwork to pass")
	}
}

func TestInvalidTickerAndImmutableAreRejected(t *testing.T) {
	idx := NewIndexer(nil, &chaincfg.TestNet4Params)
	commitTx := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	revealTx := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	idx.UpdateTransfer(&common.Block{
		Height: 27010,
		Transactions: []*common.Transaction{{
			TxId: revealTx,
			Inputs: []*common.TxInput{testTxInput(1, commitTx, 0, 1000, 27008,
				atomWitness(t, []byte{0x02, 'f', 't'}, map[string]any{"request_ticker": "bad-ticker", "bitworkc": "aaaa"}))},
			Outputs: []*common.TxOutputV2{testTxOutput(2, 100, revealTx, 0, 1000)},
		}, {
			TxId: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Inputs: []*common.TxInput{testTxInput(3, commitTx, 0, 1000, 27008,
				atomWitness(t, []byte{0x02, 'f', 't'}, map[string]any{"request_ticker": "immutable", "bitworkc": "aaaa", "i": true}))},
			Outputs: []*common.TxOutputV2{testTxOutput(4, 100, revealTx, 0, 1000)},
		}, {
			TxId: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			Inputs: []*common.TxInput{testTxInput(5, commitTx, 0, 1000, 27008,
				atomWitness(t, []byte{0x02, 'f', 't'}, map[string]any{"request_ticker": "shortpow", "bitworkc": "aaa"}))},
			Outputs: []*common.TxOutputV2{testTxOutput(6, 100, revealTx, 0, 1000)},
		}},
	})
	if idx.GetTicker("bad-ticker") != nil {
		t.Fatalf("invalid ticker was accepted")
	}
	if idx.GetTicker("immutable") != nil {
		t.Fatalf("immutable ft was accepted")
	}
	if idx.GetTicker("shortpow") != nil {
		t.Fatalf("short bitwork ticker was accepted")
	}
}

func TestDirectFtMintAndTransfer(t *testing.T) {
	idx := NewIndexer(nil, &chaincfg.TestNet4Params)
	commitTx := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	revealTx := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	block := &common.Block{
		Height: 27010,
		Transactions: []*common.Transaction{{
			TxId: revealTx,
			Inputs: []*common.TxInput{testTxInput(1, commitTx, 0, 1000, 27008,
				atomWitness(t, []byte{0x02, 'f', 't'}, map[string]any{"request_ticker": "atomt", "bitworkc": "aaaa"}))},
			Outputs: []*common.TxOutputV2{testTxOutput(2, 100, revealTx, 0, 1000)},
		}},
	}
	idx.UpdateTransfer(block)
	ticker := idx.GetTicker("atomt")
	if ticker == nil || ticker.MintedAmount != 1000 || ticker.MintedTimes != 1 {
		t.Fatalf("unexpected ticker: %#v", ticker)
	}
	if got := idx.GetUtxoAssets(2)["atomt"]; got != 1000 {
		t.Fatalf("minted utxo amount mismatch: %d", got)
	}
	mintOffsets := idx.GetAssetsWithUtxo(2)["atomt"]
	if got := mintOffsets.Size(); got != 1000 {
		t.Fatalf("minted utxo offset size mismatch: %d", got)
	}

	spendTx := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	idx.UpdateTransfer(&common.Block{
		Height: 27011,
		Transactions: []*common.Transaction{{
			TxId:    spendTx,
			Inputs:  []*common.TxInput{testTxInput(2, revealTx, 0, 1000, 27010, nil)},
			Outputs: []*common.TxOutputV2{testTxOutput(3, 101, spendTx, 0, 400), testTxOutput(4, 102, spendTx, 1, 600)},
		}},
	})
	if got := idx.GetUtxoAssets(3)["atomt"]; got != 400 {
		t.Fatalf("first transfer amount mismatch: %d", got)
	}
	if got := idx.GetUtxoAssets(4)["atomt"]; got != 600 {
		t.Fatalf("second transfer amount mismatch: %d", got)
	}
	transferOffsets := idx.GetAssetsWithUtxo(4)["atomt"]
	if got := transferOffsets.Size(); got != 600 {
		t.Fatalf("second transfer offset size mismatch: %d", got)
	}
}

func TestRegularTransferFallbackRestartsAtOutputZero(t *testing.T) {
	idx := NewIndexer(nil, &chaincfg.TestNet4Params)
	idx.heights.CustomColoring = 999999
	idx.addTicker(&Ticker{Id: 0, Name: "a", DisplayName: "a"})
	idx.addTicker(&Ticker{Id: 1, Name: "b", DisplayName: "b"})
	idx.addUtxoBalanceInMemory(&UtxoBalance{UtxoId: 1, AddressId: 10, Outpoint: "txa:0", AtomicalId: "a", Ticker: "a", Amount: 600})
	idx.addUtxoBalanceInMemory(&UtxoBalance{UtxoId: 2, AddressId: 10, Outpoint: "txb:0", AtomicalId: "b", Ticker: "b", Amount: 600})
	txid := "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	idx.UpdateTransfer(&common.Block{
		Height: 27020,
		Transactions: []*common.Transaction{{
			TxId: txid,
			Inputs: []*common.TxInput{
				testTxInput(1, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 0, 600, 27010, nil),
				testTxInput(2, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 0, 600, 27010, nil),
			},
			Outputs: []*common.TxOutputV2{
				testTxOutput(3, 11, txid, 0, 600),
				testTxOutput(4, 12, txid, 1, 700),
			},
		}},
	})
	if got := idx.GetUtxoAssets(3)["a"]; got != 600 {
		t.Fatalf("asset a should fallback to output 0, got %d", got)
	}
	if got := idx.GetUtxoAssets(3)["b"]; got != 600 {
		t.Fatalf("asset b should fallback to output 0, got %d", got)
	}
	if got := idx.GetUtxoAssets(4)["b"]; got != 0 {
		t.Fatalf("asset b should not remain on output 1 after fallback, got %d", got)
	}
}

func TestRegularTransferFifoSortsAtomicalIdsByLocationBytes(t *testing.T) {
	idx := NewIndexer(nil, &chaincfg.TestNet4Params)
	idx.heights.CustomColoring = 999999
	atomId := "56a8702bab3d2405eb9a356fd0725ca112a93a8efd1ecca06c6085e7278f0341i0"
	pepeId := "9ba68637ba32edb6370bebceaac3df4341180cbf7bac210741b12a679692d716i0"
	idx.addTicker(&Ticker{Id: 0, AtomicalId: atomId, Name: "atom", DisplayName: "atom"})
	idx.addTicker(&Ticker{Id: 1, AtomicalId: pepeId, Name: "pepe", DisplayName: "pepe"})
	idx.addUtxoBalanceInMemory(&UtxoBalance{UtxoId: 1, AddressId: 10, Outpoint: "txa:0", AtomicalId: atomId, Ticker: "atom", Amount: 2000})
	idx.addUtxoBalanceInMemory(&UtxoBalance{UtxoId: 1, AddressId: 10, Outpoint: "txa:0", AtomicalId: pepeId, Ticker: "pepe", Amount: 2000})
	txid := "779577f609f5b7dd399e9e0bc9d55a7f4c19a57c6ae30bea7ee2d4fa5880273a"
	idx.UpdateTransfer(&common.Block{
		Height: 27020,
		Transactions: []*common.Transaction{{
			TxId: txid,
			Inputs: []*common.TxInput{
				testTxInput(1, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 0, 4000, 27010, nil),
			},
			Outputs: []*common.TxOutputV2{
				testTxOutput(2, 11, txid, 0, 800),
				testTxOutput(3, 12, txid, 1, 1200),
				testTxOutput(4, 11, txid, 2, 600),
				testTxOutput(5, 11, txid, 3, 600),
				testTxOutput(6, 12, txid, 4, 800),
				testTxOutput(7, 12, txid, 5, 2496),
			},
		}},
	})
	if got := idx.GetUtxoAssets(2)["pepe"]; got != 800 {
		t.Fatalf("pepe should be first by location bytes and color output 0, got %d", got)
	}
	if got := idx.GetUtxoAssets(3)["pepe"]; got != 1200 {
		t.Fatalf("pepe should be first by location bytes and color output 1, got %d", got)
	}
	if got := idx.GetUtxoAssets(4)["atom"]; got != 600 {
		t.Fatalf("atom should color output 2 after pepe, got %d", got)
	}
	if got := idx.GetUtxoAssets(5)["atom"]; got != 600 {
		t.Fatalf("atom should color output 3 after pepe, got %d", got)
	}
	if got := idx.GetUtxoAssets(6)["atom"]; got != 800 {
		t.Fatalf("atom should color output 4 after pepe, got %d", got)
	}
	if got := idx.GetUtxoAssets(2)["atom"]; got != 0 {
		t.Fatalf("atom should not color output 0, got %d", got)
	}
}

func TestRegularTransferKeepsCleanOutputsAndBurnsRemainder(t *testing.T) {
	idx := NewIndexer(nil, &chaincfg.TestNet4Params)
	idx.heights.CustomColoring = 999999
	idx.addTicker(&Ticker{Id: 0, Name: "burn", DisplayName: "burn"})
	idx.addUtxoBalanceInMemory(&UtxoBalance{UtxoId: 1, AddressId: 10, Outpoint: "txa:0", AtomicalId: "burn", Ticker: "burn", Amount: 1000})
	txid := "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	idx.UpdateTransfer(&common.Block{
		Height: 27020,
		Transactions: []*common.Transaction{{
			TxId:   txid,
			Inputs: []*common.TxInput{testTxInput(1, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 0, 1000, 27010, nil)},
			Outputs: []*common.TxOutputV2{
				testTxOutput(2, 11, txid, 0, 600),
				testTxOutput(3, 12, txid, 1, 700),
			},
		}},
	})
	if got := idx.GetUtxoAssets(2)["burn"]; got != 600 {
		t.Fatalf("clean output should keep 600 before remainder is burned, got %d", got)
	}
	if got := idx.GetUtxoAssets(3)["burn"]; got != 0 {
		t.Fatalf("oversized output should not receive partially colored value before custom coloring, got %d", got)
	}
}

func TestSubtractKeepsHolderTouchedChangedAfterBackup(t *testing.T) {
	idx := NewIndexer(nil, &chaincfg.TestNet4Params)
	idx.addTicker(&Ticker{Id: 0, Name: "atom", DisplayName: "atom"})
	idx.addUtxoBalanceInMemory(&UtxoBalance{UtxoId: 1, AddressId: 10, Outpoint: "txa:0", AtomicalId: "atom", Ticker: "atom", Amount: 1000})
	idx.tickerIdAdded[0] = "atom"
	idx.mintsAdded = append(idx.mintsAdded, &MintInfo{Id: 1, AtomicalId: "old"})
	idx.actionsAdded = append(idx.actionsAdded, &ActionHistory{Id: 1, AtomicalId: "old"})

	backup := idx.Clone(nil)
	idx.removeUtxoBalanceInMemory(&UtxoBalance{UtxoId: 1, AddressId: 10, Outpoint: "txa:0", AtomicalId: "atom", Ticker: "atom", Amount: 1000})
	idx.addUtxoBalanceInMemory(&UtxoBalance{UtxoId: 2, AddressId: 10, Outpoint: "txb:0", AtomicalId: "atom", Ticker: "atom", Amount: 600})
	idx.tickerIdAdded[0] = "atom-new"
	idx.mintsAdded = []*MintInfo{{Id: 2, AtomicalId: "new"}}
	idx.actionsAdded = []*ActionHistory{{Id: 2, AtomicalId: "new"}}

	idx.Subtract(backup)

	holderKey := GetTickerHolderKey("atom", 10)
	if got := idx.holderTouched[holderKey]; got != 600 {
		t.Fatalf("later holder update should remain pending after subtract, got %d", got)
	}
	if got := idx.tickerIdAdded[0]; got != "atom-new" {
		t.Fatalf("later ticker id update should remain pending after subtract, got %s", got)
	}
	if len(idx.mintsAdded) != 1 || idx.mintsAdded[0].Id != 2 {
		t.Fatalf("later mint should remain pending after subtract: %+v", idx.mintsAdded)
	}
	if len(idx.actionsAdded) != 1 || idx.actionsAdded[0].Id != 2 {
		t.Fatalf("later action should remain pending after subtract: %+v", idx.actionsAdded)
	}
}

func TestAddUtxoBalanceReplacesExistingIndexes(t *testing.T) {
	idx := NewIndexer(nil, &chaincfg.TestNet4Params)
	idx.addTicker(&Ticker{Id: 0, Name: "atom", DisplayName: "atom"})

	first := &UtxoBalance{UtxoId: 1, AddressId: 10, Outpoint: "txa:0", AtomicalId: "atomid", Ticker: "atom", Amount: 1000}
	second := &UtxoBalance{UtxoId: 1, AddressId: 10, Outpoint: "txa:0", AtomicalId: "atomid", Ticker: "atom", Amount: 1000}
	idx.addUtxoBalanceInMemory(first)
	idx.addUtxoBalanceInMemory(second)

	count, amount := idx.tickerUtxoSummaryLocked("atom")
	if count != 1 || amount != 1000 {
		t.Fatalf("duplicate add should replace existing balance, got count=%d amount=%d", count, amount)
	}
	if got := idx.tickerUtxos["atom"][1]; got != 1000 {
		t.Fatalf("ticker utxo helper should not double count, got %d", got)
	}
	if got := idx.tickerHolders["atom"][10]; got != 1000 {
		t.Fatalf("holder helper should not double count, got %d", got)
	}

	idx.removeUtxoBalanceInMemory(second)
	count, amount = idx.tickerUtxoSummaryLocked("atom")
	if count != 0 || amount != 0 {
		t.Fatalf("remove after duplicate add should clear balance, got count=%d amount=%d", count, amount)
	}
	if got := idx.tickerUtxos["atom"][1]; got != 0 {
		t.Fatalf("ticker utxo helper should be cleared, got %d", got)
	}
	if got := idx.tickerHolders["atom"][10]; got != 0 {
		t.Fatalf("holder helper should be cleared, got %d", got)
	}
}

func TestRegularTransferAfterCustomColoringBurnsRemainderWithoutFallback(t *testing.T) {
	idx := NewIndexer(nil, &chaincfg.TestNet4Params)
	idx.heights.CustomColoring = 27000
	dragonId := "dc0038f5313f5fbbcfc51aaab7370e43507bdc661760f55ba634aefb5ad15c57i0"
	sophonId := "360533d31e6f3c535acf7a70686ab42cf477b3f7ceaf12ab1d30be218b1726a9i0"
	idx.addTicker(&Ticker{Id: 0, AtomicalId: dragonId, Name: "dragon", DisplayName: "dragon"})
	idx.addTicker(&Ticker{Id: 1, AtomicalId: sophonId, Name: "sophon", DisplayName: "sophon"})
	idx.addUtxoBalanceInMemory(&UtxoBalance{UtxoId: 1, AddressId: 10, Outpoint: "txa:0", AtomicalId: dragonId, Ticker: "dragon", Amount: 910520})
	idx.addUtxoBalanceInMemory(&UtxoBalance{UtxoId: 2, AddressId: 10, Outpoint: "txb:0", AtomicalId: sophonId, Ticker: "sophon", Amount: 699140})
	txid := "4ac9d8494dd0cd5c3e92a2e7b760a03dcb23d9d9996c6487f682bff4e60bbd28"
	idx.UpdateTransfer(&common.Block{
		Height: 27020,
		Transactions: []*common.Transaction{{
			TxId: txid,
			Inputs: []*common.TxInput{
				testTxInput(1, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 0, 910520, 27010, nil),
				testTxInput(2, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 0, 699140, 27010, nil),
			},
			Outputs: []*common.TxOutputV2{
				testTxOutput(3, 11, txid, 0, 1200),
				testTxOutput(4, 11, txid, 1, 546),
				testTxOutput(5, 11, txid, 2, 908774),
				testTxOutput(6, 12, txid, 3, 600),
				testTxOutput(7, 12, txid, 4, 600),
				testTxOutput(8, 12, txid, 5, 66684),
			},
		}},
	})
	if got := idx.GetUtxoAssets(3)["dragon"]; got != 1200 {
		t.Fatalf("dragon should color output 0, got %d", got)
	}
	if got := idx.GetUtxoAssets(5)["dragon"]; got != 908774 {
		t.Fatalf("dragon should color output 2, got %d", got)
	}
	if got := idx.GetUtxoAssets(3)["sophon"]; got != 0 {
		t.Fatalf("sophon should not fallback to output 0, got %d", got)
	}
	if got := idx.GetUtxoAssets(6)["sophon"]; got != 600 {
		t.Fatalf("sophon should color output 3 after dragon, got %d", got)
	}
	if got := idx.GetUtxoAssets(8)["sophon"]; got != 66684 {
		t.Fatalf("sophon should burn remainder after output 5, got %d", got)
	}
}

func TestRegularTransferSkipsOpFalseOpReturnOutputs(t *testing.T) {
	idx := NewIndexer(nil, &chaincfg.TestNet4Params)
	idx.heights.CustomColoring = 27000
	idx.addTicker(&Ticker{Id: 0, Name: "skip", DisplayName: "skip"})
	idx.addUtxoBalanceInMemory(&UtxoBalance{UtxoId: 1, AddressId: 10, Outpoint: "txa:0", AtomicalId: "skipid", Ticker: "skip", Amount: 1000})
	txid := "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	opFalseReturn := testTxOutput(2, 11, txid, 0, 700)
	opFalseReturn.OutValue.PkScript = []byte{txscript.OP_FALSE, txscript.OP_RETURN}
	idx.UpdateTransfer(&common.Block{
		Height: 27020,
		Transactions: []*common.Transaction{{
			TxId:   txid,
			Inputs: []*common.TxInput{testTxInput(1, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 0, 1000, 27010, nil)},
			Outputs: []*common.TxOutputV2{
				opFalseReturn,
				testTxOutput(3, 12, txid, 1, 600),
				testTxOutput(4, 13, txid, 2, 700),
			},
		}},
	})
	if got := idx.GetUtxoAssets(2)["skip"]; got != 0 {
		t.Fatalf("OP_FALSE OP_RETURN output should be skipped, got %d", got)
	}
	if got := idx.GetUtxoAssets(3)["skip"]; got != 600 {
		t.Fatalf("first spendable output should receive 600, got %d", got)
	}
	if got := idx.GetUtxoAssets(4)["skip"]; got != 400 {
		t.Fatalf("second spendable output should receive remaining 400, got %d", got)
	}
}

func TestCustomColorBeforeActivationUsesRegularFallback(t *testing.T) {
	idx := NewIndexer(nil, &chaincfg.TestNet4Params)
	idx.heights.CustomColoring = 999999
	idx.addTicker(&Ticker{Id: 0, Name: "early", DisplayName: "early"})
	idx.addUtxoBalanceInMemory(&UtxoBalance{UtxoId: 1, AddressId: 10, Outpoint: "txa:0", AtomicalId: "earlyid", Ticker: "early", Amount: 1000})
	txid := "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	idx.UpdateTransfer(&common.Block{
		Height: 27020,
		Transactions: []*common.Transaction{{
			TxId: txid,
			Inputs: []*common.TxInput{testTxInput(1, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 0, 1000, 27010,
				atomWitnessPayload(t, []byte{0x01, 'z'}, map[string]any{"earlyid": map[string]any{"1": uint64(1000)}}))},
			Outputs: []*common.TxOutputV2{
				testTxOutput(2, 11, txid, 0, 600),
				testTxOutput(3, 12, txid, 1, 700),
			},
		}},
	})
	if got := idx.GetUtxoAssets(2)["early"]; got != 600 {
		t.Fatalf("pre-activation z should use regular fallback and keep 600, got %d", got)
	}
	if got := idx.GetUtxoAssets(3)["early"]; got != 0 {
		t.Fatalf("pre-activation z should not use custom payload, got %d", got)
	}
}

func TestCustomColorNestedPayloadAssignsPartialOutputs(t *testing.T) {
	idx := NewIndexer(nil, &chaincfg.TestNet4Params)
	idx.heights.CustomColoring = 27000
	idx.addTicker(&Ticker{Id: 0, Name: "custom", DisplayName: "custom"})
	idx.addUtxoBalanceInMemory(&UtxoBalance{UtxoId: 1, AddressId: 10, Outpoint: "txa:0", AtomicalId: "customid", Ticker: "custom", Amount: 1000})
	txid := "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	idx.UpdateTransfer(&common.Block{
		Height: 27020,
		Transactions: []*common.Transaction{{
			TxId: txid,
			Inputs: []*common.TxInput{testTxInput(1, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 0, 1000, 27010,
				atomWitnessPayload(t, []byte{0x01, 'z'}, map[string]any{"customid": map[any]any{"1": uint64(500), "2": uint64(500)}}))},
			Outputs: []*common.TxOutputV2{
				testTxOutput(2, 11, txid, 0, 600),
				testTxOutput(3, 12, txid, 1, 700),
				testTxOutput(4, 13, txid, 2, 700),
			},
		}},
	})
	if got := idx.GetUtxoAssets(2)["custom"]; got != 0 {
		t.Fatalf("custom payload should skip output 0, got %d", got)
	}
	if got := idx.GetUtxoAssets(3)["custom"]; got != 500 {
		t.Fatalf("custom payload should color output 1 with 500, got %d", got)
	}
	if got := idx.GetUtxoAssets(4)["custom"]; got != 500 {
		t.Fatalf("custom payload should color output 2 with 500, got %d", got)
	}
}

func TestCustomColorFromNonZeroInputAppliesWhenPayloadNamesAtomical(t *testing.T) {
	idx := NewIndexer(nil, &chaincfg.TestNet4Params)
	idx.heights.CustomColoring = 27000
	idx.addTicker(&Ticker{Id: 0, Name: "pc", DisplayName: "pc"})
	idx.addUtxoBalanceInMemory(&UtxoBalance{UtxoId: 1, AddressId: 10, Outpoint: "txa:0", AtomicalId: "pcid", Ticker: "pc", Amount: 333})
	txid := "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	idx.UpdateTransfer(&common.Block{
		Height: 27020,
		Transactions: []*common.Transaction{{
			TxId: txid,
			Inputs: []*common.TxInput{
				testTxInput(1, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 0, 333, 27010, nil),
				testTxInput(5, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 0, 1000, 27010,
					atomWitnessPayload(t, []byte{0x01, 'z'}, map[string]any{"pcid": map[string]any{"1": uint64(333)}})),
			},
			Outputs: []*common.TxOutputV2{
				testTxOutput(2, 11, txid, 0, 546),
				testTxOutput(3, 12, txid, 1, 546),
			},
		}},
	})
	if got := idx.GetUtxoAssets(2)["pc"]; got != 0 {
		t.Fatalf("custom op from input 1 should skip output 0, got %d", got)
	}
	if got := idx.GetUtxoAssets(3)["pc"]; got != 333 {
		t.Fatalf("custom op from input 1 should color output 1, got %d", got)
	}
}

func TestCustomColorAppliesToAtomicalSpentByNonZeroOpInput(t *testing.T) {
	idx := NewIndexer(nil, &chaincfg.TestNet4Params)
	idx.heights.CustomColoring = 27000
	atomicalId := "536737aadfaffa17233bca342be2571e14916f6a29003ff4766d515283e68e90i0"
	idx.addTicker(&Ticker{Id: 0, AtomicalId: atomicalId, Name: "electron", DisplayName: "electron"})
	idx.addUtxoBalanceInMemory(&UtxoBalance{UtxoId: 5, AddressId: 10, Outpoint: "txb:0", AtomicalId: atomicalId, Ticker: "electron", Amount: 546})
	txid := "bdd01faeb5531bf00dd392565bccf05fec3534c37a8779a01a2cf520a818885e"
	idx.UpdateTransfer(&common.Block{
		Height: 27020,
		Transactions: []*common.Transaction{{
			TxId: txid,
			Inputs: []*common.TxInput{
				testTxInput(1, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 0, 333, 27010, nil),
				testTxInput(5, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 0, 546, 27010,
					atomWitnessPayload(t, []byte{0x01, 'z'}, map[string]any{atomicalId: map[string]any{"1": uint64(546)}})),
			},
			Outputs: []*common.TxOutputV2{
				testTxOutput(2, 11, txid, 0, 546),
				testTxOutput(3, 11, txid, 1, 546),
			},
		}},
	})
	if got := idx.GetUtxoAssets(2)["electron"]; got != 0 {
		t.Fatalf("custom op input asset should skip output 0, got %d", got)
	}
	if got := idx.GetUtxoAssets(3)["electron"]; got != 546 {
		t.Fatalf("custom op input asset should color output 1, got %d", got)
	}
}

func TestSplitFromNonZeroInputUsesRegularColoring(t *testing.T) {
	idx := NewIndexer(nil, &chaincfg.TestNet4Params)
	idx.heights.CustomColoring = 999999
	idx.addTicker(&Ticker{Id: 0, Name: "split", DisplayName: "split"})
	idx.addUtxoBalanceInMemory(&UtxoBalance{UtxoId: 1, AddressId: 10, Outpoint: "txa:0", AtomicalId: "splitid", Ticker: "split", Amount: 1000})
	txid := "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	idx.UpdateTransfer(&common.Block{
		Height: 27020,
		Transactions: []*common.Transaction{{
			TxId: txid,
			Inputs: []*common.TxInput{
				testTxInput(1, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 0, 1000, 27010, nil),
				testTxInput(5, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 0, 1000, 27010,
					atomWitnessPayload(t, []byte{0x01, 'y'}, map[string]any{"splitid": uint64(600)})),
			},
			Outputs: []*common.TxOutputV2{
				testTxOutput(2, 11, txid, 0, 600),
				testTxOutput(3, 12, txid, 1, 700),
			},
		}},
	})
	if got := idx.GetUtxoAssets(2)["split"]; got != 600 {
		t.Fatalf("split op from input 1 should fall back to regular coloring, got %d", got)
	}
	if got := idx.GetUtxoAssets(3)["split"]; got != 0 {
		t.Fatalf("split op from input 1 should not skip output 0 and color output 1, got %d", got)
	}
}

func TestDftDeployAndMint(t *testing.T) {
	idx := NewIndexer(nil, &chaincfg.TestNet4Params)
	deployCommit := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	deployTx := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	idx.UpdateTransfer(&common.Block{
		Height: 27010,
		Transactions: []*common.Transaction{{
			TxId: deployTx,
			Inputs: []*common.TxInput{testTxInput(1, deployCommit, 0, 1000, 27008,
				atomWitness(t, []byte{0x03, 'd', 'f', 't'}, map[string]any{
					"request_ticker": "dftx",
					"bitworkc":       "aaaa",
					"mint_height":    uint64(27010),
					"mint_amount":    uint64(600),
					"max_mints":      uint64(2),
				}))},
			Outputs: []*common.TxOutputV2{testTxOutput(2, 100, deployTx, 0, 546)},
		}},
	})
	if ticker := idx.GetTicker("dftx"); ticker == nil || ticker.MintAmount != 600 {
		t.Fatalf("unexpected dft ticker: %#v", ticker)
	}

	mintCommit := "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	mintTx := "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	idx.UpdateTransfer(&common.Block{
		Height: 27011,
		Transactions: []*common.Transaction{{
			TxId: mintTx,
			Inputs: []*common.TxInput{testTxInput(3, mintCommit, 0, 1000, 27010,
				atomWitness(t, []byte{0x03, 'd', 'm', 't'}, map[string]any{"mint_ticker": "dftx"}))},
			Outputs: []*common.TxOutputV2{testTxOutput(4, 101, mintTx, 0, 600)},
		}},
	})
	if got := idx.GetUtxoAssets(4)["dftx"]; got != 600 {
		t.Fatalf("dft mint amount mismatch: %d", got)
	}

	upperMintTx := "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeef"
	idx.UpdateTransfer(&common.Block{
		Height: 27012,
		Transactions: []*common.Transaction{{
			TxId: upperMintTx,
			Inputs: []*common.TxInput{testTxInput(9, mintCommit, 0, 1000, 27010,
				atomWitness(t, []byte{0x03, 'd', 'm', 't'}, map[string]any{"mint_ticker": "DFTX"}))},
			Outputs: []*common.TxOutputV2{testTxOutput(10, 104, upperMintTx, 0, 600)},
		}},
	})
	if got := idx.GetUtxoAssets(10)["dftx"]; got != 0 {
		t.Fatalf("uppercase dft mint ticker should be rejected, got %d", got)
	}

	oldCommit := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	oldMintTx := "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	idx.UpdateTransfer(&common.Block{
		Height: 27150,
		Transactions: []*common.Transaction{{
			TxId: oldMintTx,
			Inputs: []*common.TxInput{testTxInput(5, oldCommit, 0, 1000, 27010,
				atomWitness(t, []byte{0x03, 'd', 'm', 't'}, map[string]any{"mint_ticker": "dftx"}))},
			Outputs: []*common.TxOutputV2{testTxOutput(6, 102, oldMintTx, 0, 600)},
		}},
	})
	if got := idx.GetUtxoAssets(6)["dftx"]; got != 600 {
		t.Fatalf("old dft commit mint should be accepted, got %d", got)
	}

	ticker := idx.getTickerLocked("dftx")
	ticker.MintedTimes = 0
	ticker.MintedAmount = 0
	overflowCommit := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccd"
	overflowMintTx := "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe"
	idx.UpdateTransfer(&common.Block{
		Height: 27151,
		Transactions: []*common.Transaction{{
			TxId: overflowMintTx,
			Inputs: []*common.TxInput{testTxInput(7, overflowCommit, 0, 1000, 27150,
				atomWitness(t, []byte{0x03, 'd', 'm', 't'}, map[string]any{"mint_ticker": "dftx"}))},
			Outputs: []*common.TxOutputV2{testTxOutput(8, 103, overflowMintTx, 0, 600)},
		}},
	})
	if got := idx.GetUtxoAssets(8)["dftx"]; got != 0 {
		t.Fatalf("dft mint over max_mints should be rejected by mint history count, got %d", got)
	}
}

func TestDirectTickerUsesEarliestCommitTx(t *testing.T) {
	idx := NewIndexer(nil, &chaincfg.TestNet4Params)
	earlyCommit := "00009a3a225b4ed471f613c81589dc118f144ae780b0241616922dd476db8f52"
	lateCommit := "00004abda1e4de3e171feffe8bf95c92f5760996d7d662f2978ed4b8130b686d"
	lateReveal := "bf79bf0c0da339de2f77098e3897838dc69313389378e2b49da60aa3ed11e977"
	lateInput := testTxInput(11, lateCommit, 0, 1000, 27010,
		atomWitness(t, []byte{0x02, 'f', 't'}, map[string]any{"request_ticker": "shadow", "bitworkc": "0000"}))
	lateInput.OutTxIndex = 1750
	idx.UpdateTransfer(&common.Block{
		Height: 27010,
		Transactions: []*common.Transaction{{
			TxId:    lateReveal,
			Inputs:  []*common.TxInput{lateInput},
			Outputs: []*common.TxOutputV2{testTxOutput(12, 100, lateReveal, 0, 20000)},
		}},
	})
	if got := idx.GetUtxoAssets(12)["shadow"]; got != 20000 {
		t.Fatalf("late commit ticker should be accepted first, got %d", got)
	}

	earlyReveal := "74918ef1cc92dad430757395222bf7152698b1565ae896c3867c3305c8c65ebc"
	earlyInput := testTxInput(13, earlyCommit, 0, 1000, 27010,
		atomWitness(t, []byte{0x02, 'f', 't'}, map[string]any{"request_ticker": "shadow", "bitworkc": "0000"}))
	earlyInput.OutTxIndex = 1749
	idx.UpdateTransfer(&common.Block{
		Height: 27011,
		Transactions: []*common.Transaction{{
			TxId:    earlyReveal,
			Inputs:  []*common.TxInput{earlyInput},
			Outputs: []*common.TxOutputV2{testTxOutput(14, 101, earlyReveal, 0, 10000)},
		}},
	})
	ticker := idx.GetTicker("shadow")
	if ticker == nil || ticker.AtomicalId != compactId(earlyCommit, 0) || ticker.MaxSupply != 10000 {
		t.Fatalf("expected earliest commit ticker, got %#v", ticker)
	}
	if got := idx.GetUtxoAssets(12)["shadow"]; got != 0 {
		t.Fatalf("late ticker mint should be removed, got %d", got)
	}
	if got := idx.GetUtxoAssets(14)["shadow"]; got != 10000 {
		t.Fatalf("early ticker mint should remain, got %d", got)
	}
}

func TestPerpetualDftMintRequiresExpectedBitwork(t *testing.T) {
	idx := NewIndexer(nil, &chaincfg.TestNet4Params)
	deployCommit := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	deployTx := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	idx.UpdateTransfer(&common.Block{
		Height: 27010,
		Transactions: []*common.Transaction{{
			TxId: deployTx,
			Inputs: []*common.TxInput{testTxInput(1, deployCommit, 0, 1000, 27008,
				atomWitness(t, []byte{0x03, 'd', 'f', 't'}, map[string]any{
					"request_ticker": "perp",
					"bitworkc":       "aaaa",
					"mint_height":    uint64(27010),
					"mint_amount":    uint64(600),
					"max_mints":      uint64(2),
					"md":             uint64(1),
					"bv":             strings.Repeat("d", 256),
					"bci":            uint64(1),
				}))},
			Outputs: []*common.TxOutputV2{testTxOutput(2, 100, deployTx, 0, 546)},
		}},
	})
	if ticker := idx.GetTicker("perp"); ticker == nil || ticker.MintMode != "perpetual" || ticker.Bcs != 64 || ticker.Brs != 64 {
		t.Fatalf("unexpected perpetual ticker: %#v", ticker)
	}

	badMintTx := "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	idx.UpdateTransfer(&common.Block{
		Height: 27011,
		Transactions: []*common.Transaction{{
			TxId: badMintTx,
			Inputs: []*common.TxInput{testTxInput(3, badMintTx, 0, 1000, 27010,
				atomWitness(t, []byte{0x03, 'd', 'm', 't'}, map[string]any{"mint_ticker": "perp"}))},
			Outputs: []*common.TxOutputV2{testTxOutput(4, 101, badMintTx, 0, 600)},
		}},
	})
	if got := idx.GetUtxoAssets(4)["perp"]; got != 0 {
		t.Fatalf("bad perpetual mint should be rejected, got %d", got)
	}

	goodCommit := "dddd000000000000000000000000000000000000000000000000000000000000"
	goodMintTx := "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	idx.UpdateTransfer(&common.Block{
		Height: 27012,
		Transactions: []*common.Transaction{{
			TxId: goodMintTx,
			Inputs: []*common.TxInput{testTxInput(5, goodCommit, 0, 1000, 27011,
				atomWitness(t, []byte{0x03, 'd', 'm', 't'}, map[string]any{"mint_ticker": "perp"}))},
			Outputs: []*common.TxOutputV2{testTxOutput(6, 102, goodMintTx, 0, 600)},
		}},
	})
	if got := idx.GetUtxoAssets(6)["perp"]; got != 600 {
		t.Fatalf("good perpetual mint amount mismatch: %d", got)
	}
}

func TestDftMintWaitsForTickerEffectiveDelay(t *testing.T) {
	idx := NewIndexer(nil, &chaincfg.TestNet4Params)
	deployCommit := "0000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	deployTx := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	idx.UpdateTransfer(&common.Block{
		Height: 27010,
		Transactions: []*common.Transaction{{
			TxId: deployTx,
			Inputs: []*common.TxInput{testTxInput(1, deployCommit, 0, 1000, 27010,
				atomWitness(t, []byte{0x03, 'd', 'f', 't'}, map[string]any{
					"request_ticker": "wait",
					"bitworkc":       "0000",
					"mint_height":    uint64(0),
					"mint_amount":    uint64(1000),
					"max_mints":      uint64(1),
				}))},
			Outputs: []*common.TxOutputV2{testTxOutput(2, 100, deployTx, 0, 546)},
		}},
	})

	earlyMintTx := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	idx.UpdateTransfer(&common.Block{
		Height: 27012,
		Transactions: []*common.Transaction{{
			TxId: earlyMintTx,
			Inputs: []*common.TxInput{testTxInput(3, earlyMintTx, 0, 1000, 27012,
				atomWitness(t, []byte{0x03, 'd', 'm', 't'}, map[string]any{"mint_ticker": "wait"}))},
			Outputs: []*common.TxOutputV2{testTxOutput(4, 101, earlyMintTx, 0, 1000)},
		}},
	})
	if got := idx.GetUtxoAssets(4)["wait"]; got != 0 {
		t.Fatalf("early DFT mint before ticker effective delay should be rejected, got %d", got)
	}

	validMintTx := "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	idx.UpdateTransfer(&common.Block{
		Height: 27013,
		Transactions: []*common.Transaction{{
			TxId: validMintTx,
			Inputs: []*common.TxInput{testTxInput(5, validMintTx, 0, 1000, 27013,
				atomWitness(t, []byte{0x03, 'd', 'm', 't'}, map[string]any{"mint_ticker": "wait"}))},
			Outputs: []*common.TxOutputV2{testTxOutput(6, 102, validMintTx, 0, 1000)},
		}},
	})
	if got := idx.GetUtxoAssets(6)["wait"]; got != 1000 {
		t.Fatalf("DFT mint after ticker effective delay should be accepted, got %d", got)
	}
}
