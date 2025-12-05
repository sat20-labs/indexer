package ord0_9_0

import (
	"github.com/sat20-labs/indexer/common"
	ordCommon "github.com/sat20-labs/indexer/indexer/ord/common"
)

type TransactionInscription struct {
	Inscription *ordCommon.Inscription
	TxInIndex   int
	TxInOffset  int
	Err         error
}

type InscriptionResult struct {
	Inscription *TransactionInscription
	IsCursed    bool
	CurseReason ordCommon.Curse
	Err         error
}

func GetInscriptionsFromTransaction(tx *common.Transaction) []*TransactionInscription {
	result := []*TransactionInscription{}
	for index, txIn := range tx.Inputs {
		inscriptions, err := Parse(txIn.Witness)
		if err != nil {
			common.Log.Debugf("Failed to parse inscriptions in input %d: %v", index, err)
			txInscription := &TransactionInscription{
				Inscription: nil,
				TxInIndex:   index,
				TxInOffset:  -1,
				Err:         err,
			}
			result = append(result, txInscription)
			continue
		}
		common.Log.Debugf("Found %d inscriptions in input %d", len(inscriptions), index)
		for offset, inscription := range inscriptions {
			txInscription := &TransactionInscription{
				Inscription: inscription,
				TxInIndex:   index,
				TxInOffset:  offset,
				Err:         nil,
			}
			result = append(result, txInscription)
		}
	}
	common.Log.Debugf("Found %d inscriptions in transaction", len(result))
	return result
}

func GetInscriptionCurseStatus(tx *common.Transaction) []*InscriptionResult {
	inscriptions := GetInscriptionsFromTransaction(tx)
	results := make([]*InscriptionResult, 0, len(inscriptions))
	for _, inscription := range inscriptions {
		curseStatus := &InscriptionResult{
			Inscription: inscription,
			IsCursed:    false,
			CurseReason: ordCommon.NoCurse,
			Err:         inscription.Err,
		}
		if inscription.Inscription != nil {
			if inscription.Inscription.UnrecognizedEvenField {
				curseStatus.IsCursed = true
				curseStatus.CurseReason = ordCommon.UnrecognizedEvenField
			} else if inscription.TxInIndex != 0 {
				curseStatus.IsCursed = true
				curseStatus.CurseReason = ordCommon.NotInFirstInput
			} else if inscription.TxInOffset != 0 {
				curseStatus.IsCursed = true

			}
			curseStatus.CurseReason = ordCommon.NotAtOffsetZero
		}
		results = append(results, curseStatus)
	}
	return results
}
