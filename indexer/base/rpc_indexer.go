package base

import (
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/sat20-labs/indexer/common"

	"github.com/sat20-labs/indexer/indexer/db"
)

type SatSearchingStatus struct {
	Utxo    string
	Address string
	Status  int // 0 finished; 1 searching; -1 error.
	Ts      int64
}

type RpcIndexer struct {
	BaseIndexer

	// 接收前端api访问的实例，隔离内存访问
	mutex              sync.RWMutex
	deletedUtxoMap     map[uint64]bool
	addedUtxoMap       map[uint64]string
	bSearching         bool
	satSearchingStatus map[int64]*SatSearchingStatus
}

func NewRpcIndexer(base *BaseIndexer) *RpcIndexer {
	indexer := &RpcIndexer{
		BaseIndexer:        *base.Clone(false),
		bSearching:         false,
		deletedUtxoMap:     make(map[uint64]bool),
		addedUtxoMap:       make(map[uint64]string),
		satSearchingStatus: make(map[int64]*SatSearchingStatus),
	}

	return indexer
}

// 仅用于前端RPC数据查询时，更新地址数据
func (b *RpcIndexer) UpdateServiceInstance() {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	for _, v := range b.delUTXOs {
		b.deletedUtxoMap[v.UtxoId] = true
	}
	for k, v := range b.utxoIndex.Index {
		b.addedUtxoMap[v.UtxoId] = k
	}
}

// sync
func (b *RpcIndexer) GetOrdinalsWithUtxo(utxo string) (uint64, []*common.Range, error) {

	// 有可能还没有写入数据库，所以先读缓存
	utxoInfo, ok := b.utxoIndex.Index[utxo]
	if ok {
		return utxoInfo.UtxoId, nil, nil
	}

	if err := common.CheckUtxoFormat(utxo); err != nil {
		return 0, nil, err
	}

	output := &common.UtxoValueInDB{}
	
	key := db.GetUTXODBKey(utxo)
	//err := db.GetValueFromDB(key, txn, output)
	err := db.GetValueFromDBWithProto3(key, b.db, output)
	
	if err != nil {
		return common.INVALID_ID, nil, err
	}

	_, ok = b.deletedUtxoMap[output.UtxoId]
	if ok {
		return 0, nil, fmt.Errorf("utxo %s is spent", utxo)
	}

	return output.UtxoId, nil, nil
}

func (b *RpcIndexer) GetUtxoInfo(utxo string) (*common.UtxoInfo, error) {

	// 有可能还没有写入数据库，所以先读缓存
	utxoInfo, ok := b.utxoIndex.Index[utxo]
	if ok {
		value := &common.UtxoInfo{
			UtxoId:   utxoInfo.UtxoId,
			Value:    utxoInfo.OutValue.Value,
			PkScript: utxoInfo.OutValue.PkScript,
			Ordinals: nil,
		}
		return value, nil
	}

	if err := common.CheckUtxoFormat(utxo); err != nil {
		return nil, err
	}

	output := &common.UtxoValueInDB{}
	key := db.GetUTXODBKey(utxo)
	//err := db.GetValueFromDB(key, txn, output)
	err := db.GetValueFromDBWithProto3(key, b.db, output)
	if err != nil {
		return nil, err
	}

	_, ok = b.deletedUtxoMap[output.UtxoId]
	if ok {
		return nil, fmt.Errorf("utxo %s is spent", utxo)
	}

	info := common.UtxoInfo{}
	address, err := b.GetAddressByID(output.AddressId)
	if err != nil {
		return nil, err
	}
	
	pkScript, err := base64.StdEncoding.DecodeString(address)
	if err != nil {
		return nil, err
	}
	
	info.UtxoId = output.UtxoId
	info.Value = output.Value
	info.PkScript = pkScript
	info.Ordinals = nil

	return &info, nil
}

// only for api access
func (b *RpcIndexer) getAddressValue2(address string, ldb common.KVDB) *common.AddressValueV2 {
	b.mutex.Lock()
	value, ok := b.addressValueMap[address]
	if !ok {
		data, err := db.GetAddressDataFromDBV2(ldb, address)
		if err == nil {
			value = common.ToAddressValueV2(data)
			b.addressValueMap[address] = value
			ok = true
		}
	}
	b.mutex.Unlock()

	return value
}

