package test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	ic "github.com/libp2p/go-libp2p/core/crypto"

	dkvs "github.com/sat20-labs/indexer/dkvs"
	dkvs_pb "github.com/sat20-labs/indexer/dkvs/pb"
)

func TestDkvs(t *testing.T) {
	//relayAddr := "/ip4/156.251.179.31/tcp/9000/p2p/12D3KooWSYLNGkmanka9QS7kV5CS8kqLZBT2PUwxX7WqL63jnbGx"

	
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

	tKey := "/" + dkvs.PUBSERVICE_DAUTH + "/" + hash("dkvs-k001-aa28")
	tValue1 := []byte("world1")
	tValue2 := []byte("mtv2")
	tValue3 := []byte("mtv3")
	tValue4 := []byte("mtv4")
	ttl := dkvs.GetTtlFromDuration(time.Hour)
	issuetime := dkvs.TimeNow()

	data := dkvs.GetRecordSignData(tKey, tValue1, pkBytes, issuetime, ttl)
	sigData1, err := priv.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(tKey, tValue1, pkBytes, issuetime, ttl, sigData1)
	if err != nil {
		t.Fatal(err)
	}

	data = dkvs.GetRecordSignData(tKey, tValue2, pkBytes, issuetime, ttl)
	sigData2, err := priv.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(tKey, tValue2, pkBytes, issuetime, ttl, sigData2)
	if err != nil {
		t.Fatal(err)
	}
	value, _, _, _, sign, err := kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue2) || !bytes.Equal(sign, sigData2) {
		t.Fatal(err)
	}

	// sigData3, err := priv.Sign(tValue3)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// err = dkvs.Put(tKey, tValue3, pkBytes, issuetime, ttl, sigData3)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	priv2, err := dkvs.GetPriKeyBySeed("mtv3")
	if err != nil {
		t.Fatal(err)
	}
	pubKey2 := priv2.GetPublic()
	pkBytes2, err := ic.MarshalPublicKey(pubKey2)
	if err != nil {
		t.Fatal(err)
	}

	data = dkvs.GetRecordSignData(tKey, tValue3, pkBytes2, issuetime, ttl)
	sigData3, err := priv2.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(tKey, tValue3, pkBytes2, issuetime, ttl, sigData3)
	if err != nil {
		fmt.Println(err)
	} else {
		t.Fatal(err)
	}
	value, _, _, _, sign, err = kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue2) || !bytes.Equal(sign, sigData2) {
		t.Fatal(err)
	}

	data = dkvs.GetRecordSignData(tKey, tValue4, pkBytes, issuetime, ttl)
	sigData4, err := priv.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(tKey, tValue4, pkBytes, issuetime, ttl, sigData4)
	if err != nil {
		t.Fatal(err)
	}

	value, _, _, _, sign, err = kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue4) || !bytes.Equal(sign, sigData4) {
		t.Fatal(err)
	}

	// use a pubkey as key
	tKey = "/" + dkvs.PUBSERVICE_DAUTH + "/" + bytesToHexString(pkBytes)
	data = dkvs.GetRecordSignData(tKey, tValue4, pkBytes2, issuetime, ttl)
	sigData5, err := priv2.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(tKey, tValue4, pkBytes2, issuetime, ttl, sigData5)
	if err == nil {
		t.Fatal(err)
	}

	data = dkvs.GetRecordSignData(tKey, tValue4, pkBytes, issuetime, ttl)
	sigData6, err := priv.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(tKey, tValue4, pkBytes, issuetime, ttl, sigData6)
	if err != nil {
		t.Fatal(err)
	}

	// node.Stop()
}


