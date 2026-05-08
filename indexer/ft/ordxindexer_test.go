package ft

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"unsafe"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/base"
	indexer "github.com/sat20-labs/indexer/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
	"github.com/sat20-labs/indexer/indexer/exotic"
	"github.com/sat20-labs/indexer/indexer/nft"
)

func TestIntervalTree(t *testing.T) {
	// 创建一个区间树
	tree := indexer.NewRBTress()

	// 插入一些区间
	tree.Put(&common.Range{Start: 1, Size: 5}, "UTXO(1)")
	tree.Put(&common.Range{Start: 1, Size: 5}, "UTXO(1.1)")
	tree.Put(&common.Range{Start: 1, Size: 4}, "UTXO(1.2)")
	tree.Put(&common.Range{Start: 1, Size: 6}, "UTXO(1.3)")
	tree.Put(&common.Range{Start: 7, Size: 4}, "UTXO(2)")
	tree.Put(&common.Range{Start: 13, Size: 7}, "UTXO(3)")
	tree.Put(&common.Range{Start: 26, Size: 10}, "UTXO(4)")
	tree.Put(&common.Range{Start: 38, Size: 12}, "UTXO(5)")
	printRBTree(tree)

	// 查询与给定区间相交的所有区间
	key := common.Range{Start: 4, Size: 26}
	intersections := tree.FindIntersections(&key)
	for _, v := range intersections {
		fmt.Printf("Intersections: %s %d-%d\n", v.Value.(string), v.Rng.Start, v.Rng.Size)
	}

	printRBTree(tree)
	tree.RemoveRange(&key)

	tree.Put(&key, "UTXO(6)")
	printRBTree(tree)
}

func printRBTree(tree *indexer.RangeRBTree) {
	fmt.Println(tree)
	fmt.Printf("\n")
}

func TestSplitRange(t *testing.T) {

	{
		tree := indexer.NewRBTress()

		// 测试数据
		rangeA := common.Range{Start: 5, Size: 2}
		rangeB := common.Range{Start: 1, Size: 10}

		tree.AddMintInfo(&rangeA, "utxo_A")
		printRBTree(tree)
		tree.AddMintInfo(&rangeB, "utxo_B")
		printRBTree(tree)
	}

	{
		tree := indexer.NewRBTress()

		// 测试数据
		rangeA := common.Range{Start: 1, Size: 10}
		rangeB := common.Range{Start: 5, Size: 2}

		tree.AddMintInfo(&rangeA, "utxo_A")
		printRBTree(tree)
		tree.AddMintInfo(&rangeB, "utxo_B")
		printRBTree(tree)
	}

	{
		tree := indexer.NewRBTress()

		// 测试数据
		rangeA := common.Range{Start: 1, Size: 5}
		rangeB := common.Range{Start: 4, Size: 6}

		tree.AddMintInfo(&rangeA, "utxo_A")
		printRBTree(tree)
		tree.AddMintInfo(&rangeB, "utxo_B")
		printRBTree(tree)
	}

	{
		tree := indexer.NewRBTress()

		// 测试数据
		rangeA := common.Range{Start: 4, Size: 6}
		rangeB := common.Range{Start: 1, Size: 5}

		tree.AddMintInfo(&rangeA, "utxo_A")
		printRBTree(tree)
		tree.AddMintInfo(&rangeB, "utxo_B")
		printRBTree(tree)
	}

}

func TestPizzaRange(t *testing.T) {

	tree := indexer.NewRBTress()

	// 测试数据
	for i, rng := range exotic.PizzaRanges {
		tree.AddMintInfo(rng, strconv.Itoa(i))
	}

	if len(exotic.PizzaRanges) != tree.Size() {
		t.Fatalf("")
	}

	printRBTree(tree)

}

