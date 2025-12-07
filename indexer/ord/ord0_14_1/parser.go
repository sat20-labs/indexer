package ord0_14_1

import (
	"github.com/btcsuite/btcd/wire"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/ord/bitcoin0.30.1/blockdata/opcodes"
	"github.com/sat20-labs/indexer/indexer/ord/bitcoin0.30.1/blockdata/script"
	ordCommon "github.com/sat20-labs/indexer/indexer/ord/common"
)

type RawEnvelope struct {
	Input   uint32
	Offset  uint32
	Payload [][]byte
	Pushnum bool
	Stutter bool
}

type ParsedEnvelope struct {
	Input   uint32
	Offset  uint32
	Payload *ordCommon.Inscription
	Pushnum bool
	Stutter bool
}

func fromRawEnvelope(raw *RawEnvelope) *ParsedEnvelope {
	var bodyIndex *int
	for i, push := range raw.Payload {
		if i%2 == 0 && len(push) == 0 {
			bodyIndex = new(int)
			*bodyIndex = i
			break
		}
	}

	headerEnd := len(raw.Payload)
	if bodyIndex != nil {
		headerEnd = *bodyIndex
	}
	fields := make(map[string][][]byte)
	incompleteField := false
	for i := 0; i < headerEnd; i += 2 {
		if i+1 < headerEnd {
			keyBytes := raw.Payload[i]
			valueBytes := raw.Payload[i+1]
			keyStr := string(keyBytes)
			fields[keyStr] = append(fields[keyStr], valueBytes)
		} else {
			incompleteField = true
		}
	}
	duplicateField := false
	for _, values := range fields {
		if len(values) > 1 {
			duplicateField = true
			break
		}
	}

	contentEncoding := CONTENT_ENCODING_TAG.removeField(fields)
	contentType := CONTENT_TYPE_TAG.removeField(fields)
	delegate := DELEGATE_TAG.removeField(fields)
	metadata := METADATA_TAG.removeField(fields)
	metaprotocol := METAPROTOCOL_TAG.removeField(fields)
	parent := PARENT_TAG.removeField(fields)
	pointer := POINTER_TAG.removeField(fields)
	runeName := RUNE_NAME_TAG.removeField(fields)

	unrecognizedEvenField := false
	for tagStr := range fields {
		if len(tagStr) > 0 {
			tagByte := tagStr[0]
			if tagByte%2 == 0 {
				unrecognizedEvenField = true
				break
			}
		}
	}

	var body []byte
	if bodyIndex != nil {
		for i := *bodyIndex + 1; i < len(raw.Payload); i++ {
			push := raw.Payload[i]
			body = append(body, push...)
		}
	}
	inscription := &ordCommon.Inscription{
		Body:                  body,
		ContentEncoding:       contentEncoding,
		ContentType:           contentType,
		Delegate:              delegate,
		DuplicateField:        duplicateField,
		IncompleteField:       incompleteField,
		Metadata:              metadata,
		Metaprotocol:          metaprotocol,
		Parent:                parent,
		Pointer:               pointer,
		RuneName:              runeName,
		UnrecognizedEvenField: unrecognizedEvenField,
	}
	return &ParsedEnvelope{
		Input:   raw.Input,
		Offset:  raw.Offset,
		Payload: inscription,
		Pushnum: raw.Pushnum,
		Stutter: raw.Stutter,
	}
}

func ParseEnvelopesFromTransaction(tx *common.Transaction) []*ParsedEnvelope {
	rawEnvelopes := RawEnvelope{}.FromTransaction(tx)
	parsedEnvelopes := make([]*ParsedEnvelope, 0, len(rawEnvelopes))
	for _, raw := range rawEnvelopes {
		parsedEnvelopes = append(parsedEnvelopes, fromRawEnvelope(raw))
	}
	return parsedEnvelopes
}

func ParseEnvelopesFromTxInput(txInput wire.TxWitness, inputindex int) []*ParsedEnvelope {
	rawEnvelopes := RawEnvelope{}.FromTxInput(txInput, inputindex)
	parsedEnvelopes := make([]*ParsedEnvelope, 0, len(rawEnvelopes))
	for _, raw := range rawEnvelopes {
		parsedEnvelopes = append(parsedEnvelopes, fromRawEnvelope(raw))
	}
	return parsedEnvelopes
}

func (re RawEnvelope) FromTransaction(tx *common.Transaction) []*RawEnvelope {
	var envelopes []*RawEnvelope
	for i, input := range tx.Inputs {
		tapscript := ordCommon.GetTapscriptBytes(input.Witness)
		if tapscript != nil {
			inputEnvelopes, err := re.fromTapscript(tapscript, i)
			if err == nil {
				envelopes = append(envelopes, inputEnvelopes...)
			}
		}
	}
	return envelopes
}

func (re RawEnvelope) FromTxInput(input wire.TxWitness, inputindex int) []*RawEnvelope {
	var envelopes []*RawEnvelope
	tapscript := ordCommon.GetTapscriptBytes(input)
	if tapscript != nil {
		inputEnvelopes, err := re.fromTapscript(tapscript, inputindex)
		if err == nil {
			envelopes = append(envelopes, inputEnvelopes...)
		}
	}
	return envelopes
}

