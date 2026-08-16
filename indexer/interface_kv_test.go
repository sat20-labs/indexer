package indexer

import (
	"strings"
	"testing"
	"time"

	"github.com/sat20-labs/indexer/common"
)

func TestValidateKVWriteRequestLimitsDistinctKeysPerPubKey(t *testing.T) {
	pubkey := []byte{1, 2, 3}
	values := make([]*common.KeyValue, maxKVKeysPerPubKey)
	for i := range values {
		values[i] = &common.KeyValue{Key: string(rune(i)), PubKey: pubkey}
	}

	if _, err := validateKVWriteRequest(values); err != nil {
		t.Fatalf("expected %d keys to be accepted: %v", maxKVKeysPerPubKey, err)
	}

	values = append(values, &common.KeyValue{Key: "overflow", PubKey: pubkey})
	if _, err := validateKVWriteRequest(values); err == nil {
		t.Fatal("expected request with more than 128 keys to be rejected")
	}
}

func TestValidateKVWriteRequestLimitsDataSize(t *testing.T) {
	pubkey := []byte{1, 2, 3}
	tooLargeValue := &common.KeyValue{
		Key:    "large",
		Value:  []byte(strings.Repeat("x", maxKVValueBytes+1)),
		PubKey: pubkey,
	}
	if _, err := validateKVWriteRequest([]*common.KeyValue{tooLargeValue}); err == nil {
		t.Fatal("expected an oversized value to be rejected")
	}

	values := make([]*common.KeyValue, 11)
	for i := range values {
		values[i] = &common.KeyValue{
			Key:    string(rune(i)),
			Value:  []byte(strings.Repeat("x", maxKVValueBytes)),
			PubKey: pubkey,
		}
	}
	if _, err := validateKVWriteRequest(values); err == nil {
		t.Fatal("expected an oversized aggregate request to be rejected")
	}
}

func TestRegistrationFreshUsesForwardAge(t *testing.T) {
	now := time.Unix(1_000_000, 0).Unix()
	inside := now - int64(supportedKeyGracePeriod.Seconds()) + 1
	expired := now - int64(supportedKeyGracePeriod.Seconds())

	if !isRegistrationFresh(inside, now) {
		t.Fatal("registration inside grace period should be fresh")
	}
	if isRegistrationFresh(expired, now) {
		t.Fatal("registration at grace-period boundary should be expired")
	}
	if isRegistrationFresh(now+1, now) {
		t.Fatal("future registration timestamp should not be accepted")
	}
}
