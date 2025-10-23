package common

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

func TestTxAssets(t *testing.T) {
	inputAssets := []TxAssets{
		{
			{
				Name: AssetName{
					Protocol: "runes",
					Type: "f",
					Ticker: "65103_1",
				},
				Amount: *NewDecimal(200, 0),
				BindingSat: 0,
			},
		},
		nil,
	}

	inputValues := []int64{710, 10000}

	outputAssets := []TxAssets{
		{
			{
				Name: AssetName{
					Protocol: "ordx",
					Type: "f",
					Ticker: "dogcoin",
				},
				Amount: *NewDecimal(10000, 0),
				BindingSat: 1,
			},
		},
		nil,
		nil,
		nil,
	}

	outputValues := []int64{3084, 10, 2413, 0}


	var totalInTxAssets TxAssets
	var totalSatoshiIn int64

	for _, assets := range inputAssets {
		totalInTxAssets.Merge(assets)
	}
	for _, value := range inputValues {
		totalSatoshiIn += value
	}

	var totalSatoshiOut int64
	for i, out := range outputAssets{
		err := totalInTxAssets.Split(out)
		if err != nil {
			t.Fatalf("invalid TxOut asset with index %d, (%s)", i, err.Error())
		}
		totalSatoshiOut += outputValues[i]
	}

	if totalSatoshiOut > totalSatoshiIn {
		t.Fatal()
	}

}

func TestRangeMerge(t *testing.T) {
	dest := []*Range{
		{Start: 10, Size: 5},  // [10, 14]
		{Start: 20, Size: 5},  // [20, 24]
	}
	newRange := &Range{Start: 13, Size: 10} // [13, 22]

	merged := MergeRange(dest, newRange)
	fmt.Printf("%v", merged)
}

func TestMergeRange(t *testing.T) {
	tests := []struct {
		name     string
		input    []*Range
		newRange *Range
		expected []*Range
	}{
		{
			name:     "empty input",
			input:    nil,
			newRange: &Range{Start: 5, Size: 3}, // [5,7]
			expected: []*Range{{Start: 5, Size: 3}},
		},
		{
			name: "non-overlapping insert at end",
			input: []*Range{
				{Start: 0, Size: 3}, // [0,2]
				{Start: 10, Size: 2}, // [10,11]
			},
			newRange: &Range{Start: 20, Size: 5}, // [20,24]
			expected: []*Range{
				{Start: 0, Size: 3},
				{Start: 10, Size: 2},
				{Start: 20, Size: 5},
			},
		},
		{
			name: "merge with overlap",
			input: []*Range{
				{Start: 10, Size: 5}, // [10,14]
				{Start: 20, Size: 5}, // [20,24]
			},
			newRange: &Range{Start: 13, Size: 10}, // [13,22]
			expected: []*Range{
				{Start: 10, Size: 15}, // [10,24]
			},
		},
		{
			name: "merge with multiple overlaps",
			input: []*Range{
				{Start: 10, Size: 5}, // [10,14]
				{Start: 16, Size: 4}, // [16,19]
				{Start: 21, Size: 3}, // [21,23]
			},
			newRange: &Range{Start: 13, Size: 10}, // [13,22]
			expected: []*Range{
				{Start: 10, Size: 14}, // [10,23]
			},
		},
		{
			name: "adjacent ranges should merge",
			input: []*Range{
				{Start: 0, Size: 5}, // [0,4]
			},
			newRange: &Range{Start: 5, Size: 3}, // [5,7]
			expected: []*Range{
				{Start: 0, Size: 8}, // [0,7]
			},
		},
		{
			name: "insert inside existing range",
			input: []*Range{
				{Start: 10, Size: 10}, // [10,19]
			},
			newRange: &Range{Start: 13, Size: 2}, // [13,14]
			expected: []*Range{
				{Start: 10, Size: 10},
			},
		},
		{
			name: "surround existing range",
			input: []*Range{
				{Start: 20, Size: 3}, // [20,22]
			},
			newRange: &Range{Start: 18, Size: 10}, // [18,27]
			expected: []*Range{
				{Start: 18, Size: 10}, // [18,27]
			},
		},
		{
			name: "merge all overlapping and non-overlapping ranges",
			input: []*Range{
				{Start: 1, Size: 2},  // [1,2]
				{Start: 4, Size: 2},  // [4,5]
				{Start: 10, Size: 2}, // [10,11]
			},
			newRange: &Range{Start: 2, Size: 3}, // [2,4]
			expected: []*Range{
				{Start: 1, Size: 5}, // [1,5]
				{Start: 10, Size: 2},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := MergeRange(test.input, test.newRange)
			if !equalRanges(result, test.expected) {
				t.Errorf("Expected %+v, got %+v", test.expected, result)
			}
		})
	}
}