func buildUnbindScript(t *testing.T, ticker string, vout int) []byte {
	t.Helper()

	builder := txscript.NewScriptBuilder().
		AddOp(txscript.OP_RETURN).
		AddOp(txscript.OP_16).
		AddInt64(int64(txscript.OP_DATA_40)).
		AddData([]byte(ticker)).
		AddInt64(int64(vout))

	script, err := builder.Script()
	if err != nil {
		t.Fatalf("build unbind script: %v", err)
	}
	return script
}

func buildFreezeScript(t *testing.T, ticker, address string, height int) []byte {
	t.Helper()

	builder := txscript.NewScriptBuilder().
		AddOp(txscript.OP_RETURN).
		AddOp(txscript.OP_16).
		AddInt64(int64(txscript.OP_DATA_43)).
		AddData([]byte(ticker)).
		AddData([]byte(address)).
		AddInt64(int64(height))

	script, err := builder.Script()
	if err != nil {
		t.Fatalf("build freeze script: %v", err)
	}
	return script
}

func buildUnfreezeScript(t *testing.T, ticker, address string) []byte {
	t.Helper()

	builder := txscript.NewScriptBuilder().
		AddOp(txscript.OP_RETURN).
		AddOp(txscript.OP_16).
		AddInt64(int64(txscript.OP_DATA_44)).
		AddData([]byte(ticker)).
		AddData([]byte(address))

	script, err := builder.Script()
	if err != nil {
		t.Fatalf("build unfreeze script: %v", err)
	}
	return script
}

func testHolder(addressId uint64, ticker string, bindingSat int, offsets common.AssetOffsets) *HolderInfo {
	return &HolderInfo{
		AddressId: addressId,
		Tickers: map[string]*common.AssetAbbrInfo{
			ticker: {
				BindingSat: bindingSat,
				Offsets:    offsets,
			},
		},
	}
}

func newTestFTIndexer() *FTIndexer {
	return &FTIndexer{
		enableHeight:             0,
		tickerMap:                make(map[string]*TickInfo),
		holderInfo:               make(map[uint64]*HolderInfo),
		utxoMap:                  make(map[string]map[uint64]int64),
		holderActionList:         make([]*HolderAction, 0),
		tickerAdded:              make(map[string]*common.Ticker),
		actionBufferMap:          make(map[uint64][]*ActionInfo),
		unbindHistory:            make([]*common.UnbindHistory, 0),
		freezeHistory:            make([]*common.FreezeHistory, 0),
		freezeStates:             make(map[string]map[uint64]*common.FreezeState),
		freezeTouched:            make(map[string]*common.FreezeState),
		freezeDeleted:            make(map[string]*common.FreezeState),
		pendingHistoricalFreezes: make(map[int][]*common.FreezeDirective),
		pendingHistoricalKeys:    make(map[string]bool),
		reloadFreezeDirectives:   make(map[string]*common.FreezeDirective),
	}
}

func setPrivateField(target any, field string, value any) {
	v := reflect.ValueOf(target).Elem().FieldByName(field)
	reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}

func attachTestDeployNft(p *FTIndexer, nftId int64, ownerAddressId uint64) {
	n := &nft.NftIndexer{}
	setPrivateField(n, "nftIdToinscriptionMap", map[int64]*common.Nft{
		nftId: {
			Base:           &common.InscribeBaseContent{Id: nftId},
			OwnerAddressId: ownerAddressId,
		},
	})
	p.nftIndexer = n
}

func TestParseUnbindScript(t *testing.T) {
	script := buildUnbindScript(t, "pearl", 2)

	ticker, got, matched, err := ParseUnbindScript(script)
	if err != nil {
		t.Fatalf("ParseUnbindScript error: %v", err)
	}
	if !matched {
		t.Fatalf("expected unbind script to match")
	}
	if ticker != "pearl" {
		t.Fatalf("unexpected ticker %s", ticker)
	}
	if got != 2 {
		t.Fatalf("unexpected vout %d", got)
	}
}

