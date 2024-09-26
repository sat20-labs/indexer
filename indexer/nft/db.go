package nft

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
)

func initStatusFromDB(db *badger.DB) *common.NftStatus {
	stats := &common.NftStatus{}
	db.View(func(txn *badger.Txn) error {
		err := common.GetValueFromDB([]byte(NFT_STATUS_KEY), txn, stats)
		if err == badger.ErrKeyNotFound {
			common.Log.Info("initStatusFromDB no stats found in db")
			stats.Version = NFT_DB_VERSION
		} else if err != nil {
			common.Log.Panicf("initStatusFromDB failed. %v", err)
			return err
		}
		common.Log.Infof("nft stats: %v", stats)

		if stats.Version != NFT_DB_VERSION {
			common.Log.Panicf("nft data version inconsistent %s", NFT_DB_VERSION)
		}

		return nil
	})

	return stats
}

func getNftsWithAddressFromDB(addressId uint64, db *badger.DB) []int64 {
	result := make([]int64, 0)
	err := db.View(func(txn *badger.Txn) error {
		prefixBytes := []byte(fmt.Sprintf("%s%d-", DB_PREFIX_INSCADDR, addressId))
		prefixOptions := badger.DefaultIteratorOptions
		prefixOptions.Prefix = prefixBytes
		it := txn.NewIterator(prefixOptions)
		defer it.Close()
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			key := string(item.Key())

			_, nftId, err := ParseAddressKey(key)
			if err == nil {
				result = append(result, nftId)
			}
		}

		return nil
	})

	if err != nil {
		common.Log.Panicf("getNftsWithAddressFromDB Error: %v", err)
	}

	return result
}

func loadNftFromDB(sat int64, value *common.NftsInSat, txn *badger.Txn) error {
	key := GetSatKey(sat)
	// return common.GetValueFromDB([]byte(key), txn, value)
	return common.GetValueFromDBWithProto3([]byte(key), txn, value)
}

func loadUtxoValueFromDB(utxoId uint64, value *NftsInUtxo, txn *badger.Txn) error {
	key := GetUtxoKey(utxoId)
	// return common.GetValueFromDB([]byte(key), txn, value)
	return common.GetValueFromDBWithProto3([]byte(key), txn, value)
}

func hasNftInUtxo(utxoId uint64, txn *badger.Txn) bool {
	key := GetUtxoKey(utxoId)
	_, err := txn.Get([]byte(key))
	return err == nil
}

func GetSatKey(sat int64) string {
	return fmt.Sprintf("%s%d", DB_PREFIX_NFT, sat)
}

func GetUtxoKey(UtxoId uint64) string {
	return fmt.Sprintf("%s%d", DB_PREFIX_UTXO, UtxoId)
}

func GetInscriptionIdKey(id string) string {
	return fmt.Sprintf("%s%s", DB_PREFIX_INSC, id)
}

func GetInscriptionAddressKey(addrId uint64, nftId int64) string {
	return fmt.Sprintf("%s%d-%d", DB_PREFIX_INSCADDR, addrId, nftId)
}

func ParseSatKey(input string) (int64, error) {
	if !strings.HasPrefix(input, DB_PREFIX_NFT) {
		return -1, fmt.Errorf("invalid string format, %s", input)
	}
	str := strings.TrimPrefix(input, DB_PREFIX_NFT)
	sat, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return -1, fmt.Errorf("invalid string format, %s", input)
	}
	return sat, nil
}

func ParseUtxoKey(input string) (uint64, error) {
	if !strings.HasPrefix(input, DB_PREFIX_UTXO) {
		return common.INVALID_ID, fmt.Errorf("invalid string format, %s", input)
	}
	str := strings.TrimPrefix(input, DB_PREFIX_UTXO)
	return strconv.ParseUint(str, 10, 64)
}

func ParseAddressKey(input string) (uint64, int64, error) {
	if !strings.HasPrefix(input, DB_PREFIX_INSCADDR) {
		return common.INVALID_ID, -1, fmt.Errorf("invalid string format, %s", input)
	}
	parts := strings.Split(input, "-")
	if len(parts) != 3 {
		return common.INVALID_ID, -1, fmt.Errorf("invalid string format, %s", input)
	}
	addressId, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return common.INVALID_ID, -1, fmt.Errorf("invalid string format, %s", input)
	}
	nftId, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return common.INVALID_ID, -1, fmt.Errorf("invalid string format, %s", input)
	}
	return addressId, nftId, nil
}
