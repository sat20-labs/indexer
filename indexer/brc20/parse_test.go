package brc20

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/sat20-labs/indexer/common"
	"github.com/stretchr/testify/assert"
)


func GetRawData(txID string, network string) ([][]byte, error) {
	url := ""
	switch network {
	case "testnet":
		url = fmt.Sprintf("https://mempool.space/testnet/api/tx/%s", txID)
	case "testnet4":
		url = fmt.Sprintf("https://mempool.space/testnet4/api/tx/%s", txID)
	case "mainnet":
		url = fmt.Sprintf("https://mempool.space/api/tx/%s", txID)
	default:
		return nil, fmt.Errorf("unsupported network: %s", network)
	}

	response, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve transaction data for %s from the API, error: %v", txID, err)

	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to retrieve transaction data for %s from the API, error: %v", txID, err)
	}

	var data map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response for %s, error: %v", txID, err)
	}
	txWitness := data["vin"].([]interface{})[0].(map[string]interface{})["witness"].([]interface{})

	if len(txWitness) < 2 {
		return nil, fmt.Errorf("failed to retrieve witness for %s", txID)
	}

	var rawData [][]byte = make([][]byte, len(txWitness))
	for i, v := range txWitness {
		rawData[i], err = hex.DecodeString(v.(string))
		if err != nil {
			return nil, fmt.Errorf("failed to decode hex string to byte array for %s, error: %v", txID, err)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex string to byte array for %s, error: %v", txID, err)
	}
	return rawData, nil
}

func TestParser_ord20(t *testing.T) {
	// input 0, output 0
	{
		rawData, err := GetRawData("06ccbac09747a62b2b6c000786b0c5c34ded98d50e8e84986f1c4884bc60e461", "testnet4")
		if err != nil {
			common.Log.Info(err)
			assert.True(t, false)
		}
		fields, envelopes, err := common.ParseInscription(rawData)
		if err != nil {
			common.Log.Info(err)
			assert.True(t, false)
		}

		if len(fields) != 1 {
			assert.True(t, false)
		}

		for i, insc := range envelopes {
			r := IsCursed(insc, i, 70526)
			fmt.Printf("%d is curesed %v\n", i, r)
		}
	}

	{
		rawData, err := GetRawData("94ab452a9716cd4fd557bb6bad845e7b15173e01213caadd01752057ec799bb4", "testnet4")
		if err != nil {
			common.Log.Info(err)
			assert.True(t, false)
		}
		fields, envelopes, err := common.ParseInscription(rawData)
		if err != nil {
			common.Log.Info(err)
			assert.True(t, false)
		}

		if len(fields) != 1 {
			assert.True(t, false)
		}

		for i, insc := range envelopes {
			r := IsCursed(insc, i, 70526)
			fmt.Printf("%d is curesed %v\n", i, r)
		}
	}

}