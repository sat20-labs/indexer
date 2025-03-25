package wire

type KeyValue struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Validity  uint64 `json:"validity"`
	Ttl       uint64 `json:"ttl"`
	Pubkey    string `json:"pubkey"`
	Signature string `json:"signature"`
}

type GetValueReq struct {
	Keys []string `json:"keys"`
}

type GetValueResp struct {
	BaseResp
	Data []*KeyValue `json:"data"`
}

type PutKValueReq struct {
	KValues []*KeyValue `json:"values"`
}

type PutKValueResp struct {
	BaseResp
	Succeeded []string `json:"succeeded"`
}

type DelKValueReq struct {
	Keys []string `json:"keys"`
}

type DelKValueResp struct {
	BaseResp
	Deleted []string `json:"deleted"`
}
