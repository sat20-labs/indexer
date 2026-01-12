package indexer

import (
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/config"
	base_indexer "github.com/sat20-labs/indexer/indexer/base"
	"github.com/sat20-labs/indexer/indexer/brc20"
	inCommon "github.com/sat20-labs/indexer/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
	"github.com/sat20-labs/indexer/indexer/exotic"
	"github.com/sat20-labs/indexer/indexer/ft"
	"github.com/sat20-labs/indexer/indexer/nft"
	"github.com/sat20-labs/indexer/indexer/ns"
	"github.com/sat20-labs/indexer/indexer/runes"

	"github.com/btcsuite/btcd/chaincfg"
)

type IndexerMgr struct {
	cfg   *config.YamlConf
	dbDir string
	// data from blockchain
	baseDB   common.KVDB
	exoticDB common.KVDB
	ftDB     common.KVDB
	nsDB     common.KVDB
	nftDB    common.KVDB
	brc20DB  common.KVDB
	runesDB  common.KVDB
	// data from market
	localDB common.KVDB
	kvDB    common.KVDB

	// 保护这两个数据
	reloading     int32
	rpcProcessing int32

	// 配置参数
	chaincfgParam   *chaincfg.Params
	ordxFirstHeight int
	ordFirstHeight  int
	maxIndexHeight  int
	periodFlushToDB int

	//mpn         *mpn.MemPoolNode
	miniMempool *MiniMemPool

	brc20Indexer *brc20.BRC20Indexer
	RunesIndexer *runes.Indexer
	exotic       *exotic.ExoticIndexer
	ftIndexer    *ft.FTIndexer
	ns           *ns.NameService
	nft          *nft.NftIndexer

	// 跑数据
	lastCheckHeight int
	base            *base_indexer.BaseIndexer
	// 备份所有需要写入数据库的数据
	baseBackupDB   *base_indexer.BaseIndexer
	exoticBackupDB *exotic.ExoticIndexer
	brc20BackupDB  *brc20.BRC20Indexer
	runesBackupDB  *runes.Indexer
	ftBackupDB     *ft.FTIndexer
	nsBackupDB     *ns.NameService
	nftBackupDB    *nft.NftIndexer

	/////////////////////////////////
	mutex sync.RWMutex                           // 保护下面的数据
	clmap map[common.TickerName]map[string]int64 // collections map, ticker -> inscriptionId -> asset amount
	//registerPubKey map[string]int64  // pubkey -> refresh time (注册时间， 挖矿地址刷新时间)
	// 接收前端api访问的实例，隔离内存访问
	rpcService *base_indexer.RpcIndexer
	// 本地缓存，在区块更新时清空
	addressToNftMap  map[string][]*common.Nft
	addressToNameMap map[string][]*common.Nft
	/////////////////////////////////
}

var instance *IndexerMgr

func NewIndexerMgr(
	yamlcfg *config.YamlConf,
) *IndexerMgr {

	if instance != nil {
		return instance
	}

	if yamlcfg.BasicIndex.PeriodFlushToDB == 0 {
		yamlcfg.BasicIndex.PeriodFlushToDB = 12
	}

	chainParam := &chaincfg.MainNetParams
	switch yamlcfg.Chain {
	case common.ChainTestnet:
		common.CHAIN = "testnet"
		chainParam = &chaincfg.TestNet4Params
	case common.ChainTestnet4:
		common.CHAIN = "testnet"
		chainParam = &chaincfg.TestNet4Params
	case common.ChainMainnet:
		chainParam = &chaincfg.MainNetParams
	default:
		chainParam = &chaincfg.MainNetParams
	}
	dbDir := yamlcfg.DB.Path
	if !filepath.IsAbs(dbDir) {
		dbDir = filepath.Clean(dbDir) + string(filepath.Separator)
	}

	mgr := &IndexerMgr{
		cfg:             yamlcfg,
		dbDir:           dbDir,
		chaincfgParam:   chainParam,
		maxIndexHeight:  int(yamlcfg.BasicIndex.MaxIndexHeight),
		periodFlushToDB: yamlcfg.BasicIndex.PeriodFlushToDB,
		miniMempool:     NewMiniMemPool(),
	}

	instance = mgr
	switch instance.chaincfgParam.Name {
	case "mainnet":
		instance.ordFirstHeight = 767430
		instance.ordxFirstHeight = 827307
	case "testnet3":
		instance.ordFirstHeight = 2413343
		instance.ordxFirstHeight = 2570589
		common.Jubilee_Height = 0
	default: // testnet4
		instance.ordFirstHeight = 0
		instance.ordxFirstHeight = 0
		common.Jubilee_Height = 0
	}

	return instance
}

