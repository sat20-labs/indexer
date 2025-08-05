package ns

import (
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
	"github.com/sat20-labs/indexer/indexer/nft"
)

// 名字注册到几百万几千万后，这个模块的加载速度查找速度
type NameService struct {
	db         *badger.DB
	status     *common.NameServiceStatus
	nftIndexer *nft.NftIndexer

	// 用于快速查找，不释放，尽可能降低内存占用
	mutex sync.RWMutex

	// 缓存

	// 状态变迁
	nameAdded   []*NameRegister // 保持顺序
	updateAdded []*NameUpdate   // 保持顺序
}

func NewNameService(db *badger.DB) *NameService {
	ns := &NameService{
		db:     db,
		status: nil,
	}
	ns.reset()
	return ns
}

// 只能被调用一次
func (p *NameService) Init(nftIndexer *nft.NftIndexer) {
	p.nftIndexer = nftIndexer
	p.status = initStatusFromDB(p.db)
}

func (p *NameService) reset() {
	p.nameAdded = make([]*NameRegister, 0)
	p.updateAdded = make([]*NameUpdate, 0)
}

func (p *NameService) Clone() *NameService {
	newInst := NewNameService(p.db)

	newInst.nameAdded = make([]*NameRegister, len(p.nameAdded))
	copy(newInst.nameAdded, p.nameAdded)

	newInst.updateAdded = make([]*NameUpdate, len(p.updateAdded))
	copy(newInst.updateAdded, p.updateAdded)

	newInst.status = p.status.Clone()

	return newInst
}

func (p *NameService) Subtract(another *NameService) {
	p.nameAdded = p.nameAdded[len(another.nameAdded):]
	p.updateAdded = p.updateAdded[len(another.updateAdded):]
}

func (p *NameService) GetNftIndexer() *nft.NftIndexer {
	return p.nftIndexer
}

// 每个Register都调用
func (p *NameService) NameRegister(reg *NameRegister) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	reg.Id = int64(p.status.NameCount)
	p.status.NameCount++
	p.nameAdded = append(p.nameAdded, reg)
}

func (p *NameService) NameUpdate(update *NameUpdate) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.updateAdded = append(p.updateAdded, update)
}

// 使用utxoMap，效率高很多
func (p *NameService) UpdateTransfer(block *common.Block) {

}

func (p *NameService) getNameInBuffer(name string) *NameRegister {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	for _, reg := range p.nameAdded {
		if reg.Name == name {
			return reg
		}
	}
	return nil
}

// 跟base数据库同步
func (p *NameService) UpdateDB() {
	//common.Log.Infof("NameService->UpdateDB start...")
	startTime := time.Now()

	buckDB := NewBuckStore(p.db)
	buckNames := make(map[int]*BuckValue)

	wb := p.db.NewWriteBatch()
	defer wb.Cancel()

	// index: name
	for _, name := range p.nameAdded {
		key := GetNameKey(name.Name)
		value := NameValueInDB{
			NftId: name.Nft.Base.Id,
			Id:    name.Id,
			Sat:   name.Nft.Base.Sat,
			Name:  name.Name,
		}
		err := db.SetDBWithProto3([]byte(key), &value, wb)
		//err := db.SetDB([]byte(key), &value, wb)
		if err != nil {
			common.Log.Panicf("NameService->UpdateDB Error setting %s in db %v", key, err)
		}

		buckNames[int(name.Id)] = &BuckValue{Name: name.Name, Sat: name.Nft.Base.Sat}
	}

	for _, update := range p.updateAdded {
		for _, kv := range update.KVs {
			key := GetKVKey(update.Name, kv.Key)
			value := &common.KeyValueInDB{Value: kv.Value, InscriptionId: update.InscriptionId}
			err := db.SetDB([]byte(key), value, wb)
			if err != nil {
				common.Log.Panicf("NameService->UpdateDB Error setting %s in db %v", key, err)
			}
		}
	}

	err := db.SetDB([]byte(NS_STATUS_KEY), p.status, wb)
	if err != nil {
		common.Log.Panicf("NameService->UpdateDB Error setting in db %v", err)
	}

	err = wb.Flush()
	if err != nil {
		common.Log.Panicf("NameService->UpdateDB Error wb flushing writes to db %v", err)
	}

	// index: id
	err = buckDB.BatchPut(buckNames)
	if err != nil {
		common.Log.Panicf("NameService->UpdateDB BatchPut %v", err)
	}

	// reset memory buffer
	p.nameAdded = make([]*NameRegister, 0)
	p.updateAdded = make([]*NameUpdate, 0)

	common.Log.Infof("NameService->UpdateDB takes %v", time.Since(startTime))
}

