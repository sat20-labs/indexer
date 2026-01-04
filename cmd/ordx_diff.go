package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"

	"github.com/sat20-labs/indexer/common"
	indexerwire "github.com/sat20-labs/indexer/rpcserver/wire"
)


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
	var data indexerwire.TickerInfoResp
	err = json.Unmarshal(respBytes, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response for %s, error: %v", url, err)
	}
	
	return data.Data, nil
}


func GetMintHistory(host, ticker string, start, limit int) (*indexerwire.MintHistoryDataV3, error) {
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
	var data indexerwire.MintHistoryRespV3
	err = json.Unmarshal(respBytes, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response for %s, error: %v", url, err)
	}
	
	return data.Data, nil
}

func GetMintDetails(host, inscriptionId string) (*indexerwire.MintDetailInfo, error) {
	
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
	var data indexerwire.MintDetailInfoResp
	err = json.Unmarshal(respBytes, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response for %s, error: %v", url, err)
	}
	
	return data.Data, nil
}

func loadMintHistory(host, ticker string, total int) map[string]*indexerwire.MintHistoryItemV3 {
	historymap := make(map[string]*indexerwire.MintHistoryItemV3)
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
	host1 := "http://192.168.1.102:8019/btc/mainnet"
	host2 := "http://127.0.0.1:8019/btc/mainnet"
	tickerName := "pearl"

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

func GetHolders(host, ticker string, start, limit int) (*indexerwire.HolderListDataV3, error) {
	var url string
	if start == 0 && limit == 0 {
		url = fmt.Sprintf("%s/v3/tick/holders/%s", host, ticker)
	} else {
		url = fmt.Sprintf("%s/v3/tick/holders/%s?start=%d&limit=%d", host, ticker, start, limit)
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
	var data indexerwire.HolderListRespV3
	err = json.Unmarshal(respBytes, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response for %s, error: %v", url, err)
	}
	
	return data.Data, nil
}


func loadAllHolders(host, ticker string, total int) map[string]string {
	result := make(map[string]string)
	limit := total
	for i := 0; i < total; i += limit {
		start := int(i)
		holders, err := GetHolders(host, ticker, start, limit)
		if err != nil {
			common.Log.Infof("GetHolders failed, %v\n", err)
			break
		}

		for _, item := range holders.Detail {
			result[item.Wallet] = item.TotalBalance
		}
	}

	return result
}


func TestCompareHolders() {
	host1 := "http://192.168.1.102:8009/btc/mainnet"
	host2 := "http://127.0.0.1:8009/btc/mainnet"
	tickerName := "pearl"

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


	if status.HoldersCount != status2.HoldersCount {
		common.Log.Errorf("holder count different %d %d", status.HoldersCount, status2.HoldersCount)
	}

	holders1 := loadAllHolders(host1, tickerName, status.HoldersCount)
	holders2 := loadAllHolders(host2, tickerName, status2.HoldersCount)

	if len(holders1) > len(holders2) {
		for k, v1 := range holders1 {
			v2, ok := holders2[k]
			if !ok {
				common.Log.Infof("missing %s in host2", k)
			}	
			if v1 != v2 {
				common.Log.Infof("%s has diferrent value %s %s", k, v1, v2)
			}
		}
	} else {
		for k, v2 := range holders2 {
			v1, ok := holders1[k]
			if !ok {
				common.Log.Infof("missing %s in host1", k)
			}	
			if v1 != v2 {
				common.Log.Infof("%s has diferrent value %s %s", k, v1, v2)
			}
		}
	}

	type pair struct {
		address string
		amt int64
	}

	mid := make([]*pair, 0, len(holders1))
	for address, amt := range holders1 {
		n, _ := strconv.ParseInt(amt, 10, 64)
		mid = append(mid, &pair{
			address: address,
			amt: n,
		})
	}
	sort.Slice(mid, func(i, j int) bool {
		return mid[i].amt > mid[j].amt
	})

	for _, item := range mid {
		fmt.Printf("\"%s\": %d,\n", item.address, item.amt)
	}

	common.Log.Info("completed")
}
