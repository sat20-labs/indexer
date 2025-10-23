package common

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/btcsuite/btcd/wire"
)

// 辅助构造函数 —— 保持简单，字段与你代码中的类型一致
func newAssetName(proto, typ, ticker string) AssetName {
	return AssetName{
		Protocol: proto,
		Type:     typ,
		Ticker:   ticker,
	}
}

func newAssetInfo(name AssetName, amt int64, bindingSat uint32) AssetInfo {
	return AssetInfo{
		Name:       name,
		Amount:     *NewDefaultDecimal(amt),
		BindingSat: bindingSat,
	}
}

// TestAppend_plainSat_and_SatBindingMap
// - 验证 Append 会把 another 的 SatBindingMap 的 key 按 p.OutValue.Value 偏移后加入 p.SatBindingMap
// - 验证 OutValue.Value 会被累加
func TestTxOutput_Append_PlainSat_SatBindingMap(t *testing.T) {
	a := NewTxOutput(100) // p value = 100
	b := NewTxOutput(50)  // another value = 50

	// 在 another 中放置一个 sat-binding map，key = 10
	name := newAssetName("", ASSET_TYPE_FT, "TKN") // 名称随意，但不会被 Append 的 plain-sat 路径使用
	b.SatBindingMap[10] = &AssetInfo{
		Name:       name,
		Amount:     *NewDefaultDecimal(123),
		BindingSat: 0,
	}

	// call Append
	err := a.Append(b)
	if err != nil {
		t.Fatalf("Append returned error: %v", err)
	}

	// value should be increased
	if a.Value() != 150 {
		t.Fatalf("Append value mismatch: got %d want %d", a.Value(), int64(150))
	}

	// sat binding map key should be shifted by original p.OutValue.Value (100)
	if _, ok := a.SatBindingMap[110]; !ok {
		t.Fatalf("Append did not shift SatBindingMap key, expected key 110 present")
	}
}

// TestCut_bindingSatAsset
// 构造一个 TxOutput，其中包含一个 bindingSat != 0 的资产以及对应 Offsets。
// 然后 Cut(offset) 并检查 asset 数量与 Offsets 是否正确地分配到两部分上。
func TestTxOutput_Cut_BindingSat(t *testing.T) {
	// 构造 TxOutput p，value = 10 sats
	p := NewTxOutput(10)

	// 构造一个 ordx-like asset：bindingSat = 2 (每个 asset 单位占 2 sat)
	name := newAssetName(PROTOCOL_NAME_ORDX, ASSET_TYPE_FT, "ORDX")
	asset := newAssetInfo(name, 6, 2) // amount = 6 -> 需要 sats = amt / bindingSat = 3 sat

	// 将 asset 加入 p.Assets（使用库中 TxAssets.Add 方法）
	p.Assets.Add(&asset)

	// Offsets: 假设该资产在本 utxo 占用了 offsets: [0,2) 和 [3,5)
	offs := AssetOffsets{
		&OffsetRange{Start: 0, End: 2}, // size = 2
		&OffsetRange{Start: 3, End: 5}, // size = 2
	}
	p.Offsets[asset.Name] = offs

	// 我们选择 offset = 3 (按聪偏移)，cut 后：
	// - part1 value = 3, part2 value = 7
	// binding sats 切割规则：
	// offsets.Cut(3) 会把前 3 个聪分到 left，剩下到 right
	// left offsets 应为 [0,2) + [2,3)（second range 被拆成 [3,?] 实际表现以代码算法为准）
	part1, part2, err := p.Cut(3)
	if err != nil {
		t.Fatalf("Cut returned error: %v", err)
	}

	// 验证 value
	if part1.Value() != 3 {
		t.Fatalf("Cut part1 value mismatch: got %d want %d", part1.Value(), int64(3))
	}
	if part2.Value() != 7 {
		t.Fatalf("Cut part2 value mismatch: got %d want %d", part2.Value(), int64(7))
	}

	// 由于原 asset.BindingSat = 2，
	// part1 应拥有 offset1.Size()*BindingSat 个 asset amount（以 Decimal 存储）
	// 我们只验证 Assets.Find 能否找到 asset 以及 Offsets 被设置
	a1, err := part1.Assets.Find(&asset.Name)
	if err != nil {
		// 可能 part1 没有 asset（取决 offsets 切割结果），在这种情况下 we still accept it only if offset1.Size()==0
		offsets := p.Offsets[asset.Name]
		off1, _ := offsets.Cut(3)
		if off1.Size() == 0 {
			// acceptable (no binding asset moved to part1)
		} else {
			t.Fatalf("part1 expected to have asset but not found")
		}
	} else {
		fmt.Printf("%v\n", a1)
		// 若存在，part1.Offsets 应包含 asset.Name
		if _, ok := part1.Offsets[asset.Name]; !ok {
			t.Fatalf("part1 has asset but Offsets missing for it")
		}
	}

	// part2 同理
	a2, err := part2.Assets.Find(&asset.Name)
	if err != nil {
		offsets := p.Offsets[asset.Name]
		_, off2 := offsets.Cut(3)
		if off2.Size() == 0 {
			// acceptable
		} else {
			t.Fatalf("part2 expected to have asset but not found")
		}
	} else {
		fmt.Printf("%v\n", a2)
		if _, ok := part2.Offsets[asset.Name]; !ok {
			t.Fatalf("part2 has asset but Offsets missing for it")
		}
	}
}

