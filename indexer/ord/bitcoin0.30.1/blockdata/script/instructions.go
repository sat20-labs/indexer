package script

import (
	"errors"

	"github.com/sat20-labs/indexer/indexer/ord/bitcoin0.30.1/blockdata/opcodes"
)

type PushDataLenLen int

const (
	Len1Byte  PushDataLenLen = 1
	Len2Bytes PushDataLenLen = 2
	Len4Bytes PushDataLenLen = 4
)

// / Something did a non-minimal push; for more information see
// / <https://github.com/bitcoin/bips/blob/master/bip-0062.mediawiki#push-operators>
var (
	/// Something did a non-minimal push; for more information see
	/// <https://github.com/bitcoin/bips/blob/master/bip-0062.mediawiki#push-operators>
	ErrNonMinimalPush = errors.New("non minimal push")
	/// Some opcode expected a parameter but it was missing or truncated.
	ErrEarlyEndOfScript = errors.New("early end of script")
	/// Tried to read an array off the stack as a number when it was more than 4 bytes.
	ErrNumericOverflow = errors.New("numeric overflow")
)

type Instructions struct {
	data           []byte
	enforceMinimal bool
	cursor         int
	peeked         *Instruction
	peekedCursor   int
}

func NewInstructions(script []byte, enforceMinimal bool) *Instructions {
	return &Instructions{
		data:           script,
		enforceMinimal: enforceMinimal,
		cursor:         0,
		peeked:         nil,
		peekedCursor:   0,
	}
}

func (ins *Instructions) Kill() {
	ins.cursor = len(ins.data)
}

func (ins *Instructions) TakeSliceOrKill(length uint32) ([]byte, error) {
	lenInt := int(length)
	if ins.cursor+lenInt > len(ins.data) {
		ins.Kill()
		return nil, ErrEarlyEndOfScript
	}
	slice := ins.data[ins.cursor : ins.cursor+lenInt]
	ins.cursor += lenInt
	return slice, nil
}

func (ins *Instructions) ReadUint(length int) (uint64, error) {
	if ins.cursor+length > len(ins.data) {
		return 0, ErrEarlyEndOfScript
	}
	var n uint64
	for i := 0; i < length; i++ {
		byteVal := uint64(ins.data[ins.cursor+i])
		shift := uint64(i) * 8
		if shift >= 64 && byteVal != 0 {
			return 0, ErrNumericOverflow
		}
		n |= byteVal << shift
	}
	ins.cursor += length
	return n, nil
}

func (ins *Instructions) NextPushDataLen(lengthType PushDataLenLen, minPushLen int) (*Instruction, error) {
	n, err := ins.ReadUint(int(lengthType))
	if err != nil {
		ins.Kill()
		return nil, err
	}
	if n > 0xFFFFFFFF {
		ins.Kill()
		return nil, ErrNumericOverflow
	}
	pushLen := uint32(n)
	if ins.enforceMinimal && pushLen < uint32(minPushLen) {
		ins.Kill()
		return nil, ErrNonMinimalPush
	}
	dataSlice, err := ins.TakeSliceOrKill(pushLen)
	if err != nil {
		return nil, err
	}
	return NewInstructionPushBytes(dataSlice), nil
}

func (ins *Instructions) internalNext() (*Instruction, error) {
	if ins.cursor >= len(ins.data) {
		return nil, nil
	}

	opByte := ins.data[ins.cursor]
	ins.cursor++

	allOp := opcodes.NewAllFromByte(opByte)
	classification := allOp.Classify(opcodes.Legacy)

	switch classification.ClassType {
	case opcodes.ClassPushBytes:
		n := classification.PushBytesValue

		var isNonMinimal bool
		if ins.enforceMinimal && n == 1 {
			if ins.cursor < len(ins.data) {
				dataByte := ins.data[ins.cursor]
				if dataByte == 0x81 || (dataByte > 0 && dataByte <= 16) {
					isNonMinimal = true
				}
			}
		}

		if isNonMinimal {
			ins.Kill()
			return nil, ErrNonMinimalPush
		} else if n == 0 && ins.cursor >= len(ins.data) {
			return NewInstructionPushBytes([]byte{}), nil
		} else {
			data, err := ins.TakeSliceOrKill(n)
			if err != nil {
				return nil, err
			}
			return NewInstructionPushBytes(data), nil
		}
	case opcodes.ClassOrdinary:
		switch allOp.Code {
		case opcodes.OP_PUSHDATA1:
			return ins.NextPushDataLen(Len1Byte, 76)
		case opcodes.OP_PUSHDATA2:
			return ins.NextPushDataLen(Len2Bytes, 0x100)
		case opcodes.OP_PUSHDATA4:
			return ins.NextPushDataLen(Len4Bytes, 0x10000)
		default:
			return NewInstructionOp(allOp), nil
		}
	default:
		return NewInstructionOp(allOp), nil
	}
}

func (ins *Instructions) Next() (*Instruction, error) {
	if ins.peeked != nil {
		instr := ins.peeked
		ins.cursor = ins.peekedCursor
		ins.peeked = nil
		ins.peekedCursor = 0
		return instr, nil
	}
	return ins.internalNext()
}

func (ins *Instructions) Peek() (*Instruction, error) {
	if ins.peeked != nil {
		return ins.peeked, nil
	}
	originalCursor := ins.cursor
	instr, err := ins.internalNext()

	if err != nil {
		return nil, err
	}
	if instr == nil {
		return nil, nil
	}
	ins.peeked = instr
	ins.peekedCursor = ins.cursor
	ins.cursor = originalCursor
	return ins.peeked, nil
}
