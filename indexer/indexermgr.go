package indexer

import (
	"sync"
	"time"

	"github.com/sat20-labs/indexer/common"
	base_indexer "github.com/sat20-labs/indexer/indexer/base"
	"github.com/sat20-labs/indexer/indexer/brc20"
	"github.com/sat20-labs/indexer/indexer/exotic"
	"github.com/sat20-labs/indexer/indexer/ft"
	"github.com/sat20-labs/indexer/indexer/nft"
	"github.com/sat20-labs/indexer/indexer/ns"
	"github.com/sat20-labs/indexer/indexer/runes"

	"github.com/btcsuite/btcd/chaincfg"

	"github.com/dgraph-io/badger/v4"
)

type IndexerMgr struct {
	dbDir string
	// data from blockchain
	baseDB  *badger.DB
	ftDB    *badger.DB
	nsDB    *badger.DB
	nftDB   *badger.DB
	brc20DB *badger.DB
	runesDB *badger.DB
	// data from market
	localDB *badger.DB

	// 配置参数
	chaincfgParam   *chaincfg.Params
	ordxFirstHeight int
	ordFirstHeight  int
	maxIndexHeight  int
	periodFlushToDB int

	brc20Indexer *brc20.BRC20Indexer
	RunesIndexer *runes.Indexer
	exotic       *exotic.ExoticIndexer
	ftIndexer    *ft.FTIndexer
	ns           *ns.NameService
	nft          *nft.NftIndexer
	clmap        map[common.TickerName]map[string]int64 // collections map, ticker -> inscriptionId -> asset amount

	mutex sync.RWMutex
	// 跑数据
	lastCheckHeight int
	compiling       *base_indexer.BaseIndexer
	// 备份所有需要写入数据库的数据
	compilingBackupDB *base_indexer.BaseIndexer
	exoticBackupDB    *exotic.ExoticIndexer
	brc20BackupDB     *brc20.BRC20Indexer
	runesBackupDB     *runes.Indexer
	ftBackupDB        *ft.FTIndexer
	nsBackupDB        *ns.NameService
	nftBackupDB       *nft.NftIndexer
	// 接收前端api访问的实例，隔离内存访问
	rpcService *base_indexer.RpcIndexer

	// 本地缓存，在区块更新时清空
	addressToNftMap  map[string][]*common.Nft
	addressToNameMap map[string][]*common.Nft
}

var instance *IndexerMgr

func NewIndexerMgr(
	dbDir string,
	chaincfgParam *chaincfg.Params,
	maxIndexHeight int,
	periodFlushToDB int,
) *IndexerMgr {

	if instance != nil {
		return instance
	}

	if periodFlushToDB == 0 {
		periodFlushToDB = 12
	}

	mgr := &IndexerMgr{
		dbDir:           dbDir,
		chaincfgParam:   chaincfgParam,
		maxIndexHeight:  maxIndexHeight,
		periodFlushToDB: periodFlushToDB,
	}

	instance = mgr
	switch instance.chaincfgParam.Name {
	case "mainnet":
		instance.ordFirstHeight = 767430
		instance.ordxFirstHeight = 827307
	case "testnet3":
		instance.ordFirstHeight = 2413343
		instance.ordxFirstHeight = 2570589
	default: // testnet4
		instance.ordFirstHeight = 0
		instance.ordxFirstHeight = 0
	}

	return instance
}

func (b *IndexerMgr) Init() {
	err := b.initDB()
	if err != nil {
		common.Log.Panicf("initDB failed. %v", err)
	}
	b.compiling = base_indexer.NewBaseIndexer(b.baseDB, b.chaincfgParam, b.maxIndexHeight, b.periodFlushToDB)
	b.compiling.Init(b.processOrdProtocol, b.forceUpdateDB)
	b.lastCheckHeight = b.compiling.GetSyncHeight()
	b.initCollections()

	dbver := b.GetBaseDBVer()
	common.Log.Infof("base db version: %s", dbver)
	if dbver != "" && dbver != common.BASE_DB_VERSION {
		common.Log.Panicf("DB version inconsistent. DB ver %s, but code base %s", dbver, common.BASE_DB_VERSION)
	}

	b.rpcService = base_indexer.NewRpcIndexer(b.compiling)

	if !instance.IsMainnet() {
		exotic.IsTestNet = true
		exotic.SatributeList = append(exotic.SatributeList, exotic.Customized)
	}

	b.exotic = exotic.NewExoticIndexer(b.compiling)
	b.exotic.Init()
	b.nft = nft.NewNftIndexer(b.nftDB)
	b.nft.Init(b.compiling)
	b.ftIndexer = ft.NewOrdxIndexer(b.ftDB)
	b.ftIndexer.InitOrdxIndexer(b.nft)
	b.ns = ns.NewNameService(b.nsDB)
	b.ns.Init(b.nft)
	b.brc20Indexer = brc20.NewIndexer(b.brc20DB)
	b.brc20Indexer.InitIndexer(b.nft)
	b.RunesIndexer = runes.NewIndexer(b.runesDB, b.chaincfgParam, b.compiling, b.rpcService)
	b.RunesIndexer.Init()

	b.compilingBackupDB = nil
	b.exoticBackupDB = nil
	b.ftBackupDB = nil
	b.brc20BackupDB = nil
	b.runesBackupDB = nil
	b.nsBackupDB = nil
	b.nftBackupDB = nil

	b.addressToNftMap = nil
	b.addressToNameMap = nil
}

