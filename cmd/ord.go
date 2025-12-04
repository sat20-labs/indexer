package main

import (
	"bufio"
	"embed"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/config"
	"github.com/sat20-labs/indexer/indexer/ord/ord0_14_1"
	"github.com/sat20-labs/indexer/indexer/ord/ord0_9_0"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

func getRawBlock(blockHash string) (string, error) {
	h, err := bitcoin_rpc.ShareBitconRpc.GetRawBlock(blockHash)
	if err != nil {
		n := 1
		for n < 10 {
			time.Sleep(time.Duration(n) * time.Second)
			n++
			h, err = bitcoin_rpc.ShareBitconRpc.GetRawBlock(blockHash)
			if err == nil {
				break
			}
		}
	}
	return h, err
}

func getBlockHash(height uint64) (string, error) {
	h, err := bitcoin_rpc.ShareBitconRpc.GetBlockHash(height)
	if err != nil {
		n := 1
		for n < 10 {
			time.Sleep(time.Duration(n) * time.Second)
			n++
			h, err = bitcoin_rpc.ShareBitconRpc.GetBlockHash(height)
			if err == nil {
				break
			}
		}
	}
	return h, err
}

func fetchBlock(height int) *common.Block {
	hash, err := getBlockHash(uint64(height))
	if err != nil {
		common.Log.Errorf("getBlockHash %d failed. %v", height, err)
		return nil
	}

	rawBlock, err := getRawBlock(hash)
	if err != nil {
		common.Log.Errorf("getRawBlock %d %s failed. %v", height, hash, err)
		return nil
	}
	blockData, err := hex.DecodeString(rawBlock)
	if err != nil {
		common.Log.Errorf("DecodeString %d %s failed. %v", height, rawBlock, err)
		return nil
	}

	// Deserialize the bytes into a btcutil.Block.
	block, err := btcutil.NewBlockFromBytes(blockData)
	if err != nil {
		common.Log.Errorf("NewBlockFromBytes %d failed. %v", height, err)
		return nil
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

		for j, v := range tx.MsgTx().TxOut {
			scyptClass, addrs, reqSig, err := txscript.ExtractPkScriptAddrs(v.PkScript, &chaincfg.MainNetParams)
			if err != nil {
				common.Log.Errorf("ExtractPkScriptAddrs %d failed. %v", height, err)
				return nil
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
					PkScript:  v.PkScript,
					ReqSig:    reqSig,
				}
			} else {
				receiver = common.ScriptPubKey{
					Addresses: addrsString,
					Type:      int(scyptClass),
					PkScript:  v.PkScript,
					ReqSig:    reqSig,
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

	return bl
}

func initRpc(conf *config.YamlConf) error {
	var host string
	var port int
	var user string
	var pass string

	host = conf.ShareRPC.Bitcoin.Host
	port = conf.ShareRPC.Bitcoin.Port
	user = conf.ShareRPC.Bitcoin.User
	pass = conf.ShareRPC.Bitcoin.Password

	err := bitcoin_rpc.InitBitconRpc(
		host,
		port,
		user,
		pass,
		false,
	)
	if err != nil {
		return err
	}
	return nil
}

type LogEntry struct {
	Type   string
	ID     string
	Height int
	Number int
}

func (e LogEntry) String() string {
	return fmt.Sprintf("type %s, id %s, height %d, number %d", e.Type, e.ID, e.Height, e.Number)
}

//go:embed brc20_curse.txt
var brc20Fs embed.FS

func Work() {
	inputPath := filepath.Join("", "brc20_curse.txt")
	// inputPath := filepath.Join("", "curse.txt")
	input, err := brc20Fs.ReadFile(inputPath)
	if err != nil {
		common.Log.Panicf("Error reading brc20_curse: %v", err)
	}
	reader := strings.NewReader(string(input))
	scanner := bufio.NewScanner(reader)
	regex := regexp.MustCompile(
		`^(?P<Type>curse|not find in old ord),\s*` +
			`height:(?P<Height>\d+),\s*` +
			`id:(?P<ID>[a-f0-9]+i\d+)[,\s，]*` +
			`num:(?P<Number>\d+)`,
	)
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		submatches := regex.FindStringSubmatch(line)
		if len(submatches) == 5 {
			var entry LogEntry
			nameMap := make(map[string]int)
			for i, name := range regex.SubexpNames() {
				if name != "" {
					nameMap[name] = i
				}
			}
			rawType := submatches[nameMap["Type"]]
			if strings.HasPrefix(rawType, "curse") {
				entry.Type = "curse"
			} else if strings.HasPrefix(rawType, "not find") {
				entry.Type = "notfind"
			}
			entry.ID = submatches[nameMap["ID"]]
			entry.Height, _ = strconv.Atoi(submatches[nameMap["Height"]])
			entry.Number, _ = strconv.Atoi(submatches[nameMap["Number"]])

			common.Log.Infof("line %d: entry %s", lineNum, entry)
			FindCurse(&entry)
			lineNum++
		} else {
			common.Log.Panic("unreach")
		}
	}
}

func FindCurse(entry *LogEntry) {
	block := fetchBlock(entry.Height)
	var findTx *common.Transaction
	var findTxIndex int
	var findTxInOffset int
	for txIndex, tx := range block.Transactions {
		txid, offset, err := common.ParseOrdInscriptionID(entry.ID)
		if err != nil {
			common.Log.Panicf("ParseOrdInscriptionID %s failed. %v", entry.ID, err)
		}

		if tx.Txid != txid {
			continue
		}
		common.Log.Infof("txIndex %d: txId %v, offset %d", txIndex, txid, offset)
		findTx = tx
		findTxIndex = txIndex
		findTxInOffset = offset
		break
	}

	curseStatusList090 := ord0_9_0.GetInscriptionCurseStatus(findTx)
	var findCurseStatus090 *ord0_9_0.InscriptionResult
	for i, status := range curseStatusList090 {
		if entry.Type == "nofind" {
			if status.Err == nil {
				common.Log.Panicf("status090 must have err,entry: %s, index: %d", entry, i)
				break
			} else {
				common.Log.Debugf("090 txIndex %d: txId %v, status %v, index: %d", findTxIndex, findTx.Txid, status.Err, i)
			}
		} else if entry.Type == "curse" {
			if status.Inscription.TxInOffset == findTxInOffset && status.IsCursed {
				findCurseStatus090 = status
				break
			}
		}
	}
	if entry.Type == "curse" {
		if findCurseStatus090 == nil {
			common.Log.Panicf("findCurseStatus090 not found, entry: %s", entry)
		} else {
			common.Log.Debugf("090 txIndex %d: txId %v, status %v", findTxIndex, findTx.Txid, findCurseStatus090)
		}
	}

	curseStatusList0141 := ord0_14_1.GetInscriptionCurseStatus(entry.Height, findTx, &chaincfg.MainNetParams)
	var findCurseStatus0141 *ord0_14_1.InscriptionResult
	for _, status := range curseStatusList0141 {
		if status.TxInOffset == uint32(findTxInOffset) && !status.IsCursed {
			findCurseStatus0141 = status
			break
		}
	}
	if findCurseStatus0141 == nil {
		common.Log.Panicf("findCurseStatus141 not found, entry: %s", entry)
	} else {
		common.Log.Debugf("141 txIndex %d: txId %v, status %v", findTxIndex, findTx.Txid, findCurseStatus090)
	}
}

func main() {
	yamlcfg := config.InitConfig("../mainnet.env")
	config.InitLog(yamlcfg)
	err := initRpc(yamlcfg)
	if err != nil {
		common.Log.Error(err)
		return
	}
	entry := &LogEntry{
		ID:     "6958b1c02016a1d02538735f7a8554f9981c75ba6b5ae0ae2d62900cb8c31331i0",
		Height: 824544,
		Number: 53393742,
		Type:   "nofind",
	}
	FindCurse(entry)
	// Work()
}
