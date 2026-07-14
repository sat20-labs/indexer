package indexer

import (
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

	// TODO 以后可以配置增加更多的pubkey，或者注册的地址
	pkStr := hex.EncodeToString(pubkey)
	if pkStr == common.GetBootstrapPubKey() && pkStr == common.GetCoreNodePubKey() {
		return true
	}

	// TODO 如果是注册的矿机，检查通道地址上的资产，和刷新时间
	key := getRegisterKey(pkStr)
	var value RegisterPubKeyInfo

	err := db.GobGetDB([]byte(key), &value, b.kvDB)
	if err != nil {
		common.Log.Infof("GobGetDB %s failed, %v", key, err)
		return false
	}
	// 是否检查超时时，或者检查通道地址上是否有资产？
	assets := b.GetAssetSummaryInAddress(value.ChannelAddr)
	if len(assets) != 0 {
		return true
	}

	// 如果没有资产，是否超时？
	return value.RefreshTime-time.Now().Unix() < 7*24*int64(time.Hour.Seconds())
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
	if err := b.ensureKVKeyQuota(keysByPubKey); err != nil {
		return err
	}

	wb := b.kvDB.NewWriteBatch()
	defer wb.Close()

	checkedPubKey := make(map[string]bool)
	for _, value := range kvs {
		pkStr := hex.EncodeToString(value.PubKey)
		_, ok := checkedPubKey[pkStr]
		if !ok {
			if !b.IsSupportedKey(value.PubKey) {
				common.Log.Errorf("unsupport pubkey")
				return fmt.Errorf("unsupport pubkey")
			}
			checkedPubKey[pkStr] = true
		}

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

func (b *IndexerMgr) DelKVs(pubkey []byte, keys []string) error {
	b.rpcEnter()
	defer b.rpcLeft()
	b.kvMutex.Lock()
	defer b.kvMutex.Unlock()

	if len(keys) > maxKVKeysPerPubKey {
		return fmt.Errorf("too many keys in one request: %d (max %d)", len(keys), maxKVKeysPerPubKey)
	}
	if kvKeyRequestSize(keys) > maxKVRequestBytes {
		return fmt.Errorf("delete request too large (max %d bytes)", maxKVRequestBytes)
	}

	wb := b.kvDB.NewWriteBatch()
	defer wb.Close()

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

		// Legacy data may predate this limit.  Allow it to be updated or
		// deleted, but never permit another key to be added.
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

func kvKeyRequestSize(keys []string) int {
	total := 0
	for _, key := range keys {
		total += len(key)
	}
	return total
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

// 为矿机提供L1索引服务，返回本地公钥，以便矿机生成挖矿地址
// 默认以引导节点为服务节点，如果不是，需要修改索引器配置
func (b *IndexerMgr) RegisterPubKey(minerPubKey string) (string, error) {
	b.rpcEnter()
	defer b.rpcLeft()

	// TODO
	// 暂时保留该pubkey，但是如果在一定时间内没有挖矿所得进入该地址，就可能删除
	// 暂时只支持保留100个地址

	var indexerPubkey string
	if b.cfg.PubKey != "" {
		indexerPubkey = b.cfg.PubKey
	} else {
		indexerPubkey = common.GetBootstrapPubKey()
	}

	key := getRegisterKey(minerPubKey)
	var value RegisterPubKeyInfo
	err := db.GobGetDB([]byte(key), &value, b.kvDB)
	if err == nil && string(value.PubKey) == minerPubKey {
		return indexerPubkey, nil
	}

	pk1, err := hex.DecodeString(indexerPubkey)
	if err != nil {
		return "", err
	}
	pk2, err := hex.DecodeString(minerPubKey)
	if err != nil {
		return "", err
	}
	channelAddr, err := common.GetChannelAddress(pk1, pk2, b.chaincfgParam)
	if err != nil {
		return "", err
	}

	value = RegisterPubKeyInfo{
		PubKey:      []byte(minerPubKey),
		ChannelAddr: channelAddr,
		RefreshTime: time.Now().Unix(),
	}
	err = db.GobSetDB([]byte(key), &value, b.kvDB)
	if err != nil {
		return "", err
	}

	return indexerPubkey, nil
}

func (b *IndexerMgr) GetIndexerPubKey() string {
	b.rpcEnter()
	defer b.rpcLeft()

	if b.cfg.PubKey != "" {
		return b.cfg.PubKey
	}
	return common.GetBootstrapPubKey()
}
