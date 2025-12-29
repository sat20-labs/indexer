package common

import (
	"testing"

)
func mustDecimal(v int64) Decimal {
	return *NewDecimal(v, 0)
}

func ordAsset(name string) *AssetInfo {
	return &AssetInfo{
		Name: AssetName{
			Protocol: "ord",
			Ticker:   name,
		},
		Amount:     mustDecimal(1),
		BindingSat: 1,
	}
}

func ordxAsset(ticker string, binding uint32, sats int64) *AssetInfo {
	return &AssetInfo{
		Name: AssetName{
			Protocol: "ordx",
			Ticker:   ticker,
		},
		Amount:     mustDecimal(int64(binding) * sats),
		BindingSat: binding,
	}
}

func brc20Asset(ticker string, amount int64) *AssetInfo {
	return &AssetInfo{
		Name: AssetName{
			Protocol: "brc20",
			Ticker:   ticker,
		},
		Amount:     mustDecimal(amount),
		BindingSat: 0,
	}
}

func NewTxOutputV2(value int64) *TxOutputV2 {
	p := &TxOutputV2{}

	// === 基础数值 ===
	p.OutValue.Value = value
	p.UtxoId = INVALID_ID
	p.OutPointStr = ""

	// === 资产相关 ===
	// Assets 本身是 slice，可以延迟 append
	p.Assets = make([]AssetInfo, 0)

	// Offsets / Invalids 在 Append / Cut 中都会直接写
	p.Offsets = make(map[AssetName]AssetOffsets)
	p.Invalids = make(map[AssetName]bool)

	// === brc20 SatBindingMap ===
	p.SatBindingMap = make(map[int64]*AssetInfo)

	// === 编译期 cursor（关键）===
	// Cut 中会「增量消费」，必须提前初始化
	p.offsetCursor = make(map[AssetName]int)
	p.satCursor = make(map[AssetName]int)

	// satKeys 是惰性初始化，但 map 本身必须存在
	p.satKeys = make(map[AssetName][]int)
	return p
}

// AppendAsset 仅用于测试构造 TxOutput
// 不做去重、不做 merge、不做 rebuild
func (p *TxOutputV2) AppendAsset(
	asset AssetInfo,
	offsets AssetOffsets,
	satBindings map[int64]*AssetInfo, // brc20 可选
) {
	// Assets
	p.Assets = append(p.Assets, asset)

	// Offsets
	if len(offsets) > 0 {
		if p.Offsets == nil {
			p.Offsets = make(map[AssetName]AssetOffsets)
		}
		p.Offsets[asset.Name] = offsets
	}

	// brc20 SatBindingMap
	if len(satBindings) > 0 {
		if p.SatBindingMap == nil {
			p.SatBindingMap = make(map[int64]*AssetInfo)
		}
		for k, v := range satBindings {
			p.SatBindingMap[k] = v
		}
	}
}

func TestTxOutputV2_Cut_WithOrdOrdXAndBrc20(t *testing.T) {
	out := NewTxOutputV2(10)

	// ===== ord：一个资产 = 一个 sat =====
	ord1 := AssetInfo{
		Name: AssetName{Protocol: "ord", Ticker: "ord-100"},
		Amount: *NewDefaultDecimal(1),
		BindingSat: 1,
	}
	out.AppendAsset(ord1, AssetOffsets{
		{Start: 2, End: 3},
	}, nil)

	ord2 := AssetInfo{
		Name: AssetName{Protocol: "ord", Ticker: "ord-200"},
		Amount: *NewDefaultDecimal(1),
		BindingSat: 1,
	}
	out.AppendAsset(ord2, AssetOffsets{
		{Start: 7, End: 8},
	}, nil)

	// ===== ordx：绑定资产，每 sat = 2 =====
	ordx := AssetInfo{
		Name: AssetName{Protocol: "ordx", Ticker: "ordx-ft"},
		Amount: *NewDefaultDecimal(20), // 10 sats * 2
		BindingSat: 2,
	}
	out.AppendAsset(ordx, AssetOffsets{
		{Start: 0, End: 10},
	}, nil)

	// ===== brc20 =====
	brc := AssetInfo{
		Name: AssetName{Protocol: "brc20", Ticker: "ordi"},
		Amount: *NewDefaultDecimal(100),
		BindingSat: 0,
	}

	brcMap := map[int64]*AssetInfo{
		1: {Name: brc.Name, Amount: *NewDefaultDecimal(30)},
		8: {Name: brc.Name, Amount: *NewDefaultDecimal(70)},
	}

	out.AppendAsset(brc, AssetOffsets{
		{Start: 1, End: 2},
		{Start: 8, End: 9},
	}, brcMap)

	// ===== 编译期对象 =====
	v2 := out

	left, err := v2.Cut(5)
	if err != nil {
		t.Fatalf("cut failed: %v", err)
	}

	// ===== left 校验 =====
	requireNoAsset(t, left, "ord", "ord-200")
	requireAsset(t, left, "ord", "ord-100", 1)
	requireAsset(t, left, "ordx", "ordx-ft", 10) // 5 sats * 2
	requireAsset(t, left, "brc20", "ordi", 30)

	right, err := v2.Cut(5)
	if err != nil {
		t.Fatalf("cut failed: %v", err)
	}
	// ===== right 校验 =====
	requireNoAsset(t, right, "ord", "ord-100")
	requireAsset(t, right, "ord", "ord-200", 1)
	requireAsset(t, right, "ordx", "ordx-ft", 10)
	requireAsset(t, right, "brc20", "ordi", 70)
}


