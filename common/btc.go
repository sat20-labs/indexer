package common

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
)

const (
	ChainTestnet  = "testnet"
	ChainTestnet4 = "testnet4"
	ChainMainnet  = "mainnet"
)

func PkScriptToAddr(pkScript []byte, chainParams *chaincfg.Params) (string, error) {
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


// GenMultiSigScript generates the non-p2sh'd multisig script for 2 of 2
// pubkeys.
func GenMultiSigScript(aPub, bPub []byte) ([]byte, error) {
	if len(aPub) != 33 || len(bPub) != 33 {
		return nil, fmt.Errorf("pubkey size error: compressed " +
			"pubkeys only")
	}

	// Swap to sort pubkeys if needed. Keys are sorted in lexicographical
	// order. The signatures within the scriptSig must also adhere to the
	// order, ensuring that the signatures for each public key appears in
	// the proper order on the stack.
	if bytes.Compare(aPub, bPub) == 1 {
		aPub, bPub = bPub, aPub
	}

	// MultiSigSize 71 bytes
	//	- OP_2: 1 byte
	//	- OP_DATA: 1 byte (pubKeyAlice length)
	//	- pubKeyAlice: 33 bytes
	//	- OP_DATA: 1 byte (pubKeyBob length)
	//	- pubKeyBob: 33 bytes
	//	- OP_2: 1 byte
	//	- OP_CHECKMULTISIG: 1 byte
	MultiSigSize := 1 + 1 + 33 + 1 + 33 + 1 + 1
	bldr := txscript.NewScriptBuilder(txscript.WithScriptAllocSize(
		MultiSigSize,
	))
	bldr.AddOp(txscript.OP_2)
	bldr.AddData(aPub) // Add both pubkeys (sorted).
	bldr.AddData(bPub)
	bldr.AddOp(txscript.OP_2)
	bldr.AddOp(txscript.OP_CHECKMULTISIG)
	return bldr.Script()
}

// WitnessScriptHash generates a pay-to-witness-script-hash public key script
// paying to a version 0 witness program paying to the passed redeem script.
func WitnessScriptHash(witnessScript []byte) ([]byte, error) {
	// P2WSHSize 34 bytes
	//	- OP_0: 1 byte
	//	- OP_DATA: 1 byte (WitnessScriptSHA256 length)
	//	- WitnessScriptSHA256: 32 bytes
	P2WSHSize := 1 + 1 + 32
	bldr := txscript.NewScriptBuilder(
		txscript.WithScriptAllocSize(P2WSHSize),
	)

	bldr.AddOp(txscript.OP_0)
	scriptHash := sha256.Sum256(witnessScript)
	bldr.AddData(scriptHash[:])
	return bldr.Script()
}

func GetP2WSHscript(a, b []byte) ([]byte, []byte, error) {
	// 根据闪电网络的规则，小的公钥放前面
	witnessScript, err := GenMultiSigScript(a, b)
	if err != nil {
		return nil, nil, err
	}

	pkScript, err := WitnessScriptHash(witnessScript)
	if err != nil {
		return nil, nil, err
	}

	return witnessScript, pkScript, nil
}

func GetBTCAddressFromPkScript(pkScript []byte, chainParams *chaincfg.Params) (string, error) {
	_, addresses, _, err := txscript.ExtractPkScriptAddrs(pkScript, chainParams)
	if err != nil {
		return "", err
	}

	if len(addresses) == 0 {
		return "", fmt.Errorf("can't generate BTC address")
	}

	return addresses[0].EncodeAddress(), nil
}

func GetP2TRAddressFromPubkey(pubKey []byte, chainParams *chaincfg.Params) (string, error) {
	key, err := btcec.ParsePubKey(pubKey)
	if err != nil {
		return "", err
	}

	taprootPubKey := txscript.ComputeTaprootKeyNoScript(key)
	addr, err := btcutil.NewAddressTaproot(schnorr.SerializePubKey(taprootPubKey), chainParams)
	if err != nil {
		return "", err
	}
	return addr.EncodeAddress(), nil
}


func GetChannelAddress(pubkeyA, pubkeyB []byte, chainParams *chaincfg.Params) (string, error) {
	// 生成P2WSH地址
	_, pkScript, err := GetP2WSHscript(pubkeyA, pubkeyB)
	if err != nil {
		return "", err
	}

	// 生成地址
	address, err := GetBTCAddressFromPkScript(pkScript, chainParams)
	if err != nil {
		return "", err
	}

	return address, nil
}


func GetCoreNodeChannelAddress(pubkey []byte, chainParams *chaincfg.Params) (string, error) {
	// 生成P2WSH地址
	bootstrappubkey, _ := hex.DecodeString(GetBootstrapPubKey())
	return GetChannelAddress(bootstrappubkey, pubkey, chainParams)
}

func GetDefaultChannelAddress(chainParams *chaincfg.Params) (string, error) {
	// 生成P2WSH地址
	bootstrappubkey, _ := hex.DecodeString(GetBootstrapPubKey())
	corenodepubkey, _ := hex.DecodeString(GetCoreNodePubKey())
	return GetChannelAddress(bootstrappubkey, corenodepubkey, chainParams)
}

func BytesToPublicKey(pubKeyBytes []byte) (*secp256k1.PublicKey, error) {
	// 检查公钥长度
	if len(pubKeyBytes) != 33 && len(pubKeyBytes) != 65 {
		return nil, fmt.Errorf("invalid public key length: %d", len(pubKeyBytes))
	}

	// 解析公钥
	pubKey, err := secp256k1.ParsePubKey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}

	return pubKey, nil
}

func VerifyMessage(pubKey *secp256k1.PublicKey, msg []byte, sig []byte) error {
	// Compute the hash of the message.
	var msgDigest []byte
	doubleHash := false
	if doubleHash {
		msgDigest = chainhash.DoubleHashB(msg)
	} else {
		msgDigest = chainhash.HashB(msg)
	}

	signature, err := ecdsa.ParseDERSignature(sig)
	if err != nil {
		return err
	}

	// Verify the signature using the public key.
	if signature.Verify(msgDigest, pubKey) {
		return nil
	} else {
		return fmt.Errorf("signature.Verify failed")
	}
}

func VerifySignOfMessage(msg, sig, pubkey []byte) error {
	key, err := BytesToPublicKey(pubkey)
	if err != nil {
		return err
	}
	return VerifyMessage(key, msg, sig)
}

func IsOpReturn(pkScript []byte) bool {
	if len(pkScript) < 1 || pkScript[0] != txscript.OP_RETURN {
		return false
	}

	// Single OP_RETURN.
	if len(pkScript) == 1 {
		return true
	}
	if len(pkScript) > txscript.MaxDataCarrierSize {
		return false
	}

	return true
}