func (b *IndexerMgr) GetBaseDB() *badger.DB {
	return b.baseDB
}

func (b *IndexerMgr) StartDaemon(stopChan chan bool) {
	n := 10
	ticker := time.NewTicker(time.Duration(n) * time.Second)

	stopIndexerChan := make(chan struct{}, 1) // 非阻塞

	if b.repair() {
		common.Log.Infof("repaired, check again.")
		return
	}

	bWantExit := false
	isRunning := false
	disableSync := false
	tick := func() {
		if disableSync {
			return
		}
		if !isRunning {
			isRunning = true
			go func() {
				ret := b.compiling.SyncToChainTip(stopIndexerChan)
				if ret == 0 {
					if b.maxIndexHeight > 0 {
						if b.maxIndexHeight <= b.compiling.GetHeight() {
							b.checkSelf()
							common.Log.Infof("reach expected height, set exit flag")
							bWantExit = true
						}
					} else {
						b.updateDB()
						b.dbgc()
						// 每周定期检查数据 （目前主网一次检查需要半个小时-1个小时，需要考虑这个影响）
						// if b.lastCheckHeight != b.compiling.GetSyncHeight() {
						// 	period := 1000
						// 	if b.compiling.GetSyncHeight()%period == 0 {
						// 		b.lastCheckHeight = b.compiling.GetSyncHeight()
						// 		b.checkSelf()
						// 	}
						// }
						if b.dbStatistic() {
							bWantExit = true
						}
					}
				} else if ret > 0 {
					// handle reorg
					b.handleReorg(ret)
				} else {
					if ret == -1 {
						common.Log.Infof("IndexerMgr inner thread exit by SIGINT signal")
						bWantExit = true
					}
				}

				isRunning = false
			}()
		}
	}

	tick()
	for !bWantExit {
		select {
		case <-ticker.C:
			if bWantExit {
				break
			}
			tick()
		case <-stopChan:
			common.Log.Info("IndexerMgr got SIGINT")
			if bWantExit {
				break
			}
			if isRunning {
				select {
				case stopIndexerChan <- struct{}{}:
					// 成功发送
				default:
					// 通道已满或没有接收者，执行其他操作
				}
				for isRunning {
					time.Sleep(time.Second / 10)
				}
				common.Log.Info("IndexerMgr inner thread exited")
			}
			bWantExit = true
		}
	}

	ticker.Stop()

	// close all
	b.closeDB()

	common.Log.Info("IndexerMgr exited.")
}

func (b *IndexerMgr) dbgc() {
	common.RunBadgerGC(b.localDB)
	common.RunBadgerGC(b.baseDB)
	common.RunBadgerGC(b.nftDB)
	common.RunBadgerGC(b.nsDB)
	common.RunBadgerGC(b.ftDB)
	common.RunBadgerGC(b.brc20DB)
	common.RunBadgerGC(b.runesDB)
}

func (b *IndexerMgr) closeDB() {
	b.dbgc()

	b.runesDB.Close()
	b.brc20DB.Close()
	b.ftDB.Close()
	b.nsDB.Close()
	b.nftDB.Close()
	b.baseDB.Close()
	b.localDB.Close()
}

func (b *IndexerMgr) checkSelf() {
	start := time.Now()
	if b.compiling.CheckSelf() &&
	b.exotic.CheckSelf() &&
	b.nft.CheckSelf(b.baseDB) &&
	b.ftIndexer.CheckSelf(b.compiling.GetSyncHeight()) &&
	b.brc20Indexer.CheckSelf(b.compiling.GetSyncHeight()) &&
	b.RunesIndexer.CheckSelf() &&
	b.ns.CheckSelf(b.baseDB) {
		common.Log.Infof("IndexerMgr.checkSelf takes %v", time.Since(start))
	}
	
}

func (b *IndexerMgr) forceUpdateDB() {
	startTime := time.Now()
	b.exotic.UpdateDB()
	b.nft.UpdateDB()
	b.ns.UpdateDB()
	b.ftIndexer.UpdateDB()
	b.brc20Indexer.UpdateDB()
	b.RunesIndexer.UpdateDB()

	common.Log.Infof("IndexerMgr.forceUpdateDB: takes: %v", time.Since(startTime))
}

