package indexer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func getKvKey(pubkey string, key string) string {
	return fmt.Sprintf("/%s/%s", pubkey, key)
}


func (b *IndexerMgr) IsSupportedKey(pubkey []byte) bool {
	// TODO 以后可以配置增加更多的pubkey
	pkStr := hex.EncodeToString(pubkey)
	if pkStr != common.GetBootstrapPubKey() && pkStr != common.GetCoreNodePubKey() {
		return false
	}
	return true
}


func (b *IndexerMgr) PutKVs(kvs []*common.KeyValue) (error) {

	wb := b.kvDB.NewWriteBatch()
	defer wb.Cancel()

	for _, value := range kvs {
		
		if !b.IsSupportedKey(value.PubKey) {
			common.Log.Errorf("unsupport pubkey")
			return fmt.Errorf("unsupport pubkey")
		}

		// 目前仅允许内置的pubkey
		pkStr := hex.EncodeToString(value.PubKey)

		sig := value.Signature
		value.Signature = nil
		msg, err := json.Marshal(value)
		if err != nil {
			common.Log.Errorf("json.Marshal failed. %v", err)
			return err
		}
		value.Signature = sig
		
		// verify the signature
		err = common.VerifySignOfMessage(msg, sig, value.PubKey)
		if err != nil {
			common.Log.Errorf("verify signature of key %s failed, %v", value.Key, err)
			return fmt.Errorf("verify signature of key %s failed, %v", value.Key, err)
		}

		key := getKvKey(pkStr, value.Key)
		err = db.SetDB([]byte(key), value, wb)
		if err != nil {
			common.Log.Errorf("setting key %s failed, %v", key, err)
			return err
		}
		common.Log.Infof("keyValue saved. %s", key)
	}

	err := wb.Flush()
	if err != nil {
		common.Log.Errorf("flushing writes to db %v", err)
		return err
	}

	return nil
}


func (b *IndexerMgr) DelKVs(pubkey []byte, keys []string) (error) {
	
	wb := b.kvDB.NewWriteBatch()
	defer wb.Cancel()

	pkStr := hex.EncodeToString(pubkey)

	for _, k := range keys {
		key := getKvKey(pkStr, k)
		err := wb.Delete([]byte(key))
		if err != nil {
			common.Log.Errorf("deleting key %s failed, %v", key, err)
			return err
		}
		common.Log.Infof("keyValue deleted. %s", key)
	}

	err := wb.Flush()
	if err != nil {
		common.Log.Errorf("flushing writes to db %v", err)
		return err
	}

	return nil
}


func (b *IndexerMgr) GetKVs(pubkey []byte, keys []string) ([]*common.KeyValue, error) {
	
	pkStr := hex.EncodeToString(pubkey)
	result := make([]*common.KeyValue, 0)
	b.kvDB.View(func(txn *badger.Txn) error {
		for _, k := range keys {
			key := getKvKey(pkStr, k)
	
			item, err := txn.Get([]byte(key))
			if err != nil {
				continue
			}
			var value common.KeyValue
			err = item.Value(func(v []byte) error {
				return db.DecodeBytes(v, &value)
			})
			if err != nil {
				common.Log.Errorf("decoding key %s failed, %v", key, err)
				continue
			}
	
			result = append(result, &value)
		}
		return nil
	})
	
	return result, nil
}

