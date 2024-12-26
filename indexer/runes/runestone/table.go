package runestone

import (
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type Table[T any] struct {
	cache *store.Cache[T]
}
