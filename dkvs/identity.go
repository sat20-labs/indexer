package dkvs

import (
	"encoding/base64"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/pnet"
	"golang.org/x/crypto/salsa20"
	"golang.org/x/crypto/sha3"
)

func LoadPrikey(prikeyHex string) (crypto.PrivKey, error) {
	data, err := base64.StdEncoding.DecodeString(prikeyHex)
	if err != nil {
		return nil, err
	}

	return crypto.UnmarshalPrivateKey(data)
}

// PNet fingerprint section is taken from github.com/ipfs/kubo/core/node/libp2p/pnet.go
// since the functions in that package were not exported.
// https://github.com/ipfs/kubo/blob/255e64e49e837afce534555f3451e2cffe9f0dcb/core/node/libp2p/pnet.go#L74

type PNetFingerprint []byte

// PNetFingerprint returns the given swarm key's fingerprint.
func pnetFingerprint(psk pnet.PSK) []byte {
	var pskArr [32]byte
	copy(pskArr[:], psk)

	enc := make([]byte, 64)
	zeros := make([]byte, 64)
	out := make([]byte, 16)

	// We encrypt data first so we don't feed PSK to hash function.
	// Salsa20 function is not reversible thus increasing our security margin.
	salsa20.XORKeyStream(enc, zeros, []byte("finprint"), &pskArr)

	// Then do Shake-128 hash to reduce its length.
	// This way if for some reason Shake is broken and Salsa20 preimage is possible,
	// attacker has only half of the bytes necessary to recreate psk.
	sha3.ShakeSum128(out, enc)

	return out
}

func GenIdenity() (crypto.PrivKey, string, error) {
	privk, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
	if err != nil {
		return privk, "", err
	}

	data, err := crypto.MarshalPrivateKey(privk)
	if err != nil {
		return privk, "", err
	}

	return privk, base64.StdEncoding.EncodeToString(data), nil
}
