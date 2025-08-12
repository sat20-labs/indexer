package base

import (
	"fmt"
	"sync"
	"time"

	"github.com/btcsuite/btcd/txscript"
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
	addressValueMap    map[string]*common.AddressValueInDB
	addressIdMap       map[uint64]string
	deletedUtxoMap     map[uint64]bool
	bSearching         bool
	satSearchingStatus map[int64]*SatSearchingStatus
}

func NewRpcIndexer(base *BaseIndexer) *RpcIndexer {
	indexer := &RpcIndexer{
		BaseIndexer:        *base.Clone(),
		bSearching:         false,
		addressValueMap:    make(map[string]*common.AddressValueInDB),
		addressIdMap:       make(map[uint64]string),
		deletedUtxoMap:     make(map[uint64]bool),
		satSearchingStatus: make(map[int64]*SatSearchingStatus),
	}

	return indexer
}

// 仅用于前端RPC数据查询时，更新地址数据
func (b *RpcIndexer) UpdateServiceInstance() {
	b.addressValueMap = b.prefechAddress()
	b.addressIdMap = make(map[uint64]string)
	for k, v := range b.addressValueMap {
		b.addressIdMap[v.AddressId] = k
	}
	for _, v := range b.delUTXOs {
		b.deletedUtxoMap[v.UtxoId] = true
	}
}

