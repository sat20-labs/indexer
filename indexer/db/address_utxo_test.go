package db

import (
	"bytes"
	"testing"
)

func TestAddressUtxoCountKeyRoundTrip(t *testing.T) {
	const addressID = uint64(0x1020304050607080)
	key := GetAddressUtxoCountKey(addressID)
	if !bytes.HasPrefix(key, GetAddressUtxoCountPrefix()) {
		t.Fatalf("count key %x missing prefix %x", key, GetAddressUtxoCountPrefix())
	}
	got, err := ParseAddressUtxoCountKey(key)
	if err != nil || got != addressID {
		t.Fatalf("ParseAddressUtxoCountKey=%x,%v want %x", got, err, addressID)
	}
}

func TestAddressUtxoCountValueRoundTrip(t *testing.T) {
	const want = uint64(987654321)
	got, err := DecodeAddressUtxoCount(EncodeAddressUtxoCount(want))
	if err != nil || got != want {
		t.Fatalf("count=%d,%v want %d", got, err, want)
	}
}