func TestGun(t *testing.T) {

	cfg := dkvs.NewDefaultDkvsConfig()
	cfg.InitMode(dkvs.LightMode)
	kv := dkvs.NewDkvs(cfg)

	seed := "zjMGsKesWSlZnayK" //dkvs.RandString(16)
	gunPrivKey, e := dkvs.GetPriKeyBySeed(seed)
	if e != nil {
		t.Fatal()
	}
	fmt.Println("seed ", seed)

	gunPubKey, err := ic.MarshalPublicKey(gunPrivKey.GetPublic())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("public key: ", bytesToHexString(gunPubKey))
	fmt.Println("public key length: ", len(bytesToHexString(gunPubKey)))

	// Gun service apply a gun name
	name := dkvs.RandString(8)
	for {
		if !kv.Has(name) {
			break
		}
		name = dkvs.RandString(8)
	}
	key := dkvs.GetGunKey(name)
	fmt.Println("name ", name)

	issueTime := dkvs.TimeNow()
	ttl := dkvs.GetTtlFromDuration(10 * time.Hour)
	issuetime := dkvs.TimeNow()
	data1 := dkvs.GetGunSignData(name, 1, gunPubKey, issueTime, ttl)
	sign1, err := gunPrivKey.Sign(data1)
	if err != nil {
		t.Fatal(err)
	}
	gunvalue := dkvs.EncodeGunValue(name, 1, issueTime, ttl, gunPubKey, sign1, nil)
	newgunvalue := dkvs.EncodeGunValue(name, 1, issueTime, ttl, gunPubKey, sign1, []byte("user data"))

	data := dkvs.GetRecordSignData(key, gunvalue, gunPubKey, issuetime, ttl)
	sigData1, err := gunPrivKey.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put((key), gunvalue, gunPubKey, issuetime, ttl, sigData1)
	if err != nil {
		t.Fatal(err)
	}

	// prevent others to apply gun name directly
	privA, err := dkvs.GetPriKeyBySeed("AAA")
	if err != nil {
		t.Fatal(err)
	}

	pubKeyA, err := ic.MarshalPublicKey(privA.GetPublic())
	if err != nil {
		t.Fatal(err)
	}

	name2 := ("gun" + dkvs.RandString(4))
	key2 := dkvs.GetGunKey(name2)
	tValue1 := []byte("world1")

	data = dkvs.GetRecordSignData(key2, tValue1, pubKeyA, issuetime, ttl)
	sigData1, err = privA.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(key2, tValue1, pubKeyA, issuetime, ttl, sigData1)
	if err == nil {
		t.Fatal(err)
	}

	// gun service transfer a name to A
	err = testTransfer(kv, key, gunvalue, gunPrivKey, issuetime, ttl, privA)
	if err != nil {
		t.Fatal(err)
	}

	// a GUN record can't be changed to a incorrect format record
	newvalue := []byte("new value")
	data = dkvs.GetRecordSignData(key, newvalue, pubKeyA, issuetime, ttl)
	sigDataA2, err := privA.Sign(data)
	if err != nil {
		t.Fatal(err)
	}

	err = kv.Put(key, newvalue, pubKeyA, issuetime, ttl, sigDataA2)
	if err == nil {
		t.Fatal(err)
	}

	newttl := ttl + 1
	gunvalue2 := dkvs.EncodeGunValue(name, 1, issueTime, newttl, gunPubKey, sign1, nil)
	data = dkvs.GetRecordSignData(key, gunvalue2, pubKeyA, issuetime, newttl)
	sigDataA3, err := privA.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(key, gunvalue, pubKeyA, issuetime, newttl, sigDataA3)
	if err == nil {
		t.Fatal(err)
	}

	// a GUN record can be changed to a correct record
	newData := dkvs.GetRecordSignData(key, newgunvalue, pubKeyA, issuetime, ttl)
	sigDataA4, err := privA.Sign(newData)
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(key, newgunvalue, pubKeyA, issuetime, ttl, sigDataA4)
	if err != nil {
		t.Fatal(err)
	}

	// the name can put another pubkey
	name22 := "/" + name + "/" + bytesToHexString(pubKeyA)
	tValue22 := []byte("world1")
	data22 := dkvs.GetRecordSignData(name22, tValue22, pubKeyA, issuetime, ttl)
	sigData22, err := privA.Sign(data22)
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(name22, tValue22, pubKeyA, issuetime, ttl, sigData22)
	if err != nil {
		t.Fatal(err)
	}

	// a gun record can be transfered.
	privB, err := dkvs.GetPriKeyBySeed("BBB")
	if err != nil {
		t.Fatal(err)
	}

	// transfer fail，只能手工测试，条件太难配置
	// err = testTransferRestore(*kv, name, gunvalue, privA, issuetime, ttl, privB)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// transfer succ
	err = testTransfer(kv, key, newgunvalue, privA, issuetime, ttl, privB)
	if err != nil {
		t.Fatal(err)
	}

	privC, err := dkvs.GetPriKeyBySeed("CCC")
	if err != nil {
		t.Fatal(err)
	}

	// A can't transfer a non-owned name to B
	err = testTransfer(kv, key, gunvalue, privA, issuetime, ttl, privC)
	if err == nil {
		t.Fatal(err)
	}

}

