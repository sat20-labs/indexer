package indexer

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

const (
	maxKVKeysPerPubKey = 128
	maxKVValueBytes    = 200 * 1024
	maxKVRequestBytes  = 2 * 1024 * 1024
)

var errKVKeyLimitReached = errors.New("KV key limit reached")

const supportedKeyGracePeriod = 7 * 24 * time.Hour

func isRegistrationFresh(refreshTime, now int64) bool {
	age := now - refreshTime
	return age >= 0 && age < int64(supportedKeyGracePeriod.Seconds())
}

type RegisterPubKeyInfo struct {
	PubKey      []byte
	ChannelAddr string
	RefreshTime int64
}

func getKvKey(pubkey string, key string) string {
	return fmt.Sprintf("/%s/%s", pubkey, key)
}

func getRegisterKey(pubkey string) string {
	return fmt.Sprintf("/register/%s", pubkey)
}

func (b *IndexerMgr) IsSupportedKey(pubkey []byte) bool {
	b.rpcEnter()
	defer b.rpcLeft()

	return b.isSupportedKey(pubkey)
}

// isSupportedKey is the internal form for callers that already hold an RPC
// admission token. Keeping the gate at the public boundary avoids nested
// rpcEnter deadlocks during DB reload admission changes.
func (b *IndexerMgr) isSupportedKey(pubkey []byte) bool {
	indexerPubKey, err := hex.DecodeString(b.indexerPubKey())
	if err == nil && bytes.Equal(pubkey, indexerPubKey) {
		return true
	}

	pkStr := hex.EncodeToString(pubkey)
	if pkStr == common.GetBootstrapPubKey() || pkStr == common.GetCoreNodePubKey() {
		return true
	}

	// TODO 如果是注册的矿机，检查通道地址上的资产，和刷新时间
	key := getRegisterKey(pkStr)
	var value RegisterPubKeyInfo

	err = db.GobGetDB([]byte(key), &value, b.kvDB)
	if err != nil {
		common.Log.Infof("GobGetDB %s failed, %v", key, err)
		return false
	}
	// Preserve the original authorization semantics. GetAssetSummaryInAddress
	// now uses internal name-index helpers, so it is safe to call while the
	// outer RPC admission token is held.
	assets := b.GetAssetSummaryInAddress(value.ChannelAddr)
	if len(assets) != 0 {
		return true
	}

	// 如果没有资产，是否超时？
	return isRegistrationFresh(value.RefreshTime, time.Now().Unix())
}