func (b *IndexerMgr) Init() {
	err := b.initDB()
	if err != nil {
		common.Log.Panicf("initDB failed. %v", err)
	}
	b.base = base_indexer.NewBaseIndexer(b.baseDB, b.chaincfgParam, b.maxIndexHeight, b.periodFlushToDB)
	b.base.Init()
	b.base.SetUpdateDBCallback(b.forceUpdateDB)
	b.base.SetBlockCallback(b.processOrdProtocol)
	b.lastCheckHeight = b.base.GetSyncHeight()
	b.initCollections()

	dbver := b.GetBaseDBVer()
	common.Log.Infof("base db version: %s", dbver)
	if dbver != "" && dbver != common.BASE_DB_VERSION {
		common.Log.Panicf("DB version inconsistent. DB ver %s, but code base %s", dbver, common.BASE_DB_VERSION)
	}

	b.rpcService = base_indexer.NewRpcIndexer(b.base)

	if !instance.IsMainnet() {
		exotic.IsTestNet = true
		exotic.SatributeList = append(exotic.SatributeList, exotic.Customized)
	}

	b.exotic = exotic.NewExoticIndexer(b.exoticDB)
	b.exotic.Init(b.base)
	b.nft = nft.NewNftIndexer(b.nftDB)
	b.nft.Init(b.base, b)
	b.ftIndexer = ft.NewOrdxIndexer(b.ftDB)
	b.ftIndexer.Init(b.nft)
	b.ns = ns.NewNameService(b.nsDB)
	b.ns.Init(b.nft)
	b.brc20Indexer = brc20.NewIndexer(b.brc20DB, b.cfg.CheckValidateFiles)
	b.brc20Indexer.Init(b.nft)
	b.RunesIndexer = runes.NewIndexer(b.runesDB, b.chaincfgParam, b.cfg.CheckValidateFiles)
	b.RunesIndexer.Init(b.base)
	b.miniMempool.init()

	b.baseBackupDB = nil
	b.exoticBackupDB = nil
	b.ftBackupDB = nil
	b.brc20BackupDB = nil
	b.runesBackupDB = nil
	b.nsBackupDB = nil
	b.nftBackupDB = nil

	b.addressToNftMap = nil
	b.addressToNameMap = nil
}

