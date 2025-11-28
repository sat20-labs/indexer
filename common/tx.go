package common

import (
	"encoding/base64"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common/pb"
)

const RANGE_IN_GLOBAL = false // true: Range 表示一个satoshi的全局编码，一个 [0, 2099999997690000) 的数字
// false: Range表示特殊聪在当前utxo中的范围。使用false，可以极大降低数据存储需求

type Range = pb.PbRange

type Input struct {
	Txid     string         `json:"txid"`
	UtxoId   uint64         `json:"utxoid"`
	Address  *ScriptPubKey  `json:"scriptPubKey"`
	Vout     int64          `json:"vout"`
	Value    int64          `json:"value"`
	Ordinals []*Range       `json:"ordinals"`
	Witness  wire.TxWitness `json:"witness"`
}

type ScriptPubKey struct {
	Addresses []string `json:"addresses"`
	Type      int      `json:"type"`
	ReqSig    int      `json:"reqSig"`
	PkScript  []byte   `json:"pkscript"`
}

type Output struct {
	Height   int           `json:"height"`
	TxId     int           `json:"txid"`
	Value    int64         `json:"value"`
	Address  *ScriptPubKey `json:"scriptPubKey"`
	N        int64         `json:"n"`
	Ordinals []*Range      `json:"ordinals"`
}

type TxInput struct {
	TxOutputV2
	Witness wire.TxWitness
	TxId    string
}

type TxOutputV2 struct {
	TxOutput
	Vout        int
	TxIndex     int
	Height      int
	AddressId   uint64
	AddressType int
}

func (p *TxOutputV2) GetAddress() string {
	switch txscript.ScriptClass(p.AddressType) {
	case txscript.NullDataTy:
		return "OP_RETURN"
	}

	var chainParams *chaincfg.Params
	if IsMainnet() {
		chainParams = &chaincfg.MainNetParams
	} else {
		chainParams = &chaincfg.TestNet4Params
	}
	_, addresses, _, _ := txscript.ExtractPkScriptAddrs(p.OutValue.PkScript, chainParams)
	if len(addresses) == 0 {
		// txscript.MultiSigTy, NonStandardTy
		return base64.StdEncoding.EncodeToString(p.OutValue.PkScript)
	}

	return addresses[0].EncodeAddress()
}

func GetPkScriptFromAddress(address string) ([]byte, error) {
	if address == "OP_RETURN" {
		return []byte{0x6a}, nil
	}
	// if address == "UNKNOWN" {
	// 	return []byte{0x51}, nil
	// }
	var chainParams *chaincfg.Params
	if IsMainnet() {
		chainParams = &chaincfg.MainNetParams
	} else {
		chainParams = &chaincfg.TestNet4Params
	}

	pkScript, err := AddrToPkScript(address, chainParams)
	if err != nil {
		// base64
		pkScript, err = base64.StdEncoding.DecodeString(address)
	}
	return pkScript, err
}


func GetAddressTypeFromAddress(address string) int {
	pkScript, err := GetPkScriptFromAddress(address)
	if err != nil {
		return int(txscript.NonStandardTy)
	}
	return GetAddressTypeFromPkScript(pkScript)
}

func GetAddressTypeFromPkScript(pkScript []byte) int {
	var chainParams *chaincfg.Params
	if IsMainnet() {
		chainParams = &chaincfg.MainNetParams
	} else {
		chainParams = &chaincfg.TestNet4Params
	}
	scriptClass, _, _, err := txscript.ExtractPkScriptAddrs(pkScript, chainParams)
	if err != nil {
		return int(txscript.NonStandardTy)
	}
	return int(scriptClass)
}

type Transaction struct {
	TxId    string        `json:"txid"`
	Inputs  []*TxInput    `json:"inputs"`
	Outputs []*TxOutputV2 `json:"outputs"`
}

type Block struct {
	Timestamp     time.Time      `json:"timestamp"`
	Height        int            `json:"height"`
	Hash          string         `json:"hash"`
	PrevBlockHash string         `json:"prevBlockHash"`
	Transactions  []*Transaction `json:"transactions"`
}

type UTXOIndex struct {
	Index map[string]*TxOutputV2
}
