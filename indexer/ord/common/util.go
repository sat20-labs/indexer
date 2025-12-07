package common

import (
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// Get Tapscript following BIP341 rules regarding accounting for an annex.
func GetTapscriptBytes(witness wire.TxWitness) []byte {
	lenWitness := len(witness)
	if lenWitness == 0 {
		return nil
	}
	lastElement := witness[lenWitness-1]
	// From BIP341:
	// If there are at least two witness elements, and the first byte of
	// the last element is 0x50, this last element is called annex a
	// and is removed from the witness stack.
	scriptPosFromLast := -1
	if lenWitness >= 2 && len(lastElement) >= 1 && lastElement[0] == txscript.TaprootAnnexTag {
		// account for the extra item removed from the end
		scriptPosFromLast = 3
	} else {
		// otherwise script is 2nd from last
		scriptPosFromLast = 2
	}
	if lenWitness >= scriptPosFromLast {
		return witness[lenWitness-scriptPosFromLast]
	}
	return nil
}