func TestTransferKey(t *testing.T) {

	cfg := dkvs.NewDefaultDkvsConfig()
	cfg.InitMode(dkvs.LightMode)
	kv := dkvs.NewDkvs(cfg)

	seed := dkvs.RandString(16)
	PrivKey1, e := dkvs.GetPriKeyBySeed(seed)
	if e != nil {
		t.Fatal()
	}
	fmt.Println("seed ", seed)

	PubKey1, err := ic.MarshalPublicKey(PrivKey1.GetPublic())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("public key: ", bytesToHexString(PubKey1))
	fmt.Println("public key length: ", len(bytesToHexString(PubKey1)))

	// apply a name
	name := dkvs.RandString(64)
	for {
		if !kv.Has(name) {
			break
		}
		name = dkvs.RandString(64)
	}
	key := "/contract/" + (name)
	fmt.Println("key ", key)

	issueTime := dkvs.TimeNow()
	ttl := dkvs.GetTtlFromDuration(10 * time.Hour)
	issuetime := dkvs.TimeNow()
	data1 := dkvs.GetGunSignData(name, 1, PubKey1, issueTime, ttl)
	sign1, err := PrivKey1.Sign(data1)
	if err != nil {
		t.Fatal(err)
	}
	gvalue := dkvs.EncodeGunValue(name, 1, issueTime, ttl, PubKey1, sign1, nil)

	data := dkvs.GetRecordSignData(key, gvalue, PubKey1, issuetime, ttl)
	sigData1, err := PrivKey1.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put((key), gvalue, PubKey1, issuetime, ttl, sigData1)
	if err != nil {
		t.Fatal(err)
	}

	privA, err := dkvs.GetPriKeyBySeed("AAA")
	if err != nil {
		t.Fatal(err)
	}

	//  transfer a name to A
	err = testTransfer(kv, key, gvalue, PrivKey1, issuetime, ttl, privA)
	if err != nil {
		t.Fatal(err)
	}

	//  can be transfered.
	privB, err := dkvs.GetPriKeyBySeed("BBB")
	if err != nil {
		t.Fatal(err)
	}

	// transfer succ
	err = testTransfer(kv, key, gvalue, privA, issuetime, ttl, privB)
	if err != nil {
		t.Fatal(err)
	}

	privC, err := dkvs.GetPriKeyBySeed("CCC")
	if err != nil {
		t.Fatal(err)
	}

	// A can't transfer a non-owned name to B
	err = testTransfer(kv, key, gvalue, privA, issuetime, ttl, privC)
	if err == nil {
		t.Fatal(err)
	}

}

func getNewRecordValue(kv *dkvs.Dkvs, key string, sk ic.PrivKey, pk []byte, cert *dkvs_pb.Cert) ([]byte, []byte, error) {

	value1, _, issuetime, ttl, _, err := kv.Get(key)
	if err != nil {
		return nil, nil, (err)
	}

	rv := dkvs.DecodeCertsRecordValue(value1)
	if rv == nil {
		return nil, nil, errors.New("DecodeCertsRecordValue fail")
	}

	rv.UserData, _ = cert.Marshal()
	value2, _ := rv.Marshal()

	data := dkvs.GetRecordSignData(key, value2, pk, issuetime, ttl)
	sign2, err := sk.Sign(data)

	return value2, sign2, err

	//err = kv.Put(key, value2, pk, issuetime, ttl, sign2)
}

