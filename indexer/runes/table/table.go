package table

import (
	"github.com/sat20-labs/indexer/indexer/runes/store"
)


type Table[T any] struct {
	Cache *store.Cache[T]
}
