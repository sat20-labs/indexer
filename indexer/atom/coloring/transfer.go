package coloring

import (
	"bytes"
	"fmt"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// ValidateRegularTransfer is the wallet/STP boundary. It supports ordinary ARC20
// transfers only and rejects burns and any Atomicals operation envelope, including
// unsupported or malformed split/custom/mint envelopes. It does not authenticate
// indexer facts, signatures, or unrelated assets; the caller must do those checks.
func ValidateRegularTransfer(tx *wire.MsgTx, inputs []Input, height int, heights ActivationHeights) (*Result, error) {
	if tx == nil || len(tx.TxIn) != len(inputs) || len(inputs) == 0 || len(tx.TxOut) == 0 {
		return nil, fmt.Errorf("missing ARC20 transaction input facts")
	}
	seen := make(map[wire.OutPoint]bool, len(tx.TxIn))
	envelope := []byte{txscript.OP_IF, 0x04, 'a', 't', 'o', 'm'}
	for _, input := range tx.TxIn {
		if input == nil || seen[input.PreviousOutPoint] {
			return nil, fmt.Errorf("nil or duplicate ARC20 transaction input")
		}
		seen[input.PreviousOutPoint] = true
		for _, witness := range input.Witness {
			if bytes.Contains(witness, envelope) {
				return nil, fmt.Errorf("Atomicals operation envelopes are not supported by regular STP transfers")
			}
		}
	}
	result, err := ColorRegular(inputs, tx.TxOut, height, heights)
	if err != nil {
		return nil, err
	}
	if len(result.Order) == 0 {
		return nil, fmt.Errorf("missing ARC20 input allocation")
	}
	if len(result.Burned) != 0 {
		return nil, fmt.Errorf("regular STP transfer would burn ARC20 assets")
	}
	return result, nil
}
