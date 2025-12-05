package opcodes

import "fmt"

type All struct {
	Code byte
}

var AllOpcodeNames = map[byte]string{
	// PUSHBYTES (0x00 - 0x4b)
	0x00: "OP_PUSHBYTES_0",
	0x01: "OP_PUSHBYTES_1",
	0x02: "OP_PUSHBYTES_2",
	0x03: "OP_PUSHBYTES_3",
	0x04: "OP_PUSHBYTES_4",
	0x05: "OP_PUSHBYTES_5",
	0x06: "OP_PUSHBYTES_6",
	0x07: "OP_PUSHBYTES_7",
	0x08: "OP_PUSHBYTES_8",
	0x09: "OP_PUSHBYTES_9",
	0x0a: "OP_PUSHBYTES_10",
	0x0b: "OP_PUSHBYTES_11",
	0x0c: "OP_PUSHBYTES_12",
	0x0d: "OP_PUSHBYTES_13",
	0x0e: "OP_PUSHBYTES_14",
	0x0f: "OP_PUSHBYTES_15",
	0x10: "OP_PUSHBYTES_16",
	0x11: "OP_PUSHBYTES_17",
	0x12: "OP_PUSHBYTES_18",
	0x13: "OP_PUSHBYTES_19",
	0x14: "OP_PUSHBYTES_20",
	0x15: "OP_PUSHBYTES_21",
	0x16: "OP_PUSHBYTES_22",
	0x17: "OP_PUSHBYTES_23",
	0x18: "OP_PUSHBYTES_24",
	0x19: "OP_PUSHBYTES_25",
	0x1a: "OP_PUSHBYTES_26",
	0x1b: "OP_PUSHBYTES_27",
	0x1c: "OP_PUSHBYTES_28",
	0x1d: "OP_PUSHBYTES_29",
	0x1e: "OP_PUSHBYTES_30",
	0x1f: "OP_PUSHBYTES_31",
	0x20: "OP_PUSHBYTES_32",
	0x21: "OP_PUSHBYTES_33",
	0x22: "OP_PUSHBYTES_34",
	0x23: "OP_PUSHBYTES_35",
	0x24: "OP_PUSHBYTES_36",
	0x25: "OP_PUSHBYTES_37",
	0x26: "OP_PUSHBYTES_38",
	0x27: "OP_PUSHBYTES_39",
	0x28: "OP_PUSHBYTES_40",
	0x29: "OP_PUSHBYTES_41",
	0x2a: "OP_PUSHBYTES_42",
	0x2b: "OP_PUSHBYTES_43",
	0x2c: "OP_PUSHBYTES_44",
	0x2d: "OP_PUSHBYTES_45",
	0x2e: "OP_PUSHBYTES_46",
	0x2f: "OP_PUSHBYTES_47",
	0x30: "OP_PUSHBYTES_48",
	0x31: "OP_PUSHBYTES_49",
	0x32: "OP_PUSHBYTES_50",
	0x33: "OP_PUSHBYTES_51",
	0x34: "OP_PUSHBYTES_52",
	0x35: "OP_PUSHBYTES_53",
	0x36: "OP_PUSHBYTES_54",
	0x37: "OP_PUSHBYTES_55",
	0x38: "OP_PUSHBYTES_56",
	0x39: "OP_PUSHBYTES_57",
	0x3a: "OP_PUSHBYTES_58",
	0x3b: "OP_PUSHBYTES_59",
	0x3c: "OP_PUSHBYTES_60",
	0x3d: "OP_PUSHBYTES_61",
	0x3e: "OP_PUSHBYTES_62",
	0x3f: "OP_PUSHBYTES_63",
	0x40: "OP_PUSHBYTES_64",
	0x41: "OP_PUSHBYTES_65",
	0x42: "OP_PUSHBYTES_66",
	0x43: "OP_PUSHBYTES_67",
	0x44: "OP_PUSHBYTES_68",
	0x45: "OP_PUSHBYTES_69",
	0x46: "OP_PUSHBYTES_70",
	0x47: "OP_PUSHBYTES_71",
	0x48: "OP_PUSHBYTES_72",
	0x49: "OP_PUSHBYTES_73",
	0x4a: "OP_PUSHBYTES_74",
	0x4b: "OP_PUSHBYTES_75",

	// Data operations
	0x4c: "OP_PUSHDATA1",
	0x4d: "OP_PUSHDATA2",
	0x4e: "OP_PUSHDATA4",
	0x4f: "OP_PUSHNUM_NEG1",

	// Numeric push operations (0x51 - 0x60)
	0x50: "OP_RESERVED",
	0x51: "OP_PUSHNUM_1",
	0x52: "OP_PUSHNUM_2",
	0x53: "OP_PUSHNUM_3",
	0x54: "OP_PUSHNUM_4",
	0x55: "OP_PUSHNUM_5",
	0x56: "OP_PUSHNUM_6",
	0x57: "OP_PUSHNUM_7",
	0x58: "OP_PUSHNUM_8",
	0x59: "OP_PUSHNUM_9",
	0x5a: "OP_PUSHNUM_10",
	0x5b: "OP_PUSHNUM_11",
	0x5c: "OP_PUSHNUM_12",
	0x5d: "OP_PUSHNUM_13",
	0x5e: "OP_PUSHNUM_14",
	0x5f: "OP_PUSHNUM_15",
	0x60: "OP_PUSHNUM_16",

	// Control flow and Stack operations
	0x61: "OP_NOP",
	0x62: "OP_VER",
	0x63: "OP_IF",
	0x64: "OP_NOTIF",
	0x65: "OP_VERIF",
	0x66: "OP_VERNOTIF",
	0x67: "OP_ELSE",
	0x68: "OP_ENDIF",
	0x69: "OP_VERIFY",
	0x6a: "OP_RETURN",
	0x6b: "OP_TOALTSTACK",
	0x6c: "OP_FROMALTSTACK",
	0x6d: "OP_2DROP",
	0x6e: "OP_2DUP",
	0x6f: "OP_3DUP",
	0x70: "OP_2OVER",
	0x71: "OP_2ROT",
	0x72: "OP_2SWAP",
	0x73: "OP_IFDUP",
	0x74: "OP_DEPTH",
	0x75: "OP_DROP",
	0x76: "OP_DUP",
	0x77: "OP_NIP",
	0x78: "OP_OVER",
	0x79: "OP_PICK",
	0x7a: "OP_ROLL",
	0x7b: "OP_ROT",
	0x7c: "OP_SWAP",
	0x7d: "OP_TUCK",
	0x7e: "OP_CAT",
	0x7f: "OP_SUBSTR",
	0x80: "OP_LEFT",
	0x81: "OP_RIGHT",
	0x82: "OP_SIZE",

	// Bitwise logic operations
	0x83: "OP_INVERT",
	0x84: "OP_AND",
	0x85: "OP_OR",
	0x86: "OP_XOR",
	0x87: "OP_EQUAL",
	0x88: "OP_EQUALVERIFY",
	0x89: "OP_RESERVED1",
	0x8a: "OP_RESERVED2",

	// Arithmetic operations
	0x8b: "OP_1ADD",
	0x8c: "OP_1SUB",
	0x8d: "OP_2MUL",
	0x8e: "OP_2DIV",
	0x8f: "OP_NEGATE",
	0x90: "OP_ABS",
	0x91: "OP_NOT",
	0x92: "OP_0NOTEQUAL",
	0x93: "OP_ADD",
	0x94: "OP_SUB",
	0x95: "OP_MUL",
	0x96: "OP_DIV",
	0x97: "OP_MOD",
	0x98: "OP_LSHIFT",
	0x99: "OP_RSHIFT",

	// Comparison operations
	0x9a: "OP_BOOLAND",
	0x9b: "OP_BOOLOR",
	0x9c: "OP_NUMEQUAL",
	0x9d: "OP_NUMEQUALVERIFY",
	0x9e: "OP_NUMNOTEQUAL",
	0x9f: "OP_LESSTHAN",
	0xa0: "OP_GREATERTHAN",
	0xa1: "OP_LESSTHANOREQUAL",
	0xa2: "OP_GREATERTHANOREQUAL",
	0xa3: "OP_MIN",
	0xa4: "OP_MAX",
	0xa5: "OP_WITHIN",

	// Cryptography and control
	0xa6: "OP_RIPEMD160",
	0xa7: "OP_SHA1",
	0xa8: "OP_SHA256",
	0xa9: "OP_HASH160",
	0xaa: "OP_HASH256",
	0xab: "OP_CODESEPARATOR",
	0xac: "OP_CHECKSIG",
	0xad: "OP_CHECKSIGVERIFY",
	0xae: "OP_CHECKMULTISIG",
	0xaf: "OP_CHECKMULTISIGVERIFY",

	// NOPs and newer opcodes
	0xb0: "OP_NOP1",
	0xb1: "OP_CLTV",
	0xb2: "OP_CSV",
	0xb3: "OP_NOP4",
	0xb4: "OP_NOP5",
	0xb5: "OP_NOP6",
	0xb6: "OP_NOP7",
	0xb7: "OP_NOP8",
	0xb8: "OP_NOP9",
	0xb9: "OP_NOP10",

	// OP_CHECKSIGADD and returns
	0xba: "OP_CHECKSIGADD",
	0xbb: "OP_RETURN_187",
	0xbc: "OP_RETURN_188",
	0xbd: "OP_RETURN_189",
	0xbe: "OP_RETURN_190",
	0xbf: "OP_RETURN_191",
	0xc0: "OP_RETURN_192",
	0xc1: "OP_RETURN_193",
	0xc2: "OP_RETURN_194",
	0xc3: "OP_RETURN_195",
	0xc4: "OP_RETURN_196",
	0xc5: "OP_RETURN_197",
	0xc6: "OP_RETURN_198",
	0xc7: "OP_RETURN_199",
	0xc8: "OP_RETURN_200",
	0xc9: "OP_RETURN_201",
	0xca: "OP_RETURN_202",
	0xcb: "OP_RETURN_203",
	0xcc: "OP_RETURN_204",
	0xcd: "OP_RETURN_205",
	0xce: "OP_RETURN_206",
	0xcf: "OP_RETURN_207",
	0xd0: "OP_RETURN_208",
	0xd1: "OP_RETURN_209",
	0xd2: "OP_RETURN_210",
	0xd3: "OP_RETURN_211",
	0xd4: "OP_RETURN_212",
	0xd5: "OP_RETURN_213",
	0xd6: "OP_RETURN_214",
	0xd7: "OP_RETURN_215",
	0xd8: "OP_RETURN_216",
	0xd9: "OP_RETURN_217",
	0xda: "OP_RETURN_218",
	0xdb: "OP_RETURN_219",
	0xdc: "OP_RETURN_220",
	0xdd: "OP_RETURN_221",
	0xde: "OP_RETURN_222",
	0xdf: "OP_RETURN_223",
	0xe0: "OP_RETURN_224",
	0xe1: "OP_RETURN_225",
	0xe2: "OP_RETURN_226",
	0xe3: "OP_RETURN_227",
	0xe4: "OP_RETURN_228",
	0xe5: "OP_RETURN_229",
	0xe6: "OP_RETURN_230",
	0xe7: "OP_RETURN_231",
	0xe8: "OP_RETURN_232",
	0xe9: "OP_RETURN_233",
	0xea: "OP_RETURN_234",
	0xeb: "OP_RETURN_235",
	0xec: "OP_RETURN_236",
	0xed: "OP_RETURN_237",
	0xee: "OP_RETURN_238",
	0xef: "OP_RETURN_239",
	0xf0: "OP_RETURN_240",
	0xf1: "OP_RETURN_241",
	0xf2: "OP_RETURN_242",
	0xf3: "OP_RETURN_243",
	0xf4: "OP_RETURN_244",
	0xf5: "OP_RETURN_245",
	0xf6: "OP_RETURN_246",
	0xf7: "OP_RETURN_247",
	0xf8: "OP_RETURN_248",
	0xf9: "OP_RETURN_249",
	0xfa: "OP_RETURN_250",
	0xfb: "OP_RETURN_251",
	0xfc: "OP_RETURN_252",
	0xfd: "OP_RETURN_253",
	0xfe: "OP_RETURN_254",
	0xff: "OP_INVALIDOPCODE",
}