func TestParseFreezeScripts(t *testing.T) {
	freezeScript := buildFreezeScript(t, "pearl", "tb1ptestaddress", 100)
	ticker, address, height, matched, err := ParseFreezeScript(freezeScript)
	if err != nil || !matched {
		t.Fatalf("ParseFreezeScript failed: matched=%v err=%v", matched, err)
	}
	if ticker != "pearl" || address != "tb1ptestaddress" || height != 100 {
		t.Fatalf("unexpected freeze parse result %s %s %d", ticker, address, height)
	}

	unfreezeScript := buildUnfreezeScript(t, "pearl", "tb1ptestaddress")
	ticker, address, matched, err = ParseUnfreezeScript(unfreezeScript)
	if err != nil || !matched {
		t.Fatalf("ParseUnfreezeScript failed: matched=%v err=%v", matched, err)
	}
	if ticker != "pearl" || address != "tb1ptestaddress" {
		t.Fatalf("unexpected unfreeze parse result %s %s", ticker, address)
	}
}

func TestBuildFreezeAuthoritySnapshot(t *testing.T) {
	p := newTestFTIndexer()
	p.tickerMap["pearl"] = &TickInfo{
		Name: "pearl",
		Ticker: &common.Ticker{
			Name:     "pearl",
			SelfMint: 100,
			Base:     &common.InscribeBaseContent{Id: 99},
		},
	}
	p.tickerMap["ruby"] = &TickInfo{
		Name: "ruby",
		Ticker: &common.Ticker{
			Name:     "ruby",
			SelfMint: 0,
			Base:     &common.InscribeBaseContent{Id: 100},
		},
	}
	attachTestDeployNft(p, 99, 7)

	snapshot := p.BuildFreezeAuthoritySnapshot()
	if len(snapshot) != 1 || snapshot["pearl"] != 7 {
		t.Fatalf("unexpected authority snapshot %+v", snapshot)
	}
}

func TestHandleTxUnbindOwnerOnly(t *testing.T) {
	p := newTestFTIndexer()

	ownerUtxo := common.ToUtxoId(100, 3, 0)
	otherUtxo := common.ToUtxoId(100, 4, 0)
	p.holderInfo[ownerUtxo] = testHolder(7, "pearl", 1, common.AssetOffsets{{Start: 0, End: 3}})
	p.holderInfo[otherUtxo] = testHolder(9, "pearl", 1, common.AssetOffsets{{Start: 0, End: 2}})
	p.utxoMap["pearl"] = map[uint64]int64{
		ownerUtxo: 3,
		otherUtxo: 2,
	}
	p.tickerMap["pearl"] = &TickInfo{
		Name: "pearl",
		Ticker: &common.Ticker{
			Name: "pearl",
			Base: &common.InscribeBaseContent{InscriptionAddress: 99},
		},
	}

	tx := &common.Transaction{
		TxId: "unbind-owner-only",
		Inputs: []*common.TxInput{
			{TxOutputV2: common.TxOutputV2{AddressId: 7}},
		},
		Outputs: []*common.TxOutputV2{
			{TxOutput: common.TxOutput{UtxoId: ownerUtxo, OutValue: wire.TxOut{Value: 10}}, AddressId: 7},
			{TxOutput: common.TxOutput{UtxoId: otherUtxo, OutValue: wire.TxOut{Value: 10}}, AddressId: 9},
			{TxOutput: common.TxOutput{OutValue: wireTxOut(buildUnbindScript(t, "pearl", 0))}},
		},
	}

	p.handleTxUnbind(tx)

	if _, ok := p.holderInfo[ownerUtxo]; ok {
		t.Fatalf("owner utxo should be unbound")
	}
	if _, ok := p.utxoMap["pearl"][ownerUtxo]; ok {
		t.Fatalf("owner utxo should be removed from utxo map")
	}
	if _, ok := p.holderInfo[otherUtxo]; !ok {
		t.Fatalf("non-owner utxo should remain")
	}
	if _, ok := p.utxoMap["pearl"][otherUtxo]; !ok {
		t.Fatalf("non-owner utxo should remain in utxo map")
	}
	if len(p.holderActionList) != 1 || p.holderActionList[0].UtxoId != ownerUtxo {
		t.Fatalf("unexpected holder actions: %+v", p.holderActionList)
	}
	if p.tickerMap["pearl"].Ticker.TotalUnbound != 3 {
		t.Fatalf("unexpected unbind total %d", p.tickerMap["pearl"].Ticker.TotalUnbound)
	}
}

