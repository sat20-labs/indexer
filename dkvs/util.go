package dkvs

import (
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// TimeFormatIpfs is the format ipfs uses to represent time in string form.
var TimeFormatIpfs = time.RFC3339Nano

// ParseRFC3339 parses an RFC3339Nano-formatted time stamp and
// returns the UTC time.
func ParseRFC3339(s string) (time.Time, error) {
	t, err := time.Parse(TimeFormatIpfs, s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// FormatRFC3339 returns the string representation of the
// UTC value of the given time in RFC3339Nano format.
func FormatRFC3339(t time.Time) string {
	return t.UTC().Format(TimeFormatIpfs)
}

func GetPriKeyBySeed(seed string) (crypto.PrivKey, error) {
	aStringToHash := []byte(seed)
	sha256Bytes := sha256.Sum256(aStringToHash)
	keySeed := int64(binary.BigEndian.Uint64(sha256Bytes[:]))
	r := rand.New(rand.NewSource(int64(keySeed)))
	// Set your own keypair
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, -1, r)
	return priv, err
}

// 生成指定长度的随机字符串
func RandString(n int) string {
	// 定义字符串的字符集
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)

	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	return string(b)
}

// RemovePrefix returns the orginal key for a dksvs key.
func RemovePrefix(dkvsKey string) string {
	return strings.Replace(dkvsKey, DefaultKeyPrefix, "", -1)
}

// ParsePeersAddrInfo parses a AddrInfos list into a list of peer id.
func ParsePeersAddrInfo(addrInfos []peer.AddrInfo) []peer.ID {
	peerIds := make([]peer.ID, len(addrInfos))
	for i, addrInfo := range addrInfos {
		peerIds[i] = addrInfo.ID
	}
	return peerIds
}


// ParseBootstrapPeer parses a bootstrap list into a list of AddrInfos.
func ParseBootstrapPeers(addrs []string) ([]peer.AddrInfo, error) {
	maddrs := make([]multiaddr.Multiaddr, len(addrs))
	for i, addr := range addrs {
		var err error
		maddrs[i], err = multiaddr.NewMultiaddr(addr)
		if err != nil {
			return nil, err
		}
	}
	return peer.AddrInfosFromP2pAddrs(maddrs...)
}

