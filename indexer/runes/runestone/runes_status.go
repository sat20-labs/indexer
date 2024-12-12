package runestone

import (
	badger "github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/db"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
)

type RunesStatus struct {
	Version       string
	Height        uint64
	Number        uint64
	ReservedRunes uint64
}

func (s *RunesStatus) Init() *RunesStatus {
	stats, err := db.Get[pb.RunesStatus]([]byte(db.STATUS_KEY))
	if err == badger.ErrKeyNotFound {
		stats.Version = db.DB_VERSION
	} else if err != nil {
		common.Log.Panicf("Runes.RunesStatus->Load: err: %v", err)
		return nil
	}
	common.Log.Infof("Runes.RunesStatus->Load: stats: %v", stats)
	if stats.Version != db.DB_VERSION {
		common.Log.Panicf("Runes.RunesStatus->Load: db version inconsistent %s", db.DB_VERSION)
	}
	ret := &RunesStatus{
		Version:       stats.Version,
		Height:        stats.Height,
		Number:        stats.Number,
		ReservedRunes: stats.ReservedRunes,
	}
	return ret
}

func (s *RunesStatus) Update() error {
	key := []byte(db.STATUS_KEY)
	value := &pb.RunesStatus{
		Version:       s.Version,
		Height:        s.Height,
		Number:        s.Number,
		ReservedRunes: s.ReservedRunes,
	}
	return db.Set(key, value)
}
