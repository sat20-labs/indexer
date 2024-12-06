package runes

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
)

func parseRuneListKey(input string) (string, error) {
	if !strings.HasPrefix(input, DB_PREFIX_RUNE) {
		return "", fmt.Errorf("invalid string format")
	}
	return strings.TrimPrefix(input, DB_PREFIX_RUNE), nil
}

func ParseMintHistoryKey(input string) (string, string, error) {
	if !strings.HasPrefix(input, DB_PREFIX_MINT_HISTORY) {
		return "", "", fmt.Errorf("invalid string format")
	}
	str := strings.TrimPrefix(input, DB_PREFIX_MINT_HISTORY)
	parts := strings.Split(str, "-")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid string format")
	}

	return parts[0], parts[1], nil
}

func parseHolderInfoKey(input string) (uint64, error) {
	if !strings.HasPrefix(input, DB_PREFIX_RUNE_HOLDER) {
		return common.INVALID_ID, fmt.Errorf("invalid string format")
	}
	str := strings.TrimPrefix(input, DB_PREFIX_RUNE_HOLDER)
	parts := strings.Split(str, "-")
	if len(parts) != 1 {
		return common.INVALID_ID, errors.New("invalid string format")
	}

	return strconv.ParseUint(parts[0], 10, 64)
}

/**
 * It must be the first INSCRIPTION encountered in the VOUTS of this transaction,
 * so it's necessary to verify that the INSCRIPTION at this position exists
 */
func tryGetFirstInscriptionId(transaction *common.Transaction) (ret *InscriptionId) {
	var id uint64 = 0
	for _, input := range transaction.Inputs {
		_, err := common.ParseInscription(input.Witness)
		if err == nil {
			inscriptionId := InscriptionId(transaction.Txid + "i" + strconv.FormatUint(id, 10))
			ret = &inscriptionId
			return
		}
		id++
	}
	return ret
}

func parserArtifact(transaction *common.Transaction) (ret *runestone.Artifact, voutIndex int, err error) {
	var msgTx wire.MsgTx
	for _, output := range transaction.Outputs {
		pkScript := output.Address.PkScript
		msgTx.AddTxOut(wire.NewTxOut(0, pkScript))
	}
	runestone := &runestone.Runestone{}
	ret, voutIndex, err = runestone.Decipher(&msgTx)
	return
}

func parseTxVoutScriptAddress(transaction *common.Transaction, voutIndex int, param chaincfg.Params) (address Address, err error) {
	output := transaction.Outputs[voutIndex]
	pkScript := output.Address.PkScript
	var addresses []btcutil.Address
	_, addresses, _, err = txscript.ExtractPkScriptAddrs(pkScript, &param)
	if err != nil {
		return
	}
	if len(addresses) == 0 {
		return "", errors.New("no address")
	}
	address = Address(addresses[0].EncodeAddress())
	return
}
