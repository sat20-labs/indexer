package runes

import (
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/dgraph-io/badger/v4"
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
	cacheLogs                     map[string]*store.CacheLog
	chaincfgParam                 *chaincfg.Params
	height                        uint64
	blockTime                     uint64
	Status                        *runestone.RunesStatus
	minimumRune                   *runestone.Rune
	burnedMap                     runestone.RuneIdLotMap
	idToEntryTbl                  *runestone.RuneIdToEntryTable
	runeToIdTbl                   *runestone.RuneToIdTable
	outpointToRuneBalancesTbl     *runestone.OutpointToRuneBalancesTable
	runeIdToAddressTbl            *runestone.RuneToAddressTable
	runeIdToOutpointTbl           *runestone.RuneIdToOutpointTable
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
		outpointToRuneBalancesTbl:     runestone.NewOutpointToRuneBalancesTable(store.NewCache[pb.OutpointToRuneBalances]()),
		runeIdToAddressTbl:            runestone.NewRuneIdToAddressTable(store.NewCache[pb.RuneIdToAddress]()),
		runeIdToOutpointTbl:           runestone.NewRuneIdToUtxoTable(store.NewCache[pb.RuneIdToOutpoint]()),
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
	for k, v := range s.cacheLogs {
		cacheLog := &store.CacheLog{
			Type:      v.Type,
			ExistInDb: v.ExistInDb,
			TimeStamp: v.TimeStamp,
		}
		if v.Val != nil {
			cacheLog.Val = make([]byte, len(v.Val))
			copy(cacheLog.Val, v.Val)
		}
		if cloneIndex.cacheLogs == nil {
			cloneIndex.cacheLogs = make(map[string]*store.CacheLog)
		}
		cloneIndex.cacheLogs[k] = cacheLog
	}
	cloneIndex.cloneTimeStamp = time.Now().UnixNano()
	return cloneIndex
}

func (b *Indexer) Subtract(backupIndexer *Indexer) {
	for k, v := range backupIndexer.cacheLogs {
		if v.TimeStamp <= b.cloneTimeStamp {
			delete(b.cacheLogs, k)
		}
	}
}

func (s *Indexer) CheckSelf() bool {
	return true
}
