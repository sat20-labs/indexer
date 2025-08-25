package base

import (
	"bytes"
	"sort"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

type UtxoValue struct {
	Utxo    string
	Address *common.ScriptPubKey
	UtxoId  uint64
	Value   int64
}

type AddressStatus struct {
	AddressId uint64
	Op        int // 0 existed; 1 added
}

type BlockProcCallback func(*common.Block)
type UpdateDBCallback func()

type BaseIndexer struct {
	db    common.KVDB
	stats *SyncStats // 数据库状态
	//reCheck bool

	// 需要clone的数据
	blockVector []*common.BlockValueInDB //
	utxoIndex   *common.UTXOIndex
	delUTXOs    []*UtxoValue // utxo->address,utxoid

	addressIdMap map[string]*AddressStatus

	lastHeight       int // 内存数据同步区块
	lastHash         string
	prevBlockHashMap map[int]string // 记录过去6个区块hash，判断哪个区块分叉
	lastSats         int64
	////////////

	blocksChan chan *common.Block

	// 配置参数
	periodFlushToDB  int
	keepBlockHistory int
	chaincfgParam    *chaincfg.Params
	maxIndexHeight   int

	blockprocCB BlockProcCallback
	updateDBCB  UpdateDBCallback
}

const BLOCK_PREFETCH = 12

func NewBaseIndexer(
	basicDB common.KVDB,
	chaincfgParam *chaincfg.Params,
	maxIndexHeight int,
	periodFlushToDB int,
) *BaseIndexer {
	indexer := &BaseIndexer{
		db:               basicDB,
		stats:            &SyncStats{},
		periodFlushToDB:  periodFlushToDB,
		keepBlockHistory: 12,
		blocksChan:       make(chan *common.Block, BLOCK_PREFETCH),
		chaincfgParam:    chaincfgParam,
		maxIndexHeight:   maxIndexHeight,
	}

	if chaincfgParam.Name != "mainnet" {
		indexer.keepBlockHistory = 72 // testnet4的分岔很多也很长
	}

	indexer.addressIdMap = make(map[string]*AddressStatus, 0)
	indexer.prevBlockHashMap = make(map[int]string)

	return indexer
}

func (b *BaseIndexer) Init() {
	dbver := b.GetBaseDBVer()
	common.Log.Infof("base db version: %s", b.GetBaseDBVer())
	if dbver != "" && dbver != common.BASE_DB_VERSION {
		common.Log.Panicf("DB version inconsistent. DB ver %s, but code base %s", dbver, common.BASE_DB_VERSION)
	}

	b.reset()
}

func (b *BaseIndexer) SetUpdateDBCallback(cb2 UpdateDBCallback) {
	b.updateDBCB = cb2
}

func (b *BaseIndexer) SetBlockCallback(cb1 BlockProcCallback) {
	b.blockprocCB = cb1
}

func (b *BaseIndexer) reset() {
	b.loadSyncStatsFromDB()

	b.blocksChan = make(chan *common.Block, BLOCK_PREFETCH)

	b.blockVector = make([]*common.BlockValueInDB, 0)
	b.utxoIndex = common.NewUTXOIndex()
	b.delUTXOs = make([]*UtxoValue, 0)
}

// 只保存UpdateDB需要用的数据
func (b *BaseIndexer) Clone() *BaseIndexer {
	startTime := time.Now()
	newInst := NewBaseIndexer(b.db, b.chaincfgParam, b.maxIndexHeight, b.periodFlushToDB)

	newInst.utxoIndex = common.NewUTXOIndex()
	for key, value := range b.utxoIndex.Index {
		newInst.utxoIndex.Index[key] = value
	}
	newInst.delUTXOs = make([]*UtxoValue, len(b.delUTXOs))
	copy(newInst.delUTXOs, b.delUTXOs)

	newInst.addressIdMap = make(map[string]*AddressStatus)
	for k, v := range b.addressIdMap {
		newInst.addressIdMap[k] = v
	}
	newInst.blockVector = make([]*common.BlockValueInDB, len(b.blockVector))
	copy(newInst.blockVector, b.blockVector)

	newInst.lastHash = b.lastHash
	newInst.lastHeight = b.lastHeight
	newInst.lastSats = b.lastSats
	newInst.stats = b.stats.Clone()
	newInst.blockprocCB = b.blockprocCB
	newInst.updateDBCB = b.updateDBCB

	common.Log.Infof("BaseIndexer->clone takes %v", time.Since(startTime))

	return newInst
}

// 在 UpdateDB 用到的数据，这里需要先剪去，这些剪去的数据，当作已经备份到数据库
func (b *BaseIndexer) Subtract(another *BaseIndexer) {
	for key := range another.utxoIndex.Index {
		delete(b.utxoIndex.Index, key)
	}

	// 会导致 UpdateServiceInstance 找不到addressId
	// for k := range another.addressIdMap {
	// 	delete(b.addressIdMap, k)
	// }

	l := len(another.delUTXOs)
	//b.delUTXOs = b.delUTXOs[l:] 不会释放前面的内存
	b.delUTXOs = append([]*UtxoValue(nil), b.delUTXOs[l:]...) // 释放前面删除的切片

	l = len(another.blockVector)
	// b.blockVector = b.blockVector[l:]
	b.blockVector = append([]*common.BlockValueInDB(nil), b.blockVector[l:]...)
}

func needMerge(rngs []*common.Range) bool {
	len1 := len(rngs)
	if len1 < 2 {
		return false
	}

	r1 := rngs[0]
	for i := 1; i < len1; i++ {
		r2 := rngs[i]
		if r1.Start+r1.Size == r2.Start {
			return true
		}
		r1 = r2
	}

	return false
}

func (b *BaseIndexer) Repair() {

}

// only call in compiling data
func (b *BaseIndexer) forceUpdateDB() {
	/* TODO 优化 NftIndexer->UpdateDB

	2025-08-21 10:21:22 [info] default: BaseIndexer->updateBasicDB 883100 start...
	2025-08-21 10:21:30 [info] default: BaseIndexer.prefechAddress add 379442, del 564247, address 337455 in 7.975174159s
	2025-08-21 10:21:32 [info] default: BaseIndexer.updateBasicDB-> add utxos 376540 (+ 2902), cost: 1.612725272s
	2025-08-21 10:21:32 [info] default: BaseIndexer.updateBasicDB-> delete utxos 564247, cost: 295.996357ms
	2025-08-21 10:21:35 [info] default: BaseIndexer.updateBasicDB-> flush db,  cost: 2.141388775s
	2025-08-21 10:21:35 [info] default: BaseIndexer.updateBasicDB: cost: 12.027970259s
	2025-08-21 10:21:35 [info] default: InitRarityDB 883100 takes 14.283141ms
	2025-08-21 10:21:35 [info] default: ExoticIndexer->UpdateDB takes 14.65488ms
	2025-08-21 10:22:03 [info] default: NftIndexer->UpdateDB takes 28.526164031s
	2025-08-21 10:22:03 [info] default: NameService->UpdateDB takes 9.490898ms
	2025-08-21 10:22:03 [info] default: OrdxIndexer->UpdateDB takse: 16.034782ms
	2025-08-21 10:22:03 [info] default: BRC20Indexer->UpdateDB takse: 1.246µs
	2025-08-21 10:22:03 [info] default: DbWrite.FlushToDB-> logs count:85453, update count:0, remove count:0, total bytes:4046793
	2025-08-21 10:22:03 [info] default: RuneIndexer.UpdateDB-> db commit success, height:883100
	2025-08-21 10:22:03 [info] default: IndexerMgr.forceUpdateDB: takes: 28.83885863s
	2025-08-21 10:22:03 [info] default: forceUpdateDB sync to height 883100
	2025-08-21 10:22:03 [info] default: processed block 883100 (2025-02-10 08:14:34) with 1909 transactions took 44.166077839s (23.135713ms per tx)

	*/
	if b.updateDBCB != nil {
		startTime := time.Now()
		b.UpdateDB()
		common.Log.Infof("BaseIndexer.updateBasicDB: cost: %v", time.Since(startTime))

		// startTime = time.Now()
		b.updateDBCB()
		// common.Log.Infof("BaseIndexer.updateOrdxDB: cost: %v", time.Since(startTime))

		common.Log.Infof("forceUpdateDB sync to height %d", b.stats.SyncHeight)
	} //else {
	// 	common.Log.Infof("don't run forceUpdateDB after entering service mode")
	// }
}


func (b *BaseIndexer) prefechAddressV2() map[string]*common.AddressValueInDB {
	// 测试下提前取的所有地址
	addressValueMap := make(map[string]*common.AddressValueInDB)

	
	startTime := time.Now()
	
	for _, v := range b.utxoIndex.Index {
		if v.Address.Type == int(txscript.NullDataTy) {
			// 只有OP_RETURN 才不记录
			if v.Value == 0 {
				continue
			}
		}
		b.addUtxo(&addressValueMap, v)
	}

	type deleteUtxo struct {
		key   []byte
		value *UtxoValue
	}

	
	for _, utxo := range b.delUTXOs {
		utxoId := utxo.UtxoId
		for _, address := range utxo.Address.Addresses {
			value, ok := addressValueMap[address]
			if ok {
				value.Utxos[utxoId] = &common.UtxoValue{Op: -1}
			} else {
				utxos := make(map[uint64]*common.UtxoValue)
				utxos[utxoId] = &common.UtxoValue{Op: -1}

				id, op := b.getAddressId(address)
				if op >= 0 {
					value = &common.AddressValueInDB{
						AddressType: uint32(utxo.Address.Type),
						AddressId:   id,
						Op:          op,
						Utxos:       utxos,
					}
					addressValueMap[address] = value
				} else {
					common.Log.Panicf("utxo %x exists but address %s not exists.", utxoId, address)
				}
			}
		}
	}

	common.Log.Infof("BaseIndexer.prefechAddress add %d, del %d, address %d in %v",
		len(b.utxoIndex.Index), len(b.delUTXOs), len(addressValueMap), time.Since(startTime))

	return addressValueMap
}


func (b *BaseIndexer) prefechAddress() map[string]*common.AddressValueInDB {
	// 测试下提前取的所有地址
	addressValueMap := make(map[string]*common.AddressValueInDB)

	// pebble数据库的优化手段: 尽可能将随机读变成按照key的顺序读
	startTime := time.Now()
	b.db.View(func(txn common.ReadBatch) error {
		for _, v := range b.utxoIndex.Index {
			if v.Address.Type == int(txscript.NullDataTy) {
				// 只有OP_RETURN 才不记录
				if v.Value == 0 {
					continue
				}
			}
			b.addUtxo(&addressValueMap, v)
		}

		type deleteUtxo struct {
			key   []byte
			value *UtxoValue
		}

		deleteUtxos := make([]*deleteUtxo, len(b.delUTXOs))
		for i, value := range b.delUTXOs {
			deleteUtxos[i] = &deleteUtxo{
				key:   db.GetUtxoIdKey(value.UtxoId),
				value: value,
			}
		}
		sort.Slice(deleteUtxos, func(i, j int) bool {
			return bytes.Compare(deleteUtxos[i].key, deleteUtxos[j].key) < 0
		})
		for _, v := range deleteUtxos {
			_, err := txn.GetRef(v.key)
			bExist := err == nil
			utxo := v.value
			utxoId := v.value.UtxoId
			for _, address := range utxo.Address.Addresses {
				value, ok := addressValueMap[address]
				if ok {
					if bExist {
						// 存在数据库中，等会去删除
						value.Utxos[utxoId] = &common.UtxoValue{Op: -1}
					} else {
						// 仅从缓存数据中删除
						delete(value.Utxos, utxoId)
					}
				} else {
					if bExist {
						// 存在数据库中，等会去删除
						utxos := make(map[uint64]*common.UtxoValue)
						utxos[utxoId] = &common.UtxoValue{Op: -1}

						id, op := b.getAddressId(address)
						if op >= 0 {
							value = &common.AddressValueInDB{
								AddressType: uint32(utxo.Address.Type),
								AddressId:   id,
								Op:          op,
								Utxos:       utxos,
							}
							addressValueMap[address] = value
						} else {
							common.Log.Panicf("utxo %x exists but address %s not exists.", utxoId, address)
						}
					}
				}
			}
		}

		return nil
	})

	common.Log.Infof("BaseIndexer.prefechAddress add %d, del %d, address %d in %v",
		len(b.utxoIndex.Index), len(b.delUTXOs), len(addressValueMap), time.Since(startTime))

	return addressValueMap
}

func (b *BaseIndexer) UpdateDB() {
	common.Log.Infof("BaseIndexer->updateBasicDB %d start...", b.lastHeight)

	// 拿到所有的addressId
	addressValueMap := b.prefechAddressV2()

	//////
	// 测试一个异常问题：blockVector 区块丢失，导致exotic索引失败
	if len(b.blockVector) > 0 && b.blockVector[0].Height > 1 {
		key := db.GetBlockDBKey(b.blockVector[0].Height - 1)
		_, err := db.GetRawValueFromDB(key, b.db)
		if err != nil {
			common.Log.Panicf("can't find the previous block %d", b.blockVector[0].Height-1)
		}
	}
	/////

	wb := b.db.NewWriteBatch()
	defer wb.Close()

	totalSubsidySats := int64(0)
	AllUtxoAdded := uint64(0)
	for _, value := range b.blockVector {
		//common.Log.Infof("blockVector %d", value.Height)
		key := db.GetBlockDBKey(value.Height)
		err := db.SetDB(key, value, wb)
		if err != nil {
			common.Log.Panicf("Error setting in db %v", err)
		}
		totalSubsidySats += value.OutputSats - value.InputSats
		AllUtxoAdded += uint64(value.OutputUtxo)
	}

	// 所有的地址都保存起来，数据太多。只保存nft相关的地址。
	// TODO 需要先询问nft模块有哪些地址需要保存
	// for k, v := range b.addressIdMap {
	// 	if v.Op > 0 {
	// 		err := db.BindAddressDBKeyToId(k, v.AddressId, wb)
	// 		if err != nil {
	// 			common.Log.Panicf("Error setting in db %v", err)
	// 		}
	// 	}
	// }

	startTime := time.Now()
	// Add the new utxos first
	utxoAdded := 0
	satsAdded := int64(0)
	utxoSkipped := 0
	for k, v := range b.utxoIndex.Index {
		//if len(v.Ordinals) == 0 {
		// 有些没有聪，一样可以花费，比如1025ca72299155eb5c2ef6c1918e7dfbdcffd04b0d13792e9773af72b827d28a:1 （testnet）
		// 这样的utxo需要保存起来
		//}
		// v.Address.Type == (txscript.NonStandardTy) 这样的utxo需要被记录下来，虽然地址是nil，ordinals也是nil
		// 比如：21e48796d17bcab49b1fea7211199c0fa1e296d2ecf4cf2f900cee62153ee331的所有输出 （testnet）
		if v.Address.Type == int(txscript.NullDataTy) {
			// 只有OP_RETURN 才不记录
			if v.Value == 0 {
				utxoSkipped++
				continue
			} else {
				// e362e21ff1d2ef78379d401d89b42ce3e0ce3e245f74b1f4cb624a8baa5d53ad:0 testnet
				common.Log.Infof("the OP_RETURN has %d sats in %s", v.Value, k)
			}
		}
		// 9173744691ac25f3cd94f35d4fc0e0a2b9d1ab17b4fe562acc07660552f95518 输出大量0sats的utxo
		key := db.GetUTXODBKey(k)
		utxoId := common.GetUtxoId(v)

		addressIds := make([]uint64, 0)
		for i, address := range v.Address.Addresses {
			addrvalue := addressValueMap[address]
			addressIds = append(addressIds, addrvalue.AddressId)
			addrkey := db.GetAddressValueDBKey(addrvalue.AddressId, utxoId, int(v.Address.Type), i)
			err := db.SetRawDB(addrkey, common.Uint64ToBytes(uint64(v.Value)), wb)
			if err != nil {
				common.Log.Panicf("Error setting in db %v", err)
			}
			if addrvalue.Op > 0 {
				err = db.BindAddressDBKeyToId(address, addrvalue.AddressId, wb)
				if err != nil {
					common.Log.Panicf("Error setting in db %v", err)
				}
			}
		}

		saveUTXO := &common.UtxoValueInDB{
			UtxoId:      utxoId,
			AddressType: uint32(v.Address.Type),
			ReqSigs:     uint32(v.Address.ReqSig),
			AddressIds:  addressIds,
			Ordinals:    v.Ordinals,
		}
		//err = db.SetDB(key, saveUTXO, wb)
		err := db.SetDBWithProto3(key, saveUTXO, wb)
		if err != nil {
			common.Log.Panicf("Error setting in db %v", err)
		}
		err = db.BindUtxoDBKeyToId(key, saveUTXO.UtxoId, wb)
		if err != nil {
			common.Log.Panicf("Error setting in db %v", err)
		}
		utxoAdded++
		satsAdded += v.Value
	}
	common.Log.Infof("BaseIndexer.updateBasicDB-> add utxos %d (+ %d), cost: %v", utxoAdded, utxoSkipped, time.Since(startTime))

	// 很多要删除的utxo，其实还没有保存到数据库
	startTime = time.Now()
	utxoDeled := 0
	for _, value := range b.delUTXOs {

		utxoDeled++
		key := db.GetUTXODBKey(value.Utxo)
		err := wb.Delete([]byte(key))
		if err != nil {
			common.Log.Errorf("BaseIndexer.updateBasicDB-> Error deleting db: %v\n", err)
		}
		err = db.UnBindUtxoId(value.UtxoId, wb)
		if err != nil {
			common.Log.Errorf("BaseIndexer.updateBasicDB-> Error deleting db: %v\n", err)
		}

		for i, address := range value.Address.Addresses {
			addrvalue, ok := addressValueMap[address]
			if ok {
				addrkey := db.GetAddressValueDBKey(addrvalue.AddressId, value.UtxoId, int(value.Address.Type), i)
				err := wb.Delete(addrkey)
				if err != nil {
					common.Log.Errorf("BaseIndexer.updateBasicDB-> Error deleting db: %v\n", err)
				}
			} else {
				// 不存在
				//common.Log.Infof("address %s not exists", value.Address)
			}
		}

	}
	common.Log.Infof("BaseIndexer.updateBasicDB-> delete utxos %d, cost: %v", utxoDeled, time.Since(startTime))

	b.stats.UtxoCount += uint64(utxoAdded)
	b.stats.UtxoCount -= uint64(utxoDeled)
	b.stats.AllUtxoCount += AllUtxoAdded
	b.stats.TotalSats += totalSubsidySats
	b.stats.SyncBlockHash = b.lastHash
	b.stats.SyncHeight = b.lastHeight
	err := db.SetDB([]byte(SyncStatsKey), b.stats, wb)
	if err != nil {
		common.Log.Panicf("BaseIndexer.updateBasicDB-> Error setting in db %v", err)
	}

	startTime = time.Now()
	err = wb.Flush()
	if err != nil {
		common.Log.Panicf("BaseIndexer.updateBasicDB-> Error satwb flushing writes to db %v", err)
	}
	common.Log.Infof("BaseIndexer.updateBasicDB-> flush db,  cost: %v", time.Since(startTime))

	// reset memory buffer
	b.blockVector = make([]*common.BlockValueInDB, 0)
	b.utxoIndex = common.NewUTXOIndex()
	b.delUTXOs = make([]*UtxoValue, 0)
	b.addressIdMap = make(map[string]*AddressStatus)
}

func (b *BaseIndexer) removeUtxo(addrmap *map[string]*common.AddressValueInDB, utxo *UtxoValue, txn common.ReadBatch) {
	utxoId := utxo.UtxoId
	key := db.GetUtxoIdKey(utxoId)
	_, err := txn.Get(key)
	bExist := err == nil
	for _, address := range utxo.Address.Addresses {
		value, ok := (*addrmap)[address]
		if ok {
			if bExist {
				// 存在数据库中，等会去删除
				value.Utxos[utxoId] = &common.UtxoValue{Op: -1}
			} else {
				// 仅从缓存数据中删除
				delete(value.Utxos, utxoId)
			}
		} else {
			if bExist {
				// 存在数据库中，等会去删除
				utxos := make(map[uint64]*common.UtxoValue)
				utxos[utxoId] = &common.UtxoValue{Op: -1}

				id, op := b.getAddressId(address)
				if op >= 0 {
					value = &common.AddressValueInDB{
						AddressType: uint32(utxo.Address.Type),
						AddressId:   id,
						Op:          op,
						Utxos:       utxos,
					}
					(*addrmap)[address] = value
				} else {
					common.Log.Panicf("utxo %x exists but address %s not exists.", utxoId, address)
				}
			}
		}
	}
}

func (b *BaseIndexer) addUtxo(addrmap *map[string]*common.AddressValueInDB, output *common.Output) {
	utxoId := common.GetUtxoId(output)
	sats := output.Value
	for _, address := range output.Address.Addresses {
		value, ok := (*addrmap)[address]
		if ok {
			utxovalue, ok := value.Utxos[utxoId]
			if ok {
				if utxovalue.Value != sats {
					utxovalue.Value = sats
					utxovalue.Op = 1
				}
			} else {
				value.Utxos[utxoId] = &common.UtxoValue{Op: 1, Value: sats}
			}
		} else {
			utxos := make(map[uint64]*common.UtxoValue)
			utxos[utxoId] = &common.UtxoValue{Op: 1, Value: sats}
			id, op := b.getAddressId(address)
			value = &common.AddressValueInDB{
				AddressType: uint32(output.Address.Type),
				AddressId:   id,
				Op:          op,
				Utxos:       utxos,
			}
			(*addrmap)[address] = value
		}
	}
}

func (b *BaseIndexer) handleReorg(currentBlock *common.Block) int {
	common.Log.Warnf("BaseIndexer.handleReorg-> reorg detected at heigh %d", currentBlock.Height)

	// clean memory and reload stats from DB
	// b.reset()
	//b.stats.ReorgsDetected = append(b.stats.ReorgsDetected, currentBlock.Height)
	b.drainBlocksChan()

	reorgHeight := currentBlock.Height
	for i := b.lastHeight - b.keepBlockHistory + 1; i <= b.lastHeight; i++ {
		blockHash, ok := b.prevBlockHashMap[i]
		if ok {
			hash, err := getBlockHash(uint64(i))
			if err == nil {
				if hash != blockHash {
					common.Log.Warnf("detected reorg at height %d, old hash %s, new hash %s", i, blockHash, hash)
					reorgHeight = i
					if i == b.lastHeight-b.keepBlockHistory+1 {
						common.Log.Panic("reorg may occur in previous block!")
					}
					break
				}
			}
		}
	}
	b.prevBlockHashMap = make(map[int]string)
	return reorgHeight
}

// syncToBlock continues from the sync height to the current height
func (b *BaseIndexer) syncToBlock(height int, stopChan chan struct{}) int {
	if b.lastHeight == height {
		common.Log.Infof("BaseIndexer.SyncToBlock-> already synced to block %d", height)
		return 0
	}

	common.Log.Infof("BaseIndexer.SyncToBlock-> currentHeight %d, targetHeight %d", b.lastHeight, height)

	// if we don't start from precisely this heigh the UTXO index is worthless
	// we need to start from exactly where we left off
	start := b.lastHeight + 1

	periodProcessedTxs := 0
	startTime := time.Now() // Record the start time

	logProgressPeriod := 1

	stopBlockFetcherChan := make(chan struct{})
	go b.spawnBlockFetcher(start, height, stopBlockFetcherChan)

	for i := start; i <= height; i++ {
		if b.maxIndexHeight > 0 && b.lastHeight >= b.maxIndexHeight {
			b.forceUpdateDB()
			break
		}

		select {
		case <-stopChan:
			common.Log.Errorf("BaseIndexer.SyncToBlock-> Graceful shutdown received")
			return -1
		default:
			block := <-b.blocksChan

			if block == nil {
				common.Log.Errorf("BaseIndexer.SyncToBlock-> fetch block failed %d", i)
				return -2
			}
			//common.Log.Infof("BaseIndexer.SyncToBlock-> get block: cost: %v", time.Since(startTime))

			// make sure that we are at the correct block height
			if block.Height != i {
				common.Log.Panicf("BaseIndexer.SyncToBlock-> expected block height %d, got %d", i, block.Height)
			}

			// detect reorgs
			if i > 0 && block.PrevBlockHash != b.lastHash {
				common.Log.WithField("BaseIndexer.SyncToBlock-> height", i).Warn("reorg detected")
				stopBlockFetcherChan <- struct{}{}
				return b.handleReorg(block)
			}

			localStartTime := time.Now()
			b.prefetchIndexesFromDB(block)
			common.Log.Infof("BaseIndexer.SyncToBlock-> prefetchIndexesFromDB: cost: %v", time.Since(localStartTime))
			localStartTime = time.Now()
			b.assignOrdinals_sat20(block)
			common.Log.Infof("BaseIndexer.SyncToBlock-> assignOrdinals: cost: %v", time.Since(localStartTime))

			// Update the sync stats
			b.lastHeight = block.Height
			b.lastHash = block.Hash
			b.prevBlockHashMap[b.lastHeight] = b.lastHash
			if len(b.prevBlockHashMap) > b.keepBlockHistory {
				delete(b.prevBlockHashMap, b.lastHeight-b.keepBlockHistory)
			}

			//localStartTime = time.Now()
			b.blockprocCB(block)
			//common.Log.Infof("BaseIndexer.SyncToBlock-> blockproc: cost: %v", time.Since(localStartTime))

			if (block.Height != 0 && block.Height%b.periodFlushToDB == 0 && height-block.Height > b.keepBlockHistory) ||
				height-block.Height == b.keepBlockHistory {
				//localStartTime = time.Now()
				b.forceUpdateDB()
				//common.Log.Infof("BaseIndexer.SyncToBlock-> forceUpdateDB: cost: %v", time.Since(localStartTime))
			}

			if i%logProgressPeriod == 0 {
				periodProcessedTxs += len(block.Transactions)
				elapsedTime := time.Since(startTime)
				timePerTx := elapsedTime / time.Duration(periodProcessedTxs)
				readableTime := block.Timestamp.Format("2006-01-02 15:04:05")
				common.Log.Infof("processed block %d (%s) with %d transactions took %v (%v per tx)\n", block.Height, readableTime, periodProcessedTxs, elapsedTime, timePerTx)
				startTime = time.Now()
				periodProcessedTxs = 0
			}
			//common.Log.Info("")
		}
	}

	//b.forceUpdateDB()

	common.Log.Infof("BaseIndexer.SyncToBlock-> already sync to block %d-%d\n", b.lastHeight, b.stats.SyncHeight)
	return 0
}

// 确保输出是第一个。只需要检查第一组的最后一个和第二组的第一个
func appendRanges(rngs1, rngs2 []*common.Range) []*common.Range {
	var r1, r2 *common.Range
	len1 := len(rngs1)
	len2 := len(rngs2)
	rngs2 = common.CloneRanges(rngs2) // 不要影响第二个参数
	if len1 > 0 {
		if len2 == 0 {
			return rngs1
		}
		r1 = rngs1[len1-1]
		r2 = rngs2[0]
		if r1.Start+r1.Size == r2.Start {
			r1.Size += r2.Size
			rngs1 = append(rngs1, rngs2[1:]...)
		} else {
			rngs1 = append(rngs1, rngs2...)
		}
		return rngs1
	} else {
		rngs1 = append(rngs1, rngs2...)
		return rngs1
	}
}

func (b *BaseIndexer) assignOrdinals_sat20(block *common.Block) {
	first := b.lastSats
	coinbaseOrdinals := []*common.Range{{Start: first, Size: 0}}
	blockValue := &common.BlockValueInDB{Height: block.Height,
		Timestamp: block.Timestamp.Unix(),
		TxAmount:  len(block.Transactions),
	}
	firstblock := block.Height
	if len(b.blockVector) > 0 {
		firstblock = b.blockVector[0].Height
	}

	addedUtxoCount := 0
	deledUtxoCount := 0

	satsInput := int64(0)
	satsOutput := int64(0)
	for _, tx := range block.Transactions[1:] {
		ranges := make([]*common.Range, 0)
		for _, input := range tx.Inputs {
			// the utxo to be spent in the format txid:vout
			utxoKey := common.GetUtxo(block.Height, input.Txid, int(input.Vout))

			// delete the utxo from the utxo index
			inputUtxo, ok := b.utxoIndex.Index[utxoKey]
			if !ok {
				common.Log.Panicf("%s does not exist in the utxo index", utxoKey)
			}
			deledUtxoCount++
			delete(b.utxoIndex.Index, utxoKey)
			utxoid := common.GetUtxoId(inputUtxo)
			if inputUtxo.Height < firstblock {
				value := &UtxoValue{Utxo: utxoKey, Address: inputUtxo.Address,
					UtxoId: utxoid, Value: inputUtxo.Value}
				b.delUTXOs = append(b.delUTXOs, value)
			}
			satsInput += inputUtxo.Value

			input.Address = inputUtxo.Address
			input.Ordinals = common.CloneRanges(inputUtxo.Ordinals)
			input.UtxoId = utxoid

			// add the utxo's ordinals to the list of ordinals to be transferred
			ranges = appendRanges(ranges, inputUtxo.Ordinals)
		}

		for _, output := range tx.Outputs {
			// transfer the ordinals to the output
			transferred, remaining := common.TransferRanges(ranges, output.Value)
			output.Ordinals = common.CloneRanges(transferred)
			ranges = remaining
			// add the output to the utxo index
			u := common.GetUtxo(block.Height, tx.Txid, int(output.N))
			b.utxoIndex.Index[u] = output
			addedUtxoCount++
			satsOutput += output.Value
		}

		// add the remaining ordinals to the coinbase ordinals
		// those are the ordinals spent on fees
		coinbaseOrdinals = appendRanges(coinbaseOrdinals, ranges)
	}

	for _, output := range block.Transactions[0].Outputs {
		u := common.GetUtxo(block.Height, block.Transactions[0].Txid, int(output.N))
		b.utxoIndex.Index[u] = output
		addedUtxoCount++
		satsOutput += output.Value
	}

	// adjust the coinbaseOrdinals[0]
	size := satsOutput - satsInput
	coinbaseOrdinals[0].Size = size
	b.lastSats += size

	for _, output := range block.Transactions[0].Outputs {
		// transfer the coinbase ordinals to the output
		transferred, remaining := common.TransferRanges(coinbaseOrdinals, output.Value)
		output.Ordinals = common.CloneRanges(transferred)
		coinbaseOrdinals = remaining
	}
	//common.Log.Infof("b.utxoIndex.Index %d, b.delUTXOs %d, added %d, deled %d",
	//		len(b.utxoIndex.Index), len(b.delUTXOs), addedUtxoCount, deledUtxoCount)

	// testnet3，height = 33995, 没有领奖励
	if len(coinbaseOrdinals) != 0 && coinbaseOrdinals[0].Size > 0 {
		common.Log.Panicf("block %d sats %d-%d wrong", block.Height, first, size)
	}

	blockValue.Ordinals.Start = first
	blockValue.Ordinals.Size = satsOutput - satsInput
	blockValue.InputUtxo = deledUtxoCount
	blockValue.OutputUtxo = addedUtxoCount
	blockValue.InputSats = satsInput
	blockValue.OutputSats = satsOutput
	blockValue.LostSats = nil // sat20, always zero. ordinals, lost sats ranges.
	b.blockVector = append(b.blockVector, blockValue)
}

func (b *BaseIndexer) getAddressIdFromTxn(address string, bGenerateNew bool, txn common.ReadBatch) (uint64, bool) {
	bExist := true
	addressId, err := db.GetAddressIdFromTxn(txn, address)
	if err == common.ErrKeyNotFound {
		bExist = false
		if bGenerateNew {
			addressId = b.generateAddressId()
		}
	} else if err != nil {
		common.Log.Panicf("GetValueFromDBWithType-> Error loading address %s from db: %v", address, err)
	}
	return addressId, bExist
}

func (b *BaseIndexer) SyncToChainTip(stopChan chan struct{}) int {
	count, err := getBlockCount()
	if err != nil {
		common.Log.Errorf("failed to get block count %v", err)
		return -2
	}

	b.stats.ChainTip = int(count)
	return b.syncToBlock(int(count), stopChan)
}

func (b *BaseIndexer) loadUtxoFromDB(utxostr string) error {
	return b.db.View(func(txn common.ReadBatch) error {
		return b.loadUtxoFromTxn(utxostr, txn)
	})
}

func (b *BaseIndexer) loadUtxoFromTxn(utxostr string, txn common.ReadBatch) error {
	utxo := &common.UtxoValueInDB{}
	dbKey := db.GetUTXODBKey(utxostr)
	err := db.GetValueFromTxnWithProto3(dbKey, txn, utxo)
	if err == common.ErrKeyNotFound {
		return err
	}
	if err != nil {
		common.Log.Errorf("failed to get value of utxo: %s, %v", utxostr, err)
		return err
	}

	var addresses common.ScriptPubKey
	for _, addressId := range utxo.AddressIds {
		address, err := db.GetAddressByID(txn, addressId)
		if err != nil {
			common.Log.Errorf("failed to get address by id %d, utxo: %s, utxoId: %d, err: %v", addressId, utxostr, utxo.UtxoId, err)
			return err
		}
		b.addressIdMap[address] = &AddressStatus{AddressId: addressId, Op: 0}
		addresses.Addresses = append(addresses.Addresses, address)
	}
	addresses.Type = int(utxo.AddressType)
	addresses.ReqSig = int(utxo.ReqSigs)

	// TODO 对于多签的utxo，目前相当于把这个utxo给第一个地址
	height, txid, vout := common.FromUtxoId(utxo.UtxoId)
	b.utxoIndex.Index[utxostr] = &common.Output{Height: height, TxId: txid,
		Value:   common.GetOrdinalsSize(utxo.Ordinals),
		Address: &addresses,
		N:       int64(vout), Ordinals: utxo.Ordinals}
	return nil
}

func (b *BaseIndexer) prefetchIndexesFromDB(block *common.Block) {
	/* TODO 在后期，这个函数经常消耗了跑区块的80%时间，有时间需要继续优化。
	2025-08-21 10:09:02 [info] default: BaseIndexer.prefetchIndexesFromDB-> prefetched 8338 in 2.779987657s
	2025-08-21 10:09:02 [info] default: BaseIndexer.SyncToBlock-> prefetchIndexesFromDB: cost: 2.780052419s
	2025-08-21 10:09:02 [info] default: BaseIndexer.SyncToBlock-> assignOrdinals: cost: 42.401982ms
	2025-08-21 10:09:02 [info] default: updateExoticTicker in 404.766µs
	2025-08-21 10:09:03 [info] default: NftIndexer.UpdateTransfer loop 3339 in 1.103972964s
	2025-08-21 10:09:03 [info] default: OrdxIndexer->UpdateTransfer loop 3339 in 1.800919ms
	2025-08-21 10:09:03 [info] default: OrdxIndexer->CheckSelf took 70.940899ms.
	2025-08-21 10:09:03 [info] default: RuneIndexer.UpdateTransfer-> handle block succ, height:882895, tx count:3339, update holder count:374, remove holder count:372, block took time:53.112239ms, tx took avg time:15.906µs
	2025-08-21 10:09:03 [info] default: processOrdProtocol 882895,is done: cost: 1.233547213s
	2025-08-21 10:09:03 [info] default: processed block 882895 (2025-02-08 22:05:46) with 3339 transactions took 4.056047383s (1.214749ms per tx)

	2025-08-21 10:09:06 [info] default: BaseIndexer.prefetchIndexesFromDB-> prefetched 7608 in 3.235794921s
	2025-08-21 10:09:06 [info] default: BaseIndexer.SyncToBlock-> prefetchIndexesFromDB: cost: 3.235867578s
	2025-08-21 10:09:06 [info] default: BaseIndexer.SyncToBlock-> assignOrdinals: cost: 40.887636ms
	2025-08-21 10:09:06 [info] default: updateExoticTicker in 392.84µs
	2025-08-21 10:09:07 [info] default: NftIndexer.UpdateTransfer loop 3749 in 1.093159612s
	2025-08-21 10:09:07 [info] default: OrdxIndexer->UpdateTransfer loop 3749 in 7.705359ms
	2025-08-21 10:09:08 [info] default: OrdxIndexer->CheckSelf took 101.788654ms.
	2025-08-21 10:09:08 [info] default: RuneIndexer.UpdateTransfer-> handle block succ, height:882896, tx count:3749, update holder count:322, remove holder count:304, block took time:54.745046ms, tx took avg time:14.602µs
	2025-08-21 10:09:08 [info] default: processOrdProtocol 882896,is done: cost: 1.270160635s
	2025-08-21 10:09:08 [info] default: processed block 882896 (2025-02-08 22:25:32) with 3749 transactions took 4.546972122s (1.212849ms per tx)
	*/

	startTime := time.Now()

	type pair struct {
		key   []byte
		value string
	}

	utxos := make([]*pair, 0)
	addressMap := make(map[string]uint64)

	//b.db.View(func(txn common.ReadBatch) error {

	for _, tx := range block.Transactions {
		for _, input := range tx.Inputs {
			if input.Vout >= 0xffffffff {
				continue
			}

			utxo := common.GetUtxo(block.Height, input.Txid, int(input.Vout))
			if _, ok := b.utxoIndex.Index[utxo]; !ok {
				utxos = append(utxos, &pair{
					key:   db.GetUTXODBKey(utxo),
					value: utxo,
				})
				// err := b.loadUtxoFromTxn(utxo, txn)
				// if err == common.ErrKeyNotFound {
				// 	continue
				// } else if err != nil {
				// 	common.Log.Panicf("failed to get value of utxo: %s, %v", utxo, err)
				// }
			}
		}

		for _, output := range tx.Outputs {
			for _, address := range output.Address.Addresses {
				_, ok := b.addressIdMap[address]
				if !ok {
					addressMap[address] = common.INVALID_ID
					// addressId, bExist := b.getAddressIdFromTxn(address, true, txn)
					// op := 1
					// if bExist {
					// 	op = 0
					// }
					// b.addressIdMap[address] = &AddressStatus{addressId, op}
				}
			}
		}
	}
	// 	return nil
	// })

	// pebble数据库的优化手段: 尽可能将随机读变成按照key的顺序读
	sort.Slice(utxos, func(i, j int) bool {
		return bytes.Compare(utxos[i].key, utxos[j].key) < 0
	})

	b.db.View(func(txn common.ReadBatch) error {
		utxoAddressMap := make(map[string][]uint64)
		addressIdMap := make(map[uint64]string)
		for _, utxo := range utxos {
			utxoValue := &common.UtxoValueInDB{}
			err := db.GetValueFromTxnWithProto3(utxo.key, txn, utxoValue)
			if err == common.ErrKeyNotFound {
				continue
			} else if err != nil {
				common.Log.Panicf("failed to get value of utxo: %s, %v", utxo, err)
				return err
			}

			for _, addressId := range utxoValue.AddressIds {
				addressIdMap[addressId] = ""
			}
			utxoAddressMap[utxo.value] = utxoValue.AddressIds

			var addresses common.ScriptPubKey
			addresses.Type = int(utxoValue.AddressType)
			addresses.ReqSig = int(utxoValue.ReqSigs)

			height, txid, vout := common.FromUtxoId(utxoValue.UtxoId)
			b.utxoIndex.Index[utxo.value] = &common.Output{Height: height, TxId: txid,
				Value:   common.GetOrdinalsSize(utxoValue.Ordinals),
				Address: &addresses,
				N:       int64(vout), Ordinals: utxoValue.Ordinals}
		}

		type addressIdPair struct {
			key   []byte
			value uint64
		}

		addressIds := make([]*addressIdPair, len(addressIdMap))
		i := 0
		for k := range addressIdMap {
			addressIds[i] = &addressIdPair{
				key:   db.GetAddressIdKey(k),
				value: k,
			}
			i++
		}
		sort.Slice(addressIds, func(i, j int) bool {
			return bytes.Compare(addressIds[i].key, addressIds[j].key) < 0
		})
		for _, v := range addressIds {
			value, err := txn.Get(v.key)
			if err != nil {
				common.Log.Panicf("failed to get value of address: %s, %v", v.key, err)
			}
			address := strings.TrimPrefix(string(value), common.DB_KEY_ADDRESS)
			addressMap[address] = v.value
			addressIdMap[v.value] = address
		}
		addresses := make([]string, len(addressMap))
		i = 0
		for k := range addressMap {
			addresses[i] = k
			i++
		}
		sort.Slice(addresses, func(i, j int) bool {
			return addresses[i] < addresses[j]
		})
		for _, address := range addresses {
			s, ok := b.addressIdMap[address]
			if !ok {
				addressId, bExist := b.getAddressIdFromTxn(address, true, txn)
				op := 1
				if bExist {
					op = 0
				}
				s = &AddressStatus{addressId, op}
				b.addressIdMap[address] = s
			}
			addressIdMap[s.AddressId] = address
		}

		for _, utxo := range utxos {
			output := b.utxoIndex.Index[utxo.value]
			addresses := utxoAddressMap[utxo.value]
			for _, addressId := range addresses {
				output.Address.Addresses = append(output.Address.Addresses, addressIdMap[addressId])
			}
		}

		common.Log.Infof("BaseIndexer.prefetchIndexesFromDB-> prefetched %d in %v", len(addressIdMap), time.Since(startTime))

		return nil
	})

}

func (b *BaseIndexer) loadSyncStatsFromDB() {

	syncStats := &SyncStats{}
	err := db.GetValueFromDB([]byte(SyncStatsKey), syncStats, b.db)
	if err == common.ErrKeyNotFound {
		common.Log.Info("BaseIndexer.LoadSyncStatsFromDB-> No sync stats found in db")
		syncStats.SyncHeight = -1
	} else if err != nil {
		common.Log.Panicf("BaseIndexer.LoadSyncStatsFromDB-> Error loading sync stats from db: %v", err)
	}
	common.Log.Infof("stats: %v", syncStats)
	common.Log.Infof("Code Ver: %s", common.ORDX_INDEXER_VERSION)
	common.Log.Infof("DB Ver: %s", b.GetBaseDBVer())

	if syncStats.ReorgsDetected == nil {
		syncStats.ReorgsDetected = make([]int, 0)
	}

	b.stats = syncStats
	b.lastHash = b.stats.SyncBlockHash
	b.lastHeight = b.stats.SyncHeight
	b.lastSats = b.stats.TotalSats

}

// triggerReorg is meant to be used for debugging and tests only
// I used it to simulate a reorg
// func (b *BaseIndexer) triggerReorg() {
// 	common.Log.Errorf("set reorg flag when test")
// 	b.lastHash = "wrong"
// }

func (b *BaseIndexer) generateAddressId() uint64 {
	id := b.stats.AddressCount
	b.stats.AddressCount++
	return id
}

// 耗时很长。仅用于在数据编译完成时验证数据，或者测试时验证数据。
func (b *BaseIndexer) CheckSelf() bool {

	common.Log.Info("BaseIndexer->checkSelf ... ")
	// for height, leak := range b.leakBlocks.SatsLeakBlocks {
	// 	common.Log.Infof("block %d leak %d", height, leak)
	// }
	// common.Log.Infof("Total leaks %d", b.leakBlocks.TotalLeakSats)

	startTime := time.Now()

	common.Log.Infof("stats: %v", b.stats)
	common.Log.Infof("Code Ver: %s", common.ORDX_INDEXER_VERSION)
	common.Log.Infof("DB Ver: %s", b.GetBaseDBVer())
	totalSats := common.FirstOrdinalInTheory(b.stats.SyncHeight + 1)
	common.Log.Infof("expected total sats %d", totalSats)
	common.Log.Infof("total leak sats %d", totalSats-b.stats.TotalSats)

	startTime2 := time.Now()
	common.Log.Infof("calculating in %s table ...", common.DB_KEY_BLOCK)
	var preValue *common.BlockValueInDB
	var leakSats int64
	for i := 0; i <= b.stats.SyncHeight; i++ {
		key := db.GetBlockDBKey(i)
		value := common.BlockValueInDB{}
		err := db.GetValueFromDB(key, &value, b.db)
		if err != nil {
			common.Log.Panicf("GetValueFromDB %s error: %v", key, err)
		}
		satsInTheory := common.SubsidyInTheory(value.Height)
		leakSats += satsInTheory - (value.OutputSats-value.InputSats) // common.GetOrdinalsSize(value.LostSats) sat20 always == 0
		if value.Height != i {
			common.Log.Panicf("block %d invalid value %d", i, value.Height)
		}
		if preValue != nil {
			if preValue.Ordinals.Start+preValue.Ordinals.Size != value.Ordinals.Start {
				common.Log.Panicf("block %d invalid range %d-%d, %d", i, preValue.Ordinals.Start, preValue.Ordinals.Size, value.Ordinals.Start)
			} 
		}
		if i == b.stats.SyncHeight {
			if b.stats.TotalSats != value.Ordinals.Start+value.Ordinals.Size {
				common.Log.Panicf("block %d invalid total sats %d-%d, %d", i, value.Ordinals.Start, value.Ordinals.Size, b.stats.TotalSats)
			}
		}

		preValue = &value
	}
	common.Log.Infof("%s table takes %v", common.DB_KEY_BLOCK, time.Since(startTime2))
	common.Log.Infof("leak sats %d, expected leak sats %d", leakSats, totalSats-b.stats.TotalSats)

	satsInUtxo := int64(0)
	utxoCount := 0
	nonZeroUtxo := 0
	addressInUtxo := 0
	addressesInT1 := make(map[uint64]bool, 0)
	utxosInT1 := make(map[uint64]bool, 0)
	startTime2 = time.Now()
	common.Log.Infof("calculating in %s table ...", common.DB_KEY_UTXO)
	b.db.BatchRead([]byte(common.DB_KEY_UTXO), false, func(k, v []byte) error {

		var value common.UtxoValueInDB
		err := db.DecodeBytesWithProto3(v, &value)
		if err != nil {
			common.Log.Panicf("item.Value error: %v", err)
		}

		// 用于打印不存在table2中的utxo
		// if value.UtxoId == 0x17453400960000 {
		// 	key := item.Key()
		// 	str, _ := db.GetUtxoByDBKey(key)
		// 	common.Log.Infof("%x %s", value.UtxoId, str)
		// }

		sats := (common.GetOrdinalsSize(value.Ordinals))
		if sats > 0 {
			nonZeroUtxo++
		}

		satsInUtxo += sats
		utxoCount++

		for _, addressId := range value.AddressIds {
			addressesInT1[addressId] = true
		}
		utxosInT1[value.UtxoId] = true

		addressInUtxo = len(addressesInT1)
		return nil
	})
	common.Log.Infof("%s table takes %v", common.DB_KEY_UTXO, time.Since(startTime2))
	common.Log.Infof("1. utxo: %d(%d), sats %d, address %d", utxoCount, nonZeroUtxo, satsInUtxo, addressInUtxo)

	satsInAddress := int64(0)
	allAddressCount := 0
	allutxoInAddress := 0
	nonZeroUtxoInAddress := 0
	addressesInT2 := make(map[uint64]bool, 0)
	utxosInT2 := make(map[uint64]bool, 0)

	startTime2 = time.Now()
	common.Log.Infof("calculating in %s table ...", common.DB_KEY_ADDRESSVALUE)
	b.db.BatchRead([]byte(common.DB_KEY_ADDRESSVALUE), false, func(k, v []byte) error {

		value := int64(common.BytesToUint64(v))

		addressId, utxoId, _, index, err := common.ParseAddressIdKey(string(k))
		if err != nil {
			common.Log.Panicf("ParseAddressIdKey %s failed: %v", string(k), err)
		}

		allutxoInAddress++
		if index == 0 {
			satsInAddress += value
			if value > 0 {
				nonZeroUtxoInAddress++
			}
		}

		addressesInT2[addressId] = true
		utxosInT2[utxoId] = true

		allAddressCount = len(addressesInT2)

		return nil
	})
	common.Log.Infof("%s table takes %v", common.DB_KEY_ADDRESSVALUE, time.Since(startTime2))
	common.Log.Infof("2. utxo: %d(%d), sats %d, address %d", allutxoInAddress, nonZeroUtxoInAddress, satsInAddress, allAddressCount)

	common.Log.Infof("utxos not in table %s", common.DB_KEY_ADDRESSVALUE)
	utxos1 := findDifferentItems(utxosInT1, utxosInT2)
	if len(utxos1) > 0 {
		//ids := b.printfUtxos(utxos1)
		//b.deleteUtxos(ids)
		// 因为badger数据库的bug，在DB_KEY_UTXO中删除的数据可能还会出现，在检查后需要重新删除，再次检查，但只重新检查一次
		// if !b.reCheck {
		// 	b.reCheck = true
		// 	return b.CheckSelf()
		// }
		b.printfUtxos(utxos1)
	}

	common.Log.Infof("utxos not in table %s", common.DB_KEY_UTXO)
	utxos2 := findDifferentItems(utxosInT2, utxosInT1)
	if len(utxos2) > 0 {
		// ids := b.printfUtxos(utxos2)
		// b.deleteUtxos(ids)
		b.printfUtxos(utxos2)
	}

	var addresses1, addresses2 map[uint64]bool
	common.Log.Infof("address not in table %s", common.DB_KEY_ADDRESSVALUE)
	b.db.View(func(txn common.ReadBatch) error {
		addresses1 = findDifferentItems(addressesInT1, addressesInT2)
		for uid := range addresses1 {
			str, _ := db.GetAddressByID(txn, uid)
			common.Log.Infof("%s", str)
		}

		common.Log.Infof("address not in table %s", common.DB_KEY_UTXO)
		addresses2 = findDifferentItems(addressesInT2, addressesInT1)
		for uid := range addresses2 {
			str, _ := db.GetAddressByID(txn, uid)
			common.Log.Infof("%s", str)
		}
		return nil
	})

	result := true
	if len(utxos1) > 0 || len(utxos2) > 0 || len(addresses1) > 0 || len(addresses2) > 0 {
		common.Log.Errorf("utxos or address differents")
		result = false
	}

	if addressInUtxo != allAddressCount {
		common.Log.Errorf("address count different %d %d", addressInUtxo, allAddressCount)
		result = false
	}

	if satsInUtxo != satsInAddress {
		common.Log.Errorf("sats different %d %d", satsInAddress, satsInUtxo)
		result = false
	}

	if nonZeroUtxo != nonZeroUtxoInAddress {
		common.Log.Errorf("utxo different %d %d", nonZeroUtxo, nonZeroUtxoInAddress)
		result = false
	}

	// testnet: block 26432 多奖励了0.001btc，2642多奖励了0.0015，所以测试网络对比数据会有异常，只在主网上验证
	// mainnet: 早期软件原因有些块没有拿到足够的奖励，比如124724
	if b.stats.TotalSats != satsInAddress {
		common.Log.Errorf("sats wrong %d %d", satsInAddress, b.stats.TotalSats)
		result = false
	}

	if result {
		common.Log.Infof("DB checked successfully, %v", time.Since(startTime))
		b.setDBVersion()
	} else {
		common.Log.Infof("DB checked failed, %v", time.Since(startTime))
	}

	return result
}

// 耗时很长。仅用于在数据编译完成时验证数据
func (b *BaseIndexer) verifyAllUtxosWithBtcd() {
	// 1. btcd同步到某个高度，断开跟互联网的连接，保持在当前高度
	// 2. indexer数据库同步到同一个高度
	// 3. indexer通过btcd，验证本地记录的所有utxo都是正确的
	// 目的：数据库在压缩后有数据错乱情况，暂时没办法解决，在压缩后必须做数据验证。
	// 验证数据分两步，一般调用checkself，只需要10分钟。最后数据跑到最新高度后，执行本函数，做一次最严格验证，可能需要4-5个小时
	b.db.BatchRead([]byte(common.DB_KEY_UTXO), false, func(k, v []byte) error {

		var value common.UtxoValueInDB

		err := db.DecodeBytesWithProto3(v, &value)
		if err != nil {
			common.Log.Errorf("item.Value error: %v", err)
			return err
		}

		return nil
	})
}

func findDifferentItems(map1, map2 map[uint64]bool) map[uint64]bool {
	differentItems := make(map[uint64]bool)
	for key := range map1 {
		if _, exists := map2[key]; !exists {
			differentItems[key] = true
		}
	}

	return differentItems
}

// only for test
func (b *BaseIndexer) printfUtxos(utxos map[uint64]bool) map[uint64]string {
	result := make(map[uint64]string)
	b.db.BatchRead([]byte(common.DB_KEY_UTXO), false, func(k, v []byte) error {

		var value common.UtxoValueInDB
		err := db.DecodeBytesWithProto3(v, &value)
		if err != nil {
			common.Log.Errorf("item.Value error: %v", err)
			return err
		}

		// 用于打印不存在table2中的utxo
		if _, ok := utxos[value.UtxoId]; ok {
			str, err := db.GetUtxoByDBKey(k)
			if err == nil {
				common.Log.Infof("%x %s %d", value.UtxoId, str, common.GetOrdinalsSize(value.Ordinals))
				result[value.UtxoId] = str
			}

			delete(utxos, value.UtxoId)
			if len(utxos) == 0 {
				return nil
			}
		}

		return nil
	})

	return result
}

// only for test
func (b *BaseIndexer) deleteUtxos(utxos map[uint64]string) {
	wb := b.db.NewWriteBatch()
	defer wb.Close()

	for utxoId, utxo := range utxos {
		key := db.GetUTXODBKey(utxo)
		err := wb.Delete([]byte(key))
		if err != nil {
			common.Log.Errorf("BaseIndexer.updateBasicDB-> Error deleting db: %v\n", err)
		} else {
			common.Log.Infof("utxo deled: %s", utxo)
		}

		err = db.UnBindUtxoId(utxoId, wb)
		if err != nil {
			common.Log.Errorf("BaseIndexer.updateBasicDB-> Error deleting db: %v\n", err)
		} else {
			common.Log.Infof("utxo unbind: %d", utxoId)
		}
	}

	err := wb.Flush()
	if err != nil {
		common.Log.Panicf("BaseIndexer.updateBasicDB-> Error satwb flushing writes to db %v", err)
	}
}

func (b *BaseIndexer) setDBVersion() {
	err := db.SetRawValueToDB([]byte(BaseDBVerKey), []byte(common.BASE_DB_VERSION), b.db)
	if err != nil {
		common.Log.Panicf("Error setting in db %v", err)
	}
}

func (b *BaseIndexer) GetBaseDBVer() string {
	value, err := db.GetRawValueFromDB([]byte(BaseDBVerKey), b.db)
	if err != nil {
		common.Log.Errorf("GetRawValueFromDB failed %v", err)
		return ""
	}

	return string(value)
}

func (b *BaseIndexer) GetBaseDB() common.KVDB {
	return b.db
}

func (b *BaseIndexer) GetSyncHeight() int {
	return b.stats.SyncHeight
}

func (b *BaseIndexer) GetSyncStats() *SyncStats {
	return b.stats
}

// 这里需要小心同步状态。设置回来的状态，是已经备份到数据库中的数据的状态(UpdateDB中修改的状态)。不要覆盖其他状态。
func (b *BaseIndexer) SetSyncStats(s *SyncStats) {
	b.stats.SyncHeight = s.SyncHeight
	b.stats.SyncBlockHash = s.SyncBlockHash
	b.stats.AllUtxoCount = s.AllUtxoCount
	b.stats.TotalSats = s.TotalSats
	b.stats.UtxoCount = s.UtxoCount
}

func (b *BaseIndexer) GetHeight() int {
	return b.lastHeight
}

func (b *BaseIndexer) GetChainTip() int {
	return b.stats.ChainTip
}

func (b *BaseIndexer) SetReorgHeight(height int) {
	b.stats.ReorgsDetected = append(b.stats.ReorgsDetected, height)
	if len(b.stats.ReorgsDetected) > 100 {
		b.stats.ReorgsDetected = b.stats.ReorgsDetected[len(b.stats.ReorgsDetected)-50:]
	}
	err := db.GobSetDB([]byte(SyncStatsKey), b.stats, b.db)
	if err != nil {
		common.Log.Panicf("Error setting in db %v", err)
	}
}

func (b *BaseIndexer) GetBlockHistory() int {
	return b.keepBlockHistory
}

func (p *BaseIndexer) GetBlockInBuffer(height int) *common.BlockValueInDB {
	for _, block := range p.blockVector {
		if block.Height == height {
			return block
		}
	}

	return nil
}

func (p *BaseIndexer) getAddressId(address string) (uint64, int) {
	value, ok := p.addressIdMap[address]
	if !ok {
		common.Log.Errorf("can't find addressId %s", address)
		return common.INVALID_ID, -1
	}
	return value.AddressId, value.Op
}
