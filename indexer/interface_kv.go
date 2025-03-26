package indexer

import (
	"encoding/hex"
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func (b *IndexerMgr) PutKVs(kvs []*common.KeyValue) ([]string, error) {

	wb := b.kvDB.NewWriteBatch()
	defer wb.Cancel()

	result := make([]string, 0)
	for _, value := range kvs {
		key := value.Key

		// 目前仅允许内置的pubkey
		pkStr := hex.EncodeToString(value.PubKey)
		if pkStr != common.BootstrapPubKey && pkStr != common.CoreNodePubKey {
			return nil, fmt.Errorf("unsupport pubkey")
		}

		// verify the signature
		err := common.VerifySignOfMessage(value.Value, value.Signature, value.PubKey)
		if err != nil {
			common.Log.Errorf("verify signature failed")
			return nil, fmt.Errorf("verify signature failed, %v", err)
		}

		err = db.SetDB([]byte(key), value, wb)
		if err != nil {
			return nil, err
		}
	}

	err := wb.Flush()
	if err != nil {
		common.Log.Errorf("flushing writes to db %v", err)
		return nil, err
	}

	return result, nil
}


func (b *IndexerMgr) DelKVs(keys []string) ([]string, error) {
	
	wb := b.kvDB.NewWriteBatch()
	defer wb.Cancel()

	result := make([]string, 0)
	for _, value := range keys {
		key := value
		err := wb.Delete([]byte(key))
		if err != nil {
			continue
		}
		result = append(result, key)
	}

	err := wb.Flush()
	if err != nil {
		common.Log.Errorf("flushing writes to db %v", err)
		return nil, err
	}

	return result, nil
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

