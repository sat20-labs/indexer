package runestone

import (
	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type Table[T any] struct {
	cache *store.Cache[T]
}

func (s *Table[T]) SetWb(wb *badger.WriteBatch) {
	s.cache.SetWb(wb)
}

func (s *Table[T]) Flush() {
	s.cache.Flush()
}
