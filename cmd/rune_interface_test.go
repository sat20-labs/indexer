package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/sat20-labs/indexer/config"
	"github.com/sat20-labs/indexer/indexer"
	"github.com/sat20-labs/indexer/indexer/runes"
	"github.com/sat20-labs/indexer/indexer/runes/table"
	"github.com/sat20-labs/indexer/share/base_indexer"
	"lukechampine.com/uint128"
)

var firstRuneName = "BESTINSLOT•XYZ"

var runesIndexer *runes.Indexer

func InitRuneTester() {
	if runesIndexer == nil {
		dbdir := "../db/testnet/"
		yamlcfg := config.YamlConf{
			Chain: "testnet",
			DB: config.DB{
				Path: dbdir,
			},
			BasicIndex: config.BasicIndex{
				MaxIndexHeight:  61680,
				PeriodFlushToDB: 20,
			},
		}
		indexerMgr := indexer.NewIndexerMgr(&yamlcfg)
		base_indexer.InitBaseIndexer(indexerMgr)
		indexerMgr.Init()
		runesIndexer = indexerMgr.RunesIndexer
		table.IsLessStorage = false
	}
}

func TestInterfaceRune(t *testing.T) {
	InitRuneTester()
	// 0
	runeIdStr := "61721_61"
	runeInfo := runesIndexer.GetRuneInfoWithId(runeIdStr)
	t.Logf("GetRuneInfoWithId return: %+v\n", runeInfo)
	// 1
	runeInfo = runesIndexer.GetRuneInfoWithName(firstRuneName)
	// common.Log.Infof("GetRuneInfo return: %+v\n", runeInfo)
	t.Logf("GetRuneInfo return: %+v\n", runeInfo)
	// 2
	isExistRune := runesIndexer.IsExistRuneWithName(firstRuneName)
	t.Logf("IsExistRune return: %+v\n", isExistRune)
	// 3
	runeInfos, total := runesIndexer.GetRuneInfos(0, 1000)
	t.Logf("GetRuneInfos return runeInfo total count: %d\n", total)
	for i, v := range runeInfos {
		t.Logf("GetRuneInfos return runeInfo %d: %+v\n", i, v)
	}
}

func TestGetHoldersWithTicks(t *testing.T) {
	InitRuneTester()
	// 11
	runeId, err := runesIndexer.GetRuneIdWithName(firstRuneName)
	if err != nil {
		t.Fatalf("GetRuneIdWithName err:%s", err.Error())
	}
	holders := runesIndexer.GetHoldersWithTick(runeId.String())
	t.Logf("GetHoldersWithTicks return holders total count: %d\n", len(holders))
	for i, v := range holders {
		t.Logf("GetHoldersWithTicks return holders, addressId: %d, value: %s\n", i, v.String())
	}
}

func TestGetAllAddressBalances(t *testing.T) {
	InitRuneTester()
	// 4
	runeId, err := runesIndexer.GetRuneIdWithName(firstRuneName)
	if err != nil {
		t.Fatalf("GetRuneIdWithName err:%s", err.Error())
	}
	addressBalance, total := runesIndexer.GetAllAddressBalances(runeId.String(), 0, 10)
	t.Logf("GetAllAddressBalances return addressBalance total count: %d\n", total)
	for i, v := range addressBalance {
		t.Logf("GetAllAddressBalances return addressBalance %d: addressId: %d, balance: %s\n", i, v.AddressId, v.Balance.String())
	}
}

func TestGetAllUtxoBalances(t *testing.T) {
	InitRuneTester()
	runeId, err := runesIndexer.GetRuneIdWithName(firstRuneName)
	if err != nil {
		t.Fatalf("GetRuneIdWithName err:%s", err.Error())
	}
	// 5
	allUtxoBalances1, total1 := runesIndexer.GetAllUtxoBalances(runeId.String(), 0, 10)
	t.Logf("GetAllUtxoBalances return utxoBalance total count: %d\n", total1)
	for i, v := range allUtxoBalances1.Balances {
		t.Logf("GetAllUtxoBalances return utxoBalance %d: %+v\n", i, v)
	}
}