func getNewRecordValue2(kv *dkvs.Dkvs, key string, sk ic.PrivKey, pk []byte, cert *dkvs_pb.Cert) ([]byte, []byte, uint64, uint64, error) {

	var rv *dkvs_pb.CertsRecordValue
	value1, _, issuetime, ttl, _, err := kv.Get(key)
	if err == nil {
		rv = dkvs.DecodeCertsRecordValue(value1)
	} else {
		rv = &dkvs_pb.CertsRecordValue{}
		issuetime = dkvs.TimeNow()
		ttl = dkvs.GetDefaultTtl()
	}

	value2, _ := dkvs.EncodeCertsRecordValueWithCert(rv, cert, nil)

	data := dkvs.GetRecordSignData(key, value2, pk, issuetime, ttl)
	sign2, err := sk.Sign(data)

	return value2, sign2, issuetime, ttl, err

	//err = kv.Put(key, value2, pk, issuetime, ttl, sign2)
}

func testTransfer(kv *dkvs.Dkvs, key string, gvalue []byte, privk1 ic.PrivKey, issuetime uint64, ttl uint64, privk2 ic.PrivKey) error {
	pubkey1, _ := ic.MarshalPublicKey(privk1.GetPublic())
	pubkey2, _ := ic.MarshalPublicKey(privk2.GetPublic())

	// B sign the record
	data := dkvs.GetRecordSignData(key, gvalue, pubkey2, issuetime, ttl)
	sign2, err := privk2.Sign(data)
	if err != nil {
		return (err)
	}

	// owner sign a transfer record
	fee := uint64(0)
	cert := dkvs.IssueCertTransferPrepare(key, fee, pubkey2, pubkey1)
	signData1 := dkvs.GetCertSignData(cert)
	if signData1 == nil {
		return (err)
	}
	sign1, err := privk1.Sign(signData1)
	if err != nil {
		return (err)
	}
	cert.IssuerSign = sign1
	//value1 := dkvs.EncodeCert(cert)
	// save cert to record
	value1, sign1, err := getNewRecordValue(kv, key, privk1, pubkey1, cert)
	if err != nil {
		return (err)
	}

	var cert2 *dkvs_pb.Cert
	if fee != 0 {
		seed := "thsgMCRQoWIPwfxJ" //dkvs.RandString(16)
		minerPrivKey, _ := dkvs.GetPriKeyBySeed(seed)
		minerPubKey, _ := ic.MarshalPublicKey(minerPrivKey.GetPublic())

		cert2 = dkvs.IssueCertTxCompleted(key, "txtxtx", fee, pubkey1, pubkey2, minerPubKey)
		signData2 := dkvs.GetCertSignData(cert2)
		if signData2 == nil {
			return (err)
		}
		sign2, err := minerPrivKey.Sign(signData2)
		if err != nil {
			return (err)
		}
		cert2.IssuerSign = sign2
	}

	// then A tranfer a name to B
	err = kv.TransferKey(key, value1, pubkey1, sign1, gvalue, pubkey2, issuetime, ttl, sign2, cert2)
	if err != nil {
		return (err)
	}

	// check
	value3, key3, issuetime3, ttl3, sign3, err3 := kv.Get(key)
	if err3 != nil {
		return (err3)
	}
	err = errors.New("new error")
	if !bytes.Equal(value3, gvalue) {
		return (err)
	}
	if !bytes.Equal(key3, pubkey2) {
		return (err)
	}
	if !bytes.Equal(sign3, sign2) {
		return (err)
	}
	if issuetime3 != issuetime {
		return err
	}
	if ttl3 != ttl {
		return (err)
	}

	return nil
}