// 耗时很长。仅用于在数据编译完成时验证数据，或者测试时验证数据。
func (p *NameService) CheckSelf(baseDB *badger.DB) bool {

	common.Log.Info("NameService->checkSelf ... ")

	startTime := time.Now()
	common.Log.Infof("stats: %v", p.status)

	// var wg sync.WaitGroup
	// wg.Add(2)

	nftIdInT1 := make(map[int64]bool, 0)
	namesInT1 := make(map[string]bool, 0)
	satsInT1 := make(map[int64]bool, 0)
	p.db.View(func(txn *badger.Txn) error {
		//defer wg.Done()

		var err error
		prefix := []byte(DB_PREFIX_NAME)
		itr := txn.NewIterator(badger.DefaultIteratorOptions)
		defer itr.Close()

		startTime2 := time.Now()
		common.Log.Infof("calculating in %s table ...", DB_PREFIX_NAME)

		for itr.Seek([]byte(prefix)); itr.ValidForPrefix([]byte(prefix)); itr.Next() {
			item := itr.Item()
			if item.IsDeletedOrExpired() {
				continue
			}
			var value NameValueInDB
			err = item.Value(func(data []byte) error {
				// return db.DecodeBytes(data, &value)
				return db.DecodeBytesWithProto3(data, &value)
			})
			if err != nil {
				common.Log.Panicf("item.Value error: %v", err)
			}

			nftIdInT1[value.NftId] = true
			namesInT1[value.Name] = true
			satsInT1[value.Sat] = true
		}

		common.Log.Infof("%s table takes %v", DB_PREFIX_NAME, time.Since(startTime2))
		return nil
	})

	bs := NewBuckStore(p.db)
	lastkey := bs.GetLastKey()
	var buckmap map[int]*BuckValue
	getbuck := func() {
		//defer wg.Done()
		buckmap = bs.GetAll()
	}
	getbuck()

	//wg.Wait()

	wrongName := make([]string, 0)
	wrongSats := make([]*BuckValue, 0)
	for _, v := range buckmap {
		_, ok := namesInT1[v.Name]
		if !ok {
			wrongName = append(wrongName, v.Name)
		}
		_, ok = satsInT1[v.Sat]
		if !ok {
			wrongSats = append(wrongSats, v)
		}
	}

	common.Log.Infof("name count: %d %d %d", p.status.NameCount, len(namesInT1), lastkey+1)
	common.Log.Infof("sats count %d", len(satsInT1))
	common.Log.Infof("inscriptionId count %d", len(nftIdInT1))
	common.Log.Infof("wrong name %d", len(wrongName))
	for i, value := range wrongName {
		if i > 10 {
			break
		}
		common.Log.Infof("wrong name %d: %s", i, value)
	}
	common.Log.Infof("wrong sat %d", len(wrongSats))
	for i, value := range wrongSats {
		if i > 10 {
			break
		}
		common.Log.Infof("wrong sat %d: %v", i, value)
	}
	if len(wrongName) != 0 || len(wrongSats) != 0 {
		common.Log.Errorf("data wrong")
		return false
	}

	count := p.status.NameCount - uint64(len(p.nameAdded))
	if count != uint64(len(namesInT1)) || count != uint64(lastkey+1) {
		common.Log.Errorf("name count different %d %d %d", count, len(namesInT1), uint64(lastkey+1))
		return false
	}

	// 1. 每个utxoId都存在baseDB中
	// 2. 两个表格中的数据相互对应: name，sat
	// 3. name的总数跟stats中一致

	common.Log.Infof("ns DB checked successfully, %v", time.Since(startTime))
	return true
}