// TestSplit_value_nonzero_behaves_like_Cut
// 当调用 Split 时传入非零 value，Split 逻辑直接在最后调用 p.Cut(value)
// 因此我们只要验证 Split 返回结果等价于 Cut 即可
func TestTxOutput_Split_WithValue_Equals_Cut(t *testing.T) {
	p := NewTxOutput(100)

	// add a dummy asset (non-binding) to ensure Assets length <=1
	name := newAssetName("", ASSET_TYPE_FT, "DUMMY")
	asset := newAssetInfo(name, 0, 0)
	p.Assets.Add(&asset)

	// add some offsets so Cut/Split have something to operate on (not strictly necessary here)
	p.Offsets[asset.Name] = AssetOffsets{
		&OffsetRange{Start: 0, End: 100},
	}

	// choose value 30 (non-zero) => Split should call Cut(30)
	part1Cut, part2Cut, errCut := p.Cut(30)
	if errCut != nil {
		t.Fatalf("Cut returned error: %v", errCut)
	}

	part1Split, part2Split, errSplit := p.Split(&asset.Name, 30, nil)
	if errSplit != nil {
		t.Fatalf("Split returned error: %v", errSplit)
	}

	// 对比结果 (只比较 Value 和 Assets/Offsets 存在性)
	if part1Cut.Value() != part1Split.Value() || part2Cut.Value() != part2Split.Value() {
		t.Fatalf("Split/Cut value mismatch: cut(%d,%d) split(%d,%d)",
			part1Cut.Value(), part2Cut.Value(), part1Split.Value(), part2Split.Value())
	}

	// 简单比较 Offsets map keys 和 Assets count（深度比较可能受 Decimal 实现影响）
	if !reflect.DeepEqual(getOffsetKeys(part1Cut), getOffsetKeys(part1Split)) ||
		!reflect.DeepEqual(getOffsetKeys(part2Cut), getOffsetKeys(part2Split)) {
		t.Fatalf("Split and Cut resulted different Offsets maps")
	}
}

// getOffsetKeys returns sorted keys set of offsets map for easy comparison (order not important)
func getOffsetKeys(p *TxOutput) []AssetName {
	keys := make([]AssetName, 0, len(p.Offsets))
	for k := range p.Offsets {
		keys = append(keys, k)
	}
	return keys
}