// only for RPC interface
func (b *RpcIndexer) GetUtxoByID(id uint64) (string, error) {
	utxo, err := db.GetUtxoByID(b.db, id)
	if err != nil {
		utxo, ok := b.addedUtxoMap[id]
		if ok {
			return utxo, nil
		}
		common.Log.Errorf("RpcIndexer->GetUtxoByID %d failed, err: %v", id, err)
	}

	return utxo, err
}

// only for RPC interface
func (b *RpcIndexer) GetAddressByID(id uint64) (string, error) {

	b.mutex.RLock()
	addrStr, ok := b.idToAddressMap[id]
	b.mutex.RUnlock()
	if ok {
		return addrStr, nil
	}

	address, err := db.GetAddressByIDFromDB(b.db, id)
	if err != nil {
		common.Log.Errorf("RpcIndexer->GetAddressByID %d failed, err: %v", id, err)
		return "", err
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.idToAddressMap[id] = address


	return address, err
}

// only for RPC interface
func (b *RpcIndexer) GetAddressId(address string) uint64 {

	b.mutex.Lock()
	defer b.mutex.Unlock()
	id, _ := b.getAddressId(address)
	if id == common.INVALID_ID {
		data, err := db.GetAddressDataFromDBV2(b.db, address)
		if err == nil {
			value := common.ToAddressValueV2(data)
			id = value.AddressId
			b.addressValueMap[address] = value
			b.idToAddressMap[id] = address
		}
	}

	return id
}

func (b *RpcIndexer) GetOrdinalsWithUtxoId(id uint64) (string, []*common.Range, error) {
	utxo, err := b.GetUtxoByID(id)
	if err != nil {
		return "", nil, err
	}
	_, result, err := b.GetOrdinalsWithUtxo(utxo)
	return utxo, result, err
}

// key: utxoId, value: btc value
func (b *RpcIndexer) GetUTXOs(address string) (map[uint64]int64, error) {
	addrValue, err := b.getUtxosWithAddress(address)

	if err != nil {
		return nil, err
	}
	return addrValue.Utxos, nil
}

// only for RPC
func (b *RpcIndexer) GetUTXOs2(address string) []string {
	addrValue, err := b.getUtxosWithAddress(address)

	if err != nil {
		common.Log.Errorf("getUtxosWithAddress %s failed, err %v", address, err)
		return nil
	}

	utxos := make([]string, 0)
	for utxoId := range addrValue.Utxos {
		utxo, err := b.GetUtxoByID(utxoId)
		if err != nil {
			common.Log.Errorf("GetUtxoByID failed. address %s, utxo id %d", address, utxoId)
			continue
		}
		utxos = append(utxos, utxo)
	}
	return utxos
}

func (b *RpcIndexer) getUtxosWithAddress(address string) (*common.AddressValueV2, error) {
	
	addressValueInDB := b.getAddressValue2(address, b.db)
	if addressValueInDB == nil {
		//indexer.Log.Infof("RpcIndexer.getUtxosWithAddress-> No address %s found in db", address)
		return nil, fmt.Errorf("not found")
	}

	return addressValueInDB, nil
}

func (b *RpcIndexer) GetBlockInfo(height int) (*common.BlockInfo, error) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	for _, block := range b.blockVector {
		if block.Height == height {
			info := common.BlockInfo{Height: height, Timestamp: block.Timestamp,
				TotalSats:  block.Ordinals.Start + block.Ordinals.Size,
				RewardSats: block.OutputSats - block.InputSats}
			return &info, nil
		}
	}

	key := db.GetBlockDBKey(height)
	block := common.BlockValueInDB{}
	err := db.GetValueFromDB(key, &block, b.db)
	if err != nil {
		return nil, err
	}

	info := common.BlockInfo{Height: height, Timestamp: block.Timestamp,
		TotalSats:  block.Ordinals.Start + block.Ordinals.Size,
		RewardSats: block.OutputSats - block.InputSats}
	return &info, nil

}
