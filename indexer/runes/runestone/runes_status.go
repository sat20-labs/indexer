package runestone

import (
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
)

type RunesStatus struct {
	Table[pb.RunesStatus]
	Version       string
	Height        uint64
	Number        uint64
	ReservedRunes uint64
}

func NewRunesStatus(s *store.Cache[pb.RunesStatus]) *RunesStatus {
	return &RunesStatus{
		Table: Table[pb.RunesStatus]{Cache: s},
	}
}

func (s *RunesStatus) Init() (ret bool) {
	stats, _ := s.Cache.GetFromDB([]byte(store.STATUS_KEY))
	common.Log.Infof("RunesStatus.Init-> stats:%v", stats)
	if stats == nil {
		s.Version = store.DB_VERSION
		ret = false
	} else {
		if stats.Version != store.DB_VERSION {
			common.Log.Panicf("RunesStatus.Init-> db version inconsistent %s", store.DB_VERSION)
		}
		s.Version = stats.Version
		s.Height = stats.Height
		s.Number = stats.Number
		s.ReservedRunes = stats.ReservedRunes
		ret = true
	}
	return
}

func (s *RunesStatus) UpdateDb() {
	key := []byte(store.STATUS_KEY)
	value := &pb.RunesStatus{
		Version:       s.Version,
		Height:        s.Height,
		Number:        s.Number,
		ReservedRunes: s.ReservedRunes,
	}
	s.Cache.SetToDB(key, value)
}

func (s *RunesStatus) Update() {
	key := []byte(store.STATUS_KEY)
	value := &pb.RunesStatus{
		Version:       s.Version,
		Height:        s.Height,
		Number:        s.Number,
		ReservedRunes: s.ReservedRunes,
	}
	s.Cache.Set(key, value)
}
