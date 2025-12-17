package nft

import (
	"sort"
	"sync"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/base"
	indexer "github.com/sat20-labs/indexer/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"

	ordCommon "github.com/sat20-labs/indexer/indexer/ord/common"
)

type SatInfo struct { // 该聪的实时位置
	AddressId uint64 
	UtxoId    uint64
	// Offset    int64 
	Nfts      *common.NftsInSat // 延迟加载
}


// 所有nft的记录
// 以后ns和ordx模块，数据变大，导致加载、跑数据等太慢，需要按照这个模块的方式来修改优化。
type NftIndexer struct {
	db       common.KVDB
	status   *common.NftStatus
	bEnabled bool
	disabledSats map[int64]bool // 所有disabled的satoshi

	baseIndexer *base.BaseIndexer
	mutex       sync.RWMutex

	// realtime buffer
	satTree *indexer.SatRBTree // key: sat, 用于范围搜索
	utxoMap map[uint64][]int64 // utxo->sats  确保utxo中包含的所有nft都列在这里
	satMap  map[int64]*SatInfo // sat->utxo

	// 状态变迁，做为buffer使用时注意数据可能过时
	nftAdded  []*common.Nft // 保持顺序
	utxoDeled []uint64
}

func NewNftIndexer(db common.KVDB) *NftIndexer {
	ns := &NftIndexer{
		db:        db,
		bEnabled:  true,
		status:    nil,
		utxoMap:   nil,
		satMap:    nil,
		utxoDeled: nil,
	}
	ns.reset()
	return ns
}

// 只能被调用一次
func (p *NftIndexer) Init(baseIndexer *base.BaseIndexer) {
	p.baseIndexer = baseIndexer
	p.status = initStatusFromDB(p.db)
	p.disabledSats = loadAllDisalbedSatsFromDB(p.db)
}

func (p *NftIndexer) reset() {
	//p.disabledSats = make(map[int64]bool)
	p.satTree = indexer.NewSatRBTress()
	p.utxoMap = make(map[uint64][]int64)
	p.satMap = make(map[int64]*SatInfo)
	p.nftAdded = make([]*common.Nft, 0)
	p.utxoDeled = make([]uint64, 0)
}

func (p *NftIndexer) Clone() *NftIndexer {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	newInst := NewNftIndexer(p.db)

	newInst.disabledSats = p.disabledSats // 仅在rpc中使用
	newInst.utxoMap = make(map[uint64][]int64)
	for k, v := range p.utxoMap {
		nv := make([]int64, len(v))
		copy(nv, v)
		newInst.utxoMap[k] = nv
	}
	newInst.satMap = make(map[int64]*SatInfo)
	for k, v := range p.satMap {
		newInst.satMap[k] = v
	}

	newInst.nftAdded = make([]*common.Nft, len(p.nftAdded))
	copy(newInst.nftAdded, p.nftAdded)

	newInst.utxoDeled = make([]uint64, len(p.utxoDeled))
	copy(newInst.utxoDeled, p.utxoDeled)

	newInst.status = p.status.Clone()

	return newInst
}

func (p *NftIndexer) Subtract(another *NftIndexer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 待观察
	// another.satTree.View(func(k int64, v interface{}) error {
	// 	p.satTree.Delete(k)
	// 	return nil
	// })

	for k := range another.utxoMap {
		delete(p.utxoMap, k)
	}
	for k, v := range another.satMap {
		nv, ok := p.satMap[k]
		if ok {
			if nv.UtxoId == v.UtxoId {
				// 没有变化就删除
				delete(p.satMap, k)
			}
		}
	}
	// p.nftAdded = p.nftAdded[len(another.nftAdded):]
	p.nftAdded = append([]*common.Nft(nil), p.nftAdded[len(another.nftAdded):]...)
	// p.utxoDeled = p.utxoDeled[len(another.utxoDeled):]
	p.utxoDeled = append([]uint64(nil), p.utxoDeled[len(another.utxoDeled):]...)
}

// func (p *NftIndexer) IsEnabled() bool {
// 	return p.bEnabled
// }

func (p *NftIndexer) GetBaseIndexer() *base.BaseIndexer {
	return p.baseIndexer
}

