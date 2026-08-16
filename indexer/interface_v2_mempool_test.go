package indexer

import (
	"testing"

	"github.com/sat20-labs/indexer/common"
)

func TestRemoveConfirmedPlainDuplicates(t *testing.T) {
	unconfirmed := map[string]*common.TxOutput{
		"confirmed:0": common.NewTxOutput(100),
		"pending:0":   common.NewTxOutput(200),
	}
	confirmed := map[string]struct{}{
		"confirmed:0": {},
	}

	removeConfirmedPlainDuplicates(unconfirmed, confirmed)
	if _, exists := unconfirmed["confirmed:0"]; exists {
		t.Fatal("confirmed outpoint remained in unconfirmed plain set")
	}
	if _, exists := unconfirmed["pending:0"]; !exists {
		t.Fatal("unconfirmed-only outpoint was removed")
	}
}
