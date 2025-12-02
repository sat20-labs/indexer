package opcodes

type Ordinary struct {
	Opcode All
}

func NewOrdinary(op All) *Ordinary {
	ord := TryFromAll(op)
	if ord == nil {
		return nil
	}
	return ord
}

func (ord *Ordinary) ToU8() byte {
	return ord.Opcode.Code
}

func (ord *Ordinary) String() string {
	return ord.Opcode.String()
}

func TryFromAll(op All) *Ordinary {
	switch op.Code {
	/* pushdata */
	case OP_PUSHDATA1, OP_PUSHDATA2, OP_PUSHDATA4,
		/* control flow */
		OP_IF, OP_NOTIF, OP_ELSE, OP_ENDIF, OP_VERIFY,
		/* stack */
		OP_TOALTSTACK, OP_FROMALTSTACK,
		OP_2DROP, OP_2DUP, OP_3DUP, OP_2OVER, OP_2ROT, OP_2SWAP,
		OP_DROP, OP_DUP, OP_NIP, OP_OVER, OP_PICK, OP_ROLL, OP_ROT, OP_SWAP, OP_TUCK,
		OP_IFDUP, OP_DEPTH, OP_SIZE,
		/* equality */
		OP_EQUAL, OP_EQUALVERIFY,
		/* arithmetic */
		OP_1ADD, OP_1SUB, OP_NEGATE, OP_ABS, OP_NOT, OP_0NOTEQUAL,
		OP_ADD, OP_SUB, OP_BOOLAND, OP_BOOLOR,
		OP_NUMEQUAL, OP_NUMEQUALVERIFY, OP_NUMNOTEQUAL, OP_LESSTHAN,
		OP_GREATERTHAN, OP_LESSTHANOREQUAL, OP_GREATERTHANOREQUAL,
		OP_MIN, OP_MAX, OP_WITHIN,
		/* crypto */
		OP_RIPEMD160, OP_SHA1, OP_SHA256, OP_HASH160, OP_HASH256,
		OP_CODESEPARATOR, OP_CHECKSIG, OP_CHECKSIGVERIFY,
		OP_CHECKMULTISIG, OP_CHECKMULTISIGVERIFY,
		OP_CHECKSIGADD:
		return &Ordinary{Opcode: op}
	default:
		return nil
	}
}
