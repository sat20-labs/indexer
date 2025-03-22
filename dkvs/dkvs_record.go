package dkvs

import (
	"fmt"
	"sync"
	"time"

	ic "github.com/libp2p/go-libp2p/core/crypto"
	pb "github.com/sat20-labs/indexer/dkvs/pb"
)

var (
	lastTimestampMu sync.Mutex
	lastTimestamp   uint64
)

// Create creates a new DKVS entry and signs it with the given private key.
//
// This function does not embed the public key. If you want to do that, use
// `EmbedPublicKey`.
func CreateRecord(val []byte, pk []byte, issuetime uint64, ttl uint64, sig []byte) (*pb.DkvsRecord, error) {
	return CreateRecordWithType(val, pk, issuetime, ttl, sig, 0)
}

func CreateRecordWithType(val []byte, pk []byte, issuetime uint64, ttl uint64, sig []byte, vt uint32) (*pb.DkvsRecord, error) {
	entry := new(pb.DkvsRecord)
	entry.Version = uint32(1)
	entry.ValueType = vt
	entry.PubKey = pk
	entry.Value = val
	entry.ValidityType = pb.DkvsRecord_EOL
	entry.Seq = TimestampSeq()
	entry.Validity = issuetime + ttl 
	entry.Ttl = ttl

	entry.Signature = sig

	return entry, nil
}

func ExtractPublicKey(pubKey []byte) (ic.PubKey, error) {
	if pubKey != nil {
		pk, err := ic.UnmarshalPublicKey(pubKey)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling pubkey in record: %s", err)
		}
		return pk, nil
	}
	return nil, ErrPublicIsNil
}

// Compare compares two DKVS entries. It returns:
//
// * -1 if a is older than b
// * 0 if a and b cannot be ordered (this doesn't mean that they are equal)
// * +1 if a is newer than b
//
// It returns an error when either a or b are malformed.
//
// NOTE: It *does not* validate the records, the caller is responsible for calling
// `Validate` first.
//
// NOTE: If a and b cannot be ordered by this function, you can determine their
// order by comparing their serialized byte representations (using
// `bytes.Compare`). You must do this if you are implementing a libp2p record
// validator (or you can just use the one provided for you by this package).
func Compare(a, b *pb.DkvsRecord) (int, error) {

	as := a.GetSeq()
	bs := b.GetSeq()

	if as > bs {
		return 1, nil
	} else if as < bs {
		return -1, nil
	}

	if a.Validity > b.Validity {
		return 1, nil
	} else if a.Validity < b.Validity {
		return -1, nil
	}

	return 0, nil
}

// Validates validates the given DKVS entry against the given public key.
func ValidateRecord(key string, entry *pb.DkvsRecord) error {

	// Make sure max size is respected
	if entry.Size() > MaxRecordSize {
		Logger.Error(ErrRecordSize)
		return ErrRecordSize
	}

	// Check the dkvs record signature with the public key
	if entry.GetPubKey() == nil {
		// always error if no valid signature could be found
		return ErrSignature
	}

	err := ValidateValue(key, entry.Value, entry.PubKey, entry.Validity-entry.Ttl, entry.Ttl, entry.Signature, int(entry.ValueType))
	if err != nil {
		Logger.Error(err)
		return err
	}

	switch entry.GetValidityType() {
	case pb.DkvsRecord_EOL:
		if TimeNow() > entry.Validity {
			Logger.Error(ErrExpiredRecord)
			return ErrExpiredRecord
		}
	default:
	}

	return nil
}

func ValidateValue(key string, val []byte, pubKey []byte, issuetime uint64, ttl uint64, sig []byte, valueType int) error {
	pk, err := ExtractPublicKey(pubKey)
	if err != nil {
		return err
	}

	sigData := GetRecordSignData(key, val, pubKey, issuetime, ttl)
	ok, err := pk.Verify(sigData, sig)
	if  err != nil || !ok {
		Logger.Error(err)
		return ErrSignature
	}

	var bVerify bool
	switch valueType {
	case int(pb.DkvsRecord_GUN_SIGNATURE):
		if IsGunName(key) {
			bVerify = VerifyGunRecordValue(key, val, issuetime, ttl)
		} else {
			bVerify = true
		}
	case int(pb.DkvsRecord_DEFAULT):
		bVerify = true
	}

	if !bVerify {
		Logger.Error(ErrSignature)
		return ErrSignature
	}

	return nil
}

// func dkvsEntryDataForSig(e *pb.DkvsRecord) ([]byte, error) {
// 	dataForSig := []byte("dkvs-signature:")
// 	dataForSig = append(dataForSig, e.Data...)

// 	return dataForSig, nil
// }

// EmbedPublicKey embeds the given public key in the given DKVS entry. While not
// strictly required, some nodes (e.g., DHT servers) may reject DKVS entries
// that don't embed their public keys as they may not be able to validate them
// efficiently.
func EmbedPublicKey(pk ic.PubKey, entry *pb.DkvsRecord) error {
	// We failed to extract the public key from the peer ID, embed it in the
	// record.
	pkBytes, err := ic.MarshalPublicKey(pk)
	if err != nil {
		return err
	}
	entry.PubKey = pkBytes
	return nil
}

// TimestampSeq is a helper to generate a timestamp-based sequence number for a PeerRecord.
func TimestampSeq() uint64 {
	now := uint64(time.Now().UnixMilli())
	lastTimestampMu.Lock()
	defer lastTimestampMu.Unlock()
	// Not all clocks are strictly increasing, but we need these sequence numbers to be strictly
	// increasing.
	if now <= lastTimestamp {
		now = lastTimestamp + 1
	}
	lastTimestamp = now
	return now
}
