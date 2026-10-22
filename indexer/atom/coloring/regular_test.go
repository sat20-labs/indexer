package coloring

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

var (
	idA         = strings.Repeat("0", 63) + "1i0"
	idB         = strings.Repeat("0", 63) + "2i0"
	testHeights = ActivationHeights{Activation: 10, Dmint: 20, CustomColoring: 30}
)

func outputs(values ...int64) []*wire.TxOut {
	result := make([]*wire.TxOut, 0, len(values))
	for _, value := range values {
		result = append(result, wire.NewTxOut(value, []byte{txscript.OP_TRUE}))
	}
	return result
}

func regularTx(values ...int64) *wire.MsgTx {
	tx := wire.NewMsgTx(2)
	tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(&chainhash.Hash{1}, 0), nil, nil))
	tx.TxOut = outputs(values...)
	return tx
}

func TestNetworkActivationPolicy(t *testing.T) {
	main, err := HeightsForNetwork(wire.MainNet)
	if err != nil || main.Activation != 808080 || main.CustomColoring != 848484 {
		t.Fatalf("unexpected mainnet heights: %+v, %v", main, err)
	}
	for _, network := range []wire.BitcoinNet{wire.TestNet4, wire.TestNet} {
		heights, err := HeightsForNetwork(network)
		if err != nil || heights != (ActivationHeights{27000, 27000, 27000, 27000, 27000, 27000}) {
			t.Fatalf("unexpected local/testnet4 policy for %x: %+v, %v", network, heights, err)
		}
	}
	for _, network := range []wire.BitcoinNet{wire.TestNet3, wire.SimNet, wire.BitcoinNet(0)} {
		if _, err := HeightsForNetwork(network); err == nil {
			t.Fatalf("unknown or unsupported network %x accepted", network)
		}
	}
}

func TestNativeIDOrderingAndCanonicalIdentity(t *testing.T) {
	// The vout is little endian in the native location bytes: 256 sorts before 1.
	base := strings.Repeat("ab", 32)
	if CompareAtomicalIDs(base+"i256", base+"i1") >= 0 {
		t.Fatal("vout ordering must use native location bytes")
	}
	// The displayed transaction hash is reversed before comparing it.
	first := "01" + strings.Repeat("00", 31) + "i0"
	second := strings.Repeat("00", 31) + "01i0"
	if CompareAtomicalIDs(first, second) >= 0 {
		t.Fatal("transaction hash ordering must use native location bytes")
	}
	for _, id := range []string{"quark", base + "i01", base + "i-1", base + "i4294967296", strings.ToUpper(base) + "i0", base + "i"} {
		if _, err := AtomicalIDKey(id); err == nil {
			t.Errorf("non-canonical native identity accepted: %q", id)
		}
	}
}

func TestRegularTransferOrderingAcrossActivation(t *testing.T) {
	inputs := []Input{{Value: 100, Assets: []Asset{{idB, 100}}}, {Value: 100, Assets: []Asset{{idA, 100}}}}
	for _, test := range []struct {
		height int
		order  []string
	}{{19, []string{idA, idB}}, {20, []string{idB, idA}}} {
		result, err := ColorRegular(inputs, outputs(100, 100), test.height, testHeights)
		if err != nil || !reflect.DeepEqual(result.Order, test.order) {
			t.Fatalf("height %d: order %+v, error %v", test.height, result, err)
		}
	}
	// IDs first seen in one input use native ID ordering, regardless of fact order.
	result, err := ColorRegular([]Input{{Value: 100, Assets: []Asset{{idB, 100}, {idA, 100}}}}, outputs(100, 100), 30, testHeights)
	if err != nil || !reflect.DeepEqual(result.Order, []string{idA, idB}) {
		t.Fatalf("same-input native ordering: %+v, %v", result, err)
	}
}

func TestRegularTransferPartialColoringAndFallback(t *testing.T) {
	tests := []struct {
		name     string
		assets   []Asset
		values   []int64
		partial  bool
		want     map[string][]Assignment
		burned   map[string]int64
		fallback bool
	}{
		{"small amount in dust-safe output", []Asset{{idA, 100}}, []int64{330, 2000}, true,
			map[string][]Assignment{idA: {{0, 100}}}, map[string]int64{}, false},
		{"oversized output before activation", []Asset{{idA, 100}}, []int64{330}, false,
			map[string][]Assignment{idA: {}}, map[string]int64{idA: 100}, true},
		{"partial final recipient", []Asset{{idA, 500}}, []int64{330, 330, 1000}, true,
			map[string][]Assignment{idA: {{0, 330}, {1, 170}}}, map[string]int64{}, false},
		{"capacity deficit burns remainder without fallback", []Asset{{idA, 500}}, []int64{330}, true,
			map[string][]Assignment{idA: {{0, 330}}}, map[string]int64{idA: 170}, false},
		{"later missing output restarts every asset", []Asset{{idA, 100}, {idB, 100}}, []int64{330}, true,
			map[string][]Assignment{idA: {{0, 100}}, idB: {{0, 100}}}, map[string]int64{}, true},
		{"legacy fallback retains prefix and burns remainder", []Asset{{idA, 500}, {idB, 330}}, []int64{330, 330}, false,
			map[string][]Assignment{idA: {{0, 330}}, idB: {{0, 330}}}, map[string]int64{idA: 170}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := AllocateRegular(test.assets, outputs(test.values...), test.partial)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(result.Assignments, test.want) || !reflect.DeepEqual(result.Burned, test.burned) || result.Fallback != test.fallback {
				t.Fatalf("allocation mismatch: %+v; want assignments %v, burns %v, fallback %v", result, test.want, test.burned, test.fallback)
			}
		})
	}
}

