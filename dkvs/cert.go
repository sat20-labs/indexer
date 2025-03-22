package dkvs

import (
	"bytes"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	pb "github.com/sat20-labs/indexer/dkvs/pb"
)

// 证书结构：dkvs.pb.Cert
// 证书由发行者记录在用户公钥下面，一般格式（路径可以自定义）
// addr: /服务名称/用户公钥/cert
// value: CertsRecordValue

var DefaultCertTtl = uint64(time.Duration(time.Hour * 24 * 365 * 100).Milliseconds())

const CertTransferPrepare = "TransferPrepare"
const CertTxCompleted = "TxCompleted"
const CertApprove = "Approve"

func GetCertAddr(sn string, pk []byte) string {
	return "/" + sn + "/" + BytesToHexString(pk) + "/cert"
}

func VerifyCert(cert *pb.Cert) bool {

	if cert == nil {
		return false
	}

	data := GetCertSignData(cert)

	issuepk, err := crypto.UnmarshalPublicKey(cert.IssuerPubkey)
	if err != nil {
		return false
	}

	if ok, err := issuepk.Verify(data, cert.IssuerSign); err != nil || !ok {
		return false
	}

	// check validaty
	if cert.Ttl != 0 {
		return TimeNow() <= cert.IssueTime+cert.Ttl
	}

	return true
}

func GetCertSignData(cert *pb.Cert) []byte {
	toSign := *cert
	toSign.IssuerSign = nil

	b, err := toSign.Marshal()
	if err != nil {
		return nil
	}

	return b
}

func EncodeCert(cert *pb.Cert) []byte {
	if cert == nil {
		return nil
	}

	b, err := cert.Marshal()
	if err != nil {
		return nil
	}

	return b
}

func DecodeCert(value []byte) *pb.Cert {

	var cert pb.Cert
	err := cert.Unmarshal(value)
	if err != nil {
		return nil
	}

	return &cert
}

func SearchCertByPubkey(cv []*pb.Cert, pubkey []byte) *pb.Cert {
	for _, c := range cv {
		if bytes.Equal(pubkey, c.IssuerPubkey) {
			return c
		}
	}
	return nil
}