// Helper to compare slices of ranges
func equalRanges(a, b []*Range) bool {
	if len(a) != len(b) {
		return false
	}
	sort.Slice(a, func(i, j int) bool { return a[i].Start < a[j].Start })
	sort.Slice(b, func(i, j int) bool { return b[i].Start < b[j].Start })

	for i := range a {
		if a[i].Start != b[i].Start || a[i].Size != b[i].Size {
			return false
		}
	}
	return true
}


func TestAssetOffsets_Cat(t *testing.T) {
	tests := []struct {
		name     string
		initial  AssetOffsets
		input    *OffsetRange
		expected AssetOffsets
	}{
		{
			name:     "merge adjacent ranges",
			initial:  AssetOffsets{{Start: 0, End: 10}},
			input:    &OffsetRange{Start: 10, End: 20},
			expected: AssetOffsets{{Start: 0, End: 20}},
		},
		{
			name:     "non-adjacent - append",
			initial:  AssetOffsets{{Start: 0, End: 10}},
			input:    &OffsetRange{Start: 11, End: 20},
			expected: AssetOffsets{{Start: 0, End: 10}, {Start: 11, End: 20}},
		},
		{
			name:     "empty offsets",
			initial:  AssetOffsets{},
			input:    &OffsetRange{Start: 5, End: 10},
			expected: AssetOffsets{{Start: 5, End: 10}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initial.Cat(tt.input)
			if !reflect.DeepEqual(tt.initial, tt.expected) {
				t.Errorf("expected %+v, got %+v", tt.expected, tt.initial)
			}
		})
	}
}

func TestAssetOffsets_Insert(t *testing.T) {
	tests := []struct {
		name     string
		initial  AssetOffsets
		input    *OffsetRange
		expected AssetOffsets
	}{
		{
			name:    "insert non-overlapping range",
			initial: AssetOffsets{{Start: 0, End: 5}, {Start: 10, End: 15}},
			input:   &OffsetRange{Start: 6, End: 9},
			expected: AssetOffsets{
				{Start: 0, End: 5},
				{Start: 6, End: 9},
				{Start: 10, End: 15},
			},
		},
		{
			name:    "insert and merge with previous",
			initial: AssetOffsets{{Start: 0, End: 5}, {Start: 10, End: 15}},
			input:   &OffsetRange{Start: 5, End: 8},
			expected: AssetOffsets{
				{Start: 0, End: 8},
				{Start: 10, End: 15},
			},
		},
		{
			name:    "insert and merge with next",
			initial: AssetOffsets{{Start: 0, End: 5}, {Start: 10, End: 15}},
			input:   &OffsetRange{Start: 8, End: 10},
			expected: AssetOffsets{
				{Start: 0, End: 5},
				{Start: 8, End: 15},
			},
		},
		{
			name:    "insert and merge with both",
			initial: AssetOffsets{{Start: 0, End: 5}, {Start: 10, End: 15}},
			input:   &OffsetRange{Start: 5, End: 10},
			expected: AssetOffsets{
				{Start: 0, End: 15},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initial := append(AssetOffsets{}, tt.initial...) // copy
			initial.Insert(tt.input)
			if !reflect.DeepEqual(initial, tt.expected) {
				t.Errorf("expected %+v, got %+v", tt.expected, initial)
			}
		})
	}
}

