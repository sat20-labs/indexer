package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sat20-labs/indexer/common"
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


func main() {
	//url0 := "http://192.168.1.101:8019/btc/testnet"
	host1 := "http://127.0.0.1:8019/btc/testnet"
	host2 := "http://127.0.0.1:8029/btc/testnet"

	status, err := GetNftStatus(host1)
	if err != nil {
		common.Log.Infof("GetNftStatus failed, %v", err)
		return
	}
	common.Log.Infof("status1 %d %s", status.Total, status.Version)

	status2, err := GetNftStatus(host2)
	if err != nil {
		common.Log.Infof("GetNftStatus failed, %v", err)
		return
	}
	common.Log.Infof("status2 %d %s", status2.Total, status2.Version)
	if status.Total != status2.Total {
		common.Log.Infof("different count")
		return
	}
	total := min(status.Total, status2.Total)

	for i := int64(0); i < int64(total); i++ {
		nft1, err := GetInscription(i, host1)
		if err != nil {
			common.Log.Infof("GetInscription failed, %v", err)
			break
		}
		
		nft2, err := GetInscription(i, host2)
		if err != nil {
			common.Log.Infof("GetInscription failed, %v", err)
			break
		}

		if nft1.InscriptionId != nft2.InscriptionId {
			common.Log.Infof("%d: inscription different %s %s", i, nft1.InscriptionId, nft2.InscriptionId)
			break
		}

		if nft1.Utxo != nft2.Utxo {
			common.Log.Infof("%d: %s utxo different %d %d", i, nft1.InscriptionId, nft1.OutPoint, nft2.OutPoint)
		}

		if nft1.OutPoint != nft2.OutPoint {
			common.Log.Infof("%d: %s outpoint different %d %d", i, nft1.InscriptionId, nft1.OutPoint, nft2.OutPoint)
		}

		if i % 1000 == 0 {
			common.Log.Infof("%d", i)
		}
	}

}