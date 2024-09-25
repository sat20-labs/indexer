package common

import (
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
	chainParams := &chaincfg.TestNet3Params
	switch chain {
	case ChainTestnet:
		chainParams = &chaincfg.TestNet3Params
	case ChainTestnet4:
		chainParams = &chaincfg.TestNet3Params
	case ChainMainnet:
		chainParams = &chaincfg.MainNetParams
	}
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(pkScript, chainParams)
	if err != nil {
		return "", err
	}
	return addrs[0].EncodeAddress(), nil
}

func IsValidAddr(addr string, chain string) (bool, error) {
	chainParams := &chaincfg.TestNet3Params
	switch chain {
	case ChainTestnet:
		chainParams = &chaincfg.TestNet3Params
	case ChainTestnet4:
		chainParams = &chaincfg.TestNet3Params
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
	chainParams := &chaincfg.TestNet3Params
	switch chain {
	case ChainTestnet:
		chainParams = &chaincfg.TestNet3Params
	case ChainTestnet4:
		chainParams = &chaincfg.TestNet3Params
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