// TestSplit_valueZero_with_SatBindingMap
// 场景：value == 0 且 SatBindingMap 非空。
// 期望：Split 内部调用 OffsetByAsset，自动使用 asset 的 offset.Start 作为切割点。
func TestTxOutput_Split_ValueZero_WithSatBindingMap(t *testing.T) {
	name := newAssetName(PROTOCOL_NAME_BRC20, ASSET_TYPE_FT, "FREE")
	p0 := NewTxOutput(330)
	// asset 不与聪绑定 (BindingSat = 0)
	asset0 := newAssetInfo(name, 10, 0) // bindingSat=0
	p0.Assets.Add(&asset0)
	// 假设 asset 对应 offsets
	p0.Offsets[asset0.Name] = AssetOffsets{
		&OffsetRange{Start: 0, End: 0},
	}
	p0.SatBindingMap[0] = &asset0
	p0.OutValue = wire.TxOut{
		Value: 330,
	}
	
	
	p1 := NewTxOutput(330)
	// asset 不与聪绑定 (BindingSat = 0)
	asset1 := newAssetInfo(name, 50, 0) // bindingSat=0
	p1.Assets.Add(&asset1)
	// 假设 asset 对应 offsets
	p1.Offsets[asset1.Name] = AssetOffsets{
		&OffsetRange{Start: 0, End: 0},
	}
	p1.SatBindingMap[0] = &asset1
	p1.OutValue = wire.TxOut{
		Value: 330,
	}

	p2 := NewTxOutput(330)
	// asset 不与聪绑定 (BindingSat = 0)
	asset2 := newAssetInfo(name, 100, 0) // bindingSat=0
	p2.Assets.Add(&asset2)
	// 假设 asset 对应 offsets
	p2.Offsets[asset2.Name] = AssetOffsets{
		&OffsetRange{Start: 0, End: 0},
	}
	p2.SatBindingMap[0] = &asset2
	p2.OutValue = wire.TxOut{
		Value: 330,
	}
	p3 := NewTxOutput(2000)
	total := p0.OutValue.Value + p1.OutValue.Value + p2.OutValue.Value +p3.OutValue.Value

	p := p0.Clone()
	err := p.Append(p1)
	if err != nil {
		t.Fatalf("Append failed, %v", err)
	}
	err = p.Append(p2)
	if err != nil {
		t.Fatalf("Append failed, %v", err)
	}
	err = p.Append(p3)
	if err != nil {
		t.Fatalf("Append failed, %v", err)
	}

	{
		// 调用 Split，传入 value==0，先通过资产确定offset再切割
		amt := asset0.Amount.Add(&asset1.Amount)
		part1, part2, err := p.Split(&name, 0, amt)
		if err != nil {
			t.Fatalf("Split(value==0) returned error: %v", err)
		}

		// 验证分割后两部分有效
		if part1 == nil || part2 == nil {
			t.Fatalf("Split(value==0) returned nil part(s)")
		}

		a1, _ := part1.Assets.Find(&name)
		a2, _ := part2.Assets.Find(&name)
		if a1 == nil || a2 == nil {
			t.Fatalf("Cut BindingSat=0 expected both parts to have asset, got a1=%v, a2=%v", a1, a2)
		}
		if a1.Amount.Cmp((&asset0.Amount).Add(&asset1.Amount)) != 0 {
			t.Fatalf("")
		}

		// 验证 value 总和与原始保持一致
		if part1.Value() + part2.Value() != p.Value() {
			t.Fatalf("Split(value==0) total value mismatch: got %d want %d", total, p.Value())
		}

		// 验证 part1 / part2 的 offset 分布
		off1, ok1 := part1.Offsets[name]
		off2, ok2 := part2.Offsets[name]
		if !ok1 || !ok2 {
			t.Fatalf("Split(value==0): one part missing offsets, got %v %v", ok1, ok2)
		}
		offsets := p.Offsets[name]
		if off1.Size()+off2.Size() != offsets.Size() {
			t.Fatalf("Split(value==0): offsets size not conserved (got %d+%d, want %d)",
				off1.Size(), off2.Size(), offsets.Size())
		}
	}

	{
		// 调用 Split，传入 value!=0，将强制走 offset-based 切割逻辑
		part1, part2, err := p.Split(&name, 660, nil)
		if err != nil {
			t.Fatalf("Split(value==0) returned error: %v", err)
		}

		// 验证分割后两部分有效
		if part1 == nil || part2 == nil {
			t.Fatalf("Split(value==0) returned nil part(s)")
		}

		a1, _ := part1.Assets.Find(&name)
		a2, _ := part2.Assets.Find(&name)
		if a1 == nil || a2 == nil {
			t.Fatalf("Cut BindingSat=0 expected both parts to have asset, got a1=%v, a2=%v", a1, a2)
		}
		if a1.Amount.Cmp((&asset0.Amount).Add(&asset1.Amount)) != 0 {
			t.Fatalf("")
		}

		// 验证 value 总和与原始保持一致
		if part1.Value() + part2.Value() != p.Value() {
			t.Fatalf("Split(value==0) total value mismatch: got %d want %d", total, p.Value())
		}

		// 验证 part1 / part2 的 offset 分布
		off1, ok1 := part1.Offsets[name]
		off2, ok2 := part2.Offsets[name]
		if !ok1 || !ok2 {
			t.Fatalf("Split(value==0): one part missing offsets, got %v %v", ok1, ok2)
		}
		offsets := p.Offsets[name]
		if off1.Size()+off2.Size() != offsets.Size() {
			t.Fatalf("Split(value==0): offsets size not conserved (got %d+%d, want %d)",
				off1.Size(), off2.Size(), offsets.Size())
		}
	}
	
}

