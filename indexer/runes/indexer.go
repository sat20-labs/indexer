package runes

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"lukechampine.com/uint128"
)

type Indexer struct {
	db                        *badger.DB
	wb                        *badger.WriteBatch
	chaincfgParam             *chaincfg.Params
	height                    uint64
	blockTime                 uint64
	status                    *runestone.RunesStatus
	minimumRune               *runestone.Rune
	runeLedger                *runestone.RuneLedger
	burnedMap                 runestone.RuneIdLotMap
	runeLedgerTbl             *runestone.RuneLedgerTable
	outpointToRuneBalancesTbl *runestone.OutpointToRuneBalancesTable
	idToEntryTbl              *runestone.RuneIdToEntryTable
	runeToIdTbl               *runestone.RuneToIdTable
	runeHolderTbl             *runestone.RuneHoldersTable
	runeMintHistorysTbl       *runestone.RuneMintHistorysTable
}

func NewIndexer(db *badger.DB, param *chaincfg.Params) *Indexer {
	return &Indexer{
		db:                        db,
		chaincfgParam:             param,
		runeLedger:                nil,
		burnedMap:                 nil,
		status:                    runestone.NewRunesStatus(store.NewCache[pb.RunesStatus](db)),
		outpointToRuneBalancesTbl: runestone.NewOutpointToRuneBalancesTable(store.NewCache[pb.OutpointToRuneBalances](db)),
		idToEntryTbl:              runestone.NewRuneIdToEntryTable(store.NewCache[pb.RuneEntry](db)),
		runeToIdTbl:               runestone.NewRuneToIdTable(store.NewCache[pb.RuneId](db)),
		runeLedgerTbl:             runestone.NewRuneLedgerTable(store.NewCache[pb.RuneLedger](db)),
		runeHolderTbl:             runestone.NewRuneHoldersTable(store.NewCache[pb.RuneHolders](db)),
		runeMintHistorysTbl:       runestone.NewRuneMintHistorysTable(store.NewCache[pb.RuneMintHistorys](db)),
	}
}

func (s *Indexer) Init() {
	isExist := s.status.Init()
	if !isExist && s.chaincfgParam.Net == wire.MainNet {
		firstRuneValue, err := uint128.FromString("2055900680524219742")
		if err != nil {
			common.Log.Panicf("RuneIndexer.Init-> uint128.FromString(2055900680524219742) err: %v", err)
		}
		r := runestone.Rune{
			Value: firstRuneValue,
		}
		id := &runestone.RuneId{Block: 1, Tx: 0}
		etching := "0000000000000000000000000000000000000000000000000000000000000000"
		s.runeToIdTbl.SetToDB(&r, id)

		s.status.Number = 1
		s.status.SetToDB()

		symbol := '\u29C9'
		startHeight := uint64(runestone.SUBSIDY_HALVING_INTERVAL * 4)
		endHeight := uint64(runestone.SUBSIDY_HALVING_INTERVAL * 5)
		s.idToEntryTbl.SetToDB(id, &runestone.RuneEntry{
			RuneId:       *id,
			Burned:       uint128.Uint128{},
			Divisibility: 0,
			Etching:      etching,
			Parent:       nil,
			Terms: &runestone.Terms{
				Amount: &uint128.Uint128{Hi: 0, Lo: 1},
				Cap:    &uint128.Max,
				Height: [2]*uint64{&startHeight, &endHeight},
				Offset: [2]*uint64{nil, nil},
			},
			Mints:      uint128.Uint128{},
			Number:     0,
			Premine:    uint128.Uint128{},
			SpacedRune: runestone.SpacedRune{Rune: r, Spacers: 128},
			Symbol:     &symbol,
			Timestamp:  0,
			Turbo:      true,
		})
	}
}

func (s *Indexer) Clone() *Indexer {
	return s
}

func (s *Indexer) CheckSelf() bool {
	return true
}