func (b *IndexerMgr) handleReorg(height int) {
	b.closeDB()
	b.Init()
	b.compiling.SetReorgHeight(height)
	common.Log.Infof("IndexerMgr handleReorg completed.")
}

// 为了回滚数据，我们采用这样的策略：
// 假设当前最新高度是h，那么数据库记录，最多只到（h-6），这样确保即使回滚，只需要从数据库回滚即可
// 为了保证数据库记录最高到（h-6），我们做一次数据备份，到合适实际再写入数据库
func (b *IndexerMgr) updateDB() {
	b.updateServiceInstance()

	complingHeight := b.compiling.GetHeight()
	syncHeight := b.compiling.GetSyncHeight()
	blocksInHistory := b.compiling.GetBlockHistory()

	if complingHeight-syncHeight < blocksInHistory {
		common.Log.Infof("updateDB do nothing at height %d-%d", complingHeight, syncHeight)
		return
	}

	if complingHeight-syncHeight == blocksInHistory {
		// 先备份数据在缓存
		if b.compilingBackupDB == nil {
			b.prepareDBBuffer()
			common.Log.Infof("updateDB clone data at height %d-%d", complingHeight, syncHeight)
		}
		return
	}

	// 这个区间不备份数据
	if complingHeight-syncHeight < 2*blocksInHistory {
		common.Log.Infof("updateDB do nothing at height %d-%d", complingHeight, syncHeight)
		return
	}

	// b.GetHeight()-b.GetSyncHeight() == 2*b.GetBlockHistory()

	// 到达高度时，将备份的数据写入数据库中。
	if b.compilingBackupDB != nil {
		if complingHeight-b.compilingBackupDB.GetHeight() < blocksInHistory {
			common.Log.Infof("updateDB do nothing at height %d, backup instance %d",
				complingHeight, b.compilingBackupDB.GetHeight())
			return
		}
		common.Log.Infof("updateDB do backup->forceUpdateDB() at height %d-%d", complingHeight, syncHeight)
		b.performUpdateDBInBuffer()
	}
	b.prepareDBBuffer()
	common.Log.Infof("updateDB clone data at height %d-%d", complingHeight, syncHeight)
}

func (b *IndexerMgr) performUpdateDBInBuffer() {
	b.cleanDBBuffer() // must before UpdateDB
	b.compilingBackupDB.UpdateDB()
	b.exoticBackupDB.UpdateDB()
	b.nftBackupDB.UpdateDB()
	b.nsBackupDB.UpdateDB()
	b.ftBackupDB.UpdateDB()
	b.runesBackupDB.UpdateDB()
	b.brc20BackupDB.UpdateDB()
}

func (b *IndexerMgr) prepareDBBuffer() {
	b.compilingBackupDB = b.compiling.Clone()
	b.compiling.ResetBlockVector()

	b.exoticBackupDB = b.exotic.Clone()
	b.ftBackupDB = b.ftIndexer.Clone()
	b.nsBackupDB = b.ns.Clone()
	b.nftBackupDB = b.nft.Clone()
	b.brc20BackupDB = b.brc20Indexer.Clone()
	b.runesBackupDB = b.RunesIndexer.Clone()
	common.Log.Infof("backup instance %d cloned", b.compilingBackupDB.GetHeight())
}

func (b *IndexerMgr) cleanDBBuffer() {
	b.compiling.Subtract(b.compilingBackupDB)
	b.exotic.Subtract(b.exoticBackupDB)
	b.nft.Subtract(b.nftBackupDB)
	b.ns.Subtract(b.nsBackupDB)
	b.ftIndexer.Subtract(b.ftBackupDB)
	b.brc20Indexer.Subtract(b.brc20BackupDB)
	b.RunesIndexer.Subtract(b.runesBackupDB)
}

func (b *IndexerMgr) updateServiceInstance() {
	if b.rpcService.GetHeight() == b.compiling.GetHeight() {
		return
	}

	newService := base_indexer.NewRpcIndexer(b.compiling)
	common.Log.Infof("service instance %d cloned", newService.GetHeight())

	newService.UpdateServiceInstance()
	b.mutex.Lock()
	b.rpcService = newService
	b.addressToNftMap = nil
	b.addressToNameMap = nil
	b.mutex.Unlock()
}

func (p *IndexerMgr) repair() bool {
	//p.compiling.Repair()
	return false
}

func (p *IndexerMgr) dbStatistic() bool {
	// save to latest DB first, save time.
	// if p.compilingBackupDB == nil {
	// 	p.prepareDBBuffer()
	// }
	// p.performUpdateDBInBuffer()

	//common.Log.Infof("start searching...")
	//return p.SearchPredefinedName()
	//return p.searchName()
	return false
}
