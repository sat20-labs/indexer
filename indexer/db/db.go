package db

import (

	"github.com/sat20-labs/indexer/common"
)


func RunDBGC(db common.KVDB) {
	//RunBadgerGC(db)
}

func NewKVDB(path string) common.KVDB {
	//return NewLevelDB(path)
	return NewPebbleDB(path)
	//return NewBadgerDB(path)
	//return NewLMDB(path)
	//return NewBoltDB(path)
}
