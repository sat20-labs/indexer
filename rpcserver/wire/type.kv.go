package wire

import "github.com/sat20-labs/indexer/common"

type KeyValue = common.KeyValue

type GetNonceReq struct {
	PubKey []byte `json:"pubkey"`
}

type GetNonceResp struct {
	BaseResp
	Nonce []byte `json:"Nonce"`
}

type GetValueReq struct {
	Keys 		[]string `json:"keys"`
	Nonce 		[]byte `json:"Nonce"`
	PubKey 		[]byte `json:"pubkey"`
	Signature 	[]byte `json:"signature"`
}

type GetValueResp struct {
	BaseResp
	Value  *KeyValue `json:"value"`
}

type GetValuesResp struct {
	BaseResp
	Values []*KeyValue `json:"values"`
}

type PutKValueReq struct {
	Values []*KeyValue `json:"values"`
	Nonce 		[]byte `json:"Nonce"`
	PubKey 		[]byte `json:"pubkey"`
	Signature 	[]byte `json:"signature"`
}

type PutKValueResp struct {
	BaseResp
}

type DelKValueReq struct {
	Keys 		[]string `json:"keys"`
	Nonce 		[]byte `json:"Nonce"`
	PubKey 		[]byte `json:"pubkey"`
	Signature 	[]byte `json:"signature"`
}

type DelKValueResp struct {
	BaseResp
}
