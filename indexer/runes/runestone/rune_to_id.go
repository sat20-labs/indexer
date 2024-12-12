package runestone

import (
	badger "github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/indexer/runes/db"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
)

type RuneToRuneId map[*Rune]*RuneId

type RuneToRuneIdTable struct {
}

func (s *RuneToRuneIdTable) Insert(key *Rune, value RuneId) (oldValue *RuneId, err error) {
	tableKey := []byte(db.RUNE_TO_RUNEID_KEY + key.String())
	oldPbValue, err := db.Get[pb.RuneId](tableKey)
	if err != nil {
		return nil, err
	}
	oldValue.FromPb(oldPbValue)
	pbValue := value.ToPb()
	err = db.Set(tableKey, pbValue)
	return
}
func (s *RuneToRuneIdTable) Get(key *Rune) (value *RuneId, err error) {
	tableKey := []byte(db.RUNEID_TO_ENTRY_KEY + key.String())
	pbValue, err := db.Get[pb.RuneId](tableKey)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}
	value.FromPb(pbValue)
	return
}
