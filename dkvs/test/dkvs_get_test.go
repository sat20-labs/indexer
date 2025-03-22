package test

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"testing"

	ic "github.com/libp2p/go-libp2p/core/crypto"
	dkvs "github.com/sat20-labs/indexer/dkvs"
)

func hash(key string) (hashKey string) {
	shaHash := sha512.Sum384([]byte(key))
	hashKey = hex.EncodeToString(shaHash[:])
	return
}

func bytesToHexString(input []byte) string {
	hexString := "0x"
	for _, b := range input {
		hexString += fmt.Sprintf("%02x", b)
	}
	return hexString
}

func TestDkvsGetKeyFromOtherNode(t *testing.T) {
	//relayAddr := "/ip4/156.251.179.31/tcp/9000/p2p/12D3KooWSYLNGkmanka9QS7kV5CS8kqLZBT2PUwxX7WqL63jnbGx"

	cfg := dkvs.NewDefaultDkvsConfig()
	cfg.InitMode(dkvs.LightMode)
	kv := dkvs.NewDkvs(cfg)

	// select {
	// case <-time.After(30 * time.Second):
	// 	fmt.Println("Timeout occurred")
	// }
	seed := "oIBBgepoPyhdJTYB" //dkvs.RandString(16)
	priv, err := dkvs.GetPriKeyBySeed(seed)
	if err != nil {
		t.Fatal(err)
	}
	pubKey := priv.GetPublic()
	pkBytes, err := ic.MarshalPublicKey(pubKey)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("seed: ", seed)
	fmt.Println("pubkey: ", bytesToHexString(pkBytes))

	tKey := "/" + dkvs.PUBSERVICE_DAUTH + "/" + hash("dkvs-pk001-0022")
	tValue1 := []byte("world1")
	tValue2 := []byte("mtv2")
	tValue3 := []byte("mtv3")
	tValue4 := []byte("mtv4")
	// select {
	// case <-time.After(30 * time.Second):
	// 	fmt.Println("Timeout occurred")
	// }
	value, _, _, _, _, err := kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue1) {
		t.Fatal(err)
	}

	value, _, _, _, _, err = kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue2) {
		t.Fatal(err)
	}

	value, _, _, _, _, err = kv.Get(tKey)
	if err != nil || bytes.Equal(value, tValue3) {
		t.Fatal(err)
	}

	value, _, _, _, _, err = kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue4) {
		t.Fatal(err)
	}

	// use a pubkey as key
	tKey = "/" + dkvs.PUBSERVICE_DAUTH + "/" + bytesToHexString(pkBytes)

	value, _, _, _, _, err = kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue4) {
		t.Fatal(err)
	}

}

func TestGetUnsyncedKeyFromOtherNode(t *testing.T) {
	cfg := dkvs.NewDefaultDkvsConfig()
	cfg.InitMode(dkvs.LightMode)
	kv := dkvs.NewDkvs(cfg)

	seed := "oIBBgepoPyhdJTYB" //dkvs.RandString(16)
	priv, err := dkvs.GetPriKeyBySeed(seed)
	if err != nil {
		t.Fatal(err)
	}
	pubKey := priv.GetPublic()
	pkBytes, err := ic.MarshalPublicKey(pubKey)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("seed: ", seed)
	fmt.Println("pubkey: ", bytesToHexString(pkBytes))

	tKey := "/" + dkvs.PUBSERVICE_DAUTH + "/" + hash("dkvs-usync7-001")
	tValue1 := []byte("world2")

	value, _, _, _, _, err := kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue1) {
		t.Fatal(err)
	}

	tKey = "/" + dkvs.PUBSERVICE_DAUTH + "/" + hash("dkvs-usync7-002")
	value, _, _, _, _, err = kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue1) {
		t.Fatal(err)
	}

	tKey = "/" + dkvs.PUBSERVICE_DAUTH + "/" + hash("dkvs-usync7-003")
	value, _, _, _, _, err = kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue1) {
		t.Fatal(err)
	}

}