func (op All) String() string {
	name, ok := AllOpcodeNames[op.Code]
	if ok {
		return name
	}
	return fmt.Sprintf("OP_UNKNOWN_0x%x", op.Code)
}

type Opcode = byte

const (
	OP_PUSHBYTES_0 Opcode = iota
	OP_PUSHBYTES_1
	OP_PUSHBYTES_2
	OP_PUSHBYTES_3
	OP_PUSHBYTES_4
	OP_PUSHBYTES_5
	OP_PUSHBYTES_6
	OP_PUSHBYTES_7
	OP_PUSHBYTES_8
	OP_PUSHBYTES_9
	OP_PUSHBYTES_10
	OP_PUSHBYTES_11
	OP_PUSHBYTES_12
	OP_PUSHBYTES_13
	OP_PUSHBYTES_14
	OP_PUSHBYTES_15
	OP_PUSHBYTES_16
	OP_PUSHBYTES_17
	OP_PUSHBYTES_18
	OP_PUSHBYTES_19
	OP_PUSHBYTES_20
	OP_PUSHBYTES_21
	OP_PUSHBYTES_22
	OP_PUSHBYTES_23
	OP_PUSHBYTES_24
	OP_PUSHBYTES_25
	OP_PUSHBYTES_26
	OP_PUSHBYTES_27
	OP_PUSHBYTES_28
	OP_PUSHBYTES_29
	OP_PUSHBYTES_30
	OP_PUSHBYTES_31
	OP_PUSHBYTES_32
	OP_PUSHBYTES_33
	OP_PUSHBYTES_34
	OP_PUSHBYTES_35
	OP_PUSHBYTES_36
	OP_PUSHBYTES_37
	OP_PUSHBYTES_38
	OP_PUSHBYTES_39
	OP_PUSHBYTES_40
	OP_PUSHBYTES_41
	OP_PUSHBYTES_42
	OP_PUSHBYTES_43
	OP_PUSHBYTES_44
	OP_PUSHBYTES_45
	OP_PUSHBYTES_46
	OP_PUSHBYTES_47
	OP_PUSHBYTES_48
	OP_PUSHBYTES_49
	OP_PUSHBYTES_50
	OP_PUSHBYTES_51
	OP_PUSHBYTES_52
	OP_PUSHBYTES_53
	OP_PUSHBYTES_54
	OP_PUSHBYTES_55
	OP_PUSHBYTES_56
	OP_PUSHBYTES_57
	OP_PUSHBYTES_58
	OP_PUSHBYTES_59
	OP_PUSHBYTES_60
	OP_PUSHBYTES_61
	OP_PUSHBYTES_62
	OP_PUSHBYTES_63
	OP_PUSHBYTES_64
	OP_PUSHBYTES_65
	OP_PUSHBYTES_66
	OP_PUSHBYTES_67
	OP_PUSHBYTES_68
	OP_PUSHBYTES_69
	OP_PUSHBYTES_70
	OP_PUSHBYTES_71
	OP_PUSHBYTES_72
	OP_PUSHBYTES_73
	OP_PUSHBYTES_74
	OP_PUSHBYTES_75
	OP_PUSHDATA1
	OP_PUSHDATA2
	OP_PUSHDATA4
	OP_PUSHNUM_NEG1
	OP_RESERVED
	OP_PUSHNUM_1
	OP_PUSHNUM_2
	OP_PUSHNUM_3
	OP_PUSHNUM_4
	OP_PUSHNUM_5
	OP_PUSHNUM_6
	OP_PUSHNUM_7
	OP_PUSHNUM_8
	OP_PUSHNUM_9
	OP_PUSHNUM_10
	OP_PUSHNUM_11
	OP_PUSHNUM_12
	OP_PUSHNUM_13
	OP_PUSHNUM_14
	OP_PUSHNUM_15
	OP_PUSHNUM_16
	OP_NOP
	OP_VER
	OP_IF
	OP_NOTIF
	OP_VERIF
	OP_VERNOTIF
	OP_ELSE
	OP_ENDIF
	OP_VERIFY
	OP_RETURN
	OP_TOALTSTACK
	OP_FROMALTSTACK
	OP_2DROP
	OP_2DUP
	OP_3DUP
	OP_2OVER
	OP_2ROT
	OP_2SWAP
	OP_IFDUP
	OP_DEPTH
	OP_DROP
	OP_DUP
	OP_NIP
	OP_OVER
	OP_PICK
	OP_ROLL
	OP_ROT
	OP_SWAP
	OP_TUCK
	OP_CAT
	OP_SUBSTR
	OP_LEFT
	OP_RIGHT
	OP_SIZE
	OP_INVERT
	OP_AND
	OP_OR
	OP_XOR
	OP_EQUAL
	OP_EQUALVERIFY
	OP_RESERVED1
	OP_RESERVED2
	OP_1ADD
	OP_1SUB
	OP_2MUL
	OP_2DIV
	OP_NEGATE
	OP_ABS
	OP_NOT
	OP_0NOTEQUAL
	OP_ADD
	OP_SUB
	OP_MUL
	OP_DIV
	OP_MOD
	OP_LSHIFT
	OP_RSHIFT
	OP_BOOLAND
	OP_BOOLOR
	OP_NUMEQUAL
	OP_NUMEQUALVERIFY
	OP_NUMNOTEQUAL
	OP_LESSTHAN
	OP_GREATERTHAN
	OP_LESSTHANOREQUAL
	OP_GREATERTHANOREQUAL
	OP_MIN
	OP_MAX
	OP_WITHIN
	OP_RIPEMD160
	OP_SHA1
	OP_SHA256
	OP_HASH160
	OP_HASH256
	OP_CODESEPARATOR
	OP_CHECKSIG
	OP_CHECKSIGVERIFY
	OP_CHECKMULTISIG
	OP_CHECKMULTISIGVERIFY
	OP_NOP1
	OP_CLTV
	OP_CSV
	OP_NOP4
	OP_NOP5
	OP_NOP6
	OP_NOP7
	OP_NOP8
	OP_NOP9
	OP_NOP10
	OP_CHECKSIGADD
	OP_RETURN_187
	OP_RETURN_188
	OP_RETURN_189
	OP_RETURN_190
	OP_RETURN_191
	OP_RETURN_192
	OP_RETURN_193
	OP_RETURN_194
	OP_RETURN_195
	OP_RETURN_196
	OP_RETURN_197
	OP_RETURN_198
	OP_RETURN_199
	OP_RETURN_200
	OP_RETURN_201
	OP_RETURN_202
	OP_RETURN_203
	OP_RETURN_204
	OP_RETURN_205
	OP_RETURN_206
	OP_RETURN_207
	OP_RETURN_208
	OP_RETURN_209
	OP_RETURN_210
	OP_RETURN_211
	OP_RETURN_212
	OP_RETURN_213
	OP_RETURN_214
	OP_RETURN_215
	OP_RETURN_216
	OP_RETURN_217
	OP_RETURN_218
	OP_RETURN_219
	OP_RETURN_220
	OP_RETURN_221
	OP_RETURN_222
	OP_RETURN_223
	OP_RETURN_224
	OP_RETURN_225
	OP_RETURN_226
	OP_RETURN_227
	OP_RETURN_228
	OP_RETURN_229
	OP_RETURN_230
	OP_RETURN_231
	OP_RETURN_232
	OP_RETURN_233
	OP_RETURN_234
	OP_RETURN_235
	OP_RETURN_236
	OP_RETURN_237
	OP_RETURN_238
	OP_RETURN_239
	OP_RETURN_240
	OP_RETURN_241
	OP_RETURN_242
	OP_RETURN_243
	OP_RETURN_244
	OP_RETURN_245
	OP_RETURN_246
	OP_RETURN_247
	OP_RETURN_248
	OP_RETURN_249
	OP_RETURN_250
	OP_RETURN_251
	OP_RETURN_252
	OP_RETURN_253
	OP_RETURN_254
	OP_INVALIDOPCODE
)

