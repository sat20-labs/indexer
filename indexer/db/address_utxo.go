package db

import (
	"bytes"
	"fmt"

	"github.com/sat20-labs/indexer/common"
)

const addressUtxoKeySuffixSize = 16

// GetAddressValueDBPrefix returns the compact binary prefix for all UTXOs of
// one stable address id:
//
//	av- | address_id(big endian uint64)
func GetAddressValueDBPrefix(addressID uint64) []byte {
	key := make([]byte, 0, len(common.DB_KEY_ADDRESSVALUE)+8)
	key = append(key, common.DB_KEY_ADDRESSVALUE...)
	key = append(key, common.Uint64ToBytes(addressID)...)
	return key
}

// GetAddressValueDBKey stores one address UTXO per Badger key:
//
//	av- | address_id(big endian uint64) | utxo_id(big endian uint64)
func GetAddressValueDBKey(addressID, utxoID uint64) []byte {
	key := GetAddressValueDBPrefix(addressID)
	key = append(key, common.Uint64ToBytes(utxoID)...)
	return key
}

func ParseAddressValueDBKey(key []byte) (addressID, utxoID uint64, err error) {
	prefix := []byte(common.DB_KEY_ADDRESSVALUE)
	if !bytes.HasPrefix(key, prefix) || len(key) != len(prefix)+addressUtxoKeySuffixSize {
		return common.INVALID_ID, common.INVALID_ID, fmt.Errorf("invalid address UTXO key %x", key)
	}
	offset := len(prefix)
	addressID = common.BytesToUint64(key[offset : offset+8])
	utxoID = common.BytesToUint64(key[offset+8 : offset+16])
	return addressID, utxoID, nil
}

func EncodeAddressUtxoValue(value int64) ([]byte, error) {
	if value < 0 {
		return nil, fmt.Errorf("negative address UTXO value %d", value)
	}
	return common.Uint64ToBytes(uint64(value)), nil
}

func DecodeAddressUtxoValue(value []byte) (int64, error) {
	if len(value) != 8 {
		return 0, fmt.Errorf("invalid address UTXO value length %d", len(value))
	}
	n := common.BytesToUint64(value)
	if n > uint64(^uint64(0)>>1) {
		return 0, fmt.Errorf("address UTXO value overflows int64: %d", n)
	}
	return int64(n), nil
}
