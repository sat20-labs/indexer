package brc20

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/brc20/validate"
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


type nftItem struct {
	Id                 int64  `json:"id"`
	Name               string `json:"name"`
	Sat                int64  `json:"sat"`
	Address            string `json:"address"`
	InscriptionId      string `json:"inscriptionId"`
	OutPoint           int64  `json:"outpoint"`
	Utxo               string `json:"utxo"`
	Value              int64  `json:"value"`
	BlockHeight        int    `json:"height"`
	BlockTime          int64  `json:"time"`
	InscriptionAddress string `json:"inscriptionAddress"`
	CurseType          int    `json:"curse,omitempty"`
}

type nftInfo struct {
	nftItem
	ContentType  []byte `json:"contenttype"`
	Content      []byte `json:"content"`
	MetaProtocol []byte `json:"metaprotocol"`
	MetaData     []byte `json:"metadata"`
	Parent       string `json:"parent"`
	Delegate     string `json:"delegate"`
}


type BaseResp struct {
	Code int    `json:"code" example:"0"`
	Msg  string `json:"msg" example:"ok"`
}

type NftInfoResp struct {
	BaseResp
	Data *nftInfo `json:"data"`
}



func GetInscription(id int64, host string) (*nftInfo, error) {
	
	url := fmt.Sprintf("%s/nft/nftid/%d", host, id)

	response, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve data for %s from the API, error: %v", url, err)

	}
	defer response.Body.Close()
	

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to retrieve data for %s from the API, error: %v", url, err)
	}

	respBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var data NftInfoResp
	err = json.Unmarshal(respBytes, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response for %s, error: %v", url, err)
	}
	
	return data.Data, nil
}


type NftStatusData struct {
	Version string     `json:"version"`
	Total   uint64     `json:"total"`
	Start   uint64     `json:"start"`
	Nfts    []*nftItem `json:"nfts"`
}


type NftStatusResp struct {
	BaseResp
	Data *NftStatusData `json:"data"`
}

func GetNftStatus(host string) (*NftStatusData, error) {
	
	url := fmt.Sprintf("%s/nft/status", host)

	response, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve data for %s from the API, error: %v", url, err)

	}
	defer response.Body.Close()
	

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to retrieve data for %s from the API, error: %v", url, err)
	}

	respBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var data NftStatusResp
	err = json.Unmarshal(respBytes, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response for %s, error: %v", url, err)
	}
	
	return data.Data, nil
}


func TestCompareInscription(t *testing.T) {
	//url0 := "http://192.168.1.101:8019/btc/testnet"
	host1 := "http://127.0.0.1:8019/btc/testnet"
	host2 := "http://127.0.0.1:8029/btc/testnet"

	status, err := GetNftStatus(host1)
	if err != nil {
		fmt.Printf("GetNftStatus failed, %v\n", err)
		return
	}
	fmt.Printf("status1 %d %s\n", status.Total, status.Version)

	status2, err := GetNftStatus(host2)
	if err != nil {
		fmt.Printf("GetNftStatus failed, %v\n", err)
		return
	}
	fmt.Printf("status2 %d %s\n", status2.Total, status2.Version)
	if status.Total != status2.Total {
		fmt.Printf("different count\n")
		return
	}

	for i := int64(0); i < int64(status.Total); i++ {
		nft1, err := GetInscription(i, host1)
		if err != nil {
			t.Fatal(err)
		}
		
		nft2, err := GetInscription(i, host2)
		if err != nil {
			t.Fatal(err)
		}

		if nft1.InscriptionId != nft2.InscriptionId {
			fmt.Printf("%d: inscription different %s %s\n", i, nft1.InscriptionId, nft2.InscriptionId)
			t.Fatal()
		}

		if nft1.OutPoint != nft2.OutPoint {
			fmt.Printf("%d: %s outpoint different %d %d\n", i, nft1.InscriptionId, nft1.OutPoint, nft2.OutPoint)
		}

		if i % 1000 == 0 {
			fmt.Printf("%d\n", i)
		}
	}

}

