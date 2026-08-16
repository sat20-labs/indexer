package indexer

import (
	"testing"

	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
)

func TestAllocateBoundMempoolOutputsFindsPlainChange(t *testing.T) {
	assetName := common.AssetName{
		Protocol: common.PROTOCOL_NAME_ORDX,
		Type:     common.ASSET_TYPE_FT,
		Ticker:   "test",
	}
	assetInput := common.NewTxOutput(330)
	assetInput.OutPointStr = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa:0"
	assetInput.Assets = common.TxAssets{{
		Name:       assetName,
		Amount:     *common.NewDefaultDecimal(1),
		BindingSat: 1,
	}}
	assetInput.Offsets[assetName] = common.AssetOffsets{{Start: 0, End: 1}}

	plainInput := common.NewTxOutput(100000)
	plainInput.OutPointStr = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb:0"

	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxOut(wire.NewTxOut(330, []byte{0x51}))
	tx.AddTxOut(wire.NewTxOut(99000, []byte{0x51}))

	outputs, occupied, ok := allocateBoundMempoolOutputs(tx, []*mempoolResolvedInput{
		{output: assetInput, confirmed: true, index: 0},
		{output: plainInput, confirmed: true, index: 1},
	})
	if !ok {
		t.Fatal("allocation unexpectedly unresolved")
	}
	if len(outputs) != 2 || len(occupied) != 2 {
		t.Fatalf("unexpected result sizes: outputs=%d occupied=%d", len(outputs), len(occupied))
	}
	if !occupied[0] {
		t.Fatal("asset output must be non-plain")
	}
	if occupied[1] {
		t.Fatal("change output should be classified plain")
	}
	if outputs[1].Value() != 99000 || outputs[1].HasAsset() {
		t.Fatalf("unexpected plain change: value=%d assets=%v", outputs[1].Value(), outputs[1].Assets)
	}
}

func TestMempoolAtomRegularAllocationPreservesPlainOutput(t *testing.T) {
	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxOut(wire.NewTxOut(600, []byte{0x51}))
	tx.AddTxOut(wire.NewTxOut(1000, []byte{0x51}))

	assignments, ok := mempoolAtomAssignRegular(tx, 0, 600, true)
	if !ok {
		t.Fatal("regular atom allocation should complete")
	}
	if len(assignments) != 1 || assignments[0].output != 0 || assignments[0].amount != 600 {
		t.Fatalf("unexpected assignments: %+v", assignments)
	}
}

func TestMempoolAtomRegularAllocationCanPartiallyColorOutput(t *testing.T) {
	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxOut(wire.NewTxOut(600, []byte{0x51}))
	tx.AddTxOut(wire.NewTxOut(1000, []byte{0x51}))

	assignments, ok := mempoolAtomAssignRegular(tx, 0, 800, true)
	if !ok {
		t.Fatal("regular atom allocation should complete")
	}
	if len(assignments) != 2 || assignments[0].amount != 600 || assignments[1].amount != 200 {
		t.Fatalf("unexpected assignments: %+v", assignments)
	}
}

func TestMempoolMarkSatRangeAcrossOutputs(t *testing.T) {
	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxOut(wire.NewTxOut(100, []byte{0x51}))
	tx.AddTxOut(wire.NewTxOut(100, []byte{0x51}))
	tx.AddTxOut(wire.NewTxOut(100, []byte{0x51}))

	occupied := make([]bool, 3)
	mempoolMarkSatRange(tx, 90, 110, occupied)
	if !occupied[0] || !occupied[1] || occupied[2] {
		t.Fatalf("unexpected occupied outputs: %v", occupied)
	}
}
