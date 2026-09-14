package base

import (
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

type addressUtxoScanSummary struct {
	AllUtxos         int
	AllAddresses     int
	NonZeroUtxos     int
	TotalSats        int64
	NonZeroAddresses map[uint64]bool
	NonZeroUtxoIDs   map[uint64]bool
}

// scanPersistedAddressUtxos summarizes the compact av- table. Keys are ordered
// as av-|addressID(big endian)|utxoID(big endian), so addresses are contiguous
// and the all-address count can be computed without allocating another map for
// tens of millions of address ids. Zero-sat UTXOs count toward AllUtxos and
// AllAddresses, but not toward the non-zero consistency sets used for sats.
func scanPersistedAddressUtxos(ldb common.KVDB) (*addressUtxoScanSummary, error) {
	summary := &addressUtxoScanSummary{
		NonZeroAddresses: make(map[uint64]bool),
		NonZeroUtxoIDs:   make(map[uint64]bool),
	}
	var lastAddressID uint64
	hasLastAddressID := false
	err := ldb.Scan(common.ScanOptions{Prefix: []byte(common.DB_KEY_ADDRESSVALUE)}, func(k, v []byte) error {
		addressID, utxoID, err := db.ParseAddressValueDBKey(k)
		if err != nil {
			return err
		}
		value, err := db.DecodeAddressUtxoValue(v)
		if err != nil {
			return err
		}

		summary.AllUtxos++
		if !hasLastAddressID || addressID != lastAddressID {
			summary.AllAddresses++
			lastAddressID = addressID
			hasLastAddressID = true
		}
		if value == 0 {
			return nil
		}

		summary.NonZeroUtxos++
		summary.TotalSats += value
		summary.NonZeroUtxoIDs[utxoID] = true
		summary.NonZeroAddresses[addressID] = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	return summary, nil
}

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
			UtxoCount:   pending.UtxoCount,
			Utxos:       make(map[uint64]int64),
		}
	}
	data, err := db.GetAddressDataFromDBV2(ldb, address)
	if err != nil {
		return nil
	}
	count, err := db.GetAddressUtxoCountFromDB(ldb, data.AddressId)
	if err != nil && err != common.ErrKeyNotFound {
		return nil
	}
	return &common.AddressValueV2{
		AddressId:   data.AddressId,
		AddressType: int(data.AddressType),
		Op:          0,
		UtxoCount:   count,
		Utxos:       make(map[uint64]int64),
	}
}