func TestHandleTxUnbindBeforeTransferRemovesTargetOutputTickerState(t *testing.T) {
	p := newTestFTIndexer()

	inputUtxo := common.ToUtxoId(200, 1, 0)
	outputUtxo := common.ToUtxoId(200, 2, 0)
	p.holderInfo[inputUtxo] = testHolder(7, "pearl", 1, common.AssetOffsets{{Start: 0, End: 2}})
	p.holderInfo[outputUtxo] = testHolder(7, "pearl", 1, common.AssetOffsets{{Start: 0, End: 2}})
	p.utxoMap["pearl"] = map[uint64]int64{inputUtxo: 2, outputUtxo: 2}
	p.tickerMap["pearl"] = &TickInfo{
		Name: "pearl",
		Ticker: &common.Ticker{
			Name: "pearl",
			N:    1,
			Base: &common.InscribeBaseContent{InscriptionAddress: 99},
		},
	}

	tx := &common.Transaction{
		TxId: "unbind-and-spend",
		Inputs: []*common.TxInput{
			{
				TxOutputV2: common.TxOutputV2{
					TxOutput:   *common.NewTxOutput(10),
					AddressId:  7,
					TxOutIndex: 0,
				},
			},
		},
		Outputs: []*common.TxOutputV2{
			{
				TxOutput: common.TxOutput{
					UtxoId:   outputUtxo,
					OutValue: wire.TxOut{Value: 10},
				},
				AddressId: 7,
			},
			{
				TxOutput: common.TxOutput{
					OutValue: wireTxOut(buildUnbindScript(t, "pearl", 0)),
				},
			},
		},
	}
	tx.Inputs[0].UtxoId = inputUtxo
	p.handleTxUnbind(tx)

	if _, ok := p.holderInfo[outputUtxo]; ok {
		t.Fatalf("target output utxo should be removed after unbind")
	}
	if len(p.holderActionList) == 0 || p.holderActionList[len(p.holderActionList)-1].UtxoId != outputUtxo {
		t.Fatalf("unexpected holder actions after update: %+v", p.holderActionList)
	}
}

func TestHandleTxUnbindRemovesOnlyTargetTicker(t *testing.T) {
	p := newTestFTIndexer()
	utxoId := common.ToUtxoId(300, 1, 0)
	holder := testHolder(7, "pearl", 1, common.AssetOffsets{{Start: 0, End: 3}})
	holder.Tickers["ruby"] = &common.AssetAbbrInfo{
		BindingSat: 1,
		Offsets:    common.AssetOffsets{{Start: 3, End: 5}},
	}
	p.holderInfo[utxoId] = holder
	p.utxoMap["pearl"] = map[uint64]int64{utxoId: 3}
	p.utxoMap["ruby"] = map[uint64]int64{utxoId: 2}
	p.tickerMap["pearl"] = &TickInfo{
		Name: "pearl",
		Ticker: &common.Ticker{
			Name: "pearl",
			Base: &common.InscribeBaseContent{InscriptionAddress: 99},
		},
	}
	p.tickerMap["ruby"] = &TickInfo{
		Name: "ruby",
		Ticker: &common.Ticker{
			Name: "ruby",
			Base: &common.InscribeBaseContent{InscriptionAddress: 99},
		},
	}

	tx := &common.Transaction{
		TxId: "unbind-specific-ticker",
		Inputs: []*common.TxInput{
			{TxOutputV2: common.TxOutputV2{AddressId: 7}},
		},
		Outputs: []*common.TxOutputV2{
			{TxOutput: common.TxOutput{UtxoId: utxoId, OutValue: wire.TxOut{Value: 10}}, AddressId: 7},
			{TxOutput: common.TxOutput{OutValue: wireTxOut(buildUnbindScript(t, "pearl", 0))}},
		},
	}

	p.handleTxUnbind(tx)

	if _, ok := p.holderInfo[utxoId]; !ok {
		t.Fatalf("holder should remain because other ticker still exists")
	}
	if _, ok := p.holderInfo[utxoId].Tickers["pearl"]; ok {
		t.Fatalf("target ticker should be removed")
	}
	if _, ok := p.holderInfo[utxoId].Tickers["ruby"]; !ok {
		t.Fatalf("non-target ticker should remain")
	}
	if _, ok := p.utxoMap["pearl"][utxoId]; ok {
		t.Fatalf("target ticker utxo should be deleted")
	}
	if _, ok := p.utxoMap["ruby"][utxoId]; !ok {
		t.Fatalf("non-target ticker utxo should remain")
	}
}