func (b *IndexerMgr) PutKVs(kvs []*common.KeyValue) error {
	b.rpcEnter()
	defer b.rpcLeft()
	b.kvMutex.Lock()
	defer b.kvMutex.Unlock()

	keysByPubKey, err := validateKVWriteRequest(kvs)
	if err != nil {
		return err
	}
	indexerPubKey, err := hex.DecodeString(b.indexerPubKey())
	if err != nil {
		return fmt.Errorf("invalid indexer pubkey: %w", err)
	}
	for _, value := range kvs {
		if !bytes.Equal(value.PubKey, indexerPubKey) {
			return fmt.Errorf("only indexer pubkey may write KV")
		}
	}
	if err := b.ensureKVKeyQuota(keysByPubKey); err != nil {
		return err
	}

	wb := b.kvDB.NewWriteBatch()
	defer wb.Close()

	for _, value := range kvs {
		pkStr := hex.EncodeToString(value.PubKey)

		if len(value.Value) > maxKVValueBytes {
			return fmt.Errorf("too large data %d", len(value.Value))
		}

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

	err = wb.Flush()
	if err != nil {
		common.Log.Errorf("flushing writes to db %v", err)
		return err
	}

	return nil
}

func validateKVWriteRequest(kvs []*common.KeyValue) (map[string]map[string]struct{}, error) {
	if len(kvs) == 0 {
		return nil, fmt.Errorf("empty KV request")
	}
	if len(kvs) > maxKVKeysPerPubKey {
		return nil, fmt.Errorf("too many values in one request: %d (max %d)", len(kvs), maxKVKeysPerPubKey)
	}

	keysByPubKey := make(map[string]map[string]struct{})
	totalBytes := 0
	for _, value := range kvs {
		if value == nil {
			return nil, fmt.Errorf("nil KV value")
		}
		if len(value.Value) > maxKVValueBytes {
			return nil, fmt.Errorf("too large data %d", len(value.Value))
		}

		totalBytes += len(value.Key) + len(value.Value) + len(value.PubKey) + len(value.Signature)
		if totalBytes > maxKVRequestBytes {
			return nil, fmt.Errorf("KV request too large (max %d bytes)", maxKVRequestBytes)
		}

		pkStr := hex.EncodeToString(value.PubKey)
		if _, ok := keysByPubKey[pkStr]; !ok {
			keysByPubKey[pkStr] = make(map[string]struct{})
		}
		keysByPubKey[pkStr][value.Key] = struct{}{}
	}

	for pubkey, keys := range keysByPubKey {
		if len(keys) > maxKVKeysPerPubKey {
			return nil, fmt.Errorf("too many distinct keys for pubkey %s: %d (max %d)", pubkey, len(keys), maxKVKeysPerPubKey)
		}
	}
	return keysByPubKey, nil
}

func (b *IndexerMgr) ensureKVKeyQuota(keysByPubKey map[string]map[string]struct{}) error {
	for pubkey, keys := range keysByPubKey {
		existingKeys, err := b.countKVKeys(pubkey)
		if err != nil {
			return err
		}

		newKeys := 0
		for key := range keys {
			_, err := b.kvDB.Read([]byte(getKvKey(pubkey, key)))
			if err == nil {
				continue
			}
			if !errors.Is(err, common.ErrKeyNotFound) {
				return fmt.Errorf("checking existing KV key failed: %w", err)
			}
			newKeys++
		}

		// Legacy data may predate this limit. Allow existing keys to be
		// updated, but never permit another key to be added.
		if existingKeys > maxKVKeysPerPubKey && newKeys == 0 {
			continue
		}
		if existingKeys+newKeys > maxKVKeysPerPubKey {
			return fmt.Errorf("KV key limit exceeded for pubkey %s: %d existing, %d new, max %d", pubkey, existingKeys, newKeys, maxKVKeysPerPubKey)
		}
	}
	return nil
}

func (b *IndexerMgr) countKVKeys(pubkey string) (int, error) {
	count := 0
	err := b.kvDB.BatchRead([]byte(getKvKey(pubkey, "")), false, func(_, _ []byte) error {
		count++
		if count > maxKVKeysPerPubKey {
			return errKVKeyLimitReached
		}
		return nil
	})
	if errors.Is(err, errKVKeyLimitReached) {
		return count, nil
	}
	if err != nil {
		return 0, fmt.Errorf("counting KV keys failed: %w", err)
	}
	return count, nil
}

func (b *IndexerMgr) GetKVs(pubkey []byte, keys []string) ([]*common.KeyValue, error) {
	b.rpcEnter()
	defer b.rpcLeft()

	pkStr := hex.EncodeToString(pubkey)
	result := make([]*common.KeyValue, 0)

	for _, k := range keys {
		key := getKvKey(pkStr, k)

		item, err := b.kvDB.Read([]byte(key))
		if err != nil {
			continue
		}
		var value common.KeyValue

		err = db.DecodeBytes(item, &value)
		if err != nil {
			common.Log.Errorf("decoding key %s failed, %v", key, err)
			continue
		}

		result = append(result, &value)
	}

	return result, nil
}

func (b *IndexerMgr) GetIndexerPubKey() string {
	b.rpcEnter()
	defer b.rpcLeft()

	return b.indexerPubKey()
}

func (b *IndexerMgr) indexerPubKey() string {
	if b.cfg.PubKey != "" {
		return b.cfg.PubKey
	}
	return common.GetBootstrapPubKey()
}
