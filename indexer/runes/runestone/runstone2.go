package runestone

import (
	"errors"
	"fmt"
	"math"
	"unicode/utf8"

	"github.com/btcsuite/btcd/txscript"
	"lukechampine.com/uint128"
)

func (r *Runestone) Decipher2(pkScript []byte) (*Artifact, error) {
	payload, err := r.payload2(pkScript)
	if err != nil {
		if payload != nil {
			return &Artifact{
				Cenotaph: &Cenotaph{
					Flaw: &payload.Invalid,
				}}, nil
		}

		return nil, err
	}

	integers, err := r.integers(payload.Valid)
	if err != nil {
		flaw := Varint
		return &Artifact{
			Cenotaph: &Cenotaph{
				Flaw: &flaw,
			},
		}, err
	}

	message, err := MessageFromIntegers2(integers)
	flags, err := TagTake(TagFlags, message.Fields,
		func(uint128s []uint128.Uint128) (*uint128.Uint128, error) {
			return &uint128s[0], nil
		}, 1)
	if flags == nil {
		//unwrap_or_default
		flags = &uint128.Uint128{}
	}
	var etching *Etching
	if FlagEtching.Take(flags) {
		etching = &Etching{}
		etching.Divisibility, err = TagTake(TagDivisibility, message.Fields,
			func(uint128s []uint128.Uint128) (*uint8, error) {
				divisibility := uint8(uint128s[0].Lo)
				if divisibility > MaxDivisibility {
					return nil, errors.New("divisibility too high")
				}
				return &divisibility, nil
			}, 1)
		//      premine: Tag::Premine.take(&mut fields, |[premine]| Some(premine)),
		etching.Premine, err = TagTake(TagPremine, message.Fields,
			func(uint128s []uint128.Uint128) (*uint128.Uint128, error) {
				return &uint128s[0], nil
			}, 1)
		// rune: Tag::Rune.take(&mut fields, |[rune]| Some(Rune(rune))),
		etching.Rune, err = TagTake(TagRune, message.Fields,
			func(uint128s []uint128.Uint128) (*Rune, error) {
				return &Rune{Value: uint128s[0]}, nil
			}, 1)
		//      spacers: Tag::Spacers.take(&mut fields, |[spacers]| {
		//        let spacers = u32::try_from(spacers).ok()?;
		//        (spacers <= Etching::MAX_SPACERS).then_some(spacers)
		//      }),
		etching.Spacers, err = TagTake(TagSpacers, message.Fields,
			func(uint128s []uint128.Uint128) (*uint32, error) {
				spacers := uint32(uint128s[0].Lo)
				if spacers > MaxSpacers {
					return nil, errors.New("spacers too high")
				}
				return &spacers, nil
			}, 1)
		//      symbol: Tag::Symbol.take(&mut fields, |[symbol]| {
		//        char::from_u32(u32::try_from(symbol).ok()?)
		//      }),
		etching.Symbol, err = TagTake(TagSymbol, message.Fields,
			func(uint128s []uint128.Uint128) (*rune, error) {

				symbol := rune(uint32(uint128s[0].Lo))
				if symbol > utf8.MaxRune {
					return nil, errors.New("symbol too high")
				}
				return &symbol, nil
			}, 1)
		//      terms: Flag::Terms.take(&mut flags).then(|| Terms {
		//        cap: Tag::Cap.take(&mut fields, |[cap]| Some(cap)),
		//        height: (
		//          Tag::HeightStart.take(&mut fields, |[start_height]| {
		//            u64::try_from(start_height).ok()
		//          }),
		//          Tag::HeightEnd.take(&mut fields, |[start_height]| {
		//            u64::try_from(start_height).ok()
		//          }),
		//        ),
		//        amount: Tag::Amount.take(&mut fields, |[amount]| Some(amount)),
		//        offset: (
		//          Tag::OffsetStart.take(&mut fields, |[start_offset]| {
		//            u64::try_from(start_offset).ok()
		//          }),
		//          Tag::OffsetEnd.take(&mut fields, |[end_offset]| u64::try_from(end_offset).ok()),
		//        ),
		//      }),
		if FlagTerms.Take(flags) {
			terms := Terms{}
			terms.Cap, err = TagTake(TagCap, message.Fields,
				func(uint128s []uint128.Uint128) (*uint128.Uint128, error) {
					return &uint128s[0], nil
				}, 1)
			terms.Height[0], err = TagTake(TagHeightStart, message.Fields,
				func(uint128s []uint128.Uint128) (*uint64, error) {
					h := uint128s[0].Lo
					return &h, nil
				}, 1)
			terms.Height[1], err = TagTake(TagHeightEnd, message.Fields,
				func(uint128s []uint128.Uint128) (*uint64, error) {
					h := uint128s[0].Lo
					return &h, nil
				}, 1)
			terms.Amount, err = TagTake(TagAmount, message.Fields,
				func(uint128s []uint128.Uint128) (*uint128.Uint128, error) {
					return &uint128s[0], nil
				}, 1)
			terms.Offset[0], err = TagTake(TagOffsetStart, message.Fields,
				func(uint128s []uint128.Uint128) (*uint64, error) {
					h := uint128s[0].Lo
					return &h, nil
				}, 1)
			terms.Offset[1], err = TagTake(TagOffsetEnd, message.Fields,
				func(uint128s []uint128.Uint128) (*uint64, error) {
					h := uint128s[0].Lo
					return &h, nil
				}, 1)
			etching.Terms = &terms
		}
		//      turbo: Flag::Turbo.take(&mut flags),
		etching.Turbo = FlagTurbo.Take(flags)
	}
	// let mint = Tag::Mint.take(&mut fields, |[block, tx]| {
	//      RuneId::new(block.try_into().ok()?, tx.try_into().ok()?)
	//    });
	mint, err := TagTake(TagMint, message.Fields,
		func(uint128s []uint128.Uint128) (*RuneId, error) {
			block := uint64(uint128s[0].Lo)
			tx := uint32(uint128s[1].Lo)
			return NewRuneId(block, tx)
		}, 2)
	//let pointer = Tag::Pointer.take(&mut fields, |[pointer]| {
	//      let pointer = u32::try_from(pointer).ok()?;
	//      (u64::from(pointer) < u64::try_from(transaction.output.len()).unwrap()).then_some(pointer)
	//    });
	pointer, err := TagTake(TagPointer, message.Fields,
		func(uint128s []uint128.Uint128) (*uint32, error) {
			pointer := uint32(uint128s[0].Lo)
			//if uint64(pointer) < uint64(len(transaction.TxOut)) {
				return &pointer, nil
			//}
			//return nil, errors.New("pointer too high")

		}, 1)
	//if etching
	//      .map(|etching| etching.supply().is_none())
	//      .unwrap_or_default()
	//    {
	//      flaw.get_or_insert(Flaw::SupplyOverflow);
	//    }
	if etching != nil && etching.Supply() == nil {
		message.Flaw = FlawP(SupplyOverflow)

	}
	// if flags != 0 {
	//      flaw.get_or_insert(Flaw::UnrecognizedFlag);
	//    }
	if !flags.IsZero() {
		message.Flaw = FlawP(UnrecognizedFlag)

	}
	//    if fields.keys().any(|tag| tag % 2 == 0) {
	//      flaw.get_or_insert(Flaw::UnrecognizedEvenTag);
	//    }
	for tag := range message.Fields {
		if tag%2 == 0 {
			message.Flaw = FlawP(UnrecognizedEvenTag)
		}

	}
	//if let Some(flaw) = flaw {
	//      return Some(Artifact::Cenotaph(Cenotaph {
	//        flaw: Some(flaw),
	//        mint,
	//        etching: etching.and_then(|etching| etching.rune),
	//      }));
	//    }
	if message.Flaw != nil {
		a := &Artifact{
			Cenotaph: &Cenotaph{
				Flaw: message.Flaw,
				Mint: mint,
			},
		}
		if etching != nil {
			a.Cenotaph.Etching = etching.Rune
		}
		return a, nil

	}

	return &Artifact{
		Runestone: &Runestone{
			Edicts:  message.Edicts,
			Etching: etching,
			Mint:    mint,
			Pointer: pointer,
		},
	}, nil
}