func TestHandleTxUnbindRejectsNonOwner(t *testing.T) {
	p := newTestFTIndexer()
	utxoId := common.ToUtxoId(400, 1, 0)
	p.holderInfo[utxoId] = testHolder(8, "pearl", 1, common.AssetOffsets{{Start: 0, End: 2}})
	p.utxoMap["pearl"] = map[uint64]int64{utxoId: 2}
	p.tickerMap["pearl"] = &TickInfo{
		Name: "pearl",
		Ticker: &common.Ticker{
			Name: "pearl",
			Base: &common.InscribeBaseContent{Id: 99},
		},
	}

	tx := &common.Transaction{
		TxId: "unbind-reject-non-owner",
		Inputs: []*common.TxInput{
			{TxOutputV2: common.TxOutputV2{AddressId: 7}},
		},
		Outputs: []*common.TxOutputV2{
			{TxOutput: common.TxOutput{UtxoId: utxoId, OutValue: wire.TxOut{Value: 10}}, AddressId: 8},
			{TxOutput: common.TxOutput{OutValue: wireTxOut(buildUnbindScript(t, "pearl", 0))}},
		},
	}

	p.handleTxUnbind(tx)

	if _, ok := p.holderInfo[utxoId]; !ok {
		t.Fatalf("non-owner should not be able to unbind")
	}
}

func TestHistoricalFreezeReplayAndBurn(t *testing.T) {
	p := newTestFTIndexer()
	p.tickerMap["pearl"] = &TickInfo{
		Name: "pearl",
		Ticker: &common.Ticker{
			Name:     "pearl",
			N:        1,
			SelfMint: 100,
			Base:     &common.InscribeBaseContent{Id: 99},
		},
	}
	attachTestDeployNft(p, 99, 7)
	setPrivateField(p.nftIndexer, "baseIndexer", &base.BaseIndexer{})
	baseIndexer := p.nftIndexer.GetBaseIndexer()
	setPrivateField(baseIndexer, "addressValueMap", map[string]*common.AddressValueV2{
		"tb1pfreeze": {AddressId: 88, Utxos: map[uint64]int64{}},
	})

	p.SetPendingHistoricalFreezeReplay([]*common.FreezeDirective{{
		Ticker:       "pearl",
		Address:      "tb1pfreeze",
		AddressId:    88,
		FreezeHeight: 100,
		TxId:         "freeze-tx",
	}})
	utxoId := common.ToUtxoId(99, 1, 0)
	holder := testHolder(88, "pearl", 1, common.AssetOffsets{{Start: 0, End: 2}})
	p.holderInfo[utxoId] = holder
	p.utxoMap["pearl"] = map[uint64]int64{utxoId: 2}

	p.activatePendingFreezesAtHeight(100)
	if !p.holderInfo[utxoId].Tickers["pearl"].Frozen {
		t.Fatalf("holder asset should be marked frozen on activation")
	}
	if p.tickerMap["pearl"].Ticker.TotalFrozen != 2 {
		t.Fatalf("unexpected freeze total %d", p.tickerMap["pearl"].Ticker.TotalFrozen)
	}

	tx := &common.Transaction{
		TxId: "spend-frozen",
		Inputs: []*common.TxInput{{
			TxOutputV2: common.TxOutputV2{
				TxOutput: common.TxOutput{
					UtxoId:   utxoId,
					OutValue: wire.TxOut{Value: 10},
				},
				AddressId: 88,
			},
		}},
		Outputs: []*common.TxOutputV2{{
			TxOutput: common.TxOutput{
				UtxoId:   common.ToUtxoId(100, 1, 0),
				OutValue: wire.TxOut{Value: 10},
			},
			AddressId: 9,
		}},
	}

	input := tx.Inputs[0].TxOutput.Clone()
	holderInfo := p.holderInfo[utxoId]
	p.appendHolderAssetsToInput(input, holderInfo)

	change := p.innerUpdateTransfer(tx, input)
	if !change.Zero() {
		t.Fatalf("unexpected non-zero change")
	}
	if _, ok := p.holderInfo[tx.Outputs[0].UtxoId]; ok {
		t.Fatalf("frozen asset should be burned instead of transferred")
	}
	if p.tickerMap["pearl"].Ticker.TotalBurned != 2 {
		t.Fatalf("unexpected burned total %d", p.tickerMap["pearl"].Ticker.TotalBurned)
	}
}

