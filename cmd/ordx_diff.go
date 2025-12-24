package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sat20-labs/indexer/common"
)


type TickerInfoResp struct {
	BaseResp
	Data *common.TickerInfo `json:"data"`
}

func GetTickerStatus(host, ticker string) (*common.TickerInfo, error) {
	
	url := fmt.Sprintf("%s/v3/tick/info/%s", host, ticker)

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
	var data TickerInfoResp
	err = json.Unmarshal(respBytes, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response for %s, error: %v", url, err)
	}
	
	return data.Data, nil
}


type ListResp struct {
	Start int64  `json:"start" example:"0"`
	Total uint64 `json:"total" example:"9992"`
}

// mint history
type MintHistoryRespV3 struct {
	BaseResp
	Data *MintHistoryDataV3 `json:"data"`
}

type MintHistoryDataV3 struct {
	ListResp
	Detail *MintHistoryV3 `json:"detail"`
}

type MintHistoryV3 struct {
	TypeName string               `json:"type"`
	Ticker   string               `json:"ticker,omitempty"`
	Total    int                  `json:"total,omitempty"`
	Start    int                  `json:"start,omitempty"`
	Limit    int                  `json:"limit,omitempty"`
	Items    []*MintHistoryItemV3 `json:"items,omitempty"`
}

type MintHistoryItemV3 struct {
	MintAddress    string `json:"mintaddress,omitempty" example:"bc1p9jh2caef2ejxnnh342s4eaddwzntqvxsc2cdrsa25pxykvkmgm2sy5ycc5"`
	HolderAddress  string `json:"holderaddress,omitempty"`
	Balance        string `json:"balance,omitempty" example:"546" description:"Balance of the holder"`
	InscriptionID  string `json:"inscriptionId,omitempty" example:"bac89275b4c0a0ba6aaa603d749a1c88ae3033da9f6d6e661a28fb40e8dca362i0"`
	InscriptionNum int64  `json:"inscriptionNumber,omitempty" example:"67269474" description:"Inscription number of the holder"`
}



func GetMintHistory(host, ticker string, start, limit int) (*MintHistoryDataV3, error) {
	var url string
	if start == 0 && limit == 0 {
		url = fmt.Sprintf("%s/v3/tick/history/%s", host, ticker)
	} else {
		url = fmt.Sprintf("%s/v3/tick/history/%s?start=%d&limit=%d", host, ticker, start, limit)
	}
	

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
	var data MintHistoryRespV3
	err = json.Unmarshal(respBytes, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response for %s, error: %v", url, err)
	}
	
	return data.Data, nil
}

type MintDetailInfo struct {
	ID             int64           `json:"id" example:"1"`
	Ticker         string          `json:"ticker,omitempty"`
	MintAddress    string          `json:"address,omitempty"`
	Amount         int64           `json:"amount,omitempty"`
	MintTime       int64           `json:"mintTimes,omitempty"`
	Delegate       string          `json:"delegate,omitempty"`
	Content        []byte          `json:"content,omitempty"`
	ContentType    string          `json:"contenttype,omitempty"`
	Ranges         []*common.Range `json:"ranges,omitempty"`
	InscriptionID  string          `json:"inscriptionId,omitempty"`
	InscriptionNum int64           `json:"inscriptionNumber,omitempty"`
}

type MintDetailInfoResp struct {
	BaseResp
	Data *MintDetailInfo `json:"data"`
}

func GetMintDetails(host, inscriptionId string) (*MintDetailInfo, error) {
	
	url := fmt.Sprintf("%s/mint/details/%s", host, inscriptionId)

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
	var data MintDetailInfoResp
	err = json.Unmarshal(respBytes, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response for %s, error: %v", url, err)
	}
	
	return data.Data, nil
}

func loadMintHistory(host, ticker string, total int) map[string]*MintHistoryItemV3 {
	historymap := make(map[string]*MintHistoryItemV3)
	limit := 100
	for i := 0; i < total; i += limit {
		start := int(i)
		history, err := GetMintHistory(host, ticker, start, limit)
		if err != nil {
			common.Log.Infof("GetMintHistory failed, %v\n", err)
			break
		}

		for _, item := range history.Detail.Items {
			historymap[item.InscriptionID] = item
		}
	}

	return historymap
}

func TestCompareMintHistory() {
	host1 := "http://192.168.1.102:8019/btc/testnet"
	host2 := "http://127.0.0.1:8029/btc/testnet"
	tickerName := "dogcoin"

	status, err := GetTickerStatus(host1, tickerName)
	if err != nil {
		common.Log.Infof("GetTickerStatus failed, %v\n", err)
		return
	}
	common.Log.Infof("status1 %s %d %d\n", status.TotalMinted, status.MintTimes, status.HoldersCount)

	status2, err := GetTickerStatus(host2, tickerName)
	if err != nil {
		common.Log.Infof("GetTickerStatus failed, %v\n", err)
		return
	}
	common.Log.Infof("status2 %s %d %d\n", status2.TotalMinted, status2.MintTimes, status2.HoldersCount)

	history1, err := GetMintHistory(host1, tickerName, 0, 0)
	if err != nil {
		common.Log.Infof("GetMintHistory failed, %v\n", err)
		return
	}

	history2, err := GetMintHistory(host2, tickerName, 0, 0)
	if err != nil {
		common.Log.Infof("GetMintHistory failed, %v\n", err)
		return
	}

	if history1.Total == history2.Total {
		common.Log.Infof("mint total: %d is the same, no need to compare, exit.", history1.Total)
		return
	}

	common.Log.Infof("mint total different: %d %d", history1.Total, history2.Total)
	historymap1 := loadMintHistory(host1, tickerName, int(history1.Total))
	historymap2 := loadMintHistory(host2, tickerName, int(history2.Total))

	if history1.Total > history2.Total {
		for k, item1 := range historymap1 {
			_, ok := historymap2[k]
			if !ok {
				common.Log.Infof("missing %s %s in host2", item1.InscriptionID, item1.Balance)
			}	
		}	
	} else {
		for k, item1 := range historymap2 {
			_, ok := historymap1[k]
			if !ok {
				common.Log.Infof("missing %s %s in host1", item1.InscriptionID, item1.Balance)
			}	
		}	
	}

	common.Log.Info("completed")
}