func SearchCertByName(cv []*pb.Cert, name string) *pb.Cert {
	for _, c := range cv {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func IsCertExisted(cv []*pb.Cert, cert *pb.Cert) bool {
	for _, c := range cv {
		if c.IssueTime == cert.IssueTime && c.Ttl == cert.Ttl && c.Version == cert.Version &&
			c.Name == cert.Name && c.Type == cert.Type &&
			bytes.Equal(c.UserPubkey, cert.UserPubkey) && bytes.Equal(c.Data, cert.Data) &&
			bytes.Equal(c.IssuerSign, cert.IssuerSign) && bytes.Equal(c.IssuerPubkey, cert.IssuerPubkey) {
			return true
		}
	}
	return false
}

func DecodeAndFindCertByPubkey(value []byte, pubkey []byte) *pb.Cert {
	rv := DecodeGunRecordValue(value)
	if rv == nil {
		return nil
	}

	return SearchCertByPubkey(rv.CertVect, pubkey)
}

// used to sign with private key
func GetGunSignData(name string, num uint64, gunPubkey []byte, issueTime uint64, ttl uint64) []byte {

	cert := IssueCertGun(name, num, gunPubkey, issueTime, ttl)

	b, err := cert.Marshal()
	if err != nil {
		return nil
	}

	return b
}

// generate value for a GUN record
func EncodeGunValue(name string, num uint64, issueTime uint64, ttl uint64, gunPubkey []byte, gunSign []byte, userData []byte) []byte {
	cert := IssueCertGun(name, num, gunPubkey, issueTime, ttl)
	cert.IssuerSign = gunSign

	rv := pb.CertsRecordValue{
		UserData: userData, // 在转移时放一个证书
		CertVect: []*pb.Cert{cert},
	}

	b, err := rv.Marshal()
	if err != nil {
		return nil
	}

	return b
}

// decode from value by a GUN record
func DecodeGunRecordValue(value []byte) *pb.CertsRecordValue {
	var rv pb.CertsRecordValue
	err := rv.Unmarshal(value)
	if err != nil {
		return nil
	}

	return &rv
}

func VerifyGunRecordValue(key string, value []byte, issuetime uint64, ttl uint64) bool {

	rv := DecodeGunRecordValue(value)
	if rv == nil {
		return false
	}

	cert := FindGunCert(rv.CertVect)
	if cert == nil {
		return false
	}

	var si pb.SimpleContractIssueToken
	err := si.Unmarshal(cert.Data)
	if err != nil {
		return false
	}

	return cert.Ttl == ttl && cert.IssueTime == issuetime && GetGunKey(si.Name) == key
}

func FindTransferCert(cv []*pb.Cert) *pb.Cert {
	cert := SearchCertByName(cv, CertTransferPrepare)
	if IsTransferCert(cert) {
		return cert
	}
	return nil
}

func IsTransferCert(cert *pb.Cert) bool {
	if cert != nil && cert.Name == CertTransferPrepare && VerifyCert(cert) {
		return true
	}
	return false
}

func VerifyTransferRecordValue(key string, value []byte, pk []byte) bool {

	rv := DecodeCertsRecordValue(value)
	if rv == nil {
		return false
	}
	var cert *pb.Cert
	if rv.UserData != nil {
		// try to decode it
		cert = DecodeCert(rv.UserData)
		if !IsTransferCert(cert) {
			cert = nil
		}
	}

	if cert == nil {
		cert = FindTransferCert(rv.CertVect)
		if cert == nil {
			return false
		}
	}

	var tp pb.CertDataTransferPrepare
	err := tp.Unmarshal(cert.Data)
	if err != nil {
		return false
	}

	return bytes.Equal(cert.IssuerPubkey, pk) && tp.Key == key
}

func GetCertTransferPrepare(value []byte) *pb.Cert {
	certs := DecodeCertsRecordValue(value)
	if certs == nil {
		return nil
	}
	if certs.UserData == nil {
		return nil
	}
	cert := DecodeCert(certs.UserData)
	if cert == nil {
		return nil
	}
	if !VerifyCert(cert) {
		return nil
	}
	return cert
}

func VerifyCertTransferPrepare(key string, cert *pb.Cert, oldpk, newpk []byte) bool {
	//
	var tp pb.CertDataTransferPrepare

	err := tp.Unmarshal(cert.Data)
	if err != nil {
		return false
	}

	if !bytes.Equal(cert.IssuerPubkey, oldpk) ||
		cert.Name != CertTransferPrepare || tp.Key != key {
		return false
	}

	if cert.UserPubkey != nil {
		if !bytes.Equal(cert.UserPubkey, newpk) {
			return false
		}
	}

	return true
}

func VerifyCertTxCompleted(key string, fee uint64, txcert *pb.Cert, oldpk, newpk []byte) bool {
	var tc pb.CertDataTxCompleted

	if txcert == nil {
		return false
	}
	if !VerifyCert(txcert) {
		return false
	}
	if !IsPublicServiceNameKey(PUBSERVICE_MINER, txcert.IssuerPubkey) {
		return false
	}
	if txcert.Name != CertTxCompleted {
		return false
	}

	err := tc.Unmarshal(txcert.Data)
	if err != nil {
		return false
	}

	if tc.Key != key || tc.Tx == "" || tc.Fee != fee {
		return false
	}

	if (bytes.Equal(oldpk, tc.Senderkey) && bytes.Equal(newpk, tc.Receiverkey)) ||
		(bytes.Equal(newpk, tc.Senderkey) && bytes.Equal(oldpk, tc.Receiverkey)) {
		return true
	}

	return false
}

func VerifyCertTxCompleted2(key string, transfercert, txcert *pb.Cert, oldpk, newpk []byte) bool {
	// transfercert has been verified
	var tp pb.CertDataTransferPrepare

	err := tp.Unmarshal(transfercert.Data)
	if err != nil {
		return false
	}

	if tp.Fee != 0 {
		return VerifyCertTxCompleted(key, tp.Fee, txcert, oldpk, newpk)
	}

	return true
}

func VerifyCertTransferConfirm(key string, oldvalue []byte, txcert *pb.Cert, pubkey1, pubkey2 []byte) bool {
	cert1 := GetCertTransferPrepare(oldvalue)
	if cert1 == nil {
		Logger.Error("GetCertTransferPrepare failed")
		return false
	}

	if !VerifyCertTransferPrepare(key, cert1, pubkey1, pubkey2) {
		Logger.Error("VerifyTransferCert failed")
		return false
	}

	if !VerifyCertTxCompleted2(key, cert1, txcert, pubkey1, pubkey2) {
		Logger.Error("VerifyCertTxCompleted2 failed")
		return false
	}

	return true
}



func VerifyCertApprove(cert *pb.Cert, pk []byte) bool {

	if cert == nil {
		return false
	}
	if !VerifyCert(cert) {
		return false
	}
	if !IsPublicServiceNameKey(string(cert.Data), cert.IssuerPubkey) {
		return false
	}
	if cert.Name != CertApprove {
		return false
	}

	return bytes.Equal(pk, cert.UserPubkey)
}

// used to sign with private key
func IssueCert(name string, data []byte, issuePubkey []byte, ttl uint64) *pb.Cert {

	cert := pb.Cert{
		Version:      1,
		Name:         name,
		Type:         uint32(pb.CertType_Default),
		SubType:      0,
		UserPubkey:   nil,
		Data:         data,
		IssueTime:    TimeNow(),
		Ttl:          ttl,
		IssuerPubkey: issuePubkey,
		IssuerSign:   nil,
	}

	return &cert
}

func IssueCertApprove(ns string, userPubkey []byte, issuePubkey []byte, ttl uint64) *pb.Cert {

	cert := pb.Cert{
		Version:      1,
		Name:         CertApprove,
		Type:         uint32(pb.CertType_Default),
		SubType:      0,
		UserPubkey:   userPubkey,
		Data:         []byte(ns),
		IssueTime:    TimeNow(),
		Ttl:          ttl,
		IssuerPubkey: issuePubkey,
		IssuerSign:   nil,
	}

	return &cert
}

// as a nft
func IssueCertGun(name string, num uint64, gunPubkey []byte, issueTime uint64, ttl uint64) *pb.Cert {

	var si pb.SimpleContractIssueToken
	si.Name = name
	si.MaxScore = num
	si.ReceiverKey = gunPubkey
	si.UserData = nil
	buf, err := si.Marshal()
	if err != nil {
		return nil
	}

	cert := pb.Cert{
		Version:      1,
		Name:         name,
		Type:         uint32(pb.CertType_Contract),
		SubType:      1, // new score
		UserPubkey:   nil,
		Data:         buf,
		IssueTime:    issueTime,
		Ttl:          ttl,
		IssuerPubkey: gunPubkey,
		IssuerSign:   nil,
	}

	return &cert
}

func IssueCertTransferPrepare(key string, fee uint64, receiverpk, issuePubkey []byte) *pb.Cert {

	tp := pb.CertDataTransferPrepare{
		Key: key,
		Fee: fee,
	}

	buf, err := tp.Marshal()
	if err != nil {
		return nil
	}

	cert := pb.Cert{
		Version:      1,
		Name:         CertTransferPrepare,
		Type:         uint32(pb.CertType_Default),
		SubType:      0,
		UserPubkey:   receiverpk,
		Data:         buf,
		IssueTime:    TimeNow(),
		Ttl:          DefaultCertTtl,
		IssuerPubkey: issuePubkey,
		IssuerSign:   nil,
	}

	return &cert
}

func IssueCertTxCompleted(key string, tx string, fee uint64, senderpk, receiverpk, issuerpk []byte) *pb.Cert {
	tc := pb.CertDataTxCompleted{
		Key:         key,
		Tx:          tx,
		Fee:         fee,
		Senderkey:   senderpk,
		Receiverkey: receiverpk,
	}

	buf, err := tc.Marshal()
	if err != nil {
		return nil
	}

	cert := pb.Cert{
		Version:      1,
		Name:         CertTxCompleted,
		Type:         uint32(pb.CertType_Default),
		SubType:      0,
		UserPubkey:   receiverpk,
		Data:         buf,
		IssueTime:    TimeNow(),
		Ttl:          DefaultCertTtl,
		IssuerPubkey: issuerpk,
		IssuerSign:   nil,
	}

	return &cert
}

// decode from value
func DecodeCertsRecordValue(value []byte) *pb.CertsRecordValue {
	var rv pb.CertsRecordValue
	err := rv.Unmarshal(value)
	if err != nil {
		return nil
	}

	return &rv
}

func EncodeCertsRecordValueWithCert(cr *pb.CertsRecordValue, cert *pb.Cert, userdata []byte) ([]byte, error) {

	if cr == nil {
		cr = &pb.CertsRecordValue{
			UserData: userdata, // 在转移时放一个证书
			CertVect: []*pb.Cert{cert},
		}
	} else {
		bFound := false
		for _, c := range cr.CertVect {
			if c.Name == cert.Name {
				// 不允许同名
				// cr.CertVect[i] = cert
				bFound = true
				break
				//err := errors.New("the same cert exists. " + cert.Name)
				//return nil, err
			}
		}
		cr.UserData = userdata
		if !bFound {
			cr.CertVect = append(cr.CertVect, cert)
		}
	}

	return cr.Marshal()
}