func TestAssetOffsets_Append(t *testing.T) {
	tests := []struct {
		name     string
		initial  AssetOffsets
		append   AssetOffsets
		expected AssetOffsets
	}{
		{
			name:     "append non-adjacent ranges",
			initial:  AssetOffsets{{Start: 0, End: 5}},
			append:   AssetOffsets{{Start: 6, End: 10}},
			expected: AssetOffsets{{Start: 0, End: 5}, {Start: 6, End: 10}},
		},
		{
			name:     "append adjacent - should merge",
			initial:  AssetOffsets{{Start: 0, End: 5}},
			append:   AssetOffsets{{Start: 5, End: 10}},
			expected: AssetOffsets{{Start: 0, End: 10}},
		},
		{
			name:     "append with overlap should not merge (caller responsibility)",
			initial:  AssetOffsets{{Start: 0, End: 5}},
			append:   AssetOffsets{{Start: 4, End: 10}},
			expected: AssetOffsets{{Start: 0, End: 5}, {Start: 4, End: 10}}, // <- this is how your code behaves
		},
		{
			name:     "append empty second",
			initial:  AssetOffsets{{Start: 0, End: 5}},
			append:   AssetOffsets{},
			expected: AssetOffsets{{Start: 0, End: 5}},
		},
		{
			name:     "append to empty",
			initial:  AssetOffsets{},
			append:   AssetOffsets{{Start: 2, End: 6}},
			expected: AssetOffsets{{Start: 2, End: 6}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initial := append(AssetOffsets{}, tt.initial...) // copy
			initial.Append(tt.append)
			if !reflect.DeepEqual(initial, tt.expected) {
				t.Errorf("expected %+v, got %+v", tt.expected, initial)
			}
		})
	}
}



func TestAssetOffsets_Cut(t *testing.T) {
	tests := []struct {
		name     string
		offsets  AssetOffsets
		value    int64
		expectedL AssetOffsets
		expectedR AssetOffsets
	}{
		{
			name: "cut inside one range",
			offsets: AssetOffsets{
				{Start: 0, End: 10},
			},
			value: 5,
			expectedL: AssetOffsets{
				{Start: 0, End: 5},
			},
			expectedR: AssetOffsets{
				{Start: 0, End: 5},
			},
		},
		{
			name: "cut at boundary",
			offsets: AssetOffsets{
				{Start: 0, End: 5},
				{Start: 5, End: 10},
			},
			value: 5,
			expectedL: AssetOffsets{
				{Start: 0, End: 5},
			},
			expectedR: AssetOffsets{
				{Start: 0, End: 5},
			},
		},
		{
			name: "cut beyond first range",
			offsets: AssetOffsets{
				{Start: 0, End: 5},
				{Start: 5, End: 15},
			},
			value: 7,
			expectedL: AssetOffsets{
				{Start: 0, End: 5},
				{Start: 5, End: 7},
			},
			expectedR: AssetOffsets{
				{Start: 0, End: 8},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left, right := tt.offsets.Cut(tt.value)
			if !reflect.DeepEqual(left, tt.expectedL) {
				t.Errorf("Cut() left wrong:\n got: %+v\nwant: %+v", left, tt.expectedL)
			}
			if !reflect.DeepEqual(right, tt.expectedR) {
				t.Errorf("Cut() right wrong:\n got: %+v\nwant: %+v", right, tt.expectedR)
			}
		})
	}
}

func TestAssetOffsets_Split(t *testing.T) {
	tests := []struct {
		name     string
		offsets  AssetOffsets
		amt      int64
		expectedL AssetOffsets
		expectedR AssetOffsets
	}{
		{
			name: "split inside range",
			offsets: AssetOffsets{
				{Start: 0, End: 10},
			},
			amt: 5,
			expectedL: AssetOffsets{
				{Start: 0, End: 5},
			},
			expectedR: AssetOffsets{
				{Start: 0, End: 5},
			},
		},
		{
			name: "split across two ranges",
			offsets: AssetOffsets{
				{Start: 0, End: 10},
				{Start: 10, End: 20},
			},
			amt: 15,
			expectedL: AssetOffsets{
				{Start: 0, End: 10},
				{Start: 10, End: 15},
			},
			expectedR: AssetOffsets{
				{Start: 0, End: 5},
			},
		},
		{
			name: "split exactly at boundary",
			offsets: AssetOffsets{
				{Start: 0, End: 5},
				{Start: 5, End: 10},
			},
			amt: 5,
			expectedL: AssetOffsets{
				{Start: 0, End: 5},
			},
			expectedR: AssetOffsets{
				{Start: 0, End: 5},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left, right := tt.offsets.Split(tt.amt)
			if !reflect.DeepEqual(left, tt.expectedL) {
				t.Errorf("Split() left wrong:\n got: %+v\nwant: %+v", left, tt.expectedL)
			}
			if !reflect.DeepEqual(right, tt.expectedR) {
				t.Errorf("Split() right wrong:\n got: %+v\nwant: %+v", right, tt.expectedR)
			}
		})
	}
}