func (re RawEnvelope) fromTapscript(tapscript []byte, inputIndex int) ([]*RawEnvelope, error) {
	var envelopes []*RawEnvelope
	instructions := script.NewInstructions(tapscript, false)
	stuttered := false
	envelopeOffset := 0
	for {
		instruction, err := instructions.Next()
		if err != nil {
			return nil, &ordCommon.ScriptError{Err: err}
		}
		if instruction == nil {
			break
		}
		if instruction.IsPush && len(instruction.PushBytesData) == 0 {
			stutter, envelope, err := re.fromInstructions(instructions, inputIndex, envelopeOffset, stuttered)
			if err != nil {
				return nil, err
			}

			if envelope != nil {
				envelopes = append(envelopes, envelope)
				envelopeOffset++
			} else {
				stuttered = stutter
			}
		}
	}

	return envelopes, nil
}

func (re RawEnvelope) accept(instructions *script.Instructions, target *script.Instruction) (bool, error) {
	peeked, err := instructions.Peek()
	if err != nil {
		return false, &ordCommon.ScriptError{Err: err}
	}
	if peeked == nil {
		return false, nil
	}
	if peeked.Equal(target) {
		_, err = instructions.Next()
		if err != nil {
			return false, &ordCommon.ScriptError{Err: err}
		}
		return true, nil
	}
	return false, nil
}

func (re RawEnvelope) fromInstructions(instructions *script.Instructions, input int, offset int, stutter bool) (bool, *RawEnvelope, error) {
	accepted, err := re.accept(instructions, script.NewInstructionOp(&opcodes.All{Code: opcodes.OP_IF}))
	if err != nil {
		return false, nil, err
	}
	if !accepted {
		peeked, _ := instructions.Peek()
		stutterCheck := peeked != nil && peeked.Equal(script.NewInstructionPushBytes([]byte{}))
		return stutterCheck, nil, nil
	}

	accepted, err = re.accept(instructions, script.NewInstructionPushBytes(ordCommon.PROTOCOL_ID))
	if err != nil {
		return false, nil, err
	}
	if !accepted {
		peeked, _ := instructions.Peek()
		stutterCheck := peeked != nil && peeked.Equal(script.NewInstructionPushBytes([]byte{}))
		return stutterCheck, nil, nil
	}

	pushnum := false
	var payload [][]byte
	for {
		instruction, err := instructions.Next()
		if err != nil {
			return false, nil, err
		}
		if instruction == nil {
			return false, nil, nil
		}

		if instruction.Equal(script.NewInstructionOp(&opcodes.All{Code: opcodes.OP_ENDIF})) {
			envelope := &RawEnvelope{
				Input:   uint32(input),
				Offset:  uint32(offset),
				Payload: payload,
				Pushnum: pushnum,
				Stutter: stutter,
			}
			return false, envelope, nil
		}

		if instruction.OpCodeData != nil {
			code := instruction.OpCodeData.Code
			var value []byte = nil
			isPushNum := true

			switch code {
			case opcodes.OP_PUSHNUM_NEG1:
				value = []byte{0x81}
				pushnum = true
			case opcodes.OP_PUSHNUM_1:
				value = []byte{1}
				pushnum = true
			case opcodes.OP_PUSHNUM_2:
				value = []byte{2}
				pushnum = true
			case opcodes.OP_PUSHNUM_3:
				value = []byte{3}
				pushnum = true
			case opcodes.OP_PUSHNUM_4:
				value = []byte{4}
				pushnum = true
			case opcodes.OP_PUSHNUM_5:
				value = []byte{5}
				pushnum = true
			case opcodes.OP_PUSHNUM_6:
				value = []byte{6}
				pushnum = true
			case opcodes.OP_PUSHNUM_7:
				value = []byte{7}
				pushnum = true
			case opcodes.OP_PUSHNUM_8:
				value = []byte{8}
				pushnum = true
			case opcodes.OP_PUSHNUM_9:
				value = []byte{9}
				pushnum = true
			case opcodes.OP_PUSHNUM_10:
				value = []byte{10}
				pushnum = true
			case opcodes.OP_PUSHNUM_11:
				value = []byte{11}
				pushnum = true
			case opcodes.OP_PUSHNUM_12:
				value = []byte{12}
				pushnum = true
			case opcodes.OP_PUSHNUM_13:
				value = []byte{13}
				pushnum = true
			case opcodes.OP_PUSHNUM_14:
				value = []byte{14}
				pushnum = true
			case opcodes.OP_PUSHNUM_15:
				value = []byte{15}
				pushnum = true
			case opcodes.OP_PUSHNUM_16:
				value = []byte{16}
				pushnum = true
			default:
				isPushNum = false
			}

			if isPushNum {
				payload = append(payload, value)
			} else {
				return false, nil, nil
			}
		} else if instruction.IsPush {
			payload = append(payload, instruction.PushBytesData)
		} else {
			return false, nil, nil
		}
	}
}

type EnvelopeIterator struct {
	envelopes []*ParsedEnvelope
	cursor    int
}

func NewEnvelopeIterator(envelopes []*ParsedEnvelope) *EnvelopeIterator {
	return &EnvelopeIterator{
		envelopes: envelopes,
		cursor:    0,
	}
}

func (it *EnvelopeIterator) Next() (*ParsedEnvelope, bool) {
	if it.cursor >= len(it.envelopes) {
		return nil, false
	}

	envelope := it.envelopes[it.cursor]
	it.cursor++
	return envelope, true
}

func (it *EnvelopeIterator) Peek() (*ParsedEnvelope, bool) {
	if it.cursor >= len(it.envelopes) {
		return nil, false
	}

	return it.envelopes[it.cursor], true
}