func TestInterfaceAsset(t *testing.T) {
	InitRuneTester()
	runeId, err := runesIndexer.GetRuneIdWithName(firstRuneName)
	if err != nil {
		t.Fatalf("GetRuneIdWithName err:%s", err.Error())
	}
	// 6
	firstRuneAddress := "tb1pn9dzakm6egrv90c9gsgs63axvmn6ydwemrpuwljnmz9qdk38ueqsqae936"
	addressId := runesIndexer.RpcService.GetAddressId(firstRuneAddress)
	addressAssets := runesIndexer.GetAddressAssets(addressId)
	for i, v := range addressAssets {
		t.Logf("GetAddressAssets return addressAssets %d: %+v\n", i, v)
	}

	// 7
	utxo := "d2f8fe663c83550fee4039027fc4d5053066c10b638180137f43b997cc427108:0"
	utxoInfo, err := runesIndexer.RpcService.GetUtxoInfo(utxo)
	if err != nil {
		t.Errorf("RpcService.GetUtxoInfo error: %s", err.Error())
	}
	utxoAssets := runesIndexer.GetUtxoAssets(utxoInfo.UtxoId)
	for i, v := range utxoAssets {
		t.Logf("GetUtxoAssets return utxoAssets %d: %+v\n", i, v)
	}

	// 8
	isExistAsset := runesIndexer.IsExistAsset(utxoInfo.UtxoId)
	t.Logf("IsExistAsset return: %+v\n", isExistAsset)

	// 9
	mintHistorys, total := runesIndexer.GetMintHistory(runeId.Hex(), 0, 10)
	t.Logf("GetMintHistory return txids total count: %d\n", total)
	for i, v := range mintHistorys {
		t.Logf("GetMintHistory return txids %d: %+v\n", i, v)
	}
}

func TestGetAddressMintHistory(t *testing.T) {
	InitRuneTester()
	// 10
	firstRuneAddress := "tb1pfu2ff6ycy99t02zteumkm2jtk3uwm4skp50m7tevapcpkm8vaqqq73vxqr"
	runeId, err := runesIndexer.GetRuneIdWithName(firstRuneName)
	if err != nil {
		t.Fatalf("GetRuneIdWithName err:%s", err.Error())
	}
	addressId := runesIndexer.RpcService.GetAddressId(firstRuneAddress)
	mintHistorys, total := runesIndexer.GetAddressMintHistory(runeId.Hex(), addressId, 0, 10)
	t.Logf("GetAddressMintHistory return txids total count: %d\n", total)
	for i, v := range mintHistorys {
		t.Logf("GetAddressMintHistory return txids %d: %+v\n", i, v)
	}
}

func TestCheckRunesSummary(t *testing.T) {
	InitRuneTester()
	runeId, err := runesIndexer.GetRuneIdWithName(firstRuneName)
	if err != nil {
		t.Fatalf("GetRuneIdWithName err:%s", err.Error())
	}
	t.Logf("rune: %s\n", firstRuneName)

	runeInfo := runesIndexer.GetRuneInfoWithId(runeId.String())
	_, total := runesIndexer.GetAllAddressBalances(runeId.String(), 0, 1)
	addressBalances, _ := runesIndexer.GetAllAddressBalances(runeId.String(), 0, total)
	var addressBalance uint128.Uint128
	for _, v := range addressBalances {
		addressBalance = v.Balance.Add(addressBalance)
	}

	totalAddressBalance := addressBalance.Add(runeInfo.Burned)
	if addressBalance.Add(runeInfo.Burned).Cmp(totalAddressBalance) != 0 {
		t.Errorf("all address(%d)'s total balance(%s) + burned is not equal to supply(%s)", total, totalAddressBalance.String(), runeInfo.Supply.String())
	} else {
		t.Logf("all address(%d)'s total balance(%s) + burned is equal to supply(%s)", total, totalAddressBalance.String(), runeInfo.Supply.String())
	}

	_, total = runesIndexer.GetAllUtxoBalances(runeId.String(), 0, 1)
	utxoBalances, _ := runesIndexer.GetAllUtxoBalances(runeId.String(), 0, total)
	totalUtxoBalance := utxoBalances.Total.Add(runeInfo.Burned)
	if utxoBalances.Total.Add(runeInfo.Burned).Cmp(totalUtxoBalance) != 0 {
		t.Errorf("all utxo(%d)'s total balance(%s) + burned is not equal to supply(%s)", total, totalUtxoBalance.String(), runeInfo.Supply.String())
	} else {
		t.Logf("all utxo(%d)'s total balance(%s) + burned is equal to supply(%s)", total, totalUtxoBalance.String(), runeInfo.Supply.String())
	}
}

