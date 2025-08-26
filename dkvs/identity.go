package dkvs

import (
	"encoding/base64"

	"github.com/libp2p/go-libp2p/core/crypto"
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
