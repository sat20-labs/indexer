package indexer

import (
	"fmt"
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

const (
	defaultBuildDBCacheMB = 2048
	baseBuildDBCacheMB    = 32768
	nftBuildDBCacheMB     = 16384
	brc20BuildDBCacheMB   = 8192
)

func openDB(filepath string, cacheSizeMB int) (common.KVDB, error) {

	ldb := db.NewKVDBWithCache(filepath, cacheSizeMB)
	if ldb == nil {
		return nil, fmt.Errorf("NewKVDB failed")
	}

	return ldb, nil
}

func (p *IndexerMgr) initDB() (err error) {
	common.Log.Info("InitDB-> start...")

	p.baseDB, err = openDB(p.dbDir+"base", baseBuildDBCacheMB)
	if err != nil {
		return err
	}

	p.nftDB, err = openDB(p.dbDir+"nft", nftBuildDBCacheMB)
	if err != nil {
		return err
	}

	p.nsDB, err = openDB(p.dbDir+"ns", defaultBuildDBCacheMB)
	if err != nil {
		return err
	}

	p.exoticDB, err = openDB(p.dbDir+"exotic", defaultBuildDBCacheMB)
	if err != nil {
		return err
	}

	p.ftDB, err = openDB(p.dbDir+"ft", defaultBuildDBCacheMB)
	if err != nil {
		return err
	}

	p.brc20DB, err = openDB(p.dbDir+"brc20", brc20BuildDBCacheMB)
	if err != nil {
		return err
	}

	p.runesDB, err = openDB(p.dbDir+"runes", defaultBuildDBCacheMB)
	if err != nil {
		return err
	}

	p.atomDB, err = openDB(p.dbDir+"atom", defaultBuildDBCacheMB)
	if err != nil {
		return err
	}

	p.localDB, err = openDB(p.dbDir+"local", defaultBuildDBCacheMB)
	if err != nil {
		return err
	}

	p.kvDB, err = openDB(p.dbDir+"dkvs", defaultBuildDBCacheMB)
	if err != nil {
		return err
	}

	return nil
}

func getCollectionKey(ntype, ticker string) []byte {
	return []byte("c-" + ntype + "-" + ticker)
}

func parseCollectionKey(key string) (string, string, error) {
	parts := strings.Split(key, "-")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("invalid key %s", key)
	}
	return parts[1], parts[2], nil
}

func inscriptionIdsToCollectionMap(ids []string) map[string]int64 {
	inscmap := make(map[string]int64)
	for _, id := range ids {
		inscmap[id] = 1
	}
	return inscmap
}

func (p *IndexerMgr) initCollections() {
	common.Log.Info("initCollections ...")

	p.clmap = make(map[common.TickerName]map[string]int64)
	err := p.localDB.BatchRead([]byte("c-"), false, func(k, v []byte) error {

		key := string(k)

		nty, name, err := parseCollectionKey(key)
		if err == nil {
			var ids []string

			err = db.DecodeBytes(v, &ids)
			if err == nil {
				p.clmap[common.TickerName{Protocol: common.PROTOCOL_NAME_ORDX, Type: nty, Ticker: name}] = inscriptionIdsToCollectionMap(ids)
			} else {
				common.Log.Errorln("initCollections DecodeBytes " + err.Error())
			}

		}

		return nil
	})

	if err != nil {
		common.Log.Panicf("initCollections Error: %v", err)
	}
}
