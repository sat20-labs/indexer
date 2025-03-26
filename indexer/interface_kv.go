package indexer

import (
	"encoding/hex"
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func (b *IndexerMgr) PutKVs(kvs []*common.KeyValue) (error) {

	wb := b.kvDB.NewWriteBatch()
	defer wb.Cancel()

	for _, value := range kvs {
		key := value.Key

		// 目前仅允许内置的pubkey
		pkStr := hex.EncodeToString(value.PubKey)
		if pkStr != common.BootstrapPubKey && pkStr != common.CoreNodePubKey {
			common.Log.Errorf("unsupport pubkey")
			return fmt.Errorf("unsupport pubkey")
		}

		// verify the signature
		err := common.VerifySignOfMessage(value.Value, value.Signature, value.PubKey)
		if err != nil {
			common.Log.Errorf("verify signature failed")
			return fmt.Errorf("verify signature failed, %v", err)
		}

		err = db.SetDB([]byte(key), value, wb)
		if err != nil {
			common.Log.Errorf("setting key %s failed, %v", key, err)
			return err
		}
	}

	err := wb.Flush()
	if err != nil {
		common.Log.Errorf("flushing writes to db %v", err)
		return err
	}

	return nil
}


func (b *IndexerMgr) DelKVs(keys []string) (error) {
	
	wb := b.kvDB.NewWriteBatch()
	defer wb.Cancel()

	for _, value := range keys {
		key := value
		err := wb.Delete([]byte(key))
		if err != nil {
			common.Log.Errorf("deleting key %s failed, %v", key, err)
			return err
		}
	}

	err := wb.Flush()
	if err != nil {
		common.Log.Errorf("flushing writes to db %v", err)
		return err
	}

	return nil
}


func (b *IndexerMgr) GetKVs(keys []string) ([]*common.KeyValue, error) {
	
	result := make([]*common.KeyValue, 0)
	b.kvDB.View(func(txn *badger.Txn) error {
		for _, value := range keys {
			key := value
	
			item, err := txn.Get([]byte(key))
			if err != nil {
				continue
			}
			var value common.KeyValue
			err = item.Value(func(v []byte) error {
				return db.DecodeBytes(v, &value)
			})
	
			result = append(result, &value)
		}
		return nil
	})
	
	return result, nil
}

