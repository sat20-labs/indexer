package common

import (
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/mempool"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

const (
	ChainTestnet  = "testnet"
	ChainTestnet4 = "testnet4"
	ChainMainnet  = "mainnet"
)

func PkScriptToAddr(pkScript []byte, chain string) (string, error) {
	chainParams := &chaincfg.TestNet4Params
	switch chain {
	case ChainTestnet:
		chainParams = &chaincfg.TestNet4Params
	case ChainTestnet4:
		chainParams = &chaincfg.TestNet4Params
	case ChainMainnet:
		chainParams = &chaincfg.MainNetParams
	}
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(pkScript, chainParams)
	if err != nil {
		return "", err
	}

	if len(addrs) == 0 {
		return "", fmt.Errorf("no address")
	}
	return addrs[0].EncodeAddress(), nil
}

func IsValidAddr(addr string, chain string) (bool, error) {
	chainParams := &chaincfg.TestNet4Params
	switch chain {
	case ChainTestnet:
		chainParams = &chaincfg.TestNet4Params
	case ChainTestnet4:
		chainParams = &chaincfg.TestNet4Params
	case ChainMainnet:
		chainParams = &chaincfg.MainNetParams
	default:
		return false, nil
	}
	_, err := btcutil.DecodeAddress(addr, chainParams)
	if err != nil {
		return false, err
	}
	return true, nil
}

func AddrToPkScript(addr string, chain string) ([]byte, error) {
	chainParams := &chaincfg.TestNet4Params
	switch chain {
	case ChainTestnet:
		chainParams = &chaincfg.TestNet4Params
	case ChainTestnet4:
		chainParams = &chaincfg.TestNet4Params
	case ChainMainnet:
		chainParams = &chaincfg.MainNetParams
	default:
		return nil, fmt.Errorf("invalid chain: %s", chain)
	}
	address, err := btcutil.DecodeAddress(addr, chainParams)
	if err != nil {
		return nil, err
	}
	return txscript.PayToAddrScript(address)
}

func SignalsReplacement(tx *wire.MsgTx) bool {
	for _, txIn := range tx.TxIn {
		if txIn.Sequence <= mempool.MaxRBFSequence {
			return true
		}
	}
	return false
}


func AddressToPkScript(address string, isMainnet bool) ([]byte, error) {
	var params *chaincfg.Params
	if isMainnet {
        params = &chaincfg.MainNetParams
    } else {
        params = &chaincfg.TestNet4Params
    }

    // 解析地址
    addr, err := btcutil.DecodeAddress(address, params)
    if err != nil {
        return nil, err
    }

    // 创建支付脚本
    return txscript.PayToAddrScript(addr)
}

func MultiSigToPkScript(n int, addresses []string, isMainnet bool) ([]byte, error) {
	var params *chaincfg.Params
	if isMainnet {
        params = &chaincfg.MainNetParams
    } else {
        params = &chaincfg.TestNet4Params
    }

	pubKeys := make([]*btcutil.AddressPubKey, len(addresses))
	for i, address := range addresses {
		addr, err := hex.DecodeString(address)
		if err != nil {
			return nil, fmt.Errorf("failed to decode address %s: %w", address, err)
		}

		pk, err := btcutil.NewAddressPubKey(addr, params)
		if err != nil {
			return nil, err
		}

        pubKeys[i] = pk
	}

	pkScript, err := txscript.MultiSigScript(pubKeys, n)
	if err != nil {
		return nil, fmt.Errorf("failed to create multi-sig script: %w", err)
	}

	return pkScript, nil
}
