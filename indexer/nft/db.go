package nft

import (
	"encoding/hex"
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

func getContentTypesFromDB(ldb common.KVDB) map[int]string {
	result := make(map[int]string, 0)
	err := ldb.BatchRead([]byte(DB_PREFIX_IT), false, func(k, v []byte) error {

		key := string(k)
		id, err := ParseContTypeKey(key)
		if err == nil {
			var ct string
			err = db.DecodeBytes(v, &ct)
			result[id] = ct
		}

		return nil
	})

	if err != nil {
		common.Log.Panicf("getContentTypesFromDB Error: %v", err)
	}

	return result
}

func loadNftsInSatFromDB(sat int64, value *common.NftsInSat, ldb common.KVDB) error {
	key := GetSatKey(sat)
	// return db.GetValueFromDB([]byte(key), txn, value)
	return db.GetValueFromDBWithProto3([]byte(key), ldb, value)
}

func loadNftsInSatFromTxn(sat int64, value *common.NftsInSat, txn common.ReadBatch) error {
	key := GetSatKey(sat)
	// return db.GetValueFromDB([]byte(key), txn, value)
	return db.GetValueFromTxnWithProto3([]byte(key), txn, value)
}

func loadNftFromDB(nftId int64, value *common.InscribeBaseContent, ldb common.KVDB) error {
	return ldb.View(func(rb common.ReadBatch) error {
		return loadNftFromTxn(nftId, value, rb)
	})
}

func loadNftFromTxn(nftId int64, value *common.InscribeBaseContent, txn common.ReadBatch) error {
	key := GetNftKey(nftId)
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

// 大端序下，高位字节先比较 → 字节序比较行为与整数比较行为一致。
func GetSatKey(sat int64) string {
	return fmt.Sprintf("%s%s", DB_PREFIX_SAT, hex.EncodeToString(common.Uint64ToBytes(uint64(sat)))) // 1.7.0
	//return fmt.Sprintf("%s%016d", DB_PREFIX_NFT, sat) // 1.6.0
	//return fmt.Sprintf("%s%d", DB_PREFIX_NFT, sat) // 1.5.0
}

func GetNftKey(nftId int64) string {
	return fmt.Sprintf("%s%s", DB_PREFIX_NFT, hex.EncodeToString(common.Uint64ToBytes(uint64(nftId))))
}

func GetUtxoKey(UtxoId uint64) string {
	return fmt.Sprintf("%s%s", DB_PREFIX_UTXO, hex.EncodeToString(common.Uint64ToBytes(UtxoId)))
}

func GetCTKey(id int) string {
	return fmt.Sprintf("%s%d", DB_PREFIX_IT, id)
}

func GetInscriptionIdKey(id string) string {
	return fmt.Sprintf("%s%s", DB_PREFIX_INSC, id)
}

func GetInscriptionAddressKey(addrId uint64, nftId int64) string {
	return fmt.Sprintf("%s%d_%d", DB_PREFIX_INSCADDR, addrId, nftId) // nftId 有负数，需要更改分界符
}

func GetDisabledSatKey(sat int64) string {
	return fmt.Sprintf("%s%d", DB_PREFIX_DISABLED_SAT, sat)
}

func ParseSatKey(input string) (int64, error) {
	if !strings.HasPrefix(input, DB_PREFIX_SAT) {
		return -1, fmt.Errorf("invalid string format, %s", input)
	}
	str := strings.TrimPrefix(input, DB_PREFIX_SAT)

	bytes, err := hex.DecodeString(str)
	if err != nil {
		return 0, err
	}
	if len(bytes) != 8 {
		return 0, fmt.Errorf("invalid sat: %s", str)
	}
	sat := common.BytesToUint64(bytes)

	return int64(sat), nil
}

func ParseUtxoKey(input string) (uint64, error) {
	if !strings.HasPrefix(input, DB_PREFIX_UTXO) {
		return common.INVALID_ID, fmt.Errorf("invalid string format, %s", input)
	}
	str := strings.TrimPrefix(input, DB_PREFIX_UTXO)
	bytes, err := hex.DecodeString(str)
	if err != nil {
		return 0, err
	}
	if len(bytes) != 8 {
		return 0, fmt.Errorf("invalid sat: %s", str)
	}
	utxoId := common.BytesToUint64(bytes)
	return utxoId, nil
}

func ParseContTypeKey(input string) (int, error) {
	if !strings.HasPrefix(input, DB_PREFIX_IT) {
		return -1, fmt.Errorf("invalid string format, %s", input)
	}
	str := strings.TrimPrefix(input, DB_PREFIX_IT)
	id, err := strconv.Atoi(str)
	if err != nil {
		return -1, err
	}

	return id, nil
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

func GetContentIdKey(id uint64) []byte {
	return []byte(fmt.Sprintf(DB_PREFIX_IC+"%x", id))
}

func GetContentDBKey(content string) []byte {
	return []byte(DB_PREFIX_CI + content)
}

func BindContentDBKeyToId(content string, id uint64, wb common.WriteBatch) error {
	if err := wb.Put(GetContentIdKey(id), []byte(content)); err != nil {
		return err
	}
	return wb.Put(GetContentDBKey(content), common.Uint64ToBytes(id))
}

func UnBindContentId(content string, id uint64, wb common.WriteBatch) error {
	wb.Delete(GetContentIdKey(id))
	wb.Delete(GetContentDBKey(content))
	return nil
}

func GetContentByIdFromDB(ldb common.KVDB, id uint64) (string, error) {
	key, err := ldb.Read(GetContentIdKey(id))
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(string(key), DB_PREFIX_CI), nil
}

func GetContentByIdFromTxn(txn common.ReadBatch, id uint64) (string, error) {
	key, err := txn.Get(GetContentIdKey(id))
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(string(key), DB_PREFIX_CI), nil
}

func GetContentIdFromDB(db common.KVDB, content string) (uint64, error) {
	key, err := db.Read(GetContentDBKey(content))
	if err != nil {
		return common.INVALID_ID, err
	}
	return common.BytesToUint64(key), nil
}

func GetContentIdFromTxn(db common.ReadBatch, content string) (uint64, error) {
	key, err := db.Get(GetContentDBKey(content))
	if err != nil {
		return common.INVALID_ID, err
	}
	return common.BytesToUint64(key), nil
}
