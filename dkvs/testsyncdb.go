package dkvs

import (
	"crypto/sha512"
	"encoding/hex"
	"time"

	ic "github.com/libp2p/go-libp2p/core/crypto"
)

func hash(key string) (hashKey string) {
	shaHash := sha512.Sum384([]byte(key))
	hashKey = hex.EncodeToString(shaHash[:])
	return
}

func insertKeyToDB(privkey ic.PrivKey, key string) error {
	pubkey, err := ic.MarshalPublicKey(privkey.GetPublic())
	if err != nil {
		Logger.Error(err)
		return err
	}

	issuetime := TimeNow()
	ttl := GetTtlFromDuration(time.Hour)
	key1 := hash(key)
	value1 := []byte(key + " value")
	dataToSign1 := GetRecordSignData(key1, value1, pubkey, issuetime, ttl)
	signdata1, err := privkey.Sign(dataToSign1)
	if err != nil {
		Logger.Error(err)
		return err
	}

	// rc, err := CreateRecordWithType(value1, pubkey, ttl, signdata1, 0)
	_, err = CreateRecordWithType(value1, pubkey, issuetime, ttl, signdata1, 0)
	if err != nil {
		Logger.Error(err)
		return err
	}
	// recordvalue1, err := rc.Marshal()
	// if err != nil {
	// 	return err
	// }
	// recordKey := RecordKey(key1)

	// err = SaveUnsyncKey(context.Background(), recordKey, recordvalue1)
	// if err != nil {
	// 	Logger.Error(err)
	// 	return err
	// }

	// ret, err := getUnsyncKey(context.Background(), recordKey)
	// if err != nil || !bytes.Equal(ret, recordvalue1) {
	// 	Logger.Error(err)
	// 	return err
	// }

	return nil
}

func TestSyncDB() error {

	privKey, err := GetPriKeyBySeed("mtv1")
	if err != nil {
		Logger.Error(err)
	}

	err = insertKeyToDB(privKey, "key1")
	if err != nil {
		Logger.Error(err)
		return err
	}

	err = insertKeyToDB(privKey, "key2")
	if err != nil {
		Logger.Error(err)
		return err
	}

	err = insertKeyToDB(privKey, "key3")
	if err != nil {
		Logger.Error(err)
		return err
	}

	err = insertKeyToDB(privKey, "key4")
	if err != nil {
		Logger.Error(err)
		return err
	}

	// err = deleteUnsyncKey(context.Background(), RecordKey("key3"))
	// if err != nil {
	// 	Logger.Error(err)
	// 	return err
	// }

	// _, err = getUnsyncKey(context.Background(), RecordKey("key3"))
	// if err == nil {
	// 	Logger.Error("deleted but still get it")
	// 	return err
	// }

	// err = insertKeyToDB(privKey, "key3")
	// if err != nil {
	// 	Logger.Error(err)
	// 	return err
	// }

	// err = insertKeyToDB(privKey, "key3")
	// if err != nil {
	// 	Logger.Error(err)
	// 	return err
	// }

	// OnNetworkStatusChanged(1)

	// closeUnsyncDB(context.Background())

	return nil

}