func (r *Runestone) payload2(pkScript []byte) (*Payload, error) {
	
	tokenizer := txscript.MakeScriptTokenizer(0, pkScript)
	if !tokenizer.Next() || tokenizer.Err() != nil || tokenizer.Opcode() != txscript.OP_RETURN {
		// Check for OP_RETURN
		return nil, fmt.Errorf("not OP_RETURN")
	}
	if !tokenizer.Next() || tokenizer.Err() != nil || tokenizer.Opcode() != MAGIC_NUMBER {
		// Check for protocol identifier (Runestone::MAGIC_NUMBER)
		return nil, fmt.Errorf("not RUNESTONE")
	}

	// Construct the payload by concatenating remaining data pushes
	var payload []byte
	for tokenizer.Next() {
		//is PushBytes
		if isPushBytes(tokenizer.Opcode()) {
			payload = append(payload, tokenizer.Data()...)
			continue
		} else {
			return &Payload{Invalid: Opcode}, Opcode.Error()
		}

	}
	
	if tokenizer.Err() != nil {
		return &Payload{Invalid: InvalidScript}, InvalidScript.Error()
	}

	return &Payload{Valid: payload}, nil
	
}

func MessageFromIntegers2(payload []uint128.Uint128) (*Message, error) {
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

				edict, err := EdictFromIntegers2(*next, chunk[2], chunk[3])
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

func EdictFromIntegers2(id RuneId, amount uint128.Uint128, output uint128.Uint128) (*Edict, error) {
	if output.Hi > 0 || output.Lo > math.MaxUint32 {
		return nil, errors.New("output overflow")
	}
	output32 := uint32(output.Lo)

	return &Edict{
		ID:     id,
		Amount: amount,
		Output: output32,
	}, nil
}