func TestTxOutputV2_Cut_WithOrdOrdXAndBrc20_2(t *testing.T) {
	out := NewTxOutputV2(10)

	// ===== ord：一个资产 = 一个 sat =====
	ord1 := AssetInfo{
		Name: AssetName{Protocol: "ord", Ticker: "ord-100"},
		Amount: *NewDefaultDecimal(1),
		BindingSat: 1,
	}
	out.AppendAsset(ord1, AssetOffsets{
		{Start: 2, End: 3},
	}, nil)

	ord2 := AssetInfo{
		Name: AssetName{Protocol: "ord", Ticker: "ord-200"},
		Amount: *NewDefaultDecimal(1),
		BindingSat: 1,
	}
	out.AppendAsset(ord2, AssetOffsets{
		{Start: 7, End: 8},
	}, nil)

	// ===== ordx：绑定资产，每 sat = 2 =====
	ordx := AssetInfo{
		Name: AssetName{Protocol: "ordx", Ticker: "ordx-ft"},
		Amount: *NewDefaultDecimal(14), // 10 sats * 2
		BindingSat: 2,
	}
	out.AppendAsset(ordx, AssetOffsets{
		{Start: 0, End: 1},
		{Start: 2, End: 6},
		{Start: 8, End: 10},
	}, nil)

	// ===== brc20 =====
	brc := AssetInfo{
		Name: AssetName{Protocol: "brc20", Ticker: "ordi"},
		Amount: *NewDefaultDecimal(100),
		BindingSat: 0,
	}

	brcMap := map[int64]*AssetInfo{
		1: {Name: brc.Name, Amount: *NewDefaultDecimal(30)},
		8: {Name: brc.Name, Amount: *NewDefaultDecimal(70)},
	}

	out.AppendAsset(brc, AssetOffsets{
		{Start: 1, End: 2},
		{Start: 8, End: 9},
	}, brcMap)

	// ===== 编译期对象 =====
	v2 := out

	left, err := v2.Cut(1) // 0
	if err != nil {
		t.Fatalf("cut failed: %v", err)
	}
	requireNoAsset(t, left, "ord", "ord-100")
	requireNoAsset(t, left, "ord", "ord-200",)
	requireAsset(t, left, "ordx", "ordx-ft", 2)
	requireNoAsset(t, left, "brc20", "ordi")

	left, err = v2.Cut(3) // 1-4
	if err != nil {
		t.Fatalf("cut failed: %v", err)
	}
	requireAsset(t, left, "ord", "ord-100", 1)
	requireNoAsset(t, left, "ord", "ord-200",)
	requireAsset(t, left, "ordx", "ordx-ft", 4)
	requireAsset(t, left, "brc20", "ordi", 30)

	right, err := v2.Cut(5) // 5-9
	if err != nil {
		t.Fatalf("cut failed: %v", err)
	}
	// ===== right 校验 =====
	requireNoAsset(t, right, "ord", "ord-100")
	requireAsset(t, right, "ord", "ord-200", 1)
	requireAsset(t, right, "ordx", "ordx-ft", 6)
	requireAsset(t, right, "brc20", "ordi", 70)

	right, err = v2.Cut(1) // 10
	if err != nil {
		t.Fatalf("cut failed: %v", err)
	}
	// ===== right 校验 =====
	requireNoAsset(t, right, "ord", "ord-100")
	requireNoAsset(t, right, "ord", "ord-200")
	requireAsset(t, right, "ordx", "ordx-ft", 2)
	requireNoAsset(t, right, "brc20", "ordi")
}