type ClassifyContext int

// Classification context for the opcode.
// Some opcodes like [`OP_RESERVED`] abort the script in `ClassifyContext::Legacy` context,
// but will act as `OP_SUCCESSx` in `ClassifyContext::TapScript` (see BIP342 for full list).
const (
	// TapScript Opcode used in tapscript context (BIP-342).
	TapScript ClassifyContext = iota
	// Legacy Opcode used in legacy context.
	Legacy
)

func (op All) Classify(ctx ClassifyContext) ClassifyResult {
	code := op.Code
	switch code {
	// 3 opcodes illegal in all contexts
	case OP_VERIF, OP_VERNOTIF, OP_INVALIDOPCODE:
		return ClassifyResult{ClassType: ClassIllegalOp}
	}

	// 15 opcodes illegal in Legacy context
	if ctx == Legacy {
		switch code {
		case OP_CAT, OP_SUBSTR,
			OP_LEFT, OP_RIGHT,
			OP_INVERT,
			OP_AND, OP_OR, OP_XOR,
			OP_2MUL, OP_2DIV,
			OP_MUL, OP_DIV, OP_MOD,
			OP_LSHIFT, OP_RSHIFT:
			return ClassifyResult{ClassType: ClassIllegalOp}
		}
	}

	// 87 opcodes of SuccessOp class only in TapScript context
	if ctx == TapScript {
		isSuccessOp := false
		switch {
		case code == OP_RESERVED || code == OP_VER: // 80 (OP_RESERVED) 和 98 (OP_VER)
			isSuccessOp = true
		case code >= OP_CAT && code <= OP_RIGHT: // 126..=129 (OP_CAT, OP_SUBSTR, OP_LEFT, OP_RIGHT)
			isSuccessOp = true
		case code >= OP_INVERT && code <= OP_XOR: // 131..=134 (OP_INVERT, OP_AND, OP_OR, OP_XOR)
			isSuccessOp = true
		case code >= OP_RESERVED1 && code <= OP_RESERVED2: // 137..=138 (OP_RESERVED1, OP_RESERVED2)
			isSuccessOp = true
		case code >= OP_2MUL && code <= OP_2DIV: // 141..=142 (OP_2MUL, OP_2DIV)
			isSuccessOp = true
		case code >= OP_MUL && code <= OP_RSHIFT: // 149..=153 (OP_MUL, OP_LSHIFT, OP_RSHIFT)
			isSuccessOp = true
		case code >= OP_RETURN_187 && code <= OP_RETURN_254: // 187..=254 (所有 OP_RETURN 扩展)
			isSuccessOp = true
		}
		if isSuccessOp {
			return ClassifyResult{ClassType: ClassSuccessOp}
		}
	}

	// 11 opcodes of NoOp class
	if code == OP_NOP || (code >= OP_NOP1 && code <= OP_NOP10) {
		return ClassifyResult{ClassType: ClassNoOp}
	}

	// 1 opcode for `OP_RETURN`
	if code == OP_RETURN {
		return ClassifyResult{ClassType: ClassReturnOp}
	}

	if ctx == Legacy {
		// 4 opcodes operating equally to `OP_RETURN` only in Legacy context
		switch code {
		case OP_RESERVED, OP_RESERVED1, OP_RESERVED2, OP_VER:
			return ClassifyResult{ClassType: ClassReturnOp}
		}

		// 71 opcodes operating equally to `OP_RETURN` only in Legacy context (0xba 到 0xfe)
		if code >= OP_CHECKSIGADD {
			return ClassifyResult{ClassType: ClassReturnOp}
		}
	}

	// 2 opcodes operating equally to `OP_RETURN` only in TapScript context
	if ctx == TapScript {
		switch code {
		case OP_CHECKMULTISIG, OP_CHECKMULTISIGVERIFY:
			return ClassifyResult{ClassType: ClassReturnOp}
		}
	}

	// 1 opcode of PushNum class
	if code == OP_PUSHNUM_NEG1 {
		return ClassifyResult{ClassType: ClassPushNum, PushNumValue: -1}
	}

	// 16 opcodes of PushNum class
	if code >= OP_PUSHNUM_1 && code <= OP_PUSHNUM_16 {
		numValue := 1 + int32(code-OP_PUSHNUM_1)
		return ClassifyResult{ClassType: ClassPushNum, PushNumValue: numValue}
	}

	// 76 opcodes of PushBytes class (0x00 到 0x4b)
	if code <= OP_PUSHBYTES_75 {
		return ClassifyResult{ClassType: ClassPushBytes, PushBytesValue: uint32(code)}
	}

	// opcodes of Ordinary class: 61 for Legacy and 60 for TapScript context
	return ClassifyResult{ClassType: ClassOrdinary, OrdinaryValue: NewOrdinary(op.Code)}
}

func (op All) ToU8() byte {
	return op.Code
}

func NewAllFromByte(code byte) *All {
	return &All{Code: code}
}

var (
	OP_0     = OP_PUSHBYTES_0 /// Push an empty array onto the stack.
	OP_FALSE = OP_PUSHBYTES_0 /// Empty stack is also FALSE.
	OP_TRUE  = OP_PUSHNUM_1   /// Number 1 is also TRUE.
	OP_NOP2  = OP_CLTV        /// Previously called OP_NOP2.
	OP_NOP3  = OP_CSV         /// Previously called OP_NOP3.
)

type Class int

const (
	ClassPushNum   Class = iota // Pushes the given number onto the stack.
	ClassPushBytes              // Pushes the given number of bytes onto the stack.
	ClassReturnOp               // Fails the script if executed.
	ClassSuccessOp              // Succeeds the script even if not executed. (TapScript only)
	ClassIllegalOp              // Fails the script even if not executed. (Disabled)
	ClassNoOp                   // Does nothing.
	ClassOrdinary               // Any opcode not covered above.
)

type ClassifyResult struct {
	ClassType      Class
	PushNumValue   int32
	PushBytesValue uint32
	OrdinaryValue  *Ordinary
}
