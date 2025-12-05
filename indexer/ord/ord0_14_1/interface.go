package ord0_14_1

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
	ordCommon "github.com/sat20-labs/indexer/indexer/ord/common"
)

type InscriptionResult struct {
	Inscription *ordCommon.Inscription
	TxInIndex   uint32
	TxInOffset  uint32
	IsCursed    bool
	CurseReason ordCommon.Curse
}

func GetInscriptionCurseStatus(blockHeight int, tx *common.Transaction, chainParams *chaincfg.Params) []*InscriptionResult {
	result := []*InscriptionResult{}

	jubileeHeight := 0
	switch chainParams.Net {
	case wire.MainNet:
		jubileeHeight = 824544
	case wire.TestNet4:
		jubileeHeight = 0
	}
	envelopes := NewEnvelopeIterator(ParseEnvelopesFromTransaction(tx))
	for inputIndex := range tx.Inputs {
		for {
			envelope, ok := envelopes.Peek()
			if !ok {
				break
			}
			if envelope.Input != uint32(inputIndex) {
				break
			}

			var curse ordCommon.Curse
			if blockHeight >= jubileeHeight {
				curse = ordCommon.NoCurse
			} else if envelope.Payload.UnrecognizedEvenField {
				curse = ordCommon.UnrecognizedEvenField
			} else if envelope.Payload.DuplicateField {
				curse = ordCommon.DuplicateField
			} else if envelope.Payload.IncompleteField {
				curse = ordCommon.IncompleteField
			} else if envelope.Input != 0 {
				curse = ordCommon.NotInFirstInput
			} else if envelope.Offset != 0 {
				curse = ordCommon.NotAtOffsetZero
			} else if envelope.Payload.Pointer != nil {
				curse = ordCommon.Pointer
			} else if envelope.Pushnum {
				curse = ordCommon.Pushnum
			} else if envelope.Stutter {
				curse = ordCommon.Stutter
			} else {
				curse = ordCommon.NoCurse
			} // else reinscription
			result = append(result,
				&InscriptionResult{
					Inscription: envelope.Payload,
					TxInIndex:   uint32(inputIndex),
					TxInOffset:  envelope.Offset,
					CurseReason: curse,
					IsCursed:    curse != ordCommon.NoCurse,
				})
		}
	}
	return result
}

func GetInscriptionsInTxInput(input *common.TxInput) []*InscriptionResult {
	result := []*InscriptionResult{}

	jubileeHeight := 824544
	envelopes := NewEnvelopeIterator(ParseEnvelopesFromTxInput(input))
	
	for {
		envelope, ok := envelopes.Peek()
		if !ok {
			break
		}
		if envelope.Input != uint32(input.TxIndex) {
			break
		}

		var curse ordCommon.Curse
		if input.Height >= jubileeHeight {
			curse = ordCommon.NoCurse
		} else if envelope.Payload.UnrecognizedEvenField {
			curse = ordCommon.UnrecognizedEvenField
		} else if envelope.Payload.DuplicateField {
			curse = ordCommon.DuplicateField
		} else if envelope.Payload.IncompleteField {
			curse = ordCommon.IncompleteField
		} else if envelope.Input != 0 {
			curse = ordCommon.NotInFirstInput
		} else if envelope.Offset != 0 {
			curse = ordCommon.NotAtOffsetZero
		} else if envelope.Payload.Pointer != nil {
			curse = ordCommon.Pointer
		} else if envelope.Pushnum {
			curse = ordCommon.Pushnum
		} else if envelope.Stutter {
			curse = ordCommon.Stutter
		} else {
			curse = ordCommon.NoCurse
		} // else reinscription
		result = append(result,
			&InscriptionResult{
				Inscription: envelope.Payload,
				TxInIndex:   uint32(input.TxIndex),
				TxInOffset:  envelope.Offset,
				CurseReason: curse,
				IsCursed:    curse != ordCommon.NoCurse,
			})
	}
	
	return result
}