func (b *IndexerMgr) GetBaseDB() common.KVDB {
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

	// mpnode, err := mpn.StartMPN(b.cfg, b.localDB, b, stopIndexerChan)
	// if err != nil {
	// 	common.Log.Errorf("StartMPN failed, %v", err)
	// 	return
	// }

	bWantExit := false
	isRunning := false
	disableSync := false // 启动rpc，不再同步数据
	tick := func() {
		if disableSync {
			return
		}
		if !isRunning {
			isRunning = true
			go func() {
				for !bWantExit {
					ret := b.base.SyncToChainTip(stopIndexerChan)
					if ret == 0 {
						if !bWantExit && b.base.GetHeight() == b.base.GetChainTip() {
							// IndexerMgr.updateDB 被调用后，已经进入实际运行状态，
							// 这个时候，BaseIndexer.SyncToChainTip 不能再进行数据库的内部更新，会破坏内存中的数据
							b.base.SetUpdateDBCallback(nil)
							b.updateDB()
							if b.maxIndexHeight <= 0 {
								b.miniMempool.Start(&b.cfg.ShareRPC.Bitcoin)
							}
						}

						if b.maxIndexHeight > 0 {
							if b.maxIndexHeight <= b.base.GetHeight() {
								b.updateDB()
								b.checkSelf()
								common.Log.Infof("reach expected height, set exit flag")
								bWantExit = true
							}
						}

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

					} else if ret > 0 {
						// handle reorg
						b.handleReorg(ret)
						b.base.SyncToChainTip(stopIndexerChan)
					} else {
						if ret == -1 {
							common.Log.Infof("IndexerMgr inner thread exit by SIGINT signal")
							bWantExit = true
						}
					}

					if !inCommon.STEP_RUN_MODE {
						break
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
			bWantExit = true
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
		}
	}

	ticker.Stop()

	// close all
	b.closeDB()

	// mpn.StopMPN(mpnode)

	common.Log.Info("IndexerMgr exited.")
}

func (b *IndexerMgr) dbgc() {
	db.RunDBGC(b.kvDB)
	db.RunDBGC(b.localDB)
	db.RunDBGC(b.baseDB)
	db.RunDBGC(b.nftDB)
	db.RunDBGC(b.nsDB)
	db.RunDBGC(b.exoticDB)
	db.RunDBGC(b.ftDB)
	db.RunDBGC(b.brc20DB)
	db.RunDBGC(b.runesDB)
	common.Log.Infof("dbgc completed")
}

func (b *IndexerMgr) closeDB() {
	common.Log.Infof("IndexerMgr->closeDB ")
	b.dbgc()

	if b.runesDB != nil {
		b.runesDB.Close()
		b.runesDB = nil
	}
	if b.brc20DB != nil {
		b.brc20DB.Close()
		b.brc20DB = nil
	}
	if b.ftDB != nil {
		b.ftDB.Close()
		b.ftDB = nil
	}
	if b.exoticDB != nil {
		b.exoticDB.Close()
		b.exoticDB = nil
	}
	if b.nsDB != nil {
		b.nsDB.Close()
		b.nsDB = nil
	}
	if b.nftDB != nil {
		b.nftDB.Close()
		b.nftDB = nil
	}
	if b.baseDB != nil {
		b.baseDB.Close()
		b.baseDB = nil
	}
	if b.localDB != nil {
		b.localDB.Close()
		b.localDB = nil
	}
	if b.kvDB != nil {
		b.kvDB.Close()
		b.kvDB = nil
	}
}

func (b *IndexerMgr) checkSelf() {
	start := time.Now()
	// 关闭所有无关实例
	// 检查一个关闭一个，节省空间
	ok := false
	for {
		ok = b.brc20Indexer.CheckSelf()
		if !ok {
			break
		}
		b.brc20DB.Close()
		b.brc20DB = nil

		ok = b.RunesIndexer.CheckSelf()
		if !ok {
			break
		}
		b.runesDB.Close()
		b.runesDB = nil

		ok = b.ftIndexer.CheckSelf()
		if !ok {
			break
		}
		b.ftDB.Close()
		b.ftDB = nil

		ok = b.ns.CheckSelf()
		if !ok {
			break
		}
		b.nsDB.Close()
		b.nsDB = nil
		
		ok = b.nft.CheckSelf()
		if !ok {
			break
		}
		b.nftDB.Close()
		b.nftDB = nil

		ok = b.exotic.CheckSelf()
		if !ok {
			break
		}
		b.exoticDB.Close()
		b.exoticDB = nil

		ok = b.base.CheckSelf()
		if !ok {
			break
		}
		b.baseDB.Close()
		b.baseDB = nil
		
		break
	}

	if ok {
		common.Log.Infof("IndexerMgr.checkSelf succeed. %v", time.Since(start))
	} else {
		b.closeDB()
		common.Log.Panic("IndexerMgr.checkSelf failed.")
	}
}

func (b *IndexerMgr) forceUpdateDB(wantToDelete map[string]uint64) {
	startTime := time.Now()
	b.exotic.UpdateDB()
	b.nft.UpdateDB()
	b.ns.UpdateDB()
	b.ftIndexer.UpdateDB()
	b.RunesIndexer.UpdateDB()
	b.brc20Indexer.CheckEmptyAddress(wantToDelete)
	b.brc20Indexer.UpdateDB()

	common.Log.Infof("IndexerMgr.forceUpdateDB: takes: %v", time.Since(startTime))
}

func (b *IndexerMgr) handleReorg(height int) {
	common.Log.Infof("IndexerMgr handleReorg enter...")
	// 需要等rpc都完成，再重新启动
	atomic.AddInt32(&b.reloading, 1)
	for atomic.LoadInt32(&b.rpcProcessing) > 0 {
		time.Sleep(10 * time.Millisecond)
	}
	atomic.AddInt32(&b.reloading, 1)
	defer func() {
		atomic.AddInt32(&b.reloading, -2)
	}()

	// 要确保下面的调用，没有rpc的调用
	b.miniMempool.Stop()
	b.closeDB()
	b.Init() // 数据库重新打开
	b.miniMempool.ProcessReorg()
	b.base.SetReorgHeight(height)

	common.Log.Infof("IndexerMgr handleReorg completed.")
}

func (b *IndexerMgr) rpcEnter() {
	for atomic.LoadInt32(&b.reloading) > 0 {
		time.Sleep(10 * time.Microsecond)
	}
	atomic.AddInt32(&b.rpcProcessing, 1)
	if atomic.LoadInt32(&b.reloading) > 0 {
		atomic.AddInt32(&b.rpcProcessing, -1)
		for atomic.LoadInt32(&b.reloading) > 0 {
			time.Sleep(10 * time.Microsecond)
		}
		atomic.AddInt32(&b.rpcProcessing, 1)
	}
}

func (b *IndexerMgr) rpcLeft() {
	atomic.AddInt32(&b.rpcProcessing, -1)
}

// 为了回滚数据，我们采用这样的策略：
// 假设当前最新高度是h，那么数据库记录，最多只到（h-6），这样确保即使回滚，只需要从数据库回滚即可
// 为了保证数据库记录最高到（h-6），我们做一次数据备份，到合适实际再写入数据库
func (b *IndexerMgr) updateDB() {
	b.updateServiceInstance()

	complingHeight := b.base.GetHeight()
	syncHeight := b.base.GetSyncHeight()
	blocksInHistory := b.base.GetBlockHistory()

	gap := complingHeight - syncHeight
	if gap < blocksInHistory {
		common.Log.Infof("performUpdateDBInBuffer nothing to do at height %d-%d", complingHeight, syncHeight)
	} else {
		if b.baseBackupDB == nil {
			b.prepareDBBuffer()
		}
		// 这个区间不备份数据
		if gap < 2*blocksInHistory {
			common.Log.Infof("performUpdateDBInBuffer nothing to do at height %d-%d", complingHeight, syncHeight)
			return
		}

		// 到达高度时，将备份的数据写入数据库中。
		common.Log.Infof("performUpdateDBInBuffer performUpdateDBInBuffer at height %d-%d", complingHeight, syncHeight)
		b.performUpdateDBInBuffer()

		// 备份当前高度的数据
		b.prepareDBBuffer()
	}
}

func (b *IndexerMgr) performUpdateDBInBuffer() {
	b.cleanDBBuffer() // must before UpdateDB
	wantToDelete := b.baseBackupDB.UpdateDB()
	org := make(map[string]uint64)
	for k, v := range wantToDelete {
		org[k] = v
	}
	b.exoticBackupDB.UpdateDB()
	b.nftBackupDB.UpdateDB()
	b.nsBackupDB.UpdateDB()
	b.ftBackupDB.UpdateDB()
	b.runesBackupDB.UpdateDB()
	b.brc20BackupDB.CheckEmptyAddress(wantToDelete)
	b.brc20BackupDB.UpdateDB()
	b.baseBackupDB.CleanEmptyAddress(org, wantToDelete)

	b.base.SetSyncStats(b.baseBackupDB.GetSyncStats())
}

func (b *IndexerMgr) prepareDBBuffer() {
	b.baseBackupDB = b.base.Clone(true)
	b.exoticBackupDB = b.exotic.Clone(b.baseBackupDB)
	b.runesBackupDB = b.RunesIndexer.Clone(b.baseBackupDB)
	b.nftBackupDB = b.nft.Clone(b.baseBackupDB)
	b.nsBackupDB = b.ns.Clone(b.nftBackupDB)
	b.ftBackupDB = b.ftIndexer.Clone(b.nftBackupDB)
	b.brc20BackupDB = b.brc20Indexer.Clone(b.nftBackupDB)
	common.Log.Infof("prepareDBBuffer backup instance with %d", b.baseBackupDB.GetHeight())
}

func (b *IndexerMgr) cleanDBBuffer() {
	b.base.Subtract(b.baseBackupDB)
	b.exotic.Subtract(b.exoticBackupDB)
	b.nft.Subtract(b.nftBackupDB)
	b.ns.Subtract(b.nsBackupDB)
	b.ftIndexer.Subtract(b.ftBackupDB)
	b.brc20Indexer.Subtract(b.brc20BackupDB)
	b.RunesIndexer.Subtract(b.runesBackupDB)

	common.Log.Infof("cleanDBBuffer backup instance with %d", b.baseBackupDB.GetHeight())
}

func (b *IndexerMgr) updateServiceInstance() {
	if b.rpcService.GetHeight() == b.base.GetHeight() {
		return
	}

	newService := base_indexer.NewRpcIndexer(b.base)
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

	//p.nft.Repair()
	return p.brc20Indexer.Repair()

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

func (b *IndexerMgr) GetBrc20Indexer() *brc20.BRC20Indexer {
	return b.brc20Indexer
}

func (p *IndexerMgr) GetRpcService() *base_indexer.RpcIndexer {
	return p.rpcService
}
