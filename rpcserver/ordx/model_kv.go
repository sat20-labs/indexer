package ordx

import (
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
)

func (s *Model) GetKV(key string) (*rpcwire.KeyValue, error) {

	return nil, nil
}

func (s *Model) GetKVs(keys []string) ([]*rpcwire.KeyValue, error) {

	return nil, nil
}

func (s *Model) PutKVs(kvs []*rpcwire.KeyValue) ([]string, error) {

	return nil, nil
}

func (s *Model) DelKVs(keys []string) ([]string, error) {

	return nil, nil
}
