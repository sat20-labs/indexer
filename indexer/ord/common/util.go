package common

import (
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

func GetTapscriptBytes(witness wire.TxWitness) []byte {
	// From BIP341:
	// If there are at least two witness elements, and the first byte of
	// the last element is 0x50, this last element is called annex a
	// and is removed from the witness stack.
	lenWitness := len(witness)
	if lenWitness < 2 {
		return nil
	}
	lastElement := witness[lenWitness-1]
	if len(lastElement) < 1 {
		return nil
	}
	if lastElement[0] != txscript.TaprootAnnexTag {
		// otherwise script is 2nd from last
		if lenWitness < 2 {
			return nil
		}
		return witness[lenWitness-2]
	} else {
		// account for the extra item removed from the end
		if lenWitness < 3 {
			return witness[lenWitness-3]
		}
	}
	return nil
}
