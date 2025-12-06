package ord0_14_1

import (
	"github.com/btcsuite/btcd/chaincfg"
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

func InscriptionToResult(envelope *ParsedEnvelope, blockHeight int) *InscriptionResult {
	jubileeHeight := 824544
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
	
	return &InscriptionResult{
		Inscription: envelope.Payload,
		TxInIndex:   envelope.Input,
		TxInOffset:  envelope.Offset,
		CurseReason: curse,
		IsCursed:    curse != ordCommon.NoCurse,
	}
}


func GetInscriptionCurseStatus(blockHeight int, tx *common.Transaction, chainParams *chaincfg.Params) []*InscriptionResult {
	result := []*InscriptionResult{}

	envelopes := NewEnvelopeIterator(ParseEnvelopesFromTransaction(tx))
	for {
		envelope, ok := envelopes.Peek()
		if !ok {
			break
		}
		result = append(result, InscriptionToResult(envelope, blockHeight))
		envelopes.Next()
	}
	
	return result
}

func GetInscriptionsInTxInput(input *common.Input, blockHeight, inputIndex int) []*InscriptionResult {
	result := []*InscriptionResult{}

	envelopes := NewEnvelopeIterator(ParseEnvelopesFromTxInput(input, inputIndex))

	for {
		envelope, ok := envelopes.Peek()
		if !ok {
			break
		}
		result = append(result, InscriptionToResult(envelope, blockHeight))
		envelopes.Next()
	}

	return result
}
