package exotic

import (
	"testing"

	"github.com/sat20-labs/indexer/common"
	"github.com/stretchr/testify/assert"
)

func TestIsSatInRange(t *testing.T) {

	size := common.SubsidyInTheory(MAX_SUBSIDY_HEIGHT + 1)
	assert.False(t, size == 0)

	ranges := []*common.Range{
		{
			Start: 100,
			Size:  100,
		},
		{
			Start: 250,
			Size:  50,
		},
	}

	res := IsSatInRange(ranges, 100)
	assert.True(t, res)

	res = IsSatInRange(ranges, 250)
	assert.True(t, res)

	res = IsSatInRange(ranges, 200)
	assert.False(t, res)

	res = IsSatInRange(ranges, 500)
	assert.False(t, res)
}

// func TestGetRanges(t *testing.T) {

// 	size := SubsidyInTheory(MAX_SUBSIDY_HEIGHT+1)
// 	assert.True(t, size ==0)

// 	height := int64(820000)
// 	ranges := GetRangeToBlock(int(height))
// 	sat1 := Sat(ranges[0].Start + ranges[0].Size - 1)
// 	sat2 := Sat(ranges[0].Start + ranges[0].Size)
// 	if sat1.Height() != height || sat2.Height() != height + 1 {
// 		t.Fatalf("Height, get %d", sat2.Height())
// 	}

// 	ranges = GetRangeForBlock(int(height))
// 	sat1 = Sat(ranges[0].Start + ranges[0].Size - 1)
// 	sat2 = Sat(ranges[0].Start + ranges[0].Size)
// 	if sat1.Height() != height || sat2.Height() != height + 1 {
// 		t.Fatalf("Height, get %d", sat2.Height())
// 	}

// 	ranges = GetRangesForAlpha(0, 820000)
// 	for _, rng := range ranges {
// 		sat := Sat(rng.Start)
// 		if !sat.IsAlpha() {
// 			t.Fatalf("Alpha, %d", sat)
// 		}
// 	}

// 	ranges = GetRangesForOmega(0, 820000)
// 	for _, rng := range ranges {
// 		sat := Sat(rng.Start)
// 		if !sat.IsOmega() {
// 			t.Fatalf("Omega, %d", sat)
// 		}
// 	}

// 	//rangemap := GetRangesForRodarmorRarity(6929999)
// 	rangemap := GetRangesForRodarmorRarity(808262)

// 	fmt.Printf("uncommon: %d\n", len(rangemap[Uncommon]))
// 	fmt.Printf("rare: %d\n", len(rangemap[Rare]))
// 	fmt.Printf("epic: %d\n", len(rangemap[Epic]))
// 	fmt.Printf("legendary: %d\n", len(rangemap[Legendary]))
// 	fmt.Printf("mythic: %d\n", len(rangemap[Mythic]))

// 	for _, rng := range rangemap[Uncommon] {
// 		sat := Sat(rng.Start)
// 		satribute := sat.GetRodarmorRarity()
// 		if satribute != Uncommon {
// 			t.Fatalf("Uncommon, get %s, %d", string(satribute), sat)
// 		}
// 	}

// 	for _, rng := range rangemap[Rare] {
// 		sat := Sat(rng.Start)
// 		satribute := sat.GetRodarmorRarity()
// 		if satribute != Rare {
// 			t.Fatalf("Rare, get %s", string(satribute))
// 		}
// 	}

// 	for _, rng := range rangemap[Epic] {
// 		sat := Sat(rng.Start)
// 		satribute := sat.GetRodarmorRarity()
// 		if satribute != Epic {
// 			t.Fatalf("Epic, get %s", string(satribute))
// 		}
// 	}

// 	for _, rng := range rangemap[Legendary] {
// 		sat := Sat(rng.Start)
// 		satribute := sat.GetRodarmorRarity()
// 		if satribute != Legendary {
// 			t.Fatalf("Legendary, get %s", string(satribute))
// 		}
// 	}

// 	for _, rng := range rangemap[Mythic] {
// 		sat := Sat(rng.Start)
// 		satribute := sat.GetRodarmorRarity()
// 		if satribute != Mythic {
// 			t.Fatalf("Mythic, get %s", string(satribute))
// 		}
// 	}

// 	rangemap = GetMoreRodarmorRarityRangesToHeight(808262, 808263)
// 	fmt.Printf("uncommon: %d\n", len(rangemap[Uncommon]))
// 	fmt.Printf("rare: %d\n", len(rangemap[Rare]))
// 	fmt.Printf("epic: %d\n", len(rangemap[Epic]))
// 	fmt.Printf("legendary: %d\n", len(rangemap[Legendary]))
// 	fmt.Printf("mythic: %d\n", len(rangemap[Mythic]))

// }
