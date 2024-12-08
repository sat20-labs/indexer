package runes

import (
	"sync"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dgraph-io/badger/v4"
)

type Indexer struct {
	db              *badger.DB
	chaincfgParam   *chaincfg.Params
	status          *RunesStatus
	mutex           sync.RWMutex
	runeInfoMap     *RuneInfoMap
	runeMap         *RuneMap
	mintMap         *RuneMintMap
	transferMap     *RuneTransferMap
	cnotaphMap      *RuneCenotaphMap
	addressAssetMap *AddressAssetMap
}

func New(db *badger.DB, chaincfgParam *chaincfg.Params) *Indexer {
	return &Indexer{
		db:              db,
		chaincfgParam:   chaincfgParam,
		status:          &RunesStatus{},
		runeInfoMap:     &RuneInfoMap{},
		runeMap:         &RuneMap{},
		mintMap:         &RuneMintMap{},
		transferMap:     &RuneTransferMap{},
		cnotaphMap:      &RuneCenotaphMap{},
		addressAssetMap: &AddressAssetMap{},
	}
}

func (s *Indexer) Init() {
	s.status = initStatusFromDB(s.db)
}

func (s *Indexer) Clone() *Indexer {
	newInst := New(s.db, s.chaincfgParam)
	newInst.status = s.status
	newInst.runeInfoMap = s.runeInfoMap
	newInst.runeMap = s.runeMap
	newInst.mintMap = s.mintMap
	newInst.transferMap = s.transferMap
	newInst.cnotaphMap = s.cnotaphMap
	newInst.addressAssetMap = s.addressAssetMap
	return newInst
}

func (s *Indexer) CheckSelf() bool {
	return true
}