// sync
func (b *RpcIndexer) GetOrdinalsWithUtxo(utxo string) (uint64, []*common.Range, error) {

	// 有可能还没有写入数据库，所以先读缓存
	utxoInfo, ok := b.utxoIndex.Index[utxo]
	if ok {
		return common.GetUtxoId(utxoInfo), utxoInfo.Ordinals, nil
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

	return output.UtxoId, output.Ordinals, nil
}

func (b *RpcIndexer) GetUtxoInfo(utxo string) (*common.UtxoInfo, error) {

	// 有可能还没有写入数据库，所以先读缓存
	utxoInfo, ok := b.utxoIndex.Index[utxo]
	if ok {
		value := &common.UtxoInfo{
			UtxoId:   common.GetUtxoId(utxoInfo),
			Value:    utxoInfo.Value,
			PkScript: utxoInfo.Address.PkScript,
			Ordinals: utxoInfo.Ordinals,
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
	var pkScript []byte
	addrType := output.AddressType
	reqSig := output.ReqSigs
	if addrType == uint32(txscript.MultiSigTy) {
		var addresses []string
		for _, id := range output.AddressIds {
			addr, err := b.GetAddressByID(id)
			if err != nil {
				return nil, err
			}
			addresses = append(addresses, addr)
		}
		pkScript, err = common.MultiSigToPkScript(int(reqSig), addresses, b.IsMainnet())
		if err != nil {
			return nil, err
		}
	} else if addrType == uint32(txscript.NullDataTy) {
		// TODO 这里没有获得原始的数据
		pkScript, _ = txscript.NullDataScript(nil)
	} else {
		addr, err := b.GetAddressByID(output.AddressIds[0])
		if err != nil {
			return nil, err
		}
		pkScript, err = common.AddressToPkScript(addr, b.IsMainnet())
		if err != nil {
			return nil, err
		}
	}

	info.UtxoId = output.UtxoId
	info.Value = common.GetOrdinalsSize(output.Ordinals)
	info.PkScript = pkScript
	info.Ordinals = output.Ordinals


	return &info, nil
}

// only for api access
func (b *RpcIndexer) getAddressValue2(address string) *common.AddressValueInDB {
	result := &common.AddressValueInDB{AddressId: common.INVALID_ID}
	addressId, err := db.GetAddressIdFromDB(b.db, address)
	if err == nil {
		utxos := make(map[uint64]*common.UtxoValue)
		prefix := []byte(fmt.Sprintf("%s%x-", common.DB_KEY_ADDRESSVALUE, addressId))
	
		b.db.BatchRead(prefix, false, func(k, v []byte) error {
			value := int64(common.BytesToUint64(v))
			newAddressId, utxoId, typ, _, err := common.ParseAddressIdKey(string(k))
			if err != nil {
				common.Log.Errorf("ParseAddressIdKey %s failed: %v", string(k), err)
				return err
			}
			if newAddressId != addressId {
				common.Log.Errorf("ParseAddressIdKey %s get different addressid %d, %d", string(k), newAddressId, addressId)
				return err
			}
			result.AddressType = uint32(typ)

			utxos[utxoId] = &common.UtxoValue{Op: 0, Value: value}
			return nil
		})

		result.AddressId = addressId
		result.Op = 0
		result.Utxos = utxos
	}

	b.mutex.RLock()
	value, ok := b.addressValueMap[address]
	if ok {
		result.AddressType = value.AddressType
		result.AddressId = value.AddressId
		if result.Utxos == nil {
			result.Utxos = make(map[uint64]*common.UtxoValue)
		}
		// 过滤已经删除的utxo
		for k, v := range value.Utxos {
			if v.Op > 0 {
				result.Utxos[k] = v
			} else if v.Op < 0 {
				delete(result.Utxos, k)
			}
		}
	}
	b.mutex.RUnlock()

	if result.AddressId == common.INVALID_ID {
		return nil
	}

	return result
}

// only for RPC interface
func (b *RpcIndexer) GetUtxoByID(id uint64) (string, error) {
	utxo, err := db.GetUtxoByID(b.db, id)
	if err != nil {
		for key, value := range b.utxoIndex.Index {
			if common.GetUtxoId(value) == id {
				return key, nil
			}
		}
		common.Log.Errorf("RpcIndexer->GetUtxoByID %d failed, err: %v", id, err)
	}

	return utxo, err
}

// only for RPC interface
func (b *RpcIndexer) GetAddressByID(id uint64) (string, error) {

	b.mutex.RLock()
	addrStr, ok := b.addressIdMap[id]
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
	b.addressIdMap[id] = address


	return address, err
}

// only for RPC interface
func (b *RpcIndexer) GetAddressId(address string) uint64 {

	id, err := db.GetAddressIdFromDB(b.db, address)
	if err != nil {
		id, _ = b.BaseIndexer.getAddressId(address)
		if id != common.INVALID_ID {
			err = nil
		} else {
			common.Log.Infof("getAddressId %s failed.", address)
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

func (b *RpcIndexer) searhing(sat int64) {

	var address, utxo string
	bFound := false

	// search in buffer
	for k, v := range b.utxoIndex.Index {
		if common.IsSatInRanges(sat, v.Ordinals) {
			common.Log.Infof("find sat %d in utxo %s in address %s", sat, k, v.Address.Addresses[0])
			bFound = true
			address = v.Address.Addresses[0]
			utxo = k
			break
		}
	}

	if !bFound {
		startTime := time.Now()
		common.Log.Infof("Search sat in %s table ...", common.DB_KEY_UTXO)
		var value common.UtxoValueInDB
		b.db.BatchRead([]byte(common.DB_KEY_UTXO), false, func(k, v []byte) error {
			err := db.DecodeBytesWithProto3(v, &value)
			if err != nil {
				common.Log.Errorf("item.Value error: %v", err)
				return err
			}

			if common.IsSatInRanges(sat, value.Ordinals) {
				common.Log.Infof("find sat %d in utxo %d in address %d", sat, value.UtxoId, value.AddressIds[0])
				bFound = true
				return nil
			}
			return nil
		})
		common.Log.Infof("%s table takes %v", common.DB_KEY_UTXO, time.Since(startTime))
		if bFound {
			address, _ = db.GetAddressByIDFromDB(b.db, value.AddressIds[0])
			utxo, _ = db.GetUtxoByID(b.db, value.UtxoId)
		}
	}

	b.mutex.Lock()
	status, ok := b.satSearchingStatus[sat]
	if ok {
		if bFound {
			status.Address = address
			status.Utxo = utxo
			status.Status = 0
		} else {
			status.Status = -1
		}
	}

	now := time.Now()
	threshold := now.Add(-24 * time.Hour).Unix()
	deledSats := make([]int64, 0)
	for key, value := range b.satSearchingStatus {
		if value.Ts < threshold {
			deledSats = append(deledSats, key)
		}
	}
	for _, sat := range deledSats {
		delete(b.satSearchingStatus, sat)
	}
	b.mutex.Unlock()

	b.bSearching = false
}

// address, utxo, message
func (b *RpcIndexer) FindSat(sat int64) (string, string, error) {
	b.mutex.RLock()
	status, ok := b.satSearchingStatus[sat]
	b.mutex.RUnlock()
	if ok {
		if status.Status > 0 {
			return "", "", fmt.Errorf("still running, please check again later")
		} else if status.Status == 0 {
			return status.Address, status.Utxo, nil
		} else {
			return "", "", fmt.Errorf("can't find this sat")
		}
	}

	if sat >= b.stats.TotalSats {
		return "", "", fmt.Errorf("sat too big")
	}

	if sat < 0 {
		return "", "", fmt.Errorf("sat number must be greater than 0")
	}

	if b.bSearching {
		return "", "", fmt.Errorf("another searching thread is running")
	}
	b.bSearching = true

	b.mutex.Lock()
	b.satSearchingStatus[sat] = &SatSearchingStatus{Status: 1, Ts: time.Now().Unix()}
	b.mutex.Unlock()

	go b.searhing(sat)

	return "", "", fmt.Errorf("start searhing, please check later")
}

func (b *RpcIndexer) getUtxosWithAddress(address string) (*common.AddressValue, error) {
	
	addressValueInDB := b.getAddressValue2(address)
	value := &common.AddressValue{}
	value.Utxos = make(map[uint64]int64)
	if addressValueInDB == nil {
		common.Log.Infof("RpcIndexer.getUtxosWithAddress-> No address %s found in db", address)
		return value, nil
	}

	value.AddressId = addressValueInDB.AddressId
	for utxoid, utxovalue := range addressValueInDB.Utxos {
		value.Utxos[utxoid] = utxovalue.Value
	}
	return value, nil
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
