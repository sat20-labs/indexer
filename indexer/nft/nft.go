package nft

import (
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/base"
	indexer "github.com/sat20-labs/indexer/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

type SatInfo struct {
	AddressId uint64
	Index     int
	UtxoId    uint64
}

// 所有nft的记录
// 以后ns和ordx模块，数据变大，导致加载、跑数据等太慢，需要按照这个模块的方式来修改优化。
type NftIndexer struct {
	db       *badger.DB
	status   *common.NftStatus
	bEnabled bool
	reCheck  bool

	baseIndexer *base.BaseIndexer
	mutex       sync.RWMutex

	// realtime buffer
	satTree *indexer.SatRBTree        // key: sat, 用于范围搜索
	utxoMap map[uint64]map[int64]bool // utxo->sats  确保utxo中包含的所有nft都列在这里
	satMap  map[int64]*SatInfo        // sat->utxo

	// 状态变迁，做为buffer使用时注意数据可能过时
	nftAdded  []*common.Nft // 保持顺序
	utxoDeled []uint64
}

func NewNftIndexer(db *badger.DB) *NftIndexer {
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
}

func (p *NftIndexer) reset() {
	p.satTree = indexer.NewSatRBTress()
	p.utxoMap = make(map[uint64]map[int64]bool)
	p.satMap = make(map[int64]*SatInfo)
	p.nftAdded = make([]*common.Nft, 0)
	p.utxoDeled = make([]uint64, 0)
}

func (p *NftIndexer) Clone() *NftIndexer {
	newInst := NewNftIndexer(p.db)
	newInst.utxoMap = make(map[uint64]map[int64]bool)
	for k, v := range p.utxoMap {
		newInst.utxoMap[k] = v
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

	another.satTree.View(func(k int64, v interface{}) error {
		p.satTree.Delete(k)
		return nil
	})

	for k := range another.utxoMap {
		delete(p.utxoMap, k)
	}
	for k := range another.satMap {
		delete(p.satMap, k)
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
	// addressId, _ := common.GetAddressIdFromDB(baseDB, "bc1pxvkx8ls7kfpx9kt423lqwv6q7hcshwywfnywnxxfd7udyctk93wqe8x9gh")
	// utxoId, rngs, _ := base_indexer.ShareBaseIndexer.GetOrdinalsForUTXO("4e37618704dbba19f9c26e4eaf99be03bd66e41909f9906d66e886993e32289f:0")
	// reg := &NameRegister{
	// 	InscriptionId: "4e37618704dbba19f9c26e4eaf99be03bd66e41909f9906d66e886993e32289fi0",
	// 	BlockHeight:   827307,
	// 	Sat:  rngs[0].Start,
	// 	Name: "pearl",
	// 	OwnerAddressId: addressId,
	// 	UtxoId:         utxoId,
	// }
	// p.NameRegister(reg)
	// p.UpdateDB()

	// p.db.Update(func(txn *badger.Txn) error {
	// 	key := GetNameKey("pearl")
	// 	err := txn.Delete([]byte(key))
	// 	if err != nil {
	// 		common.Log.Panicf("can't delete %s, %v", key, err)
	// 	}
	// 	return nil
	// })
}

// 每个NFT Mint都调用
func (p *NftIndexer) NftMint(nft *common.Nft) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if nft.Base.Sat < 0 {
		// unbound nft
		nft.Base.Sat = -int64(p.status.Unbound)
		p.status.Unbound++
	}

	nft.Base.Id = int64(p.status.Count)
	p.status.Count++
	p.nftAdded = append(p.nftAdded, nft)

	// 确保该nft已经加入utxomap中
	p.addSatToUtxo(nft.UtxoId, nft.Base.Sat)
	p.satMap[(nft.Base.Sat)] = &SatInfo{AddressId: nft.OwnerAddressId, Index: 0, UtxoId: nft.UtxoId}
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
	p.db.View(func(txn *badger.Txn) error {
		for _, tx := range block.Transactions[1:] {
			hasAsset := false
			for _, input := range tx.Inputs {
				sats := p.getBindingSatsWithUtxo(input.UtxoId, txn)
				for sat := range sats {

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
		return nil
	})

	common.Log.Infof("NftIndexer.UpdateTransfer loop %d in %v", len(block.Transactions), time.Since(startTime))
}

func (p *NftIndexer) innerUpdateTransfer2(output *common.Output) {
	bUpdated := false
	newUtxo := common.GetUtxoId(output)

	sats := make(map[int64]bool, 0)
	for _, r := range output.Ordinals {
		values := p.satTree.FindSatValuesWithRange(r)
		for k := range values {
			sats[k] = true
			for i, address := range output.Address.Addresses {
				newAddress := p.baseIndexer.GetAddressId(address)
				p.satMap[k] = &SatInfo{AddressId: newAddress, Index: i, UtxoId: newUtxo}
			}

			bUpdated = true
		}
	}

	if bUpdated {
		// add output utxo
		p.utxoMap[newUtxo] = sats
	}
}

func (p *NftIndexer) addSatToUtxo(utxoId uint64, sat int64) {
	p.db.View(func(txn *badger.Txn) error {
		p.getBindingSatsWithUtxo(utxoId, txn)
		return nil
	})

	satmap, ok := p.utxoMap[utxoId]
	if ok {
		satmap[sat] = true
	} else {
		satmap = make(map[int64]bool)
		satmap[sat] = true
		p.utxoMap[utxoId] = satmap
	}
}

// fast
func (p *NftIndexer) getBindingSatsWithUtxo(utxoId uint64, txn *badger.Txn) map[int64]bool {
	sats, ok := p.utxoMap[utxoId]
	if ok {
		return sats
	}

	value := NftsInUtxo{}
	err := loadUtxoValueFromDB(utxoId, &value, txn)
	if err != nil {
		//common.Log.Infof("loadUtxoValueFromDB %d failed. %v", utxoId, err)
		return nil
	}
	result := make(map[int64]bool)
	for _, sat := range value.Sats {
		result[sat] = true
	}
	p.utxoMap[utxoId] = result
	return result
}

func (p *NftIndexer) refreshNft(nft *common.Nft) {
	satinfo, ok := p.satMap[nft.Base.Sat]
	if ok {
		nft.OwnerAddressId = satinfo.AddressId
		nft.UtxoId = satinfo.UtxoId
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

func (p *NftIndexer) getNftInBuffer4(sat int64) *common.Nft {
	for _, nft := range p.nftAdded {
		if nft.Base.Sat == sat {
			p.refreshNft(nft)
			return nft
		}
	}
	return nil
}

// sat -> nfts
func (p *NftIndexer) prefetchNftsFromDB() map[int64]*common.NftsInSat {
	nftmap := make(map[int64]*common.NftsInSat)

	p.db.View(func(txn *badger.Txn) error {

		for sat, info := range p.satMap {
			_, ok := nftmap[sat]
			if !ok {
				key := GetSatKey(sat)
				oldvalue := common.NftsInSat{}
				// err := db.GetValueFromDB([]byte(key), txn, &oldvalue)
				err := db.GetValueFromDBWithProto3([]byte(key), txn, &oldvalue)
				if err == nil {
					oldvalue.OwnerAddressId = info.AddressId
					oldvalue.UtxoId = info.UtxoId
					nftmap[sat] = &oldvalue
				} //else {
				// 在p.nftAdded中，稍等再加载
				//}
			}
		}

		for _, nft := range p.nftAdded {
			value, ok := nftmap[nft.Base.Sat]
			base := nft.Base
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
				} else {
					value.OwnerAddressId = nft.OwnerAddressId
					value.UtxoId = nft.UtxoId
				}
				nftmap[base.Sat] = value
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

	nftmap := p.prefetchNftsFromDB()
	buckDB := NewBuckStore(p.db)
	buckNfts := make(map[int64]*BuckValue)

	wb := p.db.NewWriteBatch()
	defer wb.Cancel()

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
		utxoValue := NftsInUtxo{Sats: mapToVector(sats)}
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
	p.satTree = indexer.NewSatRBTress()
	p.nftAdded = make([]*common.Nft, 0)
	p.utxoMap = make(map[uint64]map[int64]bool)
	p.utxoDeled = make([]uint64, 0)
	p.satMap = make(map[int64]*SatInfo)

	common.Log.Infof("NftIndexer->UpdateDB takes %v", time.Since(startTime))
}

// 耗时很长。仅用于在数据编译完成时验证数据，或者测试时验证数据。
func (p *NftIndexer) CheckSelf(baseDB *badger.DB) bool {

	common.Log.Info("NftIndexer->checkSelf ... ")

	startTime := time.Now()
	common.Log.Infof("stats: %v", p.status)

	// var wg sync.WaitGroup
	// wg.Add(3)

	addressesInT1 := make(map[uint64]bool, 0)
	utxosInT1 := make(map[uint64]bool, 0)
	satsInT1 := make(map[uint64]uint64, 0)
	nftsInT1 := make(map[int64]bool, 0)
	p.db.View(func(txn *badger.Txn) error {
		//defer wg.Done()

		var err error
		prefix := []byte(DB_PREFIX_NFT)
		itr := txn.NewIterator(badger.DefaultIteratorOptions)
		defer itr.Close()

		startTime2 := time.Now()
		common.Log.Infof("calculating in %s table ...", DB_PREFIX_NFT)

		for itr.Seek([]byte(prefix)); itr.ValidForPrefix([]byte(prefix)); itr.Next() {
			item := itr.Item()
			if item.IsDeletedOrExpired() {
				continue
			}
			var value common.NftsInSat
			err = item.Value(func(data []byte) error {
				// return db.DecodeBytes(data, &value)
				return db.DecodeBytesWithProto3(data, &value)
			})
			if err != nil {
				common.Log.Panicf("item.Value error: %v", err)
			}

			addressesInT1[value.OwnerAddressId] = true
			utxosInT1[value.UtxoId] = true
			satsInT1[uint64(value.Sat)] = value.UtxoId
			for _, nft := range value.Nfts {
				nftsInT1[nft.Id] = true
			}
		}

		common.Log.Infof("%s table takes %v", DB_PREFIX_NFT, time.Since(startTime2))
		common.Log.Infof("1: address %d, utxo %d, sats %d, nfts %d", len(addressesInT1), len(utxosInT1), len(satsInT1), len(nftsInT1))
		return nil
	})

	// utxo的数据涉及到delete操作，但是badger的delete操作有隐藏的bug，需要检查下该utxo是否存在
	utxosInT2 := make(map[uint64]bool)
	satsInT2 := make(map[uint64]uint64)
	p.db.View(func(txn *badger.Txn) error {
		//defer wg.Done()

		var err error
		prefix := []byte(DB_PREFIX_UTXO)
		itr := txn.NewIterator(badger.DefaultIteratorOptions)
		defer itr.Close()

		startTime2 := time.Now()
		common.Log.Infof("calculating in %s table ...", DB_PREFIX_UTXO)

		for itr.Seek([]byte(prefix)); itr.ValidForPrefix([]byte(prefix)); itr.Next() {
			item := itr.Item()
			if item.IsDeletedOrExpired() {
				continue
			}
			var value NftsInUtxo
			err = item.Value(func(data []byte) error {
				// return db.DecodeBytes(data, &value)
				return db.DecodeBytesWithProto3(data, &value)
			})
			if err != nil {
				common.Log.Panicf("item.Value error: %v", err)
			}

			utxoId, err := ParseUtxoKey(string(item.Key()))
			if err != nil {
				common.Log.Panicf("item.Key error: %v", err)
			}

			utxosInT2[utxoId] = true
			for _, sat := range value.Sats {
				satsInT2[uint64(sat)] = utxoId
			}
		}

		common.Log.Infof("%s table takes %v", DB_PREFIX_UTXO, time.Since(startTime2))
		common.Log.Infof("2: utxo %d, sats %d", len(utxosInT2), len(satsInT2))
		return nil
	})

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
	common.Log.Infof("nft count: %d %d %d", p.status.Count, len(nftsInT1), lastkey+1)

	wrongAddress := make([]uint64, 0)
	wrongUtxo1 := make([]uint64, 0)
	wrongUtxo2 := make([]uint64, 0)

	//wg.Add(2)
	baseDB.View(func(txn *badger.Txn) error {
		//defer wg.Done()
		startTime2 := time.Now()
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
	baseDB.View(func(txn *badger.Txn) error {
		//defer wg.Done()
		startTime2 := time.Now()
		// 这些utxo很可能是因为delete操作的bug，遗留了下来，直接从数据库中删除是最好的办法
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
		_, ok = satsInT1[uint64(v.Sat)]
		if !ok {
			wrongSats = append(wrongSats, v.Sat)
		}
	}

	common.Log.Infof("wrong address %d", len(wrongAddress))
	common.Log.Infof("wrong name %d", len(wrongIds))
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
		common.Log.Infof("wrong name %d: %d", i, value)
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
		common.Log.Errorf("name count different %d %d %d", count, len(nftsInT1), uint64(lastkey+1))
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

	needReCheck := false
	common.Log.Infof("sats not in table %s", DB_PREFIX_NFT)
	sats1 := findDifferentItemsV2(satsInT1, satsInT2)
	if len(sats1) > 0 {
		common.Log.Errorf("sat1 wrong %d %v", len(sats1), sats1)
		// 因为badger数据库的bug，在DB_PREFIX_NFT中删除的数据可能还会出现
		deleteSats := make(map[uint64]uint64)
		baseDB.View(func(txn *badger.Txn) error {
			for sat, utxoId := range sats1 {
				_, err := txn.Get(db.GetUtxoIdKey(utxoId))
				if err != nil {
					deleteSats[sat] = utxoId
				}
			}
			return nil
		})
		if len(deleteSats) == len(sats1) {
			p.deleteSats(deleteSats)
			needReCheck = true
		} else {
			result = false
		}
	}

	common.Log.Infof("sats not in table %s", DB_PREFIX_UTXO)
	sats2 := findDifferentItemsV2(satsInT2, satsInT1)
	if len(sats2) > 0 {
		common.Log.Errorf("sats2 wrong %d", len(sats2))
		// 因为badger数据库的bug，在DB_PREFIX_NFT中删除的数据可能还会出现
		deleteSats := make(map[uint64]uint64)
		baseDB.View(func(txn *badger.Txn) error {
			for sat, utxoId := range sats2 {
				_, err := txn.Get(db.GetUtxoIdKey(utxoId))
				if err != nil {
					deleteSats[sat] = utxoId
				}
			}
			return nil
		})
		if len(deleteSats) == len(sats2) {
			p.deleteSats(deleteSats)
			needReCheck = true
		} else {
			result = false
		}
	}

	if needReCheck && !p.reCheck {
		p.reCheck = true
		return p.CheckSelf(baseDB)
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
func (b *NftIndexer) printfUtxos(utxos map[uint64]bool, ldb *badger.DB) map[uint64]string {
	result := make(map[uint64]string)
	ldb.View(func(txn *badger.Txn) error {
		var err error
		prefix := []byte(common.DB_KEY_UTXO)
		itr := txn.NewIterator(badger.DefaultIteratorOptions)
		defer itr.Close()

		for itr.Seek([]byte(prefix)); itr.ValidForPrefix([]byte(prefix)); itr.Next() {
			item := itr.Item()
			if item.IsDeletedOrExpired() {
				continue
			}
			var value common.UtxoValueInDB
			err = item.Value(func(data []byte) error {
				return db.DecodeBytesWithProto3(data, &value)
			})
			if err != nil {
				common.Log.Errorf("item.Value error: %v", err)
				continue
			}

			// 用于打印不存在table中的utxo
			if _, ok := utxos[value.UtxoId]; ok {
				key := item.Key()
				str, err := db.GetUtxoByDBKey(key)
				if err == nil {
					common.Log.Infof("%x %s %d", value.UtxoId, str, common.GetOrdinalsSize(value.Ordinals))
					result[value.UtxoId] = str
				}

				delete(utxos, value.UtxoId)
				if len(utxos) == 0 {
					return nil
				}
			}
		}

		return nil
	})

	return result
}

// only for test
func (b *NftIndexer) deleteSats(sats map[uint64]uint64) {
	wb := b.db.NewWriteBatch()
	defer wb.Cancel()

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
