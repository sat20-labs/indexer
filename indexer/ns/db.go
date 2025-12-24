package ns

import (
	"fmt"
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
	indexer "github.com/sat20-labs/indexer/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func initStatusFromDB(ldb common.KVDB) *common.NameServiceStatus {
	stats := &common.NameServiceStatus{}

	err := db.GetValueFromDB([]byte(NS_STATUS_KEY), stats, ldb)
	if err == common.ErrKeyNotFound {
		common.Log.Info("initStatusFromDB no stats found in db")
		stats.Version = NS_DB_VERSION
	} else if err != nil {
		common.Log.Panicf("initStatusFromDB failed. %v", err)
	}
	common.Log.Infof("ns stats: %v", stats)

	if stats.Version != NS_DB_VERSION {
		common.Log.Panicf("ns data version inconsistent %s", NS_DB_VERSION)
	}

	return stats
}

func initNameTreeFromDB(tree *indexer.SatRBTree, ldb common.KVDB) {
	count := 0
	startTime := time.Now()
	common.Log.Info("initNameTreeFromDB ...")
	err := ldb.BatchRead([]byte(DB_PREFIX_NAME), false, func(k, v []byte) error {

		key := string(k)
		_, err := ParseNameKey(key)
		if err == nil {
			var mint NameRegister
			err = db.DecodeBytes(v, &mint)
			if err == nil {
				BindNametoSat(tree, mint.Nft.Base.Sat, mint.Name)
			} else {
				common.Log.Errorln("initNameTreeFromDB DecodeBytes " + err.Error())
			}
		}

		count++
		return nil
	})

	if err != nil {
		common.Log.Panicf("initNameTreeFromDB Error: %v", err)
	}

	common.Log.Infof("initNameTreeFromDB loaded %d records in %v\n", count, time.Since(startTime))
}

// 没有utxo数据，utxo是变动的数据，不适合保持在buck中，避免动态数据多处保持，容易出问题。
func initNameTreeFromDB2(tree *indexer.SatRBTree, db common.KVDB) {
	startTime := time.Now()
	common.Log.Info("initNameTreeFromDB2 ...")

	buckDB := NewBuckStore(db)
	bulkMap := buckDB.GetAll()

	for _, v := range bulkMap {
		value := &RBTreeValue_Name{Name: v.Name}
		tree.Put(v.Sat, value)
	}

	// 没有utxo数据。在需要时动态加载，可能会更好

	common.Log.Infof("initNameTreeFromDB2 loaded %d records in %v\n", len(bulkMap), time.Since(startTime))
}

func loadNameFromDB(name string, value *NameValueInDB, ldb common.KVDB) error {
	key := GetNameKey(name)
	// return db.GetValueFromDB([]byte(key), txn, value)
	return db.GetValueFromDBWithProto3([]byte(key), ldb, value)
}

func loadNameWithSatIdFromDB(sat int64, name *string, ldb common.KVDB) error {
	key := GetSatKey(sat)
	return db.GetValueFromDB([]byte(key), name, ldb)
}


func loadNameProperties(name string, ldb common.KVDB) map[string]*common.KeyValueInDB {
	KVs := make(map[string]*common.KeyValueInDB)

	err := ldb.BatchRead([]byte(DB_PREFIX_KV+name+"-"), false, func(k, v []byte) error {

		_, key, err := ParseKVKey(string(k))
		if err == nil {
			var valueInDB common.KeyValueInDB
			err = db.DecodeBytes(v, &valueInDB)
			if err == nil {
				KVs[key] = &valueInDB
			} else {
				common.Log.Errorln("initNameTreeFromDB DecodeBytes " + err.Error())
			}
		}
		return nil
	})

	if err != nil {
		common.Log.Errorf("loadNameProperties %s failed. %v", name, err)
		return nil
	}

	return KVs
}

func loadValueWithKey(name, key string, ldb common.KVDB) *common.KeyValueInDB {
	kv := common.KeyValueInDB{}

	k := GetKVKey(name, key)
	err := db.GetValueFromDB([]byte(k), &kv, ldb)

	if err != nil {
		common.Log.Errorf("GetValueFromDB %s-%s failed. %v", name, key, err)
		return nil
	}

	return &kv
}

func GetNameKey(name string) string {
	return fmt.Sprintf("%s%s", DB_PREFIX_NAME, strings.ToLower(name))
}

func GetSatKey(sat int64) string {
	return fmt.Sprintf("%s%d", DB_PREFIX_SAT, sat)
}

func GetKVKey(name, key string) string {
	return fmt.Sprintf("%s%s-%s", DB_PREFIX_KV, strings.ToLower(name), key)
}

func ParseNameKey(input string) (string, error) {
	if !strings.HasPrefix(input, DB_PREFIX_NAME) {
		return "", fmt.Errorf("invalid string format")
	}
	str := strings.TrimPrefix(input, DB_PREFIX_NAME)
	return str, nil
}

func ParseKVKey(input string) (string, string, error) {
	if !strings.HasPrefix(input, DB_PREFIX_KV) {
		return "", "", fmt.Errorf("invalid string format")
	}
	str := strings.TrimPrefix(input, DB_PREFIX_KV)
	parts := strings.Split(str, "-")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid string format")
	}

	return parts[0], parts[1], nil
}
