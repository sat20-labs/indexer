package indexer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/base"
	"github.com/sat20-labs/indexer/indexer/ord"
	"github.com/sat20-labs/indexer/indexer/ord/ord0_14_1"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
	"github.com/stretchr/testify/assert"
)

func GetRawData(txID string, network string) ([][]byte, error) {
	url := ""
	switch network {
	case common.ChainTestnet:
		url = fmt.Sprintf("https://mempool.space/testnet/api/tx/%s", txID)
	case common.ChainTestnet4:
		url = fmt.Sprintf("https://mempool.space/testnet4/api/tx/%s", txID)
	case common.ChainMainnet:
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


func GetTxHexData(txID string, network string) (string, error) {
	url := ""
	switch network {
	case common.ChainTestnet:
		url = fmt.Sprintf("https://mempool.space/testnet/api/tx/%s/hex", txID)
	case common.ChainTestnet4:
		url = fmt.Sprintf("https://mempool.space/testnet4/api/tx/%s/hex", txID)
	case common.ChainMainnet:
		url = fmt.Sprintf("https://mempool.space/api/tx/%s/hex", txID)
	default:
		return "", fmt.Errorf("unsupported network: %s", network)
	}

	response, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve transaction data for %s from the API, error: %v", txID, err)

	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to retrieve transaction data for %s from the API, error: %v", txID, err)
	}

	respBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to decode hex string to byte array for %s, error: %v", txID, err)
	}
	return string(respBytes), nil
}


func GetBlock(height int, isMainnet bool) (*common.Block, error) {
	var err error
	var param *chaincfg.Params
	if isMainnet {
		err = bitcoin_rpc.InitBitconRpc(
			"192.168.1.102",
			8332,
			"jacky",
			"_RZekaGRgKQJSIOYi6vq0_CkJtjoCootamy81J2cDn0",
			false,
		)
		param = &chaincfg.MainNetParams
	} else {
		err = bitcoin_rpc.InitBitconRpc(
			"192.168.1.102",
			28332,
			"jacky",
			"123456",
			false,
		)
		param = &chaincfg.TestNet4Params
	}
	if err != nil {
		return nil, err
	}

	return base.FetchBlock(height, param), nil
}

func TestParser_ord(t *testing.T) {
	// input 0, output 0
	rawData, err := GetRawData("4e73e226998b37ea6eee0d904a17321e3c0f75abfd9c3b534845ea5ff345a9e3", "testnet4")
	if err != nil {
		t.Fatal(err)
	}
	fields, _, err := common.ParseInscription(rawData)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, len(fields) == 1)
}

func TestParser_block(t *testing.T) {
	// input 0, output 0
	block, err := GetBlock(616107, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("tx: %d\n", len(block.Transactions))
	var inCount, outCount int
	for _, tx := range block.Transactions {
		inCount += len(tx.Inputs)
		outCount += len(tx.Outputs)
	}
	fmt.Printf("inCount %d\n", inCount)
	fmt.Printf("outCount %d\n", outCount)

}


func TestParseTxFromUtxoId(t *testing.T) {
	// input 0, output 0
	//utxoId := uint64(27031631197372416)
	utxoId := common.ToUtxoId(788753,48,2)
	height, txIndex, vout := common.FromUtxoId(utxoId)
	block, err := GetBlock(height, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("TxCount: %d\n", len(block.Transactions))
	for i, tx := range block.Transactions {
		if i != txIndex {
			continue
		}

		for j, txOut := range tx.Outputs {
			if j != vout {
				continue
			}
			fmt.Printf("found %s for utxoId %d\n", txOut.OutPointStr, utxoId)
		}
		break
	}

}

func TestProcessTx(t *testing.T) {
	// input 0, output 0
	height := 892744
	hexTx, err := GetTxHexData("b6acb4825c44d63f1db29562ac534c9a7b66fb826ebd1e953f9fb9d301deb3ec", "mainnet")
	if err != nil {
		t.Fatal(err)
	}

	tx, err := DecodeMsgTx(hexTx)
	if err != nil {
		t.Fatal(err)
	}

	for i, input := range tx.TxIn {
		inscriptions := ord0_14_1.GetInscriptionsInTxInput(input.Witness, height, i)
		for _, insc := range inscriptions {
			protocol, content := ord.GetProtocol(insc)
			switch protocol {
			case "ordx":
				fmt.Printf("%s: %s\n", protocol, content)
			case "brc-20":
				content := string(insc.Inscription.Body)
				ordxBaseContent := common.ParseBrc20BaseContent(content)
				if ordxBaseContent == nil {
					common.Log.Debugf("invalid content %s", content)
					return
				}

				switch strings.ToLower(ordxBaseContent.Op) {
				case "deploy":
					deployInfo := common.ParseBrc20DeployContent(content)
					if deployInfo == nil {
					}
				case "mint":
					mintInfo := common.ParseBrc20MintContent(content)
					if mintInfo == nil {
						return
					}
				case "transfer":
					transferInfo := common.ParseBrc20TransferContent(content)
					if transferInfo == nil {
						return
					}
				}
			}
		}
	}
	

}
