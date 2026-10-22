package common

import (
	"encoding/json"
	"testing"
)

func TestAtomicalsCapacityDoesNotImplyOrdinalOffsets(t *testing.T) {
	name := AssetName{Protocol: PROTOCOL_NAME_ATOM, Type: ASSET_TYPE_FT, Ticker: "quark"}
	output := NewTxOutput(600)
	asset := AssetInfo{Name: name, Amount: *NewDefaultDecimal(100), BindingSat: 1}
	output.Assets.Add(&asset)
	if !IsBindingSat(&name) || output.GetPlainSat() != 500 {
		t.Fatal("ARC20 must retain its one-token-per-satoshi capacity")
	}
	if _, _, err := output.Cut(330); err == nil {
		t.Fatal("ordinal cutting accepted native ARC20 state")
	}
	if _, _, err := output.Split(&name, 0, NewDefaultDecimal(50)); err == nil {
		t.Fatal("ordinal splitting accepted native ARC20 state")
	}
	if _, err := output.GetAssetOffset(&name, NewDefaultDecimal(50)); err == nil {
		t.Fatal("ARC20 returned an invented ordinal offset")
	}
	whole, remainder, err := output.Cut(output.Value())
	if err != nil || remainder != nil || whole.GetAsset(&name).Int64() != 100 {
		t.Fatalf("whole-output copy lost native asset: %+v, %+v, %v", whole, remainder, err)
	}
	// Existing JSON fields carry native quantities without synthetic ranges.
	data, err := json.Marshal(output.ToAssetsInUtxo())
	if err != nil {
		t.Fatal(err)
	}
	var view AssetsInUtxo
	if err := json.Unmarshal(data, &view); err != nil {
		t.Fatal(err)
	}
	if len(view.Assets) != 1 || view.Assets[0].Amount != "100" || view.Assets[0].BindingSat != 1 || view.Assets[0].Precision != 0 || len(view.Assets[0].Offsets) != 0 {
		t.Fatalf("unexpected existing API representation: %s", data)
	}
	if got := view.ToTxOutput(); got.GetAsset(&name).Int64() != 100 || len(got.Offsets) != 0 {
		t.Fatal("existing API round trip changed ARC20 facts")
	}
	unsupported := name
	unsupported.Type = ASSET_TYPE_NFT
	if IsBindingSat(&unsupported) || IsAtomicalsFT(&unsupported) {
		t.Fatal("unsupported Atomicals NFT treated as ARC20")
	}
}

func TestAtomicalsCapacityRemainsVisibleAlongsideOrdinalAssets(t *testing.T) {
	arc := AssetName{Protocol: PROTOCOL_NAME_ATOM, Type: ASSET_TYPE_FT, Ticker: "quark"}
	ord := AssetName{Protocol: PROTOCOL_NAME_ORDX, Type: ASSET_TYPE_FT, Ticker: "sat20"}
	output := NewTxOutput(1000)
	output.Assets.Add(&AssetInfo{Name: arc, Amount: *NewDefaultDecimal(100), BindingSat: 1})
	output.Assets.Add(&AssetInfo{Name: ord, Amount: *NewDefaultDecimal(200), BindingSat: 1})
	output.Offsets[ord] = AssetOffsets{{Start: 0, End: 200}}
	if output.SizeOfBindingSats() < 300 {
		t.Fatal("mixed output silently classified native ARC20 capacity as plain sats")
	}
	if _, _, err := output.Cut(500); err == nil {
		t.Fatal("ordinal cutting accepted an output co-located with ARC20")
	}
}
