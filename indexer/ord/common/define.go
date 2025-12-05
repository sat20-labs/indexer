package common

import (
	"fmt"
)

type ScriptError struct {
	Err error
}

func (e *ScriptError) Error() string {
	return fmt.Sprintf("script error: %v", e.Err)
}

type Curse int

const (
	NoCurse               Curse = 0
	DuplicateField        Curse = 1 // 0.14.1+
	IncompleteField       Curse = 2 // 0.14.1+
	NotAtOffsetZero       Curse = 3 // 0.9.0+
	NotInFirstInput       Curse = 4 // 0.9.0+
	Pointer               Curse = 5 // 0.14.1+
	Pushnum               Curse = 6 // 0.14.1+
	Reinscription         Curse = 7 // 0.9.0+
	Stutter               Curse = 8 // 0.14.1+
	UnrecognizedEvenField Curse = 9 // 0.9.0+
)

func (c Curse) Error() string {
	switch c {
	case DuplicateField:
		return "curse: duplicate field tag encountered in the inscription envelope"
	case IncompleteField:
		return "curse: field tag found without a subsequent value"
	case NotAtOffsetZero:
		return "curse: inscription is not at offset zero within its input"
	case NotInFirstInput:
		return "curse: inscription is not in the first transaction input"
	case Pointer:
		return "curse: inscription uses the reserved pointer field"
	case Pushnum:
		return "curse: invalid pushnum opcode used in the inscription envelope"
	case Reinscription:
		return "curse: attempts to reinscribe or overwrite an existing inscription"
	case Stutter:
		return "curse: inscription envelope contains duplicate sequential data pushes (stutter)"
	case UnrecognizedEvenField:
		return "curse: contains an unrecognized field with an even tag"
	case NoCurse:
		return "curse: no curse detected"
	default:
		return fmt.Sprintf("curse: unknown condition (%d)", c)
	}
}

var (
	PROTOCOL_ID          = []byte("ord")
	BODY_TAG             = []byte{0x00}
	CONTENT_TYPE_TAG     = []byte{0x01}
	POINTER_TAG          = []byte{0x02}
	PARENT_TAG           = []byte{0x03}
	METAPROTOCOL_TAG     = []byte{0x05}
	METADATA_TAG         = []byte{0x07}
	DELEGATE_TAG         = []byte{0x0B}
	CONTENT_ENCODING_TAG = []byte{0x09}
)

type Inscription struct {
	Body                  []byte // 0.9.0+
	ContentType           []byte // 0.9.0+
	Parent                []byte // 0.9.0+
	UnrecognizedEvenField bool   // 0.9.0+
	ContentEncoding       []byte // 0.14.1+
	Delegate              []byte // 0.14.1+
	DuplicateField        bool   // 0.14.1+
	IncompleteField       bool   // 0.14.1+
	Metadata              []byte // 0.14.1+
	Metaprotocol          []byte // 0.14.1+
	Pointer               []byte // 0.14.1+
}

