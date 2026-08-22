package base

import (
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func (b *BaseIndexer) loadAddressUtxos(ldb common.KVDB, addressID uint64) (map[uint64]int64, error) {
	result := make(map[uint64]int64)
	prefix := db.GetAddressValueDBPrefix(addressID)
	if err := ldb.Scan(common.ScanOptions{Prefix: prefix}, func(k, v []byte) error {
		storedAddressID, utxoID, err := db.ParseAddressValueDBKey(k)
		if err != nil {
			return err
		}
		if storedAddressID != addressID {
			return nil
		}
		value, err := db.DecodeAddressUtxoValue(v)
		if err != nil {
			return err
		}
		result[utxoID] = value
		return nil
	}); err != nil {
		return nil, err
	}

	address := b.idToAddressMap[addressID]
	if address == "" {
		if stored, err := db.GetAddressByIDFromDB(ldb, addressID); err == nil {
			address = stored
		}
	}
	if pending := b.addressValueMap[address]; pending != nil && pending.AddressId == addressID {
		for utxoID, value := range pending.Utxos {
			result[utxoID] = value
		}
	}
	for utxoID := range b.addressUtxoDeleted[addressID] {
		delete(result, utxoID)
	}
	return result, nil
}

func (b *BaseIndexer) loadAddressMeta(address string, ldb common.KVDB) *common.AddressValueV2 {
	if pending := b.addressValueMap[address]; pending != nil {
		return &common.AddressValueV2{
			AddressId:   pending.AddressId,
			AddressType: pending.AddressType,
			Op:          pending.Op,
			Utxos:       make(map[uint64]int64),
		}
	}
	data, err := db.GetAddressDataFromDBV2(ldb, address)
	if err != nil {
		return nil
	}
	return &common.AddressValueV2{
		AddressId:   data.AddressId,
		AddressType: int(data.AddressType),
		Op:          0,
		Utxos:       make(map[uint64]int64),
	}
}