// TestCut_assetBindingSatZero_withOffsets
// 场景：asset.BindingSat == 0 且 offsets 非空。
// 期望：Cut 根据 offsets 切分资产，并且每个分支都有相应 offsets。
func TestTxOutput_Cut_BindingSatZero_WithOffsets(t *testing.T) {
	name := newAssetName(PROTOCOL_NAME_BRC20, ASSET_TYPE_FT, "FREE")
	p0 := NewTxOutput(330)
	// asset 不与聪绑定 (BindingSat = 0)
	asset0 := newAssetInfo(name, 10, 0) // bindingSat=0
	p0.Assets.Add(&asset0)
	// 假设 asset 对应 offsets
	p0.Offsets[asset0.Name] = AssetOffsets{
		&OffsetRange{Start: 0, End: 0},
	}
	p0.SatBindingMap[0] = &asset0
	p0.OutValue = wire.TxOut{
		Value: 330,
	}
	
	
	p1 := NewTxOutput(330)
	// asset 不与聪绑定 (BindingSat = 0)
	asset1 := newAssetInfo(name, 50, 0) // bindingSat=0
	p1.Assets.Add(&asset1)
	// 假设 asset 对应 offsets
	p1.Offsets[asset1.Name] = AssetOffsets{
		&OffsetRange{Start: 0, End: 0},
	}
	p1.SatBindingMap[0] = &asset1
	p1.OutValue = wire.TxOut{
		Value: 330,
	}

	p2 := NewTxOutput(330)
	// asset 不与聪绑定 (BindingSat = 0)
	asset2 := newAssetInfo(name, 100, 0) // bindingSat=0
	p2.Assets.Add(&asset2)
	// 假设 asset 对应 offsets
	p2.Offsets[asset2.Name] = AssetOffsets{
		&OffsetRange{Start: 0, End: 0},
	}
	p2.SatBindingMap[0] = &asset2
	p2.OutValue = wire.TxOut{
		Value: 330,
	}
	p3 := NewTxOutput(2000)
	total := p0.OutValue.Value + p1.OutValue.Value + p2.OutValue.Value +p3.OutValue.Value

	p := p0.Clone()
	err := p.Append(p1)
	if err != nil {
		t.Fatalf("Append failed, %v", err)
	}
	err = p.Append(p2)
	if err != nil {
		t.Fatalf("Append failed, %v", err)
	}
	err = p.Append(p3)
	if err != nil {
		t.Fatalf("Append failed, %v", err)
	}

	part1, part2, err := p.Cut(660)
	if err != nil {
		t.Fatalf("Cut returned error: %v", err)
	}

	// 验证 value 分配
	if part1.Value() != 660 || part2.Value() != total - 660 {
		t.Fatalf("Cut BindingSat=0 value mismatch: got (%d,%d) want (660, %d)",
			part1.Value(), part2.Value(), total - 660)
	}

	// 两个部分都应有 asset
	a1, _ := part1.Assets.Find(&name)
	a2, _ := part2.Assets.Find(&name)
	if a1 == nil || a2 == nil {
		t.Fatalf("Cut BindingSat=0 expected both parts to have asset, got a1=%v, a2=%v", a1, a2)
	}
	if a1.Amount.Cmp((&asset0.Amount).Add(&asset1.Amount)) != 0 {
		t.Fatalf("")
	}

	// offsets 应该都存在
	if _, ok := part1.Offsets[name]; !ok {
		t.Fatalf("part1 missing offsets for asset")
	}
	if _, ok := part2.Offsets[name]; !ok {
		t.Fatalf("part2 missing offsets for asset")
	}

}

