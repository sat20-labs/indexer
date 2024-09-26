package base

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/sat20-labs/indexer/common"
)

func TestUtxo(t *testing.T) {
	hexstring := "21e48796d17bcab49b1fea7211199c0fa1e296d2ecf4cf2f900cee62153ee331"
	hexBytes, _ := hex.DecodeString(hexstring)
	fmt.Printf("%d, %d, %d", len(hexstring), len([]byte(hexstring)), len(hexBytes))
}

func TestSubsidy(t *testing.T) {
	var tests = []struct {
		height int
		want   int64
	}{
		{0, 5000000000},
		{1, 5000000000},
		{210000, 2500000000},
		{420000, 1250000000},
		{630000, 625000000},
	}

	for _, test := range tests {
		if got := common.SubsidyInTheory(test.height); got != test.want {
			t.Errorf("subsidy(%d) = %d; want: %d", test.height, got, test.want)
		}
	}
}

func TestFirstOrdinal(t *testing.T) {
	var tests = []struct {
		height int
		want   int64
	}{
		{0, 0},
		{1, 5000000000},
		{2, 10000000000},
		{3, 15000000000},
		{210000, 1050000000000000},
		{807680, 1948550000000000},
	}

	for _, test := range tests {
		if got := common.FirstOrdinalInTheory(test.height); got != test.want {
			t.Errorf("common.FirstOrdinalInTheory(%d) = %d; want: %d", test.height, got, test.want)
		}
	}
}

func TestTransferRanges(t *testing.T) {
	var tests = []struct {
		ordinals        []*common.Range
		value           int64
		wantTransferred []*common.Range
		wantRemaining   []*common.Range
	}{
		{
			[]*common.Range{{Start: 0, Size: 100}, {Start: 100, Size: 100}, {Start: 200, Size: 100}},
			150,
			[]*common.Range{{Start: 0, Size: 100}, {Start: 100, Size: 50}},
			[]*common.Range{{Start: 150, Size: 50}, {Start: 200, Size: 100}}},
		{
			[]*common.Range{{Start: 0, Size: 100}},
			50,
			[]*common.Range{{Start: 0, Size: 50}},
			[]*common.Range{{Start: 50, Size: 50}},
		},
	}

	for _, test := range tests {
		gotTransferred, gotRemaining := common.TransferRanges(test.ordinals, test.value)

		for i, r := range gotTransferred {
			e := test.wantTransferred[i]
			if (r.Size != e.Size) || (r.Start != e.Start) {
				t.Errorf("got transferred start: %d size %d; want start %d, size %d", r.Start, r.Size, e.Start, e.Size)
			}
		}

		if len(gotRemaining) != len(test.wantRemaining) {
			t.Errorf("got remaining length: %d; expected length: %d", len(gotRemaining), len(test.wantRemaining))
		}

		for i, r := range gotRemaining {
			e := test.wantRemaining[i]
			if (r.Size != e.Size) || (r.Start != e.Start) {
				t.Errorf("got remaining start: %d, size: %d; expected start: %d, size: %d", r.Start, r.Size, e.Start, e.Size)
			}
		}

		if len(gotTransferred) != len(test.wantTransferred) {
			t.Errorf("got transferred length: %d; expected length: %d", len(gotTransferred), len(test.wantTransferred))
		}
	}
}

func TestOrdinal(t *testing.T) {
	{
		height := 124724
		sat1 := common.FirstOrdinalInTheory(height)
		sat2 := common.FirstOrdinalInTheory(height + 1)
		subsidy := common.SubsidyInTheory(height)
		leak := int64(1000001)
		fmt.Printf("height %d, sat range %d-%d, leak %d leak range %d-%d\n",
			height, sat1, sat2-1, leak, sat1+subsidy-leak, sat2-1)
	}

	{
		height := 501726
		sat1 := common.FirstOrdinalInTheory(height)
		sat2 := common.FirstOrdinalInTheory(height + 1)
		subsidy := common.SubsidyInTheory(height)
		leak := int64(1250000000)
		fmt.Printf("height %d, sat range %d-%d, leak %d leak range %d-%d\n",
			height, sat1, sat2-1, leak, sat1+subsidy-leak, sat2-1)
	}

	{
		height := 626205
		sat1 := common.FirstOrdinalInTheory(height)
		sat2 := common.FirstOrdinalInTheory(height + 1)
		subsidy := common.SubsidyInTheory(height)
		leak := int64(68)
		fmt.Printf("height %d, sat range %d-%d, leak %d leak range %d-%d\n",
			height, sat1, sat2-1, leak, sat1+subsidy-leak, sat2-1)
	}

	{
		height := 827307
		sat1 := common.FirstOrdinalInTheory(height)
		sat2 := common.FirstOrdinalInTheory(height + 1)
		subsidy := common.SubsidyInTheory(height)
		leak := int64(0)
		fmt.Printf("height %d, sat range %d-%d, leak %d leak range %d-%d\n",
			height, sat1, sat2-1, leak, sat1+subsidy-leak, sat2-1)
	}

}

func TestAppendRanges(t *testing.T) {

	rngs1 := []*common.Range{{Start: 0, Size: 100}, {Start: 100, Size: 100}, {Start: 200, Size: 100}}
	rngs2 := []*common.Range{{Start: 300, Size: 100}, {Start: 400, Size: 90}, {Start: 500, Size: 100}}

	r := appendRanges(rngs1, nil)
	printRanges(r)

	r = appendRanges(nil, rngs2)
	printRanges(r)

	r = appendRanges(rngs1, rngs2)
	printRanges(r)
}

func printRanges(rngs []*common.Range) {
	for _, r := range rngs {
		fmt.Printf("%d-%d...", r.Start, r.Start+r.Size)
	}
	fmt.Printf("\n")
}
