package dkvs

import (
	ds "github.com/ipfs/go-datastore"
	levelds "github.com/ipfs/go-ds-leveldb"
	ldbopts "github.com/syndtr/goleveldb/leveldb/opt"
)

type Datastore interface {
	ds.Batching // must be thread-safe
}

func CreateDataStore(dbRootDir string) (Datastore, error) {
	
	return CreateLevelDB(dbRootDir)
}

func CreateLevelDB(dbRootDir string) (*levelds.Datastore, error) {
	return levelds.NewDatastore(dbRootDir, &levelds.Options{
		Compression: ldbopts.NoCompression,
	})
}
