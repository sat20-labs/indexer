package runestone

import (
	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type Table[T any] struct {
	store *store.Store[T]
}

func (s *Table[T]) SetTxn(txn *badger.Txn) {
	s.store.SetTxn(txn)
}
