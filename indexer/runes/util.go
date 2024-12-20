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
func tryGetFirstInscriptionId(transaction *common.Transaction) (ret *runestone.InscriptionId) {
	for _, input := range transaction.Inputs {
		_, err := common.ParseInscription(input.Witness)
		if err == nil {
			inscriptionId := runestone.InscriptionId(transaction.Txid + "i0")
			ret = &inscriptionId
			return
		}
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

func parseTxVoutScriptAddress(transaction *common.Transaction, voutIndex int, param chaincfg.Params) (address runestone.Address, err error) {
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
	address = runestone.Address(addresses[0].EncodeAddress())
	return
}

func parseTapscript(witness wire.TxWitness) []byte {
	// From BIP341:
	// If there are at least two witness elements, and the first byte of
	// the last element is 0x50, this last element is called annex a
	// and is removed from the witness stack.
	lenWitness := len(witness)
	if lenWitness < 2 {
		return nil
	}
	lastElement := witness[lenWitness-1]
	if len(lastElement) < 1 {
		return nil
	}
	if lastElement[0] != txscript.TaprootAnnexTag {
		// otherwise script is 2nd from last
		if lenWitness < 2 {
			return nil
		}
		return witness[lenWitness-2]
	} else {
		// account for the extra item removed from the end
		if lenWitness < 3 {
			return witness[lenWitness-3]
		}
	}
	return nil
}

func parseTapscriptLegacyInstructions(tapscript []byte) (ret [][]byte) {
	// Opcode.classify(self, ctx: ClassifyContext) -> Class
	for i := 0; i < len(tapscript); {
		b := tapscript[i]
		switch b {
		// 0x65 0x66 0xff, All/IllegalOp
		case txscript.OP_VERIF, txscript.OP_VERIFY, txscript.OP_INVALIDOPCODE:
			return
		// 0x76 0xa9 0x87 0x88, Legacy/IllegalOp
		case txscript.OP_CAT, txscript.OP_SUBSTR,
			txscript.OP_LEFT, txscript.OP_RIGHT,
			txscript.OP_INVERT,
			txscript.OP_AND, txscript.OP_OR, txscript.OP_XOR,
			txscript.OP_2MUL, txscript.OP_2DIV,
			txscript.OP_MUL, txscript.OP_DIV, txscript.OP_MOD,
			txscript.OP_LSHIFT, txscript.OP_RSHIFT:
			return
		// 80, 98, 126-129, 131-134, 137-138, 141-142, 149-153, 187-254, TapScript/SuccessOp
		// case ...
		// 0x61 0xb0 0xb1 0xb2 0xb3 0xb4 0xb5 0xb6 0xb7 0xb8 0xb9, All/NoOp
		case txscript.OP_NOP,
			txscript.OP_NOP1, txscript.OP_NOP2, txscript.OP_NOP3, txscript.OP_NOP4, txscript.OP_NOP5,
			txscript.OP_NOP6, txscript.OP_NOP7, txscript.OP_NOP8, txscript.OP_NOP9, txscript.OP_NOP10:
			return
		// 0x6a, All/ReturnOp
		case txscript.OP_RETURN:
			return
		// 0x50, 0x89, 0x8a, 0x62, Legacy/ReturnOp
		case txscript.OP_RESERVED, txscript.OP_RESERVED1, txscript.OP_RESERVED2, txscript.OP_VER:
			return
		// OP_1NEGATE(OP_PUSHNUM_NEG1):0x4f, All/PushNum(-1)
		case txscript.OP_1NEGATE:
			return
		default:
			// 0xba, All/ReturnOp
			if b >= txscript.OP_CHECKSIGADD {
				return
			}
			// OP_1(OP_PUSHNUM_1):0x60, 0x51:OP_16(OP_PUSHNUM_16), All/PushNum(1 + code - OP_PUSHNUM_1)
			if b >= txscript.OP_1 && b <= txscript.OP_16 {
				return
			}
			// OP_DATA_75(OP_PUSHBYTES_75):0x4b, All/PushBytes(b)
			if b <= txscript.OP_DATA_75 {
				break
			}
			// All/Ordinary(b)
			return
		}
		var n int = int(b)
		if len(tapscript) >= n {
			slice := tapscript[i:n]
			if n > 0 {
				i += (n - 1)
			}
			ret = append(ret, slice)
		}
	}
	return
}
