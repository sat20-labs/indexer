package nft

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func initStatusFromDB(ldb common.KVDB) *common.NftStatus {
	stats := &common.NftStatus{}
	err := db.GetValueFromDB([]byte(NFT_STATUS_KEY), stats, ldb)
	if err == common.ErrKeyNotFound {
		common.Log.Info("initStatusFromDB no stats found in db")
		stats.Version = NFT_DB_VERSION
	} else if err != nil {
		common.Log.Panicf("initStatusFromDB failed. %v", err)
	}
	common.Log.Infof("nft stats: %v", stats)

	if stats.Version != NFT_DB_VERSION {
		common.Log.Panicf("nft data version inconsistent %s", NFT_DB_VERSION)
	}

	return stats
}

func getNftsWithAddressFromDB(addressId uint64, db common.KVDB) []int64 {
	result := make([]int64, 0)
	err := db.BatchRead([]byte(fmt.Sprintf("%s%d_", DB_PREFIX_INSCADDR, addressId)),
		false, func(k, v []byte) error {

			key := string(k)

			_, nftId, err := ParseAddressKey(key)
			if err == nil {
				result = append(result, nftId)
			}

			return nil
		})

	if err != nil {
		common.Log.Panicf("getNftsWithAddressFromDB Error: %v", err)
	}

	return result
}

func loadNftFromDB(sat int64, value *common.NftsInSat, ldb common.KVDB) error {
	key := GetSatKey(sat)
	// return db.GetValueFromDB([]byte(key), txn, value)
	return db.GetValueFromDBWithProto3([]byte(key), ldb, value)
}

func loadNftFromTxn(sat int64, value *common.NftsInSat, txn common.ReadBatch) error {
	key := GetSatKey(sat)
	// return db.GetValueFromDB([]byte(key), txn, value)
	return db.GetValueFromTxnWithProto3([]byte(key), txn, value)
}

func loadUtxoValueFromDB(utxoId uint64, value *NftsInUtxo, ldb common.KVDB) error {
	key := GetUtxoKey(utxoId)
	// return db.GetValueFromDB([]byte(key), txn, value)
	return db.GetValueFromDBWithProto3([]byte(key), ldb, value)
}

func loadUtxoValueFromTxn(utxoId uint64, value *NftsInUtxo, txn common.ReadBatch) error {
	key := GetUtxoKey(utxoId)
	// return db.GetValueFromDB([]byte(key), txn, value)
	return db.GetValueFromTxnWithProto3([]byte(key), txn, value)
}

func hasNftInUtxo(utxoId uint64, ldb common.KVDB) bool {
	key := GetUtxoKey(utxoId)
	_, err := ldb.Read([]byte(key))
	return err == nil
}

// 聪的十进制数字不超过16位，为了排序，这里填足够的0
func GetSatKey(sat int64) string {
	return fmt.Sprintf("%s%016d", DB_PREFIX_NFT, sat)
	//return fmt.Sprintf("%s%d", DB_PREFIX_NFT, sat) // 1.5.0
}

func GetUtxoKey(UtxoId uint64) string {
	return fmt.Sprintf("%s%d", DB_PREFIX_UTXO, UtxoId)
}

func GetInscriptionIdKey(id string) string {
	return fmt.Sprintf("%s%s", DB_PREFIX_INSC, id)
}

func GetInscriptionAddressKey(addrId uint64, nftId int64) string {
	return fmt.Sprintf("%s%d_%d", DB_PREFIX_INSCADDR, addrId, nftId)
}

func GetDisabledSatKey(sat int64) string {
	return fmt.Sprintf("%s%d", DB_PREFIX_DISABLED_SAT, sat)
}

func ParseSatKey(input string) (int64, error) {
	if !strings.HasPrefix(input, DB_PREFIX_NFT) {
		return -1, fmt.Errorf("invalid string format, %s", input)
	}
	str := strings.TrimPrefix(input, DB_PREFIX_NFT) // 注意有负数聪
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
	parts := strings.Split(input, "_")
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

func ParseDisabledSatKey(input string) (int64, error) {
	if !strings.HasPrefix(input, DB_PREFIX_DISABLED_SAT) {
		return -1, fmt.Errorf("invalid string format, %s", input)
	}
	str := strings.TrimPrefix(input, DB_PREFIX_DISABLED_SAT)
	sat, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return -1, fmt.Errorf("invalid string format, %s", input)
	}
	return sat, nil
}

func loadDisabledSatFromDB(sat int64, ldb common.KVDB) ([]byte, error) {
	key := GetDisabledSatKey(sat)
	var value []byte
	err := db.GetValueFromDB([]byte(key), &value, ldb)
	return value, err
}

func loadAllDisalbedSatsFromDB(ldb common.KVDB) map[int64]bool {
	result := make(map[int64]bool)
	ldb.BatchRead([]byte(DB_PREFIX_DISABLED_SAT), false, func(k, v []byte) error {

		sat, err := ParseDisabledSatKey(string(k))
		if err != nil {
			return nil
		}

		result[sat] = true
		return nil
	})

	return result
}

func saveDisabledSatToDB(sat int64, value []byte, ldb common.KVDB) error {
	key := GetDisabledSatKey(sat)
	return db.GobSetDB([]byte(key), value, ldb)
}
