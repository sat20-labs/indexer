package indexer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/sat20-labs/indexer/common"
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

	hash, err := bitcoin_rpc.ShareBitconRpc.GetBlockHash(uint64(height))
	if err != nil {
		return nil, err
	}

	rawBlock, err := bitcoin_rpc.ShareBitconRpc.GetRawBlock(hash)
	if err != nil {
		return nil, err
	}
	fmt.Printf("block %d size: %d\n", height, len(rawBlock))

	blockData, err := hex.DecodeString(rawBlock)
	if err != nil {
		return nil, err
	}

	// Deserialize the bytes into a btcutil.Block.
	block, err := btcutil.NewBlockFromBytes(blockData)
	if err != nil {
		return nil, err
	}

	transactions := block.Transactions()
	txs := make([]*common.Transaction, len(transactions))
	for i, tx := range transactions {
		inputs := []*common.Input{}
		outputs := []*common.Output{}

		for _, v := range tx.MsgTx().TxIn {
			txid := v.PreviousOutPoint.Hash.String()
			vout := v.PreviousOutPoint.Index
			input := &common.Input{Txid: txid, Vout: int64(vout), Witness: v.Witness}
			inputs = append(inputs, input)
		}

		// parse the raw tx values
		for j, v := range tx.MsgTx().TxOut {
			// Determine the type of the script and extract the address
			scyptClass, addrs, reqSig, err := txscript.ExtractPkScriptAddrs(v.PkScript, param)
			if err != nil {
				common.Log.Errorf("ExtractPkScriptAddrs %d failed. %v", height, err)
				return nil, err
				//common.Log.Panicf("BaseIndexer.fetchBlock-> Failed to extract address: %v", err)
			}

			addrsString := make([]string, len(addrs))
			for i, x := range addrs {
				if scyptClass == txscript.MultiSigTy {
					addrsString[i] = hex.EncodeToString(x.ScriptAddress()) // pubkey
				} else {
					addrsString[i] = x.EncodeAddress()
				}
			}

			var receiver common.ScriptPubKey

			if len(addrs) == 0 {
				address := "UNKNOWN"
				if scyptClass == txscript.NullDataTy {
					address = "OP_RETURN"
				}
				receiver = common.ScriptPubKey{
					Addresses: []string{address},
					Type:      int(scyptClass),
					PkScript: v.PkScript,
					ReqSig:   reqSig,
				}
			} else {
				receiver = common.ScriptPubKey{
					Addresses: addrsString,
					Type:      int(scyptClass),
					PkScript: v.PkScript,
					ReqSig:   reqSig,
				}
			}

			output := &common.Output{Height: height, TxId: i, Value: v.Value, Address: &receiver, N: int64(j)}
			outputs = append(outputs, output)
		}

		txs[i] = &common.Transaction{
			Txid:    tx.Hash().String(),
			Inputs:  inputs,
			Outputs: outputs,
		}
	}

	t := block.MsgBlock().Header.Timestamp
	bl := &common.Block{
		Timestamp:     t,
		Height:        height,
		Hash:          block.Hash().String(),
		PrevBlockHash: block.MsgBlock().Header.PrevBlock.String(),
		Transactions:  txs,
	}

	return bl, nil
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
