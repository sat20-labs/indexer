package runes

import (
	"sync"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
)

type Indexer struct {
	db              *badger.DB
	chaincfgParam   *chaincfg.Params
	mutex           sync.RWMutex
	runeInfoMap     *RuneInfoMap
	runeMap         *RuneMap
	mintMap         *MintMap
	transferMap     *TransferMap
	cnotaphMap      *CenotaphMap
	addressAssetMap *AddressAssetMap
}

func New(db *badger.DB, chaincfgParam *chaincfg.Params) *Indexer {
	return &Indexer{
		db:              db,
		chaincfgParam:   chaincfgParam,
		runeInfoMap:     &RuneInfoMap{},
		runeMap:         &RuneMap{},
		mintMap:         &MintMap{},
		transferMap:     &TransferMap{},
		cnotaphMap:      &CenotaphMap{},
		addressAssetMap: &AddressAssetMap{},
	}
}

func (s *Indexer) Init() {
	common.Log.Infof("RuneIndexer->Init: rune db version: %s", s.getDBVersion())
	s.setDBVersion()
}

func (s *Indexer) getDBVersion() string {
	value, err := common.GetRawValueFromDB([]byte(DB_VER_KEY), s.db)
	if err != nil {
		common.Log.Errorf("RuneIndexer->getDBVersion: GetRawValueFromDB failed %v", err)
		return ""
	}
	return string(value)
}

func (s *Indexer) setDBVersion() {
	err := common.SetRawValueToDB([]byte(DB_VER_KEY), []byte(DB_VERSION), s.db)
	if err != nil {
		common.Log.Panicf("RuneIndexer->setDBVersion: SetRawValueToDB failed %v", err)
	}
}

func (s *Indexer) Clone() *Indexer {
	newInst := New(s.db, s.chaincfgParam)
	newInst.runeInfoMap = s.runeInfoMap
	newInst.transferMap = s.transferMap
	newInst.cnotaphMap = s.cnotaphMap
	newInst.addressAssetMap = s.addressAssetMap
	return newInst
}

func (s *Indexer) CheckSelf() bool {
	return true
}
