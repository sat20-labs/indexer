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
	Succeeded []string `json:"succeeded"`
}

type DelKValueReq struct {
	Keys 		[]string `json:"keys"`
	Nonce 		[]byte `json:"Nonce"`
	PubKey 		[]byte `json:"pubkey"`
	Signature 	[]byte `json:"signature"`
}

type DelKValueResp struct {
	BaseResp
	Deleted []string `json:"deleted"`
}
