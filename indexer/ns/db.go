package ns

import (
	"fmt"
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
	indexer "github.com/sat20-labs/indexer/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"

	"github.com/dgraph-io/badger/v4"
)

func initStatusFromDB(ldb *badger.DB) *common.NameServiceStatus {
	stats := &common.NameServiceStatus{}
	ldb.View(func(txn *badger.Txn) error {
		err := db.GetValueFromDB([]byte(NS_STATUS_KEY), txn, stats)
		if err == badger.ErrKeyNotFound {
			common.Log.Info("initStatusFromDB no stats found in db")
			stats.Version = NS_DB_VERSION
		} else if err != nil {
			common.Log.Panicf("initStatusFromDB failed. %v", err)
			return err
		}
		common.Log.Infof("ns stats: %v", stats)

		if stats.Version != NS_DB_VERSION {
			common.Log.Panicf("ns data version inconsistent %s", NS_DB_VERSION)
		}

		return nil
	})

	return stats
}

func initNameTreeFromDB(tree *indexer.SatRBTree, ldb *badger.DB) {
	count := 0
	startTime := time.Now()
	common.Log.Info("initNameTreeFromDB ...")
	err := ldb.View(func(txn *badger.Txn) error {
		prefixBytes := []byte(DB_PREFIX_NAME)
		prefixOptions := badger.DefaultIteratorOptions
		prefixOptions.Prefix = prefixBytes
		it := txn.NewIterator(prefixOptions)
		defer it.Close()
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			if item.IsDeletedOrExpired() {
				continue
			}
			key := string(item.Key())

			_, err := ParseNameKey(key)
			if err == nil {
				var mint NameRegister
				value, err := item.ValueCopy(nil)
				if err != nil {
					common.Log.Errorln("initNameTreeFromDB ValueCopy " + key + " " + err.Error())
				} else {
					err = db.DecodeBytes(value, &mint)
					if err == nil {
						BindNametoSat(tree, mint.Nft.Base.Sat, mint.Name)
					} else {
						common.Log.Errorln("initNameTreeFromDB DecodeBytes " + err.Error())
					}
				}
			}

			count++
		}

		return nil
	})

	if err != nil {
		common.Log.Panicf("initNameTreeFromDB Error: %v", err)
	}

	common.Log.Infof("initNameTreeFromDB loaded %d records in %v\n", count, time.Since(startTime))
}

// 没有utxo数据，utxo是变动的数据，不适合保持在buck中，避免动态数据多处保持，容易出问题。
func initNameTreeFromDB2(tree *indexer.SatRBTree, db *badger.DB) {
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

func loadNameFromDB(name string, value *NameValueInDB, txn *badger.Txn) error {
	key := GetNameKey(name)
	// return db.GetValueFromDB([]byte(key), txn, value)
	return db.GetValueFromDBWithProto3([]byte(key), txn, value)
}

func loadNameProperties(name string, ldb *badger.DB) map[string]*common.KeyValueInDB {
	KVs := make(map[string]*common.KeyValueInDB)

	err := ldb.View(func(txn *badger.Txn) error {
		prefixBytes := []byte(DB_PREFIX_KV + name + "-")
		prefixOptions := badger.DefaultIteratorOptions
		prefixOptions.Prefix = prefixBytes
		it := txn.NewIterator(prefixOptions)
		defer it.Close()
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			if item.IsDeletedOrExpired() {
				continue
			}
			_, key, err := ParseKVKey(string(item.Key()))
			if err == nil {
				var valueInDB common.KeyValueInDB
				value, err := item.ValueCopy(nil)
				if err != nil {
					common.Log.Errorln("loadNameProperties ValueCopy " + key + " " + err.Error())
				} else {
					err = db.DecodeBytes(value, &valueInDB)
					if err == nil {
						KVs[key] = &valueInDB
					} else {
						common.Log.Errorln("initNameTreeFromDB DecodeBytes " + err.Error())
					}
				}
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

func loadValueWithKey(name, key string, ldb *badger.DB) *common.KeyValueInDB {
	kv := common.KeyValueInDB{}

	err := ldb.View(func(txn *badger.Txn) error {
		key := GetKVKey(name, key)
		return db.GetValueFromDB([]byte(key), txn, &kv)
	})

	if err != nil {
		common.Log.Errorf("GetValueFromDB %s-%s failed. %v", name, key, err)
		return nil
	}

	return &kv
}

func GetNameKey(name string) string {
	return fmt.Sprintf("%s%s", DB_PREFIX_NAME, strings.ToLower(name))
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