func (p *NftIndexer) Repair() {

	fixingUtxoMap := make(map[uint64][]int64)
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
			if sat < 0 {
				nv := make([]int64, 0)
				for _, sat := range value.Sats {
					if sat >= 0 {
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

// 每个NFT Mint都调用
func (p *NftIndexer) NftMint(nft *common.Nft) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if nft.Base.Sat >= 0 && nft.Base.CurseType == 0{
		// 检查是否同一个聪上有多个铸造
		nftsInSat := p.getNftsWithSat(nft.Base.Sat)
		if nftsInSat != nil {
			for _, n := range nftsInSat.Nfts {
				if n.CurseType == 0 {
					// 已经存在非cursed的铭文，后面的铭文都是cursed
					nft.Base.CurseType = int32(ordCommon.Reinscription)
					common.Log.Infof("%s is reinscription in sat %d, the first non-cursed inscription %s",
					nft.Base.InscriptionId, nft.Base.Sat, n.InscriptionId)
					break
				}
			}
		}
	}

	if nft.Base.CurseType != 0 && nft.Base.BlockHeight >= int32(common.Jubilee_Height) {
		nft.Base.CurseType = 0
	}

	if nft.Base.CurseType != 0 {
		p.status.CurseCount++
		nft.Base.Id = -int64(p.status.CurseCount) // 从-1开始
	} else {
		nft.Base.Id = int64(p.status.Count) // 从0开始
		p.status.Count++
	}
	p.nftAdded = append(p.nftAdded, nft)

	if nft.Base.Sat < 0 {
		// mainnet: c1e0db6368a43f5589352ed44aa1ff9af33410e4a9fd9be0f6ac42d9e4117151
		// unbound nft，负数铭文，没有绑定任何聪，也不在哪个utxo中，也没有地址，仅保存数据
		// 在Jubilee之前属于cursed铭文，Jubilee之后，正常编号
		p.status.Unbound++
		nft.Base.Sat = -int64(p.status.Unbound) // 从-1开始
		return
	}

	// 确保该nft已经加入utxomap中
	p.addSatToUtxo(nft.UtxoId, nft.Base.Sat)
	// 暂时没有加载该sat对应的所有的nft，只更新了该聪的位置
	p.satMap[(nft.Base.Sat)] = &SatInfo{
		AddressId: nft.OwnerAddressId,
		UtxoId: nft.UtxoId,
		//Offset: nft.Offset,
	}
	p.satTree.Put(nft.Base.Sat, true)

	//action := TransferAction{UtxoId: inputUtxo, Sats: v, Action: -1}
	//p.transferActionList = append(p.transferActionList, &action)

	//action2 := TransferAction{UtxoId: nft.UtxoId, AddressId: nft.OwnerAddressId, Sats: sats, Action: 1}
	//p.transferActionList = append(p.transferActionList, &action2)
}

// Mint和Transfer需要仔细协调，确保新增加的nft可以正确被转移
func (p *NftIndexer) UpdateTransfer(block *common.Block) {
	if !p.bEnabled {
		return
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	startTime := time.Now()
	p.db.View(func(txn common.ReadBatch) error {
		type pair struct {
			key   string
			value uint64
		}
		utxos := make([]*pair, 0)
		for _, tx := range block.Transactions[1:] {
			for _, input := range tx.Inputs {
				_, ok := p.utxoMap[input.UtxoId]
				if ok {
					continue
				}
				utxos = append(utxos, &pair{
					key:   GetUtxoKey(input.UtxoId),
					value: input.UtxoId,
				})
			}
		}
		// pebble数据库的优化手段: 尽可能将随机读变成按照key的顺序读
		sort.Slice(utxos, func(i, j int) bool {
			return utxos[i].key < utxos[j].key
		})
		for _, v := range utxos {
			value := NftsInUtxo{}
			err := db.GetValueFromTxnWithProto3([]byte(v.key), txn, &value)
			if err != nil {
				continue
			}
			p.utxoMap[v.value] = value.Sats
		}
		return nil
	})

	for _, tx := range block.Transactions[1:] {
		hasAsset := false
		for _, input := range tx.Inputs {
			sats := p.utxoMap[input.UtxoId]
			for _, sat := range sats {
				p.satTree.Put(sat, true)
			}
			
			if len(sats) > 0 {
				hasAsset = true
				delete(p.utxoMap, input.UtxoId)
				p.utxoDeled = append(p.utxoDeled, input.UtxoId)
			}
		}

		if hasAsset {
			for _, output := range tx.Outputs {
				p.innerUpdateTransfer2(output)
			}
		}
	}

	// 按顺序是最后一块，要放最后，保持顺序很重要
	tx := block.Transactions[0]
	for _, output := range tx.Outputs {
		p.innerUpdateTransfer2(output)
	}

	p.satTree = indexer.NewSatRBTress()
	common.Log.Infof("NftIndexer.UpdateTransfer loop %d in %v", len(block.Transactions), time.Since(startTime))
}

func (p *NftIndexer) innerUpdateTransfer2(output *common.Output) {
	bUpdated := false
	newUtxo := common.GetUtxoId(output)
	newAddress := p.baseIndexer.GetAddressId(output.Address.Addresses[0])

	sats := make([]int64, 0)
	for _, r := range output.Ordinals {
		values := p.satTree.FindSatValuesWithRange(r)
		for k := range values {
			sats = append(sats, k)	
			// offset := common.GetSatOffset(output.Ordinals, k) 对大的btc，双重循环就超级慢了
			p.satMap[k] = &SatInfo{AddressId: newAddress, UtxoId: newUtxo, /*Offset: offset*/}
			bUpdated = true
		}
	}

	if bUpdated {
		// add output utxo
		p.utxoMap[newUtxo] = sats
	}
}

func (p *NftIndexer) addSatToUtxo(utxoId uint64, sat int64) {
	p.db.View(func(txn common.ReadBatch) error {
		p.getBindingSatsWithUtxo(utxoId, txn)
		return nil
	})
	satmap := p.utxoMap[utxoId]
	p.utxoMap[utxoId] = append(satmap, sat)
}

// fast
func (p *NftIndexer) getBindingSatsWithUtxo(utxoId uint64, txn common.ReadBatch) []int64 {
	sats, ok := p.utxoMap[utxoId]
	if ok {
		return sats
	}

	value := NftsInUtxo{}
	err := loadUtxoValueFromTxn(utxoId, &value, txn)
	if err != nil {
		//common.Log.Infof("loadUtxoValueFromDB %d failed. %v", utxoId, err)
		return nil
	}

	p.utxoMap[utxoId] = value.Sats
	return value.Sats
}

func (p *NftIndexer) refreshNft(nft *common.Nft) {
	satinfo, ok := p.satMap[nft.Base.Sat]
	if ok {
		nft.OwnerAddressId = satinfo.AddressId
		nft.UtxoId = satinfo.UtxoId
		//nft.Offset = satinfo.Offset
	}
}

// 注意
func (p *NftIndexer) getNftInBuffer(id int64) *common.Nft {
	for _, nft := range p.nftAdded {
		if nft.Base.Id == id {
			p.refreshNft(nft)
			return nft
		}
	}
	return nil
}

func (p *NftIndexer) getNftInBuffer2(inscriptionId string) *common.Nft {
	for _, nft := range p.nftAdded {
		if nft.Base.InscriptionId == inscriptionId {
			p.refreshNft(nft)
			return nft
		}
	}
	return nil
}

func (p *NftIndexer) getNftInBuffer4(sat int64) []*common.Nft {
	result := make([]*common.Nft, 0)
	for _, nft := range p.nftAdded {
		if nft.Base.Sat == sat {
			p.refreshNft(nft)
			result = append(result, nft)
		}
	}
	return result
}

// sat -> nfts
func (p *NftIndexer) prefetchNftsFromDB() map[int64]*common.NftsInSat {
	nftmap := make(map[int64]*common.NftsInSat)

	p.db.View(func(txn common.ReadBatch) error {

		type pair struct {
			key   string
			value *SatInfo
			sat   int64
		}

		// 为satMap加载所有的nft数据
		loadingSats := make([]*pair, 0)
		for sat, info := range p.satMap {
			key := GetSatKey(sat)
			loadingSats = append(loadingSats, &pair{
				key:   key,
				value: info,
				sat:   sat,
			})
		}
		// pebble数据库的优化手段: 尽可能将随机读变成按照key的顺序读
		sort.Slice(loadingSats, func(i, j int) bool {
			return loadingSats[i].key < loadingSats[j].key
		})
		for _, v := range loadingSats {
			oldvalue := common.NftsInSat{}
			err := db.GetValueFromTxnWithProto3([]byte(v.key), txn, &oldvalue)
			if err == nil {
				info := v.value
				oldvalue.OwnerAddressId = info.AddressId
				oldvalue.UtxoId = info.UtxoId
				//oldvalue.Offset = info.Offset
				nftmap[v.sat] = &oldvalue
			} //else {
			// 在p.nftAdded中，稍等再加载
			//}
		}

		for _, nft := range p.nftAdded {
			base := nft.Base
			value, ok := nftmap[nft.Base.Sat]
			if ok {
				value.Nfts = append(value.Nfts, base)
			} else {
				value = &common.NftsInSat{}
				value.Nfts = []*common.InscribeBaseContent{base}
				value.Sat = base.Sat
				satInfo, ok := p.satMap[base.Sat]
				if ok {
					// updated
					value.OwnerAddressId = satInfo.AddressId
					value.UtxoId = satInfo.UtxoId
					//value.Offset = satInfo.Offset
				} else {
					value.OwnerAddressId = nft.OwnerAddressId
					value.UtxoId = nft.UtxoId
					//value.Offset = nft.Offset
				}
				nftmap[base.Sat] = value
			}
			if base.CurseType != 0 {
				value.CurseCount++
			}
		}

		return nil
	})

	return nftmap
}

// 跟base数据库同步
func (p *NftIndexer) UpdateDB() {
	//common.Log.Infof("NftIndexer->UpdateDB start...")
	startTime := time.Now()

	if !p.bEnabled {
		return
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()

	nftmap := p.prefetchNftsFromDB()
	buckDB := NewBuckStore(p.db)
	buckNfts := make(map[int64]*BuckValue)

	wb := p.db.NewWriteBatch()
	defer wb.Close()

	for _, nft := range p.nftAdded {
		key := GetInscriptionIdKey(nft.Base.InscriptionId)
		value := InscriptionInDB{Sat: nft.Base.Sat, Id: nft.Base.Id}
		err := db.SetDB([]byte(key), &value, wb)
		if err != nil {
			common.Log.Panicf("NftIndexer->UpdateDB Error setting %s in db %v", key, err)
		}

		key = GetInscriptionAddressKey(nft.Base.InscriptionAddress, nft.Base.Id)
		err = db.SetDB([]byte(key), nft.Base.Sat, wb)
		if err != nil {
			common.Log.Panicf("NftIndexer->UpdateDB Error setting %s in db %v", key, err)
		}

		buckNfts[nft.Base.Id] = &BuckValue{Sat: nft.Base.Sat}
	}

	// 处理nft的转移
	for sat, nft := range nftmap {
		key := GetSatKey(sat)
		err := db.SetDBWithProto3([]byte(key), nft, wb)
		//err := db.SetDB([]byte(key), nft, wb)
		if err != nil {
			common.Log.Panicf("NftIndexer->UpdateDB Error setting %s in db %v", key, err)
		}
	}

	for _, utxoId := range p.utxoDeled {
		utxokey := GetUtxoKey(utxoId)
		err := wb.Delete([]byte(utxokey))
		if err != nil {
			common.Log.Errorf("NftIndexer->UpdateDB Error delete %s in db %v", utxokey, err)
		}
	}

	for utxoId, sats := range p.utxoMap {
		utxokey := GetUtxoKey(utxoId)
		utxoValue := NftsInUtxo{Sats: sats}
		// err := db.SetDB([]byte(utxokey), &utxoValue, wb)
		err := db.SetDBWithProto3([]byte(utxokey), &utxoValue, wb)
		if err != nil {
			common.Log.Panicf("NftIndexer->UpdateDB Error setting %s in db %v", utxokey, err)
		}
	}

	err := db.SetDB([]byte(NFT_STATUS_KEY), p.status, wb)
	if err != nil {
		common.Log.Panicf("NftIndexer->UpdateDB Error setting in db %v", err)
	}

	err = wb.Flush()
	if err != nil {
		common.Log.Panicf("NftIndexer->UpdateDB Error wb flushing writes to db %v", err)
	}

	err = buckDB.BatchPut(buckNfts)
	if err != nil {
		common.Log.Panicf("NftIndexer->UpdateDB BatchPut %v", err)
	}

	// reset memory buffer
	p.nftAdded = make([]*common.Nft, 0)
	p.utxoMap = make(map[uint64][]int64)
	p.utxoDeled = make([]uint64, 0)
	p.satMap = make(map[int64]*SatInfo)

	common.Log.Infof("NftIndexer->UpdateDB takes %v", time.Since(startTime))
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

	curseCount := 0
	p.db.BatchRead([]byte(DB_PREFIX_NFT), false, func(k, v []byte) error {
		//defer wg.Done()

		var value common.InscribeBaseContent
		err := db.DecodeBytesWithProto3(v, &value)
		if err != nil {
			common.Log.Panicf("item.Value error: %v", err)
		}
		if value.CurseType != 0 {
			curseCount++
			//common.Log.Infof("%d %s is cursed %d", value.Id, value.InscriptionId, value.CurseType)
		}

		return nil
	})
	common.Log.Infof("curse count %d", curseCount)


	addressesInT1 := make(map[uint64]bool, 0)
	utxosInT1 := make(map[uint64]bool, 0)
	satsInT1 := make(map[uint64]uint64, 0)
	nftsInT1 := make(map[int64]bool, 0)
	startTime2 := time.Now()
	common.Log.Infof("calculating in %s table ...", DB_PREFIX_NFT)
	p.db.BatchRead([]byte(DB_PREFIX_NFT), false, func(k, v []byte) error {
		//defer wg.Done()

		var value common.NftsInSat
		err := db.DecodeBytesWithProto3(v, &value)
		if err != nil {
			common.Log.Panicf("item.Value error: %v", err)
		}
		if value.Sat < 0 {
			// 负数铭文，没有绑定到任何聪的铭文，只统计nft数量
			for _, nft := range value.Nfts {
				nftsInT1[nft.Id] = true
			}
			return nil
		}

		addressesInT1[value.OwnerAddressId] = true
		utxosInT1[value.UtxoId] = true
		satsInT1[uint64(value.Sat)] = value.UtxoId
		for _, nft := range value.Nfts {
			nftsInT1[nft.Id] = true
		}

		return nil
	})
	common.Log.Infof("%s table takes %v", DB_PREFIX_NFT, time.Since(startTime2))
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
			if sat < 0 { // 不统计负数铭文
				continue
			}
			satsInT2[uint64(sat)] = utxoId
		}
		return nil
	})
	common.Log.Infof("%s table takes %v", DB_PREFIX_UTXO, time.Since(startTime2))
	common.Log.Infof("2: utxo %d, sats %d", len(utxosInT2), len(satsInT2))

	bs := NewBuckStore(p.db)
	lastkey := bs.GetLastKey()
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
	common.Log.Infof("nft count: %d %d %d", p.status.Count-uint64(len(p.nftAdded)), len(nftsInT1), lastkey+1)

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
	baseDB.View(func(txn common.ReadBatch) error {
		//defer wg.Done()
		startTime2 = time.Now()
		for utxo := range utxosInT2 {
			key := db.GetUtxoIdKey(utxo)
			_, err := txn.Get(key)
			if err != nil {
				wrongUtxo2 = append(wrongUtxo2, utxo)
			}
		}
		common.Log.Infof("check utxosInT2 in baseDB takes %v", time.Since(startTime2))
		return nil
	})

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
		common.Log.Infof("wrong id %d: %d", i, value)
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

	count := p.status.Count - uint64(len(p.nftAdded))
	if count != uint64(len(nftsInT1)) || count != uint64(lastkey+1) {
		common.Log.Errorf("nft count different %d %d %d", count, len(nftsInT1), uint64(lastkey+1))
		result = false
	}

	common.Log.Infof("utxos not in table %s", DB_PREFIX_NFT)
	utxos1 := findDifferentItems(utxosInT1, utxosInT2)
	if len(utxos1) > 0 {
		p.printfUtxos(utxos1, baseDB)
		common.Log.Errorf("utxo1 wrong %d %v", len(utxos1), utxos1)
		result = false
	}

	common.Log.Infof("utxos not in table %s", DB_PREFIX_UTXO)
	utxos2 := findDifferentItems(utxosInT2, utxosInT1)
	if len(utxos2) > 0 {
		p.printfUtxos(utxos2, baseDB)
		common.Log.Errorf("utxo2 wrong %d", len(utxos2))
		result = false
	}

	// needReCheck := false
	common.Log.Infof("sats not in table %s", DB_PREFIX_NFT)
	sats1 := findDifferentItemsV2(satsInT1, satsInT2)
	if len(sats1) > 0 {
		common.Log.Errorf("sat1 wrong %d %v", len(sats1), sats1)
		result = false
	}

	common.Log.Infof("sats not in table %s", DB_PREFIX_UTXO)
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