func TestRegularTransferSkipsUnspendableAndZeroOutputs(t *testing.T) {
	outs := []*wire.TxOut{
		wire.NewTxOut(500, []byte{txscript.OP_RETURN}),
		wire.NewTxOut(500, []byte{txscript.OP_FALSE, txscript.OP_RETURN}),
		wire.NewTxOut(0, []byte{txscript.OP_TRUE}),
		wire.NewTxOut(330, []byte{txscript.OP_TRUE}),
	}
	result, err := ColorRegular([]Input{{Value: 1000, Assets: []Asset{{idA, 100}}}}, outs, 30, testHeights)
	if err != nil || !reflect.DeepEqual(result.Assignments[idA], []Assignment{{3, 100}}) {
		t.Fatalf("unexpected spendable output allocation: %+v, %v", result, err)
	}
}

func TestColorRegularRejectsInconsistentFacts(t *testing.T) {
	tests := []struct {
		name   string
		inputs []Input
		height int
	}{
		{"not activated", []Input{{100, []Asset{{idA, 100}}}}, 9},
		{"missing native identity", []Input{{100, []Asset{{"quark", 100}}}}, 30},
		{"negative bitcoin value", []Input{{-1, nil}}, 30},
		{"zero amount", []Input{{100, []Asset{{idA, 0}}}}, 30},
		{"amount exceeds bitcoin value", []Input{{99, []Asset{{idA, 100}}}}, 30},
		{"duplicate input allocation", []Input{{100, []Asset{{idA, 1}, {idA, 1}}}}, 30},
		{"aggregate overflow", []Input{{math.MaxInt64, []Asset{{idA, math.MaxInt64}}}, {1, []Asset{{idA, 1}}}}, 30},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ColorRegular(test.inputs, outputs(330), test.height, testHeights); err == nil {
				t.Fatal("inconsistent ARC20 input facts accepted")
			}
		})
	}
	for _, outs := range [][]*wire.TxOut{{nil}, outputs(-1)} {
		if _, err := ColorRegular([]Input{{100, []Asset{{idA, 100}}}}, outs, 30, testHeights); err == nil {
			t.Fatal("invalid output accepted")
		}
	}
}

func TestValidateRegularTransferEnforcesOrdinaryNoBurnBoundary(t *testing.T) {
	facts := []Input{{Value: 1000, Assets: []Asset{{idA, 100}}}}
	tx := regularTx(330, 600)
	result, err := ValidateRegularTransfer(tx, facts, 30, testHeights)
	if err != nil || !reflect.DeepEqual(result.Assignments[idA], []Assignment{{0, 100}}) {
		t.Fatalf("valid ordinary transfer rejected: %+v, %v", result, err)
	}
	if _, err := ValidateRegularTransfer(regularTx(50), facts, 30, testHeights); err == nil {
		t.Fatal("burn accepted")
	}
	if _, err := ValidateRegularTransfer(tx, []Input{{Value: 1000}}, 30, testHeights); err == nil {
		t.Fatal("missing ARC20 facts accepted")
	}
	if _, err := ValidateRegularTransfer(tx, nil, 30, testHeights); err == nil {
		t.Fatal("input/fact mismatch accepted")
	}
	for _, operation := range []byte{'y', 'z', 'm', 0xff} {
		tx := regularTx(330)
		tx.TxIn[0].Witness = wire.TxWitness{{txscript.OP_IF, 4, 'a', 't', 'o', 'm', operation}}
		if _, err := ValidateRegularTransfer(tx, facts, 30, testHeights); err == nil {
			t.Fatalf("unsupported or malformed Atomicals envelope %x accepted", operation)
		}
	}
	tx = regularTx(330)
	tx.AddTxIn(tx.TxIn[0])
	if _, err := ValidateRegularTransfer(tx, append(facts, facts[0]), 30, testHeights); err == nil {
		t.Fatal("duplicate input accepted")
	}
}