func TestMultiWriters(t *testing.T) {
	cfg := dkvs.NewDefaultDkvsConfig()
	cfg.InitMode(dkvs.LightMode)
	kv := dkvs.NewDkvs(cfg)


	seed1 := "thsgMCRQoWIPwfxJ" //dkvs.RandString(16)
	priv1, err := dkvs.GetPriKeyBySeed(seed1)
	if err != nil {
		t.Fatal(err)
	}
	pk1, err := ic.MarshalPublicKey(priv1.GetPublic())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("pubkey1: ", bytesToHexString(pk1))

	tKey := "/" + dkvs.PUBSERVICE_MINER + "/" + hash("dkvs-k001-aa18")
	tValue1 := []byte("world1")
	tValue2 := []byte("mtv2")
	ttl := dkvs.GetTtlFromDuration(time.Hour)
	issuetime := dkvs.TimeNow()

	sigData1, err := priv1.Sign(dkvs.GetRecordSignData(tKey, tValue1, pk1, issuetime, ttl))
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(tKey, tValue1, pk1, issuetime, ttl, sigData1)
	if err != nil {
		t.Fatal(err)
	}

	seed2 := "GeFTuvVKGRhlEgrq" //dkvs.RandString(16)
	priv2, err := dkvs.GetPriKeyBySeed(seed2)
	if err != nil {
		t.Fatal(err)
	}
	pk2, err := ic.MarshalPublicKey(priv2.GetPublic())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("pubkey2: ", bytesToHexString(pk2))

	sigData2, err := priv2.Sign(dkvs.GetRecordSignData(tKey, tValue2, pk2, issuetime, ttl))
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(tKey, tValue2, pk2, issuetime, ttl, sigData2)
	if err != nil {
		t.Fatal(err)
	}
	value, _, _, _, sign, err := kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue2) || !bytes.Equal(sign, sigData2) {
		t.Fatal(err)
	}

	err = kv.Put(tKey, tValue1, pk1, issuetime, ttl, sigData1)
	if err != nil {
		t.Fatal(err)
	}
	value, _, _, _, sign, err = kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue1) || !bytes.Equal(sign, sigData1) {
		t.Fatal(err)
	}
}

