package base

import (
	"fmt"
	"sync"
	"time"

	"github.com/btcsuite/btcd/txscript"
	"github.com/sat20-labs/indexer/common"

	"github.com/dgraph-io/badger/v4"
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
	bSearching         bool
	satSearchingStatus map[int64]*SatSearchingStatus
}

func NewRpcIndexer(base *BaseIndexer) *RpcIndexer {
	indexer := &RpcIndexer{
		BaseIndexer:        *base.Clone(),
		bSearching:         false,
		satSearchingStatus: make(map[int64]*SatSearchingStatus),
	}

	return indexer
}

// 仅用于前端RPC数据查询时，更新地址数据
func (b *RpcIndexer) UpdateServiceInstance() {
	b.addressValueMap = b.prefechAddress()
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
	err := b.db.View(func(txn *badger.Txn) error {
		key := common.GetUTXODBKey(utxo)
		//err := common.GetValueFromDB(key, txn, output)
		err := common.GetValueFromDBWithProto3(key, txn, output)
		if err != nil {
			common.Log.Errorf("GetOrdinalsForUTXO %s failed, %v", utxo, err)
			return err
		}

		return nil
	})

	if err != nil {
		return common.INVALID_ID, nil, err
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
	err := b.db.View(func(txn *badger.Txn) error {
		key := common.GetUTXODBKey(utxo)
		//err := common.GetValueFromDB(key, txn, output)
		err := common.GetValueFromDBWithProto3(key, txn, output)
		if err != nil {
			common.Log.Errorf("GetOrdinalsForUTXO %s failed, %v", utxo, err)
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
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
func (b *RpcIndexer) getAddressValue2(address string, txn *badger.Txn) *common.AddressValueInDB {
	result := &common.AddressValueInDB{AddressId: common.INVALID_ID}
	addressId, err := common.GetAddressIdFromDBTxn(txn, address)
	if err == nil {
		utxos := make(map[uint64]*common.UtxoValue)
		prefix := []byte(fmt.Sprintf("%s%x-", common.DB_KEY_ADDRESSVALUE, addressId))
		itr := txn.NewIterator(badger.DefaultIteratorOptions)
		defer itr.Close()

		for itr.Seek(prefix); itr.ValidForPrefix(prefix); itr.Next() {
			item := itr.Item()
			if item.IsDeletedOrExpired() {
				continue
			}
			value := int64(0)
			item.Value(func(data []byte) error {
				value = int64(common.BytesToUint64(data))
				return nil
			})

			newAddressId, utxoId, typ, _, err := common.ParseAddressIdKey(string(item.Key()))
			if err != nil {
				common.Log.Panicf("ParseAddressIdKey %s failed: %v", string(item.Key()), err)
			}
			if newAddressId != addressId {
				common.Log.Panicf("ParseAddressIdKey %s get different addressid %d, %d", string(item.Key()), newAddressId, addressId)
			}
			result.AddressType = uint32(typ)

			utxos[utxoId] = &common.UtxoValue{Op: 0, Value: value}
		}

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
	utxo, err := common.GetUtxoByID(b.db, id)
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
	address, err := common.GetAddressByID(b.db, id)
	if err != nil {
		for key, value := range b.addressValueMap {
			if value.AddressId == id {
				return key, nil
			}
		}
		common.Log.Errorf("RpcIndexer->GetAddressByID %d failed, err: %v", id, err)
	}

	return address, err
}

// only for RPC interface
func (b *RpcIndexer) GetAddressId(address string) uint64 {

	id, err := common.GetAddressIdFromDB(b.db, address)
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
		var value common.UtxoValueInDB
		b.db.View(func(txn *badger.Txn) error {
			var err error
			prefix := []byte(common.DB_KEY_UTXO)
			itr := txn.NewIterator(badger.DefaultIteratorOptions)
			defer itr.Close()

			startTime := time.Now()
			common.Log.Infof("Search sat in %s table ...", common.DB_KEY_UTXO)

			for itr.Seek([]byte(prefix)); itr.ValidForPrefix([]byte(prefix)); itr.Next() {
				item := itr.Item()
				if item.IsDeletedOrExpired() {
					continue
				}

				err = item.Value(func(data []byte) error {
					return common.DecodeBytesWithProto3(data, &value)
				})
				if err != nil {
					common.Log.Errorf("item.Value error: %v", err)
					continue
				}

				if common.IsSatInRanges(sat, value.Ordinals) {
					common.Log.Infof("find sat %d in utxo %d in address %d", sat, value.UtxoId, value.AddressIds[0])
					bFound = true
					break
				}
			}
			common.Log.Infof("%s table takes %v", common.DB_KEY_UTXO, time.Since(startTime))

			return nil
		})
		if bFound {
			address, _ = common.GetAddressByID(b.db, value.AddressIds[0])
			utxo, _ = common.GetUtxoByID(b.db, value.UtxoId)
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
	var addressValueInDB *common.AddressValueInDB
	b.db.View(func(txn *badger.Txn) error {
		addressValueInDB = b.getAddressValue2(address, txn)
		return nil
	})

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

	key := common.GetBlockDBKey(height)
	block := common.BlockValueInDB{}
	err := b.db.View(func(txn *badger.Txn) error {
		return common.GetValueFromDB(key, txn, &block)
	})
	if err != nil {
		return nil, err
	}

	info := common.BlockInfo{Height: height, Timestamp: block.Timestamp,
		TotalSats:  block.Ordinals.Start + block.Ordinals.Size,
		RewardSats: block.OutputSats - block.InputSats}
	return &info, nil

}