func TestCheckAllRuneInfos(t *testing.T) {
	InitRuneTester()
	status, err := getStatusData()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	var runeCount uint64 = status.Runes
	startTime := time.Now()
	var i uint64 = 16463
	var count uint64 = 0
	for ; i <= runeCount; i++ {

		runeData, err := getRuneData(i)
		if err != nil {
			t.Fatalf("getRuneData err:%s", err.Error())
		}

		if runeData.Entry.Block > runesIndexer.Status.Height {
			break
		}
		runeInfo := runesIndexer.GetRuneInfoWithName(runeData.Entry.SpacedRune)
		if runeInfo == nil {
			t.Fatal("GetRuneInfoWithName err: rune not found")
		}
		if runeInfo.Number != runeData.Entry.Number {
			t.Fatal("GetRuneInfoWithName err: number not equal")
		}
		count++
		t.Logf("number: %d, rune: %s\n", i, runeData.Entry.SpacedRune)
	}
	duration := time.Since(startTime)
	t.Logf("Total time for checking %d runes: %s", count, duration)
}

const OrdinalRpcURL = "http://192.168.10.102:81"

type RuneData struct {
	Entry struct {
		Block        uint64      `json:"block"`
		Burned       json.Number `json:"burned"`
		Divisibility uint8       `json:"divisibility"`
		Etching      string      `json:"etching"`
		Mints        uint64      `json:"mints"`
		Number       uint64      `json:"number"`
		Premine      json.Number `json:"premine"`
		SpacedRune   string      `json:"spaced_rune"`
		Symbol       string      `json:"symbol"`
		Terms        Terms       `json:"terms"`
		Timestamp    uint64      `json:"timestamp"`
		Turbo        bool        `json:"turbo"`
	} `json:"entry"`
	ID       string `json:"id"`
	Mintable bool   `json:"mintable"`
	Parent   string `json:"parent"`
}

type Terms struct {
	Amount json.Number `json:"amount"`
	Cap    json.Number `json:"cap"`
	Height [2]*int64   `json:"height"`
	Offset [2]uint64   `json:"offset"`
}

func getRuneData(runeID uint64) (*RuneData, error) {
	url := fmt.Sprintf("%s/rune/%d", OrdinalRpcURL, runeID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	var data RuneData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

type StatusData struct {
	AddressIndex            bool      `json:"address_index"`
	BlessedInscriptions     uint64    `json:"blessed_inscriptions"`
	Chain                   string    `json:"chain"`
	CursedInscriptions      uint64    `json:"cursed_inscriptions"`
	Height                  uint64    `json:"height"`
	InitialSyncTime         Duration  `json:"initial_sync_time"`
	InscriptionIndex        bool      `json:"inscription_index"`
	Inscriptions            uint64    `json:"inscriptions"`
	JsonAPI                 bool      `json:"json_api"`
	LostSats                uint64    `json:"lost_sats"`
	MinimumRuneForNextBlock string    `json:"minimum_rune_for_next_block"`
	RuneIndex               bool      `json:"rune_index"`
	Runes                   uint64    `json:"runes"`
	SatIndex                bool      `json:"sat_index"`
	Started                 time.Time `json:"started"`
	TransactionIndex        bool      `json:"transaction_index"`
	UnrecoverablyReorged    bool      `json:"unrecoverably_reorged"`
	Uptime                  Duration  `json:"uptime"`
}

type Duration struct {
	Secs  int64 `json:"secs"`
	Nanos int64 `json:"nanos"`
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v map[string]int64
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	d.Secs = v["secs"]
	d.Nanos = v["nanos"]
	return nil
}

func getStatusData() (*StatusData, error) {
	url := OrdinalRpcURL + "/status"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	var data StatusData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}
