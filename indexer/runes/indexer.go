package runes

import (
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/dgraph-io/badger/v4"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/base"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"lukechampine.com/uint128"
)

type Indexer struct {
	db                            *badger.DB
	RpcService                    *base.RpcIndexer
	BaseIndexer                   *base.BaseIndexer
	cloneTimeStamp                int64
	isUpdateing                   bool
	wb                            *badger.WriteBatch
	cacheLogs                     *cmap.ConcurrentMap[string, *store.CacheLog]
	chaincfgParam                 *chaincfg.Params
	height                        uint64
	blockTime                     uint64
	Status                        *runestone.RunesStatus
	minimumRune                   *runestone.Rune
	burnedMap                     runestone.RuneIdLotMap
	idToEntryTbl                  *runestone.RuneIdToEntryTable
	runeToIdTbl                   *runestone.RuneToIdTable
	outpointToBalancesTbl         *runestone.OutpointToBalancesTable
	runeIdAddressToBalanceTbl     *runestone.RuneIdAddressToBalanceTable
	runeIdOutpointToBalanceTbl    *runestone.RuneIdOutpointToBalanceTable
	addressOutpointToBalancesTbl  *runestone.AddressOutpointToBalancesTable
	runeIdToMintHistoryTbl        *runestone.RuneToMintHistoryTable
	addressRuneIdToMintHistoryTbl *runestone.AddressRuneIdToMintHistoryTable
}

func NewIndexer(db *badger.DB, param *chaincfg.Params, baseIndexer *base.BaseIndexer, rpcService *base.RpcIndexer) *Indexer {
	store.SetDB(db)
	return &Indexer{
		db:                            db,
		BaseIndexer:                   baseIndexer,
		RpcService:                    rpcService,
		cloneTimeStamp:                0,
		cacheLogs:                     nil,
		chaincfgParam:                 param,
		burnedMap:                     nil,
		Status:                        runestone.NewRunesStatus(store.NewCache[pb.RunesStatus]()),
		idToEntryTbl:                  runestone.NewRuneIdToEntryTable(store.NewCache[pb.RuneEntry]()),
		runeToIdTbl:                   runestone.NewRuneToIdTable(store.NewCache[pb.RuneId]()),
		outpointToBalancesTbl:         runestone.NewOutpointToBalancesTable(store.NewCache[pb.OutpointToBalances]()),
		runeIdAddressToBalanceTbl:     runestone.NewRuneIdAddressToBalanceTable(store.NewCache[pb.RuneIdAddressToBalance]()),
		runeIdOutpointToBalanceTbl:    runestone.NewRuneIdOutpointToBalancesTable(store.NewCache[pb.RuneBalance]()),
		addressOutpointToBalancesTbl:  runestone.NewAddressOutpointToBalancesTable(store.NewCache[pb.AddressOutpointToBalance]()),
		runeIdToMintHistoryTbl:        runestone.NewRuneIdToMintHistoryTable(store.NewCache[pb.RuneIdToMintHistory]()),
		addressRuneIdToMintHistoryTbl: runestone.NewAddressRuneIdToMintHistoryTable(store.NewCache[pb.AddressRuneIdToMintHistory]()),
	}
}

func (s *Indexer) Init() {
	isExist := s.Status.Init()
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

		s.Status.Number = 1
		s.Status.FlushToDB()

		symbol := defaultRuneSymbol
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
			SpacedRune: *runestone.NewSpacedRune(r, 128),
			Symbol:     &symbol,
			Timestamp:  0,
			Turbo:      true,
		})
	}
}

func (s *Indexer) Clone() *Indexer {
	cloneIndex := NewIndexer(s.db, s.chaincfgParam, s.BaseIndexer, s.RpcService)
	for log := range s.cacheLogs.IterBuffered() {
		cacheLog := &store.CacheLog{
			Type:      log.Val.Type,
			ExistInDb: log.Val.ExistInDb,
			TimeStamp: log.Val.TimeStamp,
		}
		if log.Val.Val != nil {
			cacheLog.Val = make([]byte, len(log.Val.Val))
			copy(cacheLog.Val, log.Val.Val)
		}
		if cloneIndex.cacheLogs == nil {
			cacheLogs := cmap.New[*store.CacheLog]()
			cloneIndex.cacheLogs = &cacheLogs
		}
		cloneIndex.cacheLogs.Set(log.Key, cacheLog)
	}
	cloneIndex.cloneTimeStamp = time.Now().UnixNano()
	return cloneIndex
}

func (s *Indexer) Subtract(backupIndexer *Indexer) {
	for log := range backupIndexer.cacheLogs.IterBuffered() {
		if log.Val.TimeStamp <= s.cloneTimeStamp {
			s.cacheLogs.Remove(log.Key)
		}
	}
}

func (s *Indexer) CheckSelf() bool {
	return true
}