func TestHandleTxFreezeAndUnfreezeTickerTotals(t *testing.T) {
	p := newTestFTIndexer()
	p.tickerMap["pearl"] = &TickInfo{
		Name: "pearl",
		Ticker: &common.Ticker{
			Name:     "pearl",
			SelfMint: 100,
			Base:     &common.InscribeBaseContent{Id: 99},
		},
	}
	attachTestDeployNft(p, 99, 7)
	setPrivateField(p.nftIndexer, "baseIndexer", &base.BaseIndexer{})
	baseIndexer := p.nftIndexer.GetBaseIndexer()
	setPrivateField(baseIndexer, "addressValueMap", map[string]*common.AddressValueV2{
		"tb1pfreeze": {AddressId: 88, Utxos: map[uint64]int64{}},
	})

	utxoId := common.ToUtxoId(101, 1, 0)
	p.holderInfo[utxoId] = testHolder(88, "pearl", 1, common.AssetOffsets{{Start: 0, End: 2}})
	p.utxoMap["pearl"] = map[uint64]int64{utxoId: 2}

	freezeTx := &common.Transaction{
		TxId:   "freeze-now",
		Inputs: []*common.TxInput{{TxOutputV2: common.TxOutputV2{AddressId: 7}}},
		Outputs: []*common.TxOutputV2{{
			TxOutput: common.TxOutput{
				OutValue: wireTxOut(buildFreezeScript(t, "pearl", "tb1pfreeze", 100)),
			},
		}},
	}

	p.handleTxFreeze(freezeTx, 100)
	if p.tickerMap["pearl"].Ticker.TotalFrozen != 2 {
		t.Fatalf("unexpected freeze total %d", p.tickerMap["pearl"].Ticker.TotalFrozen)
	}
	if !p.isAddressFrozen("pearl", 88) {
		t.Fatalf("address should be frozen")
	}

	unfreezeTx := &common.Transaction{
		TxId:   "unfreeze-now",
		Inputs: []*common.TxInput{{TxOutputV2: common.TxOutputV2{AddressId: 7}}},
		Outputs: []*common.TxOutputV2{{
			TxOutput: common.TxOutput{
				OutValue: wireTxOut(buildUnfreezeScript(t, "pearl", "tb1pfreeze")),
			},
		}},
	}

	p.handleTxFreeze(unfreezeTx, 101)
	if p.tickerMap["pearl"].Ticker.TotalUnfrozen != 2 {
		t.Fatalf("unexpected unfreeze total %d", p.tickerMap["pearl"].Ticker.TotalUnfrozen)
	}
	if p.isAddressFrozen("pearl", 88) {
		t.Fatalf("address should be unfrozen")
	}
}

