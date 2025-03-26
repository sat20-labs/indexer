package common

type KeyValue struct {
	Key       string `json:"key"`
	Value     []byte `json:"value"`
	PubKey    []byte `json:"pubKey"`
	Signature []byte `json:"signature"`
}
