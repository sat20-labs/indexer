package common

import (
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
