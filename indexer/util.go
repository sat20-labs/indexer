package indexer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/exotic"

	"github.com/sat20-labs/satsnet_btcd/mining/posminer/bootstrapnode"
)

// memory util
func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func GetSysMb() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return bToMb(m.Sys)
}

func GetAlloc() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return bToMb(m.Alloc)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func isValidExoticType(ty string) bool {
	for _, s := range exotic.SatributeList {
		if string(s) == ty {
			return true
		}
	}
	return false
}

func parseSatAttrString(s string) (common.SatAttr, error) {
	attr := common.SatAttr{}
	attributes := strings.Split(s, ";")
	for _, attribute := range attributes {
		pair := strings.SplitN(attribute, "=", 2)
		if len(pair) != 2 {
			return attr, fmt.Errorf("invalid attribute format: %s", attribute)
		}
		key := pair[0]
		value := pair[1]

		switch key {
		case "rar":
			if isValidExoticType(value) {
				attr.Rarity = value
			} else {
				return attr, fmt.Errorf("invalid exotic type value: %s", value)
			}
		case "trz":
			trailingZero, err := strconv.Atoi(value)
			if err != nil {
				return attr, fmt.Errorf("invalid trailing zero value: %s", value)
			}
			if trailingZero <= 0 {
				return attr, fmt.Errorf("invalid trailing zero value: %s", value)
			}
			attr.TrailingZero = trailingZero
		case "tmpl":
			attr.Template = value
		case "reg":
			attr.RegularExp = value
		}
	}

	return attr, nil
}

func skipOffsetRange(ord []*common.Range, satpoint int) []*common.Range {
	if satpoint == 0 {
		return ord
	}

	result := make([]*common.Range, 0)
	for _, rng := range ord {
		// skip the offset
		if satpoint > 0 {
			if int64(satpoint) >= (rng.Size) {
				satpoint -= int(rng.Size)
			} else {
				newRange := common.Range{Start: rng.Start + int64(satpoint), Size: rng.Size - int64(satpoint)}
				result = append(result, &newRange)
				satpoint = 0
			}
			continue
		}

		result = append(result, rng)
	}
	return result
}

func reSizeRange(ord []*common.Range, amt int64) []*common.Range {
	result := make([]*common.Range, 0)
	size := int64(0)
	for _, rng := range ord {
		if size+(rng.Size) <= amt {
			result = append(result, rng)
			size += (rng.Size)
		} else {
			newRng := common.Range{Start: rng.Start, Size: (amt - size)}
			result = append(result, &newRng)
			size += (newRng.Size)
		}

		if size == amt {
			break
		}
	}
	return result
}

func reAlignRange(ord []*common.Range, satpoint int, amt int64) []*common.Range {
	ret := skipOffsetRange(ord, satpoint)
	return reSizeRange(ret, amt)
}

func getPercentage(str string) (int, error) {
	// 只接受两位小数，或者100%
	str2 := strings.TrimSpace(str)

	var f float64
	var err error
	if strings.Contains(str2, "%") {
		str2 = strings.TrimRight(str2, "%") // 去掉百分号
		if strings.Contains(str2, ".") {
			parts := strings.Split(str2, ".")
			str3 := strings.Trim(parts[1], "0")
			if str3 != "" {
				return 0, fmt.Errorf("invalid format %s", str)
			}
		}
		f, err = strconv.ParseFloat(str2, 32)
	} else {
		regex := `^\d+(\.\d{0,2})?$`
		str2 = strings.TrimRight(str2, "0")
		var math bool
		math, err = regexp.MatchString(regex, str2)
		if err != nil || !math {
			return 0, fmt.Errorf("invalid format %s", str)
		}

		f, err = strconv.ParseFloat(str2, 32)
		f = f * 100
	}

	r := int(math.Round(f))
	if r > 100 {
		return 0, fmt.Errorf("invalid format %s", str)
	}

	return r, err
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

func GetBootstrapPubKey() []byte {
	pubkey, _ := hex.DecodeString(bootstrapnode.BootstrapPubKey)
	return pubkey
}

func GetCoreNodeChannelAddress(pubkey []byte, chainParams *chaincfg.Params) (string, error) {
	// 生成P2WSH地址
	_, pkScript, err := GetP2WSHscript(GetBootstrapPubKey(), pubkey)
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

