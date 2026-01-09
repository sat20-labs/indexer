package nft

import (
	"sort"
	"sync"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/base"
	indexerCommon "github.com/sat20-labs/indexer/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

type InscribeInfo struct {
	Input    *common.TxInput
	InOffset int64
	UtxoId   uint64
	Nft      *common.Nft
}

type SatInfo struct {
	AddressId  uint64
	UtxoId     uint64
	Offset     int64
	CurseCount int // 包括vindicated。 CurseCount+2 <= len(Nfts)，至少有两个非诅咒的铭文，意味着该sat是reinscription
	Nfts       map[*common.Nft]bool
}

func (p *SatInfo) ToNftsInSat(sat int64) *common.NftsInSat {
	nfts := &common.NftsInSat{
		Sat:            sat,
		OwnerAddressId: p.AddressId,
		UtxoId:         p.UtxoId,
		Offset:         p.Offset,
		CurseCount:     int32(p.CurseCount),
	}
	for k := range p.Nfts {
		nfts.Nfts = append(nfts.Nfts, k.Base.Id)
	}
	sort.Slice(nfts.Nfts, func(i, j int) bool {
		return nfts.Nfts[i] < nfts.Nfts[j]
	})
	return nfts
}

// 所有nft的记录
// 以后ns和ordx模块，数据变大，导致加载、跑数据等太慢，需要按照这个模块的方式来修改优化。
type NftIndexer struct {
	db           common.KVDB
	status       *common.NftStatus
	enableHeight int
	disabledSats map[int64]bool // 所有disabled的satoshi TODO 跑数据时需要禁止该功能，不要影响聪的属性

	baseIndexer     *base.BaseIndexer
	processCallback indexerCommon.BlockProcCallback
	mutex           sync.RWMutex

	// realtime buffer, utxoMap和satMap必须保持一致，utxo包含的聪，必须在satMap
	utxoMap               map[uint64]map[int64]int64 // utxo->sat->offset  确保utxo中包含的所有nft都列在这里
	satMap                map[int64]*SatInfo         // key: sat, 一个写入周期中新增加的铭文的转移结果，该sat绑定的nft都在这里
	contentMap            map[uint64]string          // contentId -> content
	contentToIdMap        map[string]uint64          //
	addedContentIdMap     map[uint64]bool
	inscriptionToNftIdMap map[string]*common.Nft // inscriptionId->nftId
	nftIdToinscriptionMap map[int64]*common.Nft  // nftId->inscriptionId

	// 暂时不需要清理
	contentTypeMap     map[int]string // ctId -> content type
	contentTypeToIdMap map[string]int //
	lastContentTypeId  int

	// 状态变迁
	//unboundNfts     []*common.Nft
	nftAdded  []*common.Nft // 保持顺序
	utxoDeled []uint64

	// 不需要备份的数据
	actionBufferMap map[int]map[int][]*InscribeInfo // txIndex-txInIndex
	nftAddedUtxoMap map[uint64][]*InscribeInfo      // 一个区块中，增量的nft在哪个输入中 utxoId->nft
}

func NewNftIndexer(db common.KVDB) *NftIndexer {
	enableHeight := 767430
	if !common.IsMainnet() {
		enableHeight = 27228
	}
	ns := &NftIndexer{
		db:           db,
		enableHeight: enableHeight,
		status:       nil,
	}
	return ns
}

// 只能被调用一次
func (p *NftIndexer) Init(baseIndexer *base.BaseIndexer,
	cb indexerCommon.BlockProcCallback) {
	p.baseIndexer = baseIndexer
	p.status = initStatusFromDB(p.db)
	p.disabledSats = loadAllDisalbedSatsFromDB(p.db)
	p.processCallback = cb

	p.utxoMap = make(map[uint64]map[int64]int64)
	p.satMap = make(map[int64]*SatInfo)
	p.nftAdded = make([]*common.Nft, 0)
	p.utxoDeled = make([]uint64, 0)

	p.contentMap = make(map[uint64]string)
	p.contentToIdMap = make(map[string]uint64)
	p.addedContentIdMap = make(map[uint64]bool)
	p.inscriptionToNftIdMap = make(map[string]*common.Nft)
	p.nftIdToinscriptionMap = make(map[int64]*common.Nft)

	p.contentTypeMap = getContentTypesFromDB(p.db)
	p.contentTypeToIdMap = make(map[string]int)
	for k, v := range p.contentTypeMap {
		p.contentTypeToIdMap[v] = k
	}
	p.lastContentTypeId = p.status.ContentTypeCount

	p.actionBufferMap = make(map[int]map[int][]*InscribeInfo)
	p.nftAddedUtxoMap = make(map[uint64][]*InscribeInfo)
}

func (p *NftIndexer) Clone(baseIndexer *base.BaseIndexer) *NftIndexer {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	newInst := NewNftIndexer(p.db)
	newInst.baseIndexer = baseIndexer

	newInst.disabledSats = p.disabledSats // 仅在rpc中使用
	newInst.utxoMap = make(map[uint64]map[int64]int64)
	for k, v := range p.utxoMap {
		nv := make(map[int64]int64)
		for s, o := range v {
			nv[s] = o
		}
		newInst.utxoMap[k] = nv
	}
	newInst.satMap = make(map[int64]*SatInfo)
	for k, v := range p.satMap {
		newV := &SatInfo{
			AddressId:  v.AddressId,
			UtxoId:     v.UtxoId,
			Offset:     v.Offset,
			CurseCount: v.CurseCount,
			Nfts:       make(map[*common.Nft]bool),
		}
		for nftId := range v.Nfts {
			newV.Nfts[nftId] = true
		}
		newInst.satMap[k] = newV
	}

	newInst.contentMap = make(map[uint64]string)
	newInst.contentToIdMap = make(map[string]uint64)
	for k, v := range p.contentMap {
		newInst.contentMap[k] = v
		newInst.contentToIdMap[v] = k
	}

	newInst.addedContentIdMap = make(map[uint64]bool)
	for k, v := range p.addedContentIdMap {
		newInst.addedContentIdMap[k] = v
	}

	newInst.inscriptionToNftIdMap = make(map[string]*common.Nft)
	for k, v := range p.inscriptionToNftIdMap {
		newInst.inscriptionToNftIdMap[k] = v
	}

	newInst.nftIdToinscriptionMap = make(map[int64]*common.Nft)
	for k, v := range p.nftIdToinscriptionMap {
		newInst.nftIdToinscriptionMap[k] = v
	}

	newInst.contentTypeMap = make(map[int]string)
	newInst.contentTypeToIdMap = make(map[string]int)
	for k, v := range p.contentTypeMap {
		newInst.contentTypeMap[k] = v
		newInst.contentTypeToIdMap[v] = k
	}

	newInst.nftAdded = make([]*common.Nft, len(p.nftAdded))
	for i, nft := range p.nftAdded {
		newInst.nftAdded[i] = nft.Clone()
	}

	newInst.utxoDeled = make([]uint64, len(p.utxoDeled))
	copy(newInst.utxoDeled, p.utxoDeled)

	newInst.status = p.status.Clone()

	return newInst
}

func (p *NftIndexer) Subtract(another *NftIndexer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for k := range another.utxoMap {
		delete(p.utxoMap, k)
	}
	for k, v := range another.satMap {
		nv, ok := p.satMap[k]
		if ok {
			if nv.UtxoId == v.UtxoId {
				// 没有变化就删除
				delete(p.satMap, k)
				// 同时必须删除utxoMap中对应的数据，保持一致
				delete(p.utxoMap, nv.UtxoId)
			}
		}
	}
	for k := range another.contentMap {
		delete(p.contentMap, k)
	}
	for k := range another.contentToIdMap {
		delete(p.contentToIdMap, k)
	}
	for k := range another.addedContentIdMap {
		delete(p.addedContentIdMap, k)
	}
	for k := range another.inscriptionToNftIdMap {
		delete(p.inscriptionToNftIdMap, k)
	}
	for k := range another.nftIdToinscriptionMap {
		delete(p.nftIdToinscriptionMap, k)
	}
	for k := range another.contentTypeMap {
		delete(p.contentTypeMap, k)
	}
	for k := range another.contentTypeToIdMap {
		delete(p.contentTypeToIdMap, k)
	}

	p.nftAdded = append([]*common.Nft(nil), p.nftAdded[len(another.nftAdded):]...)
	p.utxoDeled = append([]uint64(nil), p.utxoDeled[len(another.utxoDeled):]...)
}

// func (p *NftIndexer) IsEnabled() bool {
// 	return p.bEnabled
// }

func (p *NftIndexer) GetBaseIndexer() *base.BaseIndexer {
	return p.baseIndexer
}

func (p *NftIndexer) Repair() {

	fixingUtxoMap := make(map[uint64][]*SatOffset)
	p.db.BatchRead([]byte(DB_PREFIX_UTXO), false, func(k, v []byte) error {
		var value NftsInUtxo
		err := db.DecodeBytesWithProto3(v, &value)
		if err != nil {
			common.Log.Panicf("item.Value error: %v", err)
		}

		utxoId, err := ParseUtxoKey(string(k))
		if err != nil {
			common.Log.Panicf("item.Key error: %v", err)
		}

		for _, sat := range value.Sats {
			if sat.Sat < 0 {
				nv := make([]*SatOffset, 0)
				for _, sat := range value.Sats {
					if sat.Sat >= 0 {
						nv = append(nv, sat)
					}
				}
				fixingUtxoMap[utxoId] = nv
				break
			}
		}
		return nil
	})

	common.Log.Infof("detect %d utxo has unbound sats", len(fixingUtxoMap))

	wb := p.db.NewWriteBatch()
	defer wb.Close()

	for utxoId, sats := range fixingUtxoMap {
		utxokey := GetUtxoKey(utxoId)
		var err error
		if len(sats) == 0 {
			err = wb.Delete([]byte(utxokey))
		} else {
			utxoValue := NftsInUtxo{Sats: sats}
			err = db.SetDBWithProto3([]byte(utxokey), &utxoValue, wb)
		}

		if err != nil {
			common.Log.Panicf("NftIndexer->Repair Error setting %s in db %v", utxokey, err)
		}
	}

	err := wb.Flush()
	if err != nil {
		common.Log.Panicf("NftIndexer->Repair Flush failed. %v", err)
	}
}

func (b *NftIndexer) getContentId(content string) (uint64, error) {
	id, ok := b.contentToIdMap[content]
	if ok {
		return id, nil
	}

	var err error
	id, err = GetContentIdFromDB(b.db, content)
	if err == nil {
		b.contentToIdMap[content] = id
		b.contentMap[id] = content
	}

	return id, err
}

func (b *NftIndexer) getContentById(id uint64) (string, error) {
	content, ok := b.contentMap[id]
	if ok {
		return content, nil
	}

	var err error
	content, err = GetContentByIdFromDB(b.db, id)
	if err == nil {
		b.contentToIdMap[content] = id
		b.contentMap[id] = content
	}

	return content, err
}

func (b *NftIndexer) getInscriptionIdByNftId(id int64) (string, error) {
	nft, ok := b.nftIdToinscriptionMap[id]
	if ok {
		return nft.Base.InscriptionId, nil
	}

	var err error
	nft = b.getNftWithId(id)
	if nft != nil {
		b.inscriptionToNftIdMap[nft.Base.InscriptionId] = nft
		b.nftIdToinscriptionMap[id] = nft
	}

	return nft.Base.InscriptionId, err
}

// 注意
func (p *NftIndexer) getNftInBuffer(id int64) *common.Nft {
	return p.nftIdToinscriptionMap[id]
}

func (p *NftIndexer) getNftInBufferWithInscriptionId(inscriptionId string) *common.Nft {
	return p.inscriptionToNftIdMap[inscriptionId]
}

// 耗时很长。仅用于在数据编译完成时验证数据，或者测试时验证数据。
func (p *NftIndexer) CheckSelf(baseDB common.KVDB) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	common.Log.Info("NftIndexer->checkSelf ... ")

	startTime := time.Now()
	common.Log.Infof("stats: %v", p.status)

	// var wg sync.WaitGroup
	// wg.Add(3)

	vindicated := 0
	blessCount := 0
	curseCount := 0
	nftMap := make(map[int64]bool)
	p.db.BatchRead([]byte(DB_PREFIX_NFT), false, func(k, v []byte) error {
		//defer wg.Done()

		var value common.InscribeBaseContent
		err := db.DecodeBytesWithProto3(v, &value)
		if err != nil {
			common.Log.Panicf("item.Value error: %v", err)
		}
		if value.CurseType < 0 {
			nftMap[value.Id] = true
			curseCount++
			//common.Log.Infof("%d %s is cursed %d", value.Id, value.InscriptionId, value.CurseType)
		} else if value.CurseType == 0 {
			blessCount++
		} else {
			vindicated++
		}

		return nil
	})
	common.Log.Infof("blessed %d, cursed %d, vindicated %d, total %d", 
		blessCount, curseCount, vindicated, blessCount+curseCount+vindicated)
	for i := int64(1); i < int64(p.status.CurseCount); i++ {
		_, ok := nftMap[-i]
		if !ok {
			common.Log.Panicf("missing nft id -%d", i)
		}
	}

	addressesInT1 := make(map[uint64]bool, 0)
	utxosInT1 := make(map[uint64]bool, 0)
	satsInT1 := make(map[uint64]uint64, 0)
	nftsInT1 := make(map[int64]bool, 0)
	startTime2 := time.Now()
	common.Log.Infof("calculating in %s table ...", DB_PREFIX_SAT)
	p.db.BatchRead([]byte(DB_PREFIX_SAT), false, func(k, v []byte) error {
		//defer wg.Done()

		var value common.NftsInSat
		err := db.DecodeBytesWithProto3(v, &value)
		if err != nil {
			common.Log.Panicf("item.Value error: %v", err)
		}
		if value.Sat < 0 {
			// 负数铭文，没有绑定到任何聪的铭文，只统计nft数量
			for _, nftId := range value.Nfts {
				nftsInT1[nftId] = true
			}
			return nil
		}

		addressesInT1[value.OwnerAddressId] = true
		utxosInT1[value.UtxoId] = true
		satsInT1[uint64(value.Sat)] = value.UtxoId
		for _, nftId := range value.Nfts {
			nftsInT1[nftId] = true
		}

		return nil
	})
	common.Log.Infof("%s table takes %v", DB_PREFIX_SAT, time.Since(startTime2))
	common.Log.Infof("1: address %d, utxo %d, sats %d, nfts %d", len(addressesInT1), len(utxosInT1), len(satsInT1), len(nftsInT1))

	// utxo的数据涉及到delete操作，但是badger的delete操作有隐藏的bug，需要检查下该utxo是否存在
	utxosInT2 := make(map[uint64]bool)
	satsInT2 := make(map[uint64]uint64)
	startTime2 = time.Now()
	common.Log.Infof("calculating in %s table ...", DB_PREFIX_UTXO)
	p.db.BatchRead([]byte(DB_PREFIX_UTXO), false, func(k, v []byte) error {
		//defer wg.Done()

		var value NftsInUtxo
		err := db.DecodeBytesWithProto3(v, &value)
		if err != nil {
			common.Log.Panicf("item.Value error: %v", err)
		}

		utxoId, err := ParseUtxoKey(string(k))
		if err != nil {
			common.Log.Panicf("item.Key error: %v", err)
		}

		utxosInT2[utxoId] = true
		for _, sat := range value.Sats {
			if sat.Sat < 0 { // 不统计负数铭文
				continue
			}
			satsInT2[uint64(sat.Sat)] = utxoId
		}
		return nil
	})
	common.Log.Infof("%s table takes %v", DB_PREFIX_UTXO, time.Since(startTime2))
	common.Log.Infof("2: utxo %d, sats %d", len(utxosInT2), len(satsInT2))

	bs := NewBuckStore(p.db)
	lastkey := bs.GetLastKey() // 仅仅是正数铭文id
	var buckmap map[int64]*BuckValue
	getbuck := func() {
		//defer wg.Done()
		startTime2 := time.Now()
		buckmap = bs.GetAll()
		common.Log.Infof("%s table takes %v", DB_PREFIX_BUCK, time.Since(startTime2))
		common.Log.Infof("3: nfts %d", len(buckmap))
	}
	getbuck()

	//wg.Wait()
	common.Log.Infof("nft count: %d %d %d", p.status.Count+p.status.CurseCount-uint64(len(p.nftAdded)), len(nftsInT1), len(buckmap))

	wrongAddress := make([]uint64, 0)
	wrongUtxo1 := make([]uint64, 0)
	wrongUtxo2 := make([]uint64, 0)

	//wg.Add(2)
	baseDB.View(func(txn common.ReadBatch) error {
		//defer wg.Done()
		startTime2 = time.Now()
		for address := range addressesInT1 {
			key := db.GetAddressIdKey(address)
			_, err := txn.Get(key)
			if err != nil {
				wrongAddress = append(wrongAddress, address)
			}
		}
		common.Log.Infof("check addressesInT1 in baseDB takes %v", time.Since(startTime2))
		return nil
	})

	// 耗时很长，90w的高度，基本要10-20分钟
	// baseDB.View(func(txn common.ReadBatch) error {
	// 	//defer wg.Done()
	// 	startTime2 = time.Now()
	// 	for utxo := range utxosInT2 {
	// 		key := db.GetUtxoIdKey(utxo)
	// 		_, err := txn.Get(key)
	// 		if err != nil {
	// 			wrongUtxo2 = append(wrongUtxo2, utxo)
	// 		}
	// 	}
	// 	common.Log.Infof("check utxosInT2 in baseDB takes %v", time.Since(startTime2))
	// 	return nil
	// })

	//wg.Wait()
	common.Log.Infof("check in baseDB completed")

	wrongIds := make([]int64, 0)
	wrongSats := make([]int64, 0)
	for id, v := range buckmap {
		_, ok := nftsInT1[id]
		if !ok {
			wrongIds = append(wrongIds, id)
		}
		if v.Sat < 0 {
			continue
		}
		_, ok = satsInT1[uint64(v.Sat)]
		if !ok {
			wrongSats = append(wrongSats, v.Sat)
		}
	}

	common.Log.Infof("wrong address %d", len(wrongAddress))
	common.Log.Infof("wrong id %d", len(wrongIds))
	common.Log.Infof("wrong sat %d", len(wrongSats))
	common.Log.Infof("wrong utxo1 %d, utxo2 %d", len(wrongUtxo1), len(wrongUtxo2))
	for i, value := range wrongAddress {
		if i > 10 {
			break
		}
		common.Log.Infof("wrong address %d: %d", i, value)
	}
	for i, value := range wrongIds {
		if i > 10 {
			break
		}
		base := p.getNftBaseWithId(value)
		common.Log.Infof("wrong id %d: %d %v", i, value, base)
	}
	for i, value := range wrongSats {
		if i > 10 {
			break
		}
		common.Log.Infof("wrong sat %d: %d", i, value)
	}
	for i, value := range wrongUtxo1 {
		if i > 10 {
			break
		}
		common.Log.Infof("wrong utxo1 %d: %d", i, value)
	}
	for i, value := range wrongUtxo2 {
		if i > 10 {
			break
		}
		common.Log.Infof("wrong utxo2 %d: %d", i, value)
	}

	result := true
	if len(wrongAddress) != 0 || len(wrongIds) != 0 || len(wrongSats) != 0 || len(wrongUtxo1) != 0 {
		common.Log.Error("data wrong")
		result = false
	}

	count := p.status.Count + p.status.CurseCount - uint64(len(p.nftAdded))
	if count != uint64(len(nftsInT1)) || p.status.Count != uint64(lastkey+1) {
		common.Log.Errorf("nft count different %d %d %d", count, len(nftsInT1), uint64(lastkey+1))
		result = false
	}

	common.Log.Infof("utxos not in table %s", DB_PREFIX_UTXO)
	utxos1 := findDifferentItems(utxosInT1, utxosInT2)
	if len(utxos1) > 0 {
		p.printfUtxos(utxos1, baseDB)
		common.Log.Errorf("utxo1 wrong %d %v", len(utxos1), utxos1)
		result = false
	}

	common.Log.Infof("utxos not in table %s", DB_PREFIX_SAT)
	utxos2 := findDifferentItems(utxosInT2, utxosInT1)
	if len(utxos2) > 0 {
		p.printfUtxos(utxos2, baseDB)
		common.Log.Errorf("utxo2 wrong %d", len(utxos2))
		result = false
	}

	// needReCheck := false
	common.Log.Infof("sats not in table %s", DB_PREFIX_UTXO)
	sats1 := findDifferentItemsV2(satsInT1, satsInT2)
	if len(sats1) > 0 {
		common.Log.Errorf("sat1 wrong %d %v", len(sats1), sats1)
		result = false
	}

	common.Log.Infof("sats not in table %s", DB_PREFIX_SAT)
	sats2 := findDifferentItemsV2(satsInT2, satsInT1)
	if len(sats2) > 0 {
		common.Log.Errorf("sats2 wrong %d", len(sats2))
		result = false
	}

	// 1. 每个utxoId都存在baseDB中
	// 2. 两个表格中的数据相互对应: name，sat
	// 3. name的总数跟stats中一致
	if result {
		common.Log.Infof("nft DB checked successfully, %v", time.Since(startTime))
	}

	return result
}

