package db

import (

	"github.com/sat20-labs/indexer/common"
)


func RunDBGC(db common.KVDB) {

}

func NewKVDB(path string) common.KVDB {
	//return NewLevelDB(path)
	return NewPebbleDB(path)
}