func TestHandleTxFreezeProcessesUnfreezeOutput(t *testing.T) {
	p := newTestFTIndexer()
	p.tickerMap["pearl"] = &TickInfo{
		Name: "pearl",
		Ticker: &common.Ticker{
			Name:     "pearl",
			SelfMint: 100,
			Base:     &common.InscribeBaseContent{Id: 99},
		},
	}
	attachTestDeployNft(p, 99, 7)
	setPrivateField(p.nftIndexer, "baseIndexer", &base.BaseIndexer{})
	baseIndexer := p.nftIndexer.GetBaseIndexer()
	setPrivateField(baseIndexer, "addressValueMap", map[string]*common.AddressValueV2{
		"tb1pfreeze": {AddressId: 88, Utxos: map[uint64]int64{}},
	})

	utxoId := common.ToUtxoId(102, 1, 0)
	p.holderInfo[utxoId] = testHolder(88, "pearl", 1, common.AssetOffsets{{Start: 0, End: 2}})
	p.utxoMap["pearl"] = map[uint64]int64{utxoId: 2}
	p.setFreezeState("pearl", 88, &common.FreezeState{
		Ticker:       "pearl",
		AddressId:    88,
		FreezeHeight: 100,
		TxId:         "freeze-existing",
	})
	p.markAddressTickerFrozen("pearl", 88, true)

	tx := &common.Transaction{
		TxId:   "unfreeze-only",
		Inputs: []*common.TxInput{{TxOutputV2: common.TxOutputV2{AddressId: 7}}},
		Outputs: []*common.TxOutputV2{{
			TxOutput: common.TxOutput{
				OutValue: wireTxOut(buildUnfreezeScript(t, "pearl", "tb1pfreeze")),
			},
		}},
	}

	p.handleTxFreeze(tx, 101)

	if p.isAddressFrozen("pearl", 88) {
		t.Fatalf("unfreeze output should be processed")
	}
	if p.tickerMap["pearl"].Ticker.TotalUnfrozen != 2 {
		t.Fatalf("unexpected unfreeze total %d", p.tickerMap["pearl"].Ticker.TotalUnfrozen)
	}
	if len(p.freezeHistory) != 1 || p.freezeHistory[0].Action != common.FreezeActionUnfreeze {
		t.Fatalf("unexpected freeze history %+v", p.freezeHistory)
	}
}

func TestBackdatedFreezeRequestsReload(t *testing.T) {
	p := newTestFTIndexer()
	p.tickerMap["pearl"] = &TickInfo{
		Name: "pearl",
		Ticker: &common.Ticker{
			Name:     "pearl",
			SelfMint: 100,
			Base:     &common.InscribeBaseContent{Id: 99},
		},
	}
	attachTestDeployNft(p, 99, 7)
	setPrivateField(p.nftIndexer, "baseIndexer", &base.BaseIndexer{})
	baseIndexer := p.nftIndexer.GetBaseIndexer()
	setPrivateField(baseIndexer, "addressValueMap", map[string]*common.AddressValueV2{
		"tb1pfreeze": {AddressId: 88, Utxos: map[uint64]int64{}},
	})

	tx := &common.Transaction{
		TxId:   "freeze-reload",
		Inputs: []*common.TxInput{{TxOutputV2: common.TxOutputV2{AddressId: 7}}},
		Outputs: []*common.TxOutputV2{{
			TxOutput: common.TxOutput{
				OutValue: wireTxOut(buildFreezeScript(t, "pearl", "tb1pfreeze", 100)),
			},
		}},
	}

	p.handleTxFreeze(tx, 102)
	height, directives := p.ConsumeReloadRequest()
	if height != 100 || len(directives) != 1 {
		t.Fatalf("unexpected reload request height=%d directives=%d", height, len(directives))
	}
}