func findDifferentItemsV2(map1, map2 map[uint64]uint64) map[uint64]uint64 {
	differentItems := make(map[uint64]uint64)
	for key, v := range map1 {
		if _, exists := map2[key]; !exists {
			differentItems[key] = v
		}
	}

	return differentItems
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
func (b *NftIndexer) printfUtxos(utxos map[uint64]bool, ldb common.KVDB) map[uint64]string {
	result := make(map[uint64]string)
	ldb.BatchRead([]byte(common.DB_KEY_UTXO), false, func(k, v []byte) error {

		var value common.UtxoValueInDB
		err := db.DecodeBytesWithProto3(v, &value)
		if err != nil {
			common.Log.Errorf("item.Value error: %v", err)
			return nil
		}

		// 用于打印不存在table中的utxo
		if _, ok := utxos[value.UtxoId]; ok {

			str, err := db.GetUtxoByDBKey(k)
			if err == nil {
				common.Log.Infof("%x %s %d", value.UtxoId, str, value.Value)
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
func (b *NftIndexer) deleteSats(sats map[uint64]uint64) {
	wb := b.db.NewWriteBatch()
	defer wb.Close()

	for sat := range sats {
		key := GetSatKey(int64(sat))
		err := wb.Delete([]byte(key))
		if err != nil {
			common.Log.Errorf("NftIndexer.deleteSats-> Error deleting db: %v\n", err)
		} else {
			common.Log.Infof("sat deled: %d", sat)
		}
	}

	err := wb.Flush()
	if err != nil {
		common.Log.Panicf("NftIndexer.deleteSats-> Error satwb flushing writes to db %v", err)
	}
}
