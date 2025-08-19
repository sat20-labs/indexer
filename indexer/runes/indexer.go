package runes

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/dgraph-io/badger/v4"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/base"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"github.com/sat20-labs/indexer/indexer/runes/table"
	"lukechampine.com/uint128"
)

type Indexer struct {
	dbWrite                       *store.DbWrite
	RpcService                    *base.RpcIndexer
	BaseIndexer                   *base.BaseIndexer
	isUpdateing                   bool
	chaincfgParam                 *chaincfg.Params
	height                        uint64
	blockTime                     uint64
	Status                        *table.RunesStatus
	minimumRune                   *runestone.Rune
	burnedMap                     table.RuneIdLotMap
	HolderUpdateCount             int
	HolderRemoveCount             int
	idToEntryTbl                  *table.RuneIdToEntryTable
	runeToIdTbl                   *table.RuneToIdTable
	outpointToBalancesTbl         *table.OutpointToBalancesTable
	runeIdAddressToBalanceTbl     *table.RuneIdAddressToBalanceTable
	runeIdOutpointToBalanceTbl    *table.RuneIdOutpointToBalanceTable
	addressOutpointToBalancesTbl  *table.AddressOutpointToBalancesTable
	runeIdAddressToCountTbl       *table.RuneIdAddressToCountTable
	runeIdToMintHistoryTbl        *table.RuneToMintHistoryTable
	addressRuneIdToMintHistoryTbl *table.AddressRuneIdToMintHistoryTable
}

func NewIndexer(db *badger.DB, param *chaincfg.Params, baseIndexer *base.BaseIndexer, rpcService *base.RpcIndexer) *Indexer {
	logs := cmap.New[*store.DbLog]()
	dbWrite := store.NewDbWrite(db, &logs)
	table.IsLessStorage = true
	return &Indexer{
		BaseIndexer:                   baseIndexer,
		RpcService:                    rpcService,
		dbWrite:                       dbWrite,
		chaincfgParam:                 param,
		burnedMap:                     nil,
		Status:                        table.NewRunesStatus(store.NewCache[pb.RunesStatus](dbWrite)),
		idToEntryTbl:                  table.NewRuneIdToEntryTable(store.NewCache[pb.RuneEntry](dbWrite)),
		runeToIdTbl:                   table.NewRuneToIdTable(store.NewCache[pb.RuneId](dbWrite)),
		outpointToBalancesTbl:         table.NewOutpointToBalancesTable(store.NewCache[pb.OutpointToBalances](dbWrite)),
		runeIdAddressToBalanceTbl:     table.NewRuneIdAddressToBalanceTable(store.NewCache[pb.RuneIdAddressToBalance](dbWrite)),
		runeIdOutpointToBalanceTbl:    table.NewRuneIdOutpointToBalancesTable(store.NewCache[pb.RuneBalance](dbWrite)),
		addressOutpointToBalancesTbl:  table.NewAddressOutpointToBalancesTable(store.NewCache[pb.AddressOutpointToBalance](dbWrite)),
		runeIdAddressToCountTbl:       table.NewRuneIdAddressToCountTable(store.NewCache[pb.RuneIdAddressToCount](dbWrite)),
		runeIdToMintHistoryTbl:        table.NewRuneIdToMintHistoryTable(store.NewCache[pb.RuneIdToMintHistory](dbWrite)),
		addressRuneIdToMintHistoryTbl: table.NewAddressRuneIdToMintHistoryTable(store.NewCache[pb.AddressRuneIdToMintHistory](dbWrite)),
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
		s.Status.UpdateDb()

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
	s.minimumRune = runestone.MinimumAtHeight(s.chaincfgParam.Net, uint64(s.Status.Height))
}

func (s *Indexer) Clone() *Indexer {
	cloneIndex := NewIndexer(s.dbWrite.Db, s.chaincfgParam, s.BaseIndexer, s.RpcService)
	cloneIndex.height = s.height
	cloneIndex.Status.Version = s.Status.Version
	cloneIndex.Status.Height = s.Status.Height
	cloneIndex.Status.Number = s.Status.Number
	cloneIndex.Status.ReservedRunes = s.Status.ReservedRunes
	s.dbWrite.Clone(cloneIndex.dbWrite)
	lastRuneInfosCacheTimeStamp = 0
	runeInfosCache = nil
	runeMintHistoryCache = cmap.New[*MintHistoryInfo]()

	return cloneIndex
}

func (s *Indexer) Subtract(backupIndexer *Indexer) {
	backupIndexer.dbWrite.Subtract(s.dbWrite)
}

func (s *Indexer) CheckSelf() bool {
	common.Log.Infof("total runes: %d", len(s.GetAllRuneIds()))

	var firstRuneName = ""
	switch s.chaincfgParam.Net {
	case wire.TestNet4:
		firstRuneName = "BESTINSLOT•XYZ"
		if s.height < 30562 {
			return true
		}
	case wire.MainNet:
		firstRuneName = "UNCOMMON•GOODS"
		if s.height < 840000 {
			return true
		}
	default:
		common.Log.Panicf("RuneIndexer.CheckSelf-> unknown net:%d", s.chaincfgParam.Net)
	}
	runeId, err := s.GetRuneIdWithName(firstRuneName)
	if err != nil {
		common.Log.Panicf("GetRuneIdWithName err:%s", err.Error())
	}
	common.Log.Debugf("rune: %s\n", firstRuneName)

	runeInfo := s.GetRuneInfoWithId(runeId.String())
	_, total := s.GetAllAddressBalances(runeId.String(), 0, 1)
	addressBalances, _ := s.GetAllAddressBalances(runeId.String(), 0, total)
	var addressBalance uint128.Uint128
	for _, v := range addressBalances {
		addressBalance = v.Balance.Add(addressBalance)
	}

	totalAddressBalance := addressBalance.Add(runeInfo.Burned)
	if addressBalance.Add(runeInfo.Burned).Cmp(totalAddressBalance) != 0 {
		common.Log.Errorf("all address(%d)'s total balance(%s) + burned is not equal to supply(%s)", total, totalAddressBalance.String(), runeInfo.Supply.String())
		return false
	}

	_, total = s.GetAllUtxoBalances(runeId.String(), 0, 1)
	utxoBalances, _ := s.GetAllUtxoBalances(runeId.String(), 0, total)
	totalUtxoBalance := utxoBalances.Total.Add(runeInfo.Burned)
	if utxoBalances.Total.Add(runeInfo.Burned).Cmp(totalUtxoBalance) != 0 {
		common.Log.Errorf("all utxo(%d)'s total balance(%s) + burned is not equal to supply(%s)", total, totalUtxoBalance.String(), runeInfo.Supply.String())
		return false
	}

	return true
}