func TestMultiWritersWithCert(t *testing.T) {
	cfg := dkvs.NewDefaultDkvsConfig()
	cfg.InitMode(dkvs.LightMode)
	kv := dkvs.NewDkvs(cfg)


	seed1 := "oIBBgepoPyhdJTYB" // dauth
	priv1, err := dkvs.GetPriKeyBySeed(seed1)
	if err != nil {
		t.Fatal(err)
	}
	pk1, err := ic.MarshalPublicKey(priv1.GetPublic())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("pubkey1: ", bytesToHexString(pk1))

	tKey := "/" + dkvs.PUBSERVICE_DAUTH + "/" + hash("dkvs-k001-aa18")
	tValue1 := []byte("world1")
	tValue2 := []byte("mtv2")
	tValue3 := []byte("hope3")
	ttl := dkvs.GetTtlFromDuration(time.Hour)
	issuetime := dkvs.TimeNow()

	sigData1, err := priv1.Sign(dkvs.GetRecordSignData(tKey, tValue1, pk1, issuetime, ttl))
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(tKey, tValue1, pk1, issuetime, ttl, sigData1)
	if err != nil {
		t.Fatal(err)
	}

	seed2 := dkvs.RandString(16)
	priv2, err := dkvs.GetPriKeyBySeed(seed2)
	if err != nil {
		t.Fatal(err)
	}
	pk2, err := ic.MarshalPublicKey(priv2.GetPublic())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("pubkey2: ", bytesToHexString(pk2))

	// apply a cert from dauth
	cert := dkvs.IssueCertApprove(dkvs.PUBSERVICE_DAUTH, pk2, pk1, ttl)
	signData1 := dkvs.GetCertSignData(cert)
	if signData1 == nil {
		t.Fatal()
	}
	sign1, err := priv1.Sign(signData1)
	if err != nil {
		t.Fatal(err)
	}
	cert.IssuerSign = sign1
	addr := dkvs.GetCertAddr(dkvs.PUBSERVICE_DAUTH, pk2)
	certvalue1, certsign1, issuetime2, ttl2, err := getNewRecordValue2(kv, addr, priv1, pk1, cert)
	if err != nil {
		t.Fatal(err)
	}

	err = kv.Put(addr, certvalue1, pk1, issuetime2, ttl2, certsign1)
	if err != nil {
		t.Fatal(err)
	}

	// pk2 try to overwrite the key
	sigData2, err := priv2.Sign(dkvs.GetRecordSignData(tKey, tValue2, pk2, issuetime, ttl))
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(tKey, tValue2, pk2, issuetime, ttl, sigData2)
	if err != nil {
		t.Fatal(err)
	}
	value, pubkey, _, _, sign, err := kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue2) || !bytes.Equal(sign, sigData2) || !bytes.Equal(pk2, pubkey) {
		t.Fatal(err)
	}

	err = kv.Put(tKey, tValue1, pk1, issuetime, ttl, sigData1)
	if err != nil {
		t.Fatal(err)
	}
	value, pubkey, _, _, sign, err = kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue1) || !bytes.Equal(sign, sigData1) || !bytes.Equal(pk1, pubkey) {
		t.Fatal(err)
	}

	// apply a cert from dauth
	seed3 := dkvs.RandString(16)
	priv3, err := dkvs.GetPriKeyBySeed(seed3)
	if err != nil {
		t.Fatal(err)
	}
	pk3, err := ic.MarshalPublicKey(priv3.GetPublic())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("pubkey3: ", bytesToHexString(pk3))
	cert3 := dkvs.IssueCertApprove(dkvs.PUBSERVICE_DAUTH, pk3, pk1, ttl)
	signData3 := dkvs.GetCertSignData(cert3)
	if signData3 == nil {
		t.Fatal()
	}
	sign3, err := priv1.Sign(signData3)
	if err != nil {
		t.Fatal(err)
	}
	cert3.IssuerSign = sign3
	addr3 := dkvs.GetCertAddr(dkvs.PUBSERVICE_DAUTH, pk3)
	certvalue3, certsign3, issuetime3, ttl3, err := getNewRecordValue2(kv, addr3, priv1, pk1, cert3)
	if err != nil {
		t.Fatal(err)
	}

	err = kv.Put(addr3, certvalue3, pk1, issuetime3, ttl3, certsign3)
	if err != nil {
		t.Fatal(err)
	}

	sigData3, err := priv3.Sign(dkvs.GetRecordSignData(tKey, tValue3, pk3, issuetime, ttl))
	if err != nil {
		t.Fatal(err)
	}

	// three writers
	err = kv.Put(tKey, tValue3, pk3, issuetime, ttl, sigData3)
	if err != nil {
		t.Fatal(err)
	}
	value, pubkey, _, _, sign, err = kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue3) || !bytes.Equal(sign, sigData3) || !bytes.Equal(pk3, pubkey) {
		t.Fatal(err)
	}

	err = kv.Put(tKey, tValue2, pk2, issuetime, ttl, sigData2)
	if err != nil {
		t.Fatal(err)
	}
	value, pubkey, _, _, sign, err = kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue2) || !bytes.Equal(sign, sigData2) || !bytes.Equal(pk2, pubkey) {
		t.Fatal(err)
	}

	err = kv.Put(tKey, tValue3, pk3, issuetime, ttl, sigData3)
	if err != nil {
		t.Fatal(err)
	}
	value, pubkey, _, _, sign, err = kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue3) || !bytes.Equal(sign, sigData3) || !bytes.Equal(pk3, pubkey) {
		t.Fatal(err)
	}

	err = kv.Put(tKey, tValue1, pk1, issuetime, ttl, sigData1)
	if err != nil {
		t.Fatal(err)
	}
	value, pubkey, _, _, sign, err = kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue1) || !bytes.Equal(sign, sigData1) || !bytes.Equal(pk1, pubkey) {
		t.Fatal(err)
	}

	err = kv.Put(tKey, tValue3, pk3, issuetime, ttl, sigData3)
	if err != nil {
		t.Fatal(err)
	}
	value, pubkey, _, _, sign, err = kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue3) || !bytes.Equal(sign, sigData3) || !bytes.Equal(pk3, pubkey) {
		t.Fatal(err)
	}

	err = kv.Put(tKey, tValue2, pk2, issuetime, ttl, sigData2)
	if err != nil {
		t.Fatal(err)
	}
	value, pubkey, _, _, sign, err = kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue2) || !bytes.Equal(sign, sigData2) || !bytes.Equal(pk2, pubkey) {
		t.Fatal(err)
	}

	err = kv.Put(tKey, tValue3, pk3, issuetime, ttl, sigData3)
	if err != nil {
		t.Fatal(err)
	}
	value, pubkey, _, _, sign, err = kv.Get(tKey)
	if err != nil || !bytes.Equal(value, tValue3) || !bytes.Equal(sign, sigData3) || !bytes.Equal(pk3, pubkey) {
		t.Fatal(err)
	}
}