func TestParseValidateData(t *testing.T) {
	validateHolderData, err := validate.ReadBRC20HolderCSV("./validate/ordi-holders.csv")
	if err != nil {
		common.Log.Panicf("ReadBRC20HolderCSV failed, %v", err)
	}
	
	heightToHolderRecords := make(map[int]map[string]*validate.BRC20HolderCSVRecord)
	for _, record := range validateHolderData {
		holders, ok := heightToHolderRecords[record.LastHeight]
		if !ok {
			holders = make(map[string]*validate.BRC20HolderCSVRecord)
			heightToHolderRecords[record.LastHeight] = holders
		}
		holders[record.Address] = record
	}

	fmt.Printf("len %d", len(heightToHolderRecords))
}

func TestParseValidateDir(t *testing.T) {
	var err error
	validateHolderData, err := validate.ReadBRC20HolderCSVDir("./validate/holders")
	if err != nil {
		common.Log.Panicf("ReadBRC20HolderCSVDir failed, %v", err)
	}
	
	var startHeight, endHeight int
	startHeight = 0xffffffff

	heightToHolderRecords := make(map[int]map[string]map[string]*validate.BRC20HolderCSVRecord)
	for _, record := range validateHolderData {
		tickerToHolders, ok := heightToHolderRecords[record.LastHeight]
		if !ok {
			tickerToHolders = make(map[string]map[string]*validate.BRC20HolderCSVRecord)
			heightToHolderRecords[record.LastHeight] = tickerToHolders
		}
		holders, ok := tickerToHolders[record.Token]
		if !ok {
			holders = make(map[string]*validate.BRC20HolderCSVRecord)
			tickerToHolders[record.Token] = holders
		}
		holders[record.Address] = record

		if record.LastHeight > endHeight {
			endHeight = record.LastHeight
		}
		if record.LastHeight < startHeight {
			startHeight = record.LastHeight
		}
	}

	// 928228-928300
	fmt.Printf("len %d, height %d-%d\n", len(heightToHolderRecords), startHeight, endHeight)
}


func TestParseCompressFile(t *testing.T) {
	err := validate.SplitCSVFile("./validate/ordi.csv", "./validate/ordi", 40000, "ordi")
	if err != nil {
		t.Fatal(err)
	}
}


func TestParseValidateDir_history(t *testing.T) {
	var err error
	validateData, err := validate.ReadBRC20CSVDir("./validate/ordi")
	if err != nil {
		common.Log.Panicf("ReadBRC20CSVDir failed, %v", err)
	}

	fmt.Printf("len %d", len(validateData))
}


func TestParseValidateData_history(t *testing.T) {
	validateHolderData, err := validate.ReadBRC20CSV("./validate/ordi.csv")
	if err != nil {
		common.Log.Panicf("ReadBRC20CSV failed, %v", err)
	}

	fmt.Printf("len %d", len(validateHolderData))
}


func TestCompareValidateFile(t *testing.T) {

	validateData1, err := validate.ReadBRC20CSVDir("./validate/ordi")
	if err != nil {
		common.Log.Panicf("ReadBRC20CSVDir failed, %v", err)
	}

	validateData2, err := validate.ReadBRC20CSV("./validate/ordi_records.csv")
	if err != nil {
		common.Log.Panicf("ReadBRC20CSVDir failed, %v", err)
	}

	// validateData2, err := validate.ReadBRC20CSV("./validate/ordi.csv")
	// if err != nil {
	// 	common.Log.Panicf("ReadBRC20CSVDir failed, %v", err)
	// }

	diff1 := findDiffInMap(validateData1, validateData2)
	fmt.Printf("diff1 %d\n", len(diff1))
	for _, d := range diff1 {
		fmt.Printf("%v\n", validateData1[d])
	}

	diff2 := findDiffInMap(validateData2, validateData1)
	fmt.Printf("diff2 %d\n", len(diff2))
	for _, d := range diff2 {
		fmt.Printf("%v\n", validateData2[d])
	}

}
