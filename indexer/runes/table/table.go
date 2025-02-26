package table

import (
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

var IsLessStorage bool

type Table[T any] struct {
	Cache *store.Cache[T]
}
