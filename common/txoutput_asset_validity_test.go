package common

import (
	"fmt"
	"testing"
)

func TestTxOutput_GetAssetV2PlainSat(t *testing.T) {
	for _, bound := range []bool{false, true} {
		t.Run(fmt.Sprintf("bound=%t", bound), func(t *testing.T) {
			output := NewTxOutput(50_000)
			want := int64(50_000)
			if bound {
				name := newAssetName(PROTOCOL_NAME_ORDX, ASSET_TYPE_FT, "BOUND")
				asset := newAssetInfo(name, 200, 2)
				output.Assets.Add(&asset)
				output.Offsets[name] = AssetOffsets{&OffsetRange{Start: 0, End: 100}}
				want -= 100
			}
			for _, name := range []*AssetName{nil, &ASSET_PLAIN_SAT} {
				amount, invalid := output.GetAssetV2(name)
				if amount == nil || amount.Cmp(NewDefaultDecimal(want)) != 0 {
					t.Fatalf("plain amount = %v, want %d", amount, want)
				}
				if invalid {
					t.Error("spendable plain sats reported as invalid")
				}
			}
		})
	}
}

func TestTxOutput_GetAssetV2AssetValidity(t *testing.T) {
	name := newAssetName(PROTOCOL_NAME_BRC20, ASSET_TYPE_FT, "TOKEN")
	for _, invalid := range []bool{false, true} {
		t.Run(fmt.Sprintf("invalid=%t", invalid), func(t *testing.T) {
			output := NewTxOutput(1000)
			asset := newAssetInfo(name, 123, 0)
			output.Assets.Add(&asset)
			output.Invalids[name] = invalid
			amount, gotInvalid := output.GetAssetV2(&name)
			if amount == nil || amount.Cmp(&asset.Amount) != 0 || gotInvalid != invalid {
				t.Fatalf("GetAssetV2 = (%v, %t), want (123, %t)", amount, gotInvalid, invalid)
			}
			if (output.GetAsset(&name) == nil) != invalid {
				t.Fatal("GetAsset spendability disagrees with the invalid flag")
			}
		})
	}
}
