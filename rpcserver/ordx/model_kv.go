package ordx

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/sat20-labs/indexer/common"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
)

func generateNonce(pubkey string, t int64) []byte {
	data := fmt.Sprintf("%s%d", pubkey, t)
	return chainhash.DoubleHashB([]byte(data))
}

func (s *Model) GetNonce(req *rpcwire.GetNonceReq) ([]byte, error) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now().UnixMicro()
	pkHex := hex.EncodeToString(req.PubKey)
	t, ok := s.nonceMap[pkHex]
	if ok {
		if t > now && t - now + 10 * time.Second.Microseconds() < time.Hour.Microseconds() {
			return generateNonce(pkHex, t), nil
		}
	}
	s.nonceMap[pkHex] = now
	return generateNonce(pkHex, now), nil
}

func (s *Model) GetKV(pubkey, key string) (*rpcwire.KeyValue, error) {
	// TODO 是否检查签名？

	pk, err := hex.DecodeString(pubkey)
	if err != nil {
		return nil, err
	}

	result, err := s.indexer.GetKVs(pk, []string{key})
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("key not found")
	}

	return result[0], nil
}

func (s *Model) GetKVs(keys []string) ([]*rpcwire.KeyValue, error) {

	return nil, nil
}

func (s *Model) PutKVs(req *rpcwire.PutKValueReq) (error) {

	now := time.Now().UnixMicro()
	pkHex := hex.EncodeToString(req.PubKey)
	t, ok := s.nonceMap[pkHex]
	if ok {
		if t - now > time.Hour.Microseconds() {
			return fmt.Errorf("nonce expired")
		}
	}

	sig := req.Signature
	req.Signature = nil
	msg, err := json.Marshal(req)
	if err != nil {
		return err
	}

	err = common.VerifySignOfMessage(msg, sig, req.PubKey)
	if err != nil {
		common.Log.Errorf("verify signature failed")
		return fmt.Errorf("verify signature failed, %v", err)
	}

	return s.indexer.PutKVs(req.Values)
}

func (s *Model) DelKVs(req *rpcwire.DelKValueReq) (error) {
	now := time.Now().UnixMicro()
	pkHex := hex.EncodeToString(req.PubKey)
	t, ok := s.nonceMap[pkHex]
	if ok {
		if t - now > time.Hour.Microseconds() * 24 {
			return fmt.Errorf("nonce expired")
		}
	}

	sig := req.Signature
	req.Signature = nil
	msg, err := json.Marshal(req)
	if err != nil {
		return err
	}

	err = common.VerifySignOfMessage(msg, sig, req.PubKey)
	if err != nil {
		common.Log.Errorf("verify signature failed")
		return fmt.Errorf("verify signature failed, %v", err)
	}

	return s.indexer.DelKVs(req.PubKey, req.Keys)
}