func TestDkvsTTL(t *testing.T) {
	maxttl := dkvs.GetMaxTtl()
	fmt.Printf("maxttl: %v", maxttl)

	cfg := dkvs.NewDefaultDkvsConfig()
	cfg.InitMode(dkvs.LightMode)
	kv := dkvs.NewDkvs(cfg)


	seed := "oIBBgepoPyhdJTYC"
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

	tKey := "/" + dkvs.PUBSERVICE_DAUTH + "/" + hash("dkvs-k001-aa44")
	tValue1 := []byte("world1")
	ttl := dkvs.GetTtlFromDuration(15 * time.Second)
	issuetime := dkvs.TimeNow()
	fmt.Printf("tKey: %v", tKey)
	data := dkvs.GetRecordSignData(tKey, tValue1, pkBytes, issuetime, ttl)
	sigData1, err := priv.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(tKey, tValue1, pkBytes, issuetime, ttl, sigData1)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-time.After(20 * time.Second):
		fmt.Println("Timeout occurred")
	}
	//等待时手工从网络获取这个key
	tValue2 := []byte("world2")
	data = dkvs.GetRecordSignData(tKey, tValue2, pkBytes, issuetime, maxttl)
	sigData2, err := priv.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(tKey, tValue2, pkBytes, issuetime, maxttl, sigData2)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-time.After(20 * time.Second):
		fmt.Println("Timeout occurred")
	}
	//等待时手工从网络获取这个key

	//删除这个key
	data = dkvs.GetRecordSignData(tKey, tValue1, pkBytes, 0, ttl)
	sigData3, err := priv.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(tKey, tValue1, pkBytes, 0, ttl, sigData3)
	if err == nil {
		t.Fatal(err)
	}
	if err != nil {
		fmt.Printf("记录过期错误：%s", err.Error())
	}
	issuetime = dkvs.TimeNow()
	ttl = 5000 //代表5000毫秒，ttl与issuetime都是毫秒
	data = dkvs.GetRecordSignData(tKey, nil, pkBytes, issuetime, ttl)
	sigData4, err := priv.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(tKey, nil, pkBytes, issuetime, ttl, sigData4)
	if err != nil {
		fmt.Printf("记录错误：%s", err.Error())
	}
	select {
	case <-time.After(30 * time.Second):
		fmt.Println("Timeout occurred")
	}
}

func TestDkvsDbMaxRecordAge(t *testing.T) {
	cfg := dkvs.NewDefaultDkvsConfig()
	cfg.InitMode(dkvs.LightMode)
	kv := dkvs.NewDkvs(cfg)


	seed := "oIBBgepoPyhdJTYC"
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

	tKey := "/" + dkvs.PUBSERVICE_DAUTH + "/" + hash("dkvs-k001-aa25")
	tValue1 := []byte("world1")
	ttl := dkvs.GetTtlFromDuration(time.Minute)
	issuetime := dkvs.TimeNow()
	fmt.Printf("tKey: %v", tKey)
	data := dkvs.GetRecordSignData(tKey, tValue1, pkBytes, issuetime, ttl)
	sigData1, err := priv.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(tKey, tValue1, pkBytes, issuetime, ttl, sigData1)
	if err != nil {
		t.Fatal(err)
	}
	select {}
	//等待时手工从网络获取这个key
}
