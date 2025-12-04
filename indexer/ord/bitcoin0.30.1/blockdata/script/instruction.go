package script

import (
	"bytes"

	"github.com/sat20-labs/indexer/indexer/ord/bitcoin0.30.1/blockdata/opcodes"
)

type Instruction struct {
	IsPush        bool
	PushBytesData []byte
	OpCodeData    *opcodes.All
}

func NewInstructionPushBytes(data []byte) *Instruction {
	return &Instruction{
		IsPush:        true,
		PushBytesData: data,
		OpCodeData:    &opcodes.All{},
	}
}

func NewInstructionOp(op *opcodes.All) *Instruction {
	return &Instruction{
		IsPush:        false,
		PushBytesData: nil,
		OpCodeData:    op,
	}
}

func (i Instruction) Equal(other *Instruction) bool {
	if i.IsPush != other.IsPush {
		return false
	}
	if i.IsPush {
		return bytes.Equal(i.PushBytesData, other.PushBytesData)
	} else {
		return i.OpCodeData.Code == other.OpCodeData.Code
	}
}
