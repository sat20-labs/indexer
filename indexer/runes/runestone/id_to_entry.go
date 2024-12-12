package runestone

import (
	badger "github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/indexer/runes/db"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
)

type RuneIdToEntry map[*RuneId]*RuneEntry

type RuneIdToEntryTable struct {
}

func (s *RuneIdToEntryTable) Insert(key *RuneId, value RuneEntry) (oldValue *RuneEntry, err error) {
	tableKey := []byte(db.RUNEID_TO_ENTRY_KEY + key.String())
	oldPbValue, err := db.Get[pb.RuneEntry](tableKey)
	if err != nil {
		return nil, err
	}
	oldValue.FromPb(oldPbValue)
	pbValue := value.ToPb()
	err = db.Set(tableKey, pbValue)
	return
}
func (s *RuneIdToEntryTable) Get(key *RuneId) (value *RuneEntry, err error) {
	tableKey := []byte(db.RUNEID_TO_ENTRY_KEY + key.String())
	pbValue, err := db.Get[pb.RuneEntry](tableKey)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}
	value.FromPb(pbValue)
	return
}
