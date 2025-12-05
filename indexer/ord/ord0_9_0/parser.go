package ord0_9_0

import (
	"bytes"
	"errors"

	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/indexer/ord/bitcoin0.30.1/blockdata/opcodes"
	"github.com/sat20-labs/indexer/indexer/ord/bitcoin0.30.1/blockdata/script"
	ordCommon "github.com/sat20-labs/indexer/indexer/ord/common"
)

type InscriptionParser struct {
	instructions      *script.Instructions
	peekedInstruction *script.Instruction
}

func Parse(witness wire.TxWitness) ([]*ordCommon.Inscription, error) {
	tapscriptBytes := ordCommon.GetTapscriptBytes(witness)
	if len(tapscriptBytes) == 0 {
		return nil, ErrNoTapscript
	}
	parser := &InscriptionParser{
		instructions: script.NewInstructions(tapscriptBytes, false),
	}
	return parser.parseInscriptions()
}

func (p *InscriptionParser) parseInscriptions() ([]*ordCommon.Inscription, error) {
	var inscriptions []*ordCommon.Inscription
	for {
		current, err := p.parseOneInscription()
		if errors.Is(err, ErrNoInscription) {
			break
		}
		if err != nil {
			return nil, err
		}
		inscriptions = append(inscriptions, current)
	}
	return inscriptions, nil
}

func (p *InscriptionParser) peek() (*script.Instruction, error) {
	if p.peekedInstruction != nil {
		return p.peekedInstruction, nil
	}

	instr, err := p.instructions.Next()
	if instr == nil && err == nil {
		return nil, ErrNoInscription
	}

	if err != nil {
		return nil, &ordCommon.ScriptError{Err: err}
	}

	p.peekedInstruction = instr
	return p.peekedInstruction, nil
}

func (p *InscriptionParser) advance() (*script.Instruction, error) {
	if p.peekedInstruction != nil {
		instr := p.peekedInstruction
		p.peekedInstruction = nil
		return instr, nil
	}
	instr, err := p.instructions.Next()
	if instr == nil && err == nil {
		return nil, ErrNoInscription
	}
	if err != nil {
		return nil, &ordCommon.ScriptError{Err: err}
	}
	return instr, nil
}

func (p *InscriptionParser) accept(target *script.Instruction) (bool, error) {
	next, err := p.peek()
	if err != nil {
		if _, ok := err.(*ordCommon.ScriptError); ok {
			return false, err
		}
		return false, nil
	}

	if next.Equal(target) {
		p.advance()
		return true, nil
	}
	return false, nil
}

func (p *InscriptionParser) expectPush() ([]byte, error) {
	instr, err := p.advance()
	if err != nil {
		return nil, err
	}
	if instr.IsPush {
		return instr.PushBytesData, nil
	}
	return nil, ErrInvalidInscription
}

func (p *InscriptionParser) advanceIntoInscriptionEnvelope() error {
	sequence := []*script.Instruction{
		script.NewInstructionPushBytes([]byte{}),
		script.NewInstructionOp(&opcodes.All{Code: opcodes.NewOrdinary(opcodes.OP_IF).Opcode}),
		script.NewInstructionPushBytes(ordCommon.PROTOCOL_ID),
	}

	for {
		matched, err := p.matchInstructions(sequence)
		if err != nil {
			return err
		}
		if matched {
			return nil
		}
		_, err = p.advance()
		if err != nil {
			return err
		}
	}
}

func (p *InscriptionParser) matchInstructions(instructions []*script.Instruction) (bool, error) {
	tempParser := *p
	for _, target := range instructions {
		next, err := tempParser.advance()
		if err != nil {
			return false, nil
		}
		if !next.Equal(target) {
			return false, nil
		}
	}
	*p = tempParser
	return true, nil
}

func (p *InscriptionParser) parseOneInscription() (*ordCommon.Inscription, error) {
	err := p.advanceIntoInscriptionEnvelope()
	if err != nil {
		return nil, err
	}

	fields := make(map[string][]byte)

fieldLoop:
	for {
		instr, err := p.advance()
		if err != nil {
			return nil, ErrInvalidInscription
		}

		switch {
		case instr.IsPush && bytes.Equal(instr.PushBytesData, ordCommon.BODY_TAG):
			var body []byte
			endIf := script.NewInstructionOp(&opcodes.All{Code: opcodes.NewOrdinary(opcodes.OP_ENDIF).Opcode})
			for {
				accepted, err := p.accept(endIf)
				if err != nil {
					return nil, err
				}
				if accepted {
					break
				}

				pushData, err := p.expectPush()
				if err != nil {
					return nil, ErrInvalidInscription
				}
				body = append(body, pushData...)
			}
			fields[string(ordCommon.BODY_TAG)] = body
			break fieldLoop

		case instr.IsPush:
			tag := instr.PushBytesData
			tagStr := string(tag)

			if _, exists := fields[tagStr]; exists {
				return nil, ErrInvalidInscription
			}

			value, err := p.expectPush()
			if err != nil {
				return nil, ErrInvalidInscription
			}
			fields[tagStr] = value

		case instr.OpCodeData.Code == opcodes.OP_ENDIF:
			break fieldLoop

		default:
			return nil, ErrInvalidInscription
		}
	}

	body := fields[string(ordCommon.BODY_TAG)]
	contentType := fields[string(ordCommon.CONTENT_TYPE_TAG)]
	parent := fields[string(ordCommon.PARENT_TAG)]

	delete(fields, string(ordCommon.BODY_TAG))
	delete(fields, string(ordCommon.CONTENT_TYPE_TAG))
	delete(fields, string(ordCommon.PARENT_TAG))

	unrecognizedEvenField := false
	for tagStr := range fields {
		tag := []byte(tagStr)
		if len(tag) > 0 {
			lsb := tag[0]
			if lsb%2 == 0 {
				unrecognizedEvenField = true
				break
			}
		}
	}

	return &ordCommon.Inscription{
		Body:                  body,
		ContentType:           contentType,
		Parent:                parent,
		UnrecognizedEvenField: unrecognizedEvenField,
	}, nil
}
