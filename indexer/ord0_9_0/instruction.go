package ord090

import (
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

func ParseTapscript(witness wire.TxWitness) []byte {
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

func ParseTapscriptLegacyInstructions(tapscript []byte) (ret [][]byte) {
	// Opcode.classify(self, ctx: ClassifyContext) -> Class
	availDataLen := len(tapscript)
	for i := 0; i < len(tapscript); {
		b := tapscript[i]
		i++
		switch b {
		// 0x65 0x66 0xff, All/IllegalOp
		case txscript.OP_VERIF, txscript.OP_VERIFY, txscript.OP_INVALIDOPCODE:
			availDataLen--
			continue
		// 0x76 0xa9 0x87 0x88, Legacy/IllegalOp
		case txscript.OP_CAT, txscript.OP_SUBSTR,
			txscript.OP_LEFT, txscript.OP_RIGHT,
			txscript.OP_INVERT,
			txscript.OP_AND, txscript.OP_OR, txscript.OP_XOR,
			txscript.OP_2MUL, txscript.OP_2DIV,
			txscript.OP_MUL, txscript.OP_DIV, txscript.OP_MOD,
			txscript.OP_LSHIFT, txscript.OP_RSHIFT:
			availDataLen--
			continue
		// 80, 98, 126-129, 131-134, 137-138, 141-142, 149-153, 187-254, TapScript/SuccessOp
		// case ...
		// 0x61 0xb0 0xb1 0xb2 0xb3 0xb4 0xb5 0xb6 0xb7 0xb8 0xb9, All/NoOp
		case txscript.OP_NOP,
			txscript.OP_NOP1, txscript.OP_NOP2, txscript.OP_NOP3, txscript.OP_NOP4, txscript.OP_NOP5,
			txscript.OP_NOP6, txscript.OP_NOP7, txscript.OP_NOP8, txscript.OP_NOP9, txscript.OP_NOP10:
			availDataLen--
			continue
		// 0x6a, All/ReturnOp
		case txscript.OP_RETURN:
			availDataLen--
			continue
		// 0x50, 0x89, 0x8a, 0x62, Legacy/ReturnOp
		case txscript.OP_RESERVED, txscript.OP_RESERVED1, txscript.OP_RESERVED2, txscript.OP_VER:
			availDataLen--
			continue
		// OP_1NEGATE(OP_PUSHNUM_NEG1):0x4f, All/PushNum(-1)
		case txscript.OP_1NEGATE:
			availDataLen--
			continue
		default:
			// 0xba, All/ReturnOp
			if b >= txscript.OP_CHECKSIGADD {
				availDataLen--
				continue
			}
			// OP_1(OP_PUSHNUM_1):0x60, 0x51:OP_16(OP_PUSHNUM_16), All/PushNum(1 + code - OP_PUSHNUM_1)
			if b >= txscript.OP_1 && b <= txscript.OP_16 {
				availDataLen--
				continue
			}
			// OP_DATA_75(OP_PUSHBYTES_75):0x4b, All/PushBytes(b)
			if b <= txscript.OP_DATA_75 {
				break
			}
			// All/Ordinary(b)
			availDataLen--
			continue
		}
		n := int(uint(b))
		var slice []byte
		if availDataLen >= n {
			end := i + n
			if n > 0 && end <= len(tapscript) {
				slice = tapscript[i:end]
				ret = append(ret, slice)
				i += n
			}
			availDataLen = len(tapscript) - end
		} else if availDataLen != 0 {
			end := len(tapscript)
			slice = tapscript[i:end]
			i = end
			ret = append(ret, slice)
		}
	}
	return
}
