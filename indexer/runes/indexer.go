package runes

import (
	"sync"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/indexer/runes/db"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
)

type Indexer struct {
	chaincfgParam         *chaincfg.Params
	height                uint64
	status                runestone.RunesStatus
	minimumRune           *runestone.Rune
	mutex                 sync.RWMutex
	runeInfoMap           *RuneInfoMap
	runeMap               *RuneMap
	mintMap               *RuneMintMap
	transferMap           *RuneTransferMap
	cnotaphMap            *RuneCenotaphMap
	addressAssetMap       *AddressAssetMap
	outpointToBalancesTbl runestone.OutpointToBalancesTable
	runeIdToEntryTbl      runestone.RuneIdToEntryTable
	runeToRuneIdTbl       runestone.RuneToRuneIdTable
}

func New(newDb *badger.DB, chaincfgParam *chaincfg.Params) *Indexer {
	db.SetDB(newDb)
	return &Indexer{
		chaincfgParam:   chaincfgParam,
		runeInfoMap:     &RuneInfoMap{},
		runeMap:         &RuneMap{},
		mintMap:         &RuneMintMap{},
		transferMap:     &RuneTransferMap{},
		cnotaphMap:      &RuneCenotaphMap{},
		addressAssetMap: &AddressAssetMap{},
	}
}

func (s *Indexer) Init() {
	s.status.Init()
}

func (s *Indexer) Clone() *Indexer {
	newInst := New(db.GetDB(), s.chaincfgParam)
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
