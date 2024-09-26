package ns

import (
	"fmt"
	"strings"

	"github.com/sat20-labs/indexer/common"
	indexer "github.com/sat20-labs/indexer/indexer/common"
)

func ParseKeyValue(kvstr string) *KeyValue {
	parts := strings.Split(kvstr, "=")
	if len(parts) != 2 {
		common.Log.Errorf("ParseKeyValue failed. %s", kvstr)
		return nil
	}

	return &KeyValue{Key: parts[0], Value: parts[1]}
}

func ParseKVs(kvs []string) []*KeyValue {
	result := make([]*KeyValue, 0)
	for _, kv := range kvs {
		r := ParseKeyValue(kv)
		if r != nil {
			result = append(result, r)
		}
	}
	return result
}

func BindNametoSat(p *indexer.SatRBTree, sat int64, name string) error {
	value := p.FindNode((sat))
	if value != nil {
		if value.(*RBTreeValue_Name).Name != name {
			return fmt.Errorf("sat %d has bound to name %s", sat, value.(*RBTreeValue_Name).Name)
		}
	} else {
		value = &RBTreeValue_Name{Name: name}
		p.Put(sat, value)
	}

	return nil
}
