package runestone

import (
	"github.com/btcsuite/btcd/wire"
	"lukechampine.com/uint128"
)

type Message struct {
	Flaw   *Flaw
	Edicts []Edict
	Fields map[Tag][]uint128.Uint128
}

func NewMessageFromTx2Payload(tx *wire.MsgTx, payload []uint128.Uint128) (*Message, error) {
	var edicts []Edict
	fields := make(map[Tag][]uint128.Uint128)
	var flaw *Flaw

	for i := 0; i < len(payload); i += 2 {
		tag := Tag(payload[i].Lo)

		if TagBody == tag {
			id := RuneId{}
			for j := i + 1; j < len(payload); j += 4 {
				if j+3 >= len(payload) {
					flaw = FlawP(TrailingIntegers)
					break
				}

				chunk := payload[j : j+4]
				next, err := id.Next(chunk[0], chunk[1])
				if err != nil {
					flaw = FlawP(EdictRuneId)
					break
				}

				edict, err := NewEdictFromTx(tx, *next, chunk[2], chunk[3])
				if err != nil {
					flaw = FlawP(EdictOutput)
					break
				}

				id = *next
				edicts = append(edicts, *edict)
			}
			break
		}

		if i+1 < len(payload) {
			value := payload[i+1]
			fields[tag] = append(fields[tag], value)
		} else {
			flaw = FlawP(TruncatedField)
			break
		}
	}

	return &Message{
		Flaw:   flaw,
		Edicts: edicts,
		Fields: fields,
	}, nil
}

// MessageFromIntegers2
func NewMessageFromPayload(payload []uint128.Uint128) (*Message, error) {
	var edicts []Edict
	fields := make(map[Tag][]uint128.Uint128)
	var flaw *Flaw

	for i := 0; i < len(payload); i += 2 {
		tag := Tag(payload[i].Lo)

		if TagBody == tag {
			id := RuneId{}
			for j := i + 1; j < len(payload); j += 4 {
				if j+3 >= len(payload) {
					flaw = FlawP(TrailingIntegers)
					break
				}

				chunk := payload[j : j+4]
				next, err := id.Next(chunk[0], chunk[1])
				if err != nil {
					flaw = FlawP(EdictRuneId)
					break
				}

				edict, err := NewEdict(*next, chunk[2], chunk[3])
				if err != nil {
					flaw = FlawP(EdictOutput)
					break
				}

				id = *next
				edicts = append(edicts, *edict)
			}
			break
		}

		if i+1 < len(payload) {
			value := payload[i+1]
			fields[tag] = append(fields[tag], value)
		} else {
			flaw = FlawP(TruncatedField)
			break
		}
	}

	return &Message{
		Flaw:   flaw,
		Edicts: edicts,
		Fields: fields,
	}, nil
}

func (m *Message) takeFlags() uint128.Uint128 {
	u, _ := TagTake[uint128.Uint128](TagFlags, m.Fields, func(flags []uint128.Uint128) (*uint128.Uint128, error) {
		return &flags[0], nil
	})
	return *u
}
