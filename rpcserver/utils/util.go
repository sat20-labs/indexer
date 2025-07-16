package utils

import (
	"github.com/sat20-labs/indexer/share/base_indexer"
)

func IsUtxoSpent(utxo string) bool {

	return base_indexer.ShareBaseIndexer.IsUtxoSpent(utxo)
}