func TestGetUnbindHistoryQueries(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("ft-unbind-history-%d", os.Getpid()))
	_ = os.RemoveAll(tempDir)
	defer os.RemoveAll(tempDir)

	kvdb := db.NewKVDB(tempDir)
	defer kvdb.Close()

	item1 := &common.UnbindHistory{
		Ticker:    "pearl",
		AddressId: 7,
		UtxoId:    common.ToUtxoId(500, 1, 0),
		Amount:    3,
		Offsets:   common.AssetOffsets{{Start: 0, End: 3}},
	}
	wb := kvdb.NewWriteBatch()
	if err := db.SetDB([]byte(GetUnbindHistoryKey(item1.Ticker, item1.AddressId, item1.UtxoId)), item1, wb); err != nil {
		t.Fatalf("persist history: %v", err)
	}
	if err := wb.Flush(); err != nil {
		t.Fatalf("flush history: %v", err)
	}
	wb.Close()

	p := newTestFTIndexer()
	p.db = kvdb
	p.unbindHistory = append(p.unbindHistory, &common.UnbindHistory{
		Ticker:    "pearl",
		AddressId: 8,
		UtxoId:    common.ToUtxoId(501, 1, 0),
		Amount:    2,
		Offsets:   common.AssetOffsets{{Start: 1, End: 3}},
	})

	history, total := p.GetUnbindHistory("pearl", 0, 10)
	if total != 2 || len(history) != 2 {
		t.Fatalf("unexpected history total=%d len=%d", total, len(history))
	}

	addrHistory, total := p.GetUnbindHistoryWithAddress(7, "pearl", 0, 10)
	if total != 1 || len(addrHistory) != 1 {
		t.Fatalf("unexpected address history total=%d len=%d", total, len(addrHistory))
	}
	if addrHistory[0].UtxoId != item1.UtxoId || addrHistory[0].Amount != item1.Amount {
		t.Fatalf("unexpected address history item %+v", addrHistory[0])
	}
}

func TestGetFreezeHistoryQueries(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("ft-freeze-history-%d", os.Getpid()))
	_ = os.RemoveAll(tempDir)
	defer os.RemoveAll(tempDir)

	kvdb := db.NewKVDB(tempDir)
	defer kvdb.Close()

	item1 := &common.FreezeHistory{
		Ticker:        "pearl",
		AddressId:     7,
		TxId:          "freeze-db",
		Action:        common.FreezeActionFreeze,
		Amount:        3,
		FreezeHeight:  100,
		ConfirmHeight: 101,
	}
	wb := kvdb.NewWriteBatch()
	if err := db.SetDB([]byte(GetFreezeHistoryKey(item1.Ticker, item1.AddressId, item1.Action, item1.TxId)), item1, wb); err != nil {
		t.Fatalf("persist freeze history: %v", err)
	}
	if err := wb.Flush(); err != nil {
		t.Fatalf("flush freeze history: %v", err)
	}
	wb.Close()

	p := newTestFTIndexer()
	p.db = kvdb
	p.freezeHistory = append(p.freezeHistory, &common.FreezeHistory{
		Ticker:        "pearl",
		AddressId:     8,
		TxId:          "unfreeze-mem",
		Action:        common.FreezeActionUnfreeze,
		Amount:        2,
		FreezeHeight:  102,
		ConfirmHeight: 102,
	})

	history, total := p.GetFreezeHistory("pearl", 0, 10)
	if total != 2 || len(history) != 2 {
		t.Fatalf("unexpected freeze history total=%d len=%d", total, len(history))
	}

	addrHistory, total := p.GetFreezeHistoryWithAddress(7, "pearl", 0, 10)
	if total != 1 || len(addrHistory) != 1 {
		t.Fatalf("unexpected freeze address history total=%d len=%d", total, len(addrHistory))
	}
	if addrHistory[0].TxId != item1.TxId || addrHistory[0].Action != item1.Action || addrHistory[0].Amount != item1.Amount {
		t.Fatalf("unexpected freeze address history item %+v", addrHistory[0])
	}
}

func wireTxOut(pkScript []byte) wire.TxOut {
	return wire.TxOut{PkScript: pkScript}
}
