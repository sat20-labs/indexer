package dkvs

import (
	"bytes"
	"errors"
	"strings"

	"github.com/gogo/protobuf/proto"
	record "github.com/libp2p/go-libp2p-record"
	pb "github.com/sat20-labs/indexer/dkvs/pb"
)

var _ record.Validator = Validator{}

// RecordKey returns the libp2p record key for a given peer ID.
func RecordKey(key string) string {
	//Comment out do not use /dkvs need to check
	return DefaultKeyPrefix + key
}

// Validator is an Dkvs record validator that satisfies the libp2p record
// validator interface.
type Validator struct {
}

// Validate validates an dkvs record.
func (v Validator) Validate(fullkey string, value []byte) error {
	//Is it necessary to verify the namespace, this needs to be discussed
	ns, key, err := record.SplitKey(fullkey)
	if err != nil || ns != DKVS_NAMESPACE {
		return ErrInvalidPath
	}

	// Parse the value into an IpnsEntry
	entry := new(pb.DkvsRecord)
	err = proto.Unmarshal(value, entry)
	if err != nil {
		return ErrBadRecord
	}

	return ValidateRecord("/"+key, entry)
}

// Select selects the best record by checking which has the highest sequence
// number and latest EOL.
//
// This function returns an error if any of the records fail to parse. Validate
// your records first!
func (v Validator) Select(fullkey string, vals [][]byte) (int, error) {
	ns, key, err := record.SplitKey(fullkey)
	if err != nil || ns != DKVS_NAMESPACE {
		return -1, ErrInvalidPath
	}

	vp, err := verifyPubKey("/"+key, vals[0], vals[1])
	if vp != 0 {
		Logger.Error(err)
		return vp, err
	}
	var recs []*pb.DkvsRecord
	for _, v := range vals {
		e := new(pb.DkvsRecord)
		if err := proto.Unmarshal(v, e); err != nil {
			return -1, err
		}
		recs = append(recs, e)
	}
	return selectRecord(recs, vals)
}

func selectRecord(recs []*pb.DkvsRecord, vals [][]byte) (int, error) {
	switch len(recs) {
	case 0:
		Logger.Error("no usable records in given set")
		return -1, errors.New("no usable records in given set")
	case 1:
		return 0, nil
	}

	var i int
	for j := 1; j < len(recs); j++ {
		cmp, err := Compare(recs[i], recs[j])
		if err != nil {
			Logger.Error(err)
			return -1, err
		}
		if cmp == 0 {
			cmp = bytes.Compare(vals[i], vals[j])
		}
		if cmp < 0 {
			i = j
		}
	}
	return i, nil
}

// Verify that the public key of the new record is the same as the old public key to prevent the record from being overwritten
func verifyPubKey(key string, Val1 []byte, Val2 []byte) (int, error) {
	var newRecord *pb.DkvsRecord
	var oldRecord *pb.DkvsRecord
	record1 := new(pb.DkvsRecord)
	if err := proto.Unmarshal(Val1, record1); err != nil {
		return -1, err
	}
	record2 := new(pb.DkvsRecord)
	if err := proto.Unmarshal(Val2, record2); err != nil {
		return -1, err
	}

	if record1.Seq > record2.Seq {
		newRecord = record1
		oldRecord = record2
	} else {
		newRecord = record2
		oldRecord = record1
	}

	switch oldRecord.ValidityType {
	case pb.DkvsRecord_EOL:
		// 正常状态，只检查是否是公钥相同
		cmp := bytes.Compare(newRecord.GetPubKey(), oldRecord.GetPubKey())
		if cmp != 0 {
			// 检查是否是TransferKey
			if newRecord.Data != nil {
				var prepareRecord pb.DkvsRecord
				err := prepareRecord.Unmarshal(newRecord.Data)
				if err != nil {
					Logger.Error(err)
					return -1, ErrPublicKeyMismatch
				}

				var txcert *pb.Cert
				if prepareRecord.Data != nil {
					var c pb.Cert
					err := c.Unmarshal(prepareRecord.Data)
					if err != nil {
						Logger.Error(err)
						return -1, err
					}
					txcert = &c
				}

				if !VerifyCertTransferConfirm(key, prepareRecord.Value, txcert, oldRecord.PubKey, newRecord.PubKey) {
					Logger.Error(ErrSignature)
					return -1, ErrPublicKeyMismatch
				}

				Logger.Debugf("key %s transfer from %s to %s\n", key, BytesToHexString(oldRecord.PubKey), BytesToHexString(newRecord.PubKey))
				return 0, nil
			} else {
				subkeys := strings.Split(key, "/")
				if len(subkeys) < 3 { // subkeys[0] = ""
					return -1, ErrDifferentPublicKey
				}
				if IsPublicServiceName(subkeys[1]) || isApprovedService(subkeys[1]) {
					b11 := IsPublicServiceNameKey(subkeys[1], oldRecord.PubKey)
					b21 := IsPublicServiceNameKey(subkeys[1], newRecord.PubKey)

					if b11 && b21 {
						// 同一个公共服务，如果是已经注册的服务，可以相互修改数据
						Logger.Debugf("%s and %s are public service %s\n", BytesToHexString(oldRecord.PubKey), BytesToHexString(newRecord.PubKey), subkeys[1])
						return 0, nil
					} else {
						b12 := isApprovedPubkey(subkeys[1], oldRecord.PubKey) || b11
						b22 := isApprovedPubkey(subkeys[1], newRecord.PubKey) || b21
						// 新老记录有服务颁发的证书，证明可以修改（要读取证书）
						if b12 && b22 {
							Logger.Debugf("%s and %s have cert of public service %s\n", BytesToHexString(oldRecord.PubKey), BytesToHexString(newRecord.PubKey), subkeys[1])
							return 0, nil
						}
					}
				}
			}
			
			Logger.Error(ErrDifferentPublicKey)
			return -1, ErrDifferentPublicKey
		}
		

	default:
	}

	return 0, nil
}