func requireAsset(
	t *testing.T,
	out *TxOutput,
	protocol, ticker string,
	expect int64,
) {
	for _, a := range out.Assets {
		if a.Name.Protocol == protocol && a.Name.Ticker == ticker {
			if a.Amount.Int64() != expect {
				t.Fatalf(
					"%s:%s expect %d, got %d",
					protocol, ticker, expect, a.Amount.Int64(),
				)
			}
			return
		}
	}
	t.Fatalf("asset %s:%s not found", protocol, ticker)
}

func requireNoAsset(
	t *testing.T,
	out *TxOutput,
	protocol, ticker string,
) {
	for _, a := range out.Assets {
		if a.Name.Protocol == protocol && a.Name.Ticker == ticker {
			t.Fatalf("asset %s:%s should not exist", protocol, ticker)
		}
	}
}

func TestTxOutputV2_SequentialCut(t *testing.T) {
	out := NewTxOutputV2(10)

	// ord：只占用 sat 2 => [2,3)
	ord := AssetInfo{
		Name: AssetName{
			Protocol: "ord",
			Ticker:   "ord-2",
		},
		Amount:     *NewDefaultDecimal(1),
		BindingSat: 1,
	}
	out.AppendAsset(ord, AssetOffsets{
		{Start: 2, End: 3},
	}, nil)

	// ordx：覆盖 [0,10)，每 sat = 1
	ordx := AssetInfo{
		Name: AssetName{
			Protocol: "ordx",
			Ticker:   "ordx-ft",
		},
		Amount:     *NewDefaultDecimal(10),
		BindingSat: 1,
	}
	out.AppendAsset(ordx, AssetOffsets{
		{Start: 0, End: 10},
	}, nil)

	v2 := out

	// -------- Cut 1: [0,3)
	p1, err := v2.Cut(3)
	if err != nil {
		t.Fatal(err)
	}

	requireAsset(t, p1, "ordx", "ordx-ft", 3)
	requireAsset(t, p1, "ord", "ord-2", 1)

	// -------- Cut 2: [3,7)
	p2, err := v2.Cut(4)
	if err != nil {
		t.Fatal(err)
	}

	requireAsset(t, p2, "ordx", "ordx-ft", 4)
	requireNoAsset(t, p2, "ord", "ord-2")

	// -------- Cut 3: [7,10)
	p3, err := v2.Cut(3)
	if err != nil {
		t.Fatal(err)
	}

	requireAsset(t, p3, "ordx", "ordx-ft", 3)
	requireNoAsset(t, p3, "ord", "ord-2")
}


func TestTxOutputV2_Cut_BRC20_Off2(t *testing.T) {
	out := NewTxOutputV2(10)

	brc := AssetInfo{
		Name: AssetName{Protocol: "brc20", Ticker: "ordi"},
		Amount: *NewDefaultDecimal(50),
		BindingSat: 0,
	}

	out.AppendAsset(brc, AssetOffsets{
		{Start: 8, End: 9},
	}, map[int64]*AssetInfo{
		8: {
			Name: brc.Name,
			Amount: *NewDefaultDecimal(50),
		},
	})

	v2 := out

	_, err := v2.Cut(5)
	if err != nil {
		t.Fatal(err)
	}

	right, err := v2.Cut(5)
	if err != nil {
		t.Fatal(err)
	}

	requireAsset(t, right, "brc20", "ordi", 50)
}

func TestTxOutputV2_Cut_Ord_CrossBoundary(t *testing.T) {
	out := NewTxOutputV2(5)

	ord := AssetInfo{
		Name: AssetName{Protocol: "ord", Ticker: "ord-3"},
		Amount: *NewDefaultDecimal(1),
		BindingSat: 1,
	}

	out.AppendAsset(ord, AssetOffsets{
		{Start: 3, End: 4},
	}, nil)

	v2 := out

	left, err := v2.Cut(3)
	if err != nil {
		t.Fatal(err)
	}
	requireNoAsset(t, left, "ord", "ord-3")

	right, err := v2.Cut(3)
	if err != nil {
		t.Fatal(err)
	}
	requireAsset(t, right, "ord", "ord-3", 1)
}

func TestTxOutputV2_Cut_AtEnd(t *testing.T) {
	out := NewTxOutputV2(5)

	ordx := AssetInfo{
		Name: AssetName{Protocol: "ordx", Ticker: "ordx-ft"},
		Amount: *NewDefaultDecimal(5),
		BindingSat: 1,
	}
	out.AppendAsset(ordx, AssetOffsets{
		{Start: 0, End: 5},
	}, nil)

	v2 := out

	all, err := v2.Cut(5)
	if err != nil {
		t.Fatal(err)
	}
	requireAsset(t, all, "ordx", "ordx-ft", 5)

	none, _ := v2.Cut(5)
	if none != nil {
		t.Fatal("right part should be nil")
	}
}
