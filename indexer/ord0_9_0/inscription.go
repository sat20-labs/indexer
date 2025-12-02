package ord090

import (
	"log"

	"github.com/sat20-labs/indexer/common"
)

type Inscription struct {
	Body                  []byte
	ContentType           []byte
	Parent                []byte
	UnrecognizedEvenField bool
}

type TransactionInscription struct {
	Inscription Inscription
	TxInIndex   uint32
	TxInOffset  uint32
}


func FromTransaction(tx *common.Transaction) []TransactionInscription {
	result := []TransactionInscription{}
	parser := TapscriptParser{}
	for index, txIn := range tx.Inputs {
		inscriptions, err := parser.Parse(txIn.Witness)
		if err != nil {
			log.Printf("DEBUG: Failed to parse inscriptions in input %d: %v", index, err)
			continue
		}
		log.Printf("DEBUG: Found %d inscriptions in input %d", len(inscriptions), index)

		for offset, inscription := range inscriptions {
			txInscription := TransactionInscription{
				Inscription: inscription,
				TxInIndex:   uint32(index),
				TxInOffset:  uint32(offset),
			}
			result = append(result, txInscription)
		}
	}
	log.Printf("DEBUG: Found %d inscriptions in transaction", len(result))
	return result
}
