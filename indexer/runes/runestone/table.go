package runestone

import (
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

var IsLessStorage bool

type Table[T any] struct {
	cache *store.Cache[T]
}
