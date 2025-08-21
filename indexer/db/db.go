package db

import (
	"bytes"
	"fmt"
)

func IterateRangeInDB(db KVDB, prefix, startKey, endKey []byte, 
	processFunc func(key, value []byte) error) error {
    return db.BatchReadV2(prefix, startKey, false, func(k, v []byte) error {
        // 检查是否超过结束键
        if len(endKey) > 0 && bytes.Compare(k, endKey) > 0 {
            return fmt.Errorf("reach the endkey") // 作为特殊信号来终止迭代
        }
        return processFunc(k, v)
    })
}

