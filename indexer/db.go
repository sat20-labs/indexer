package indexer

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

const defaultBadgerBlockCacheTotalMB = 16 * 1024

var badgerBlockCacheWeights = map[string]int{
	"base":   40,
	"nft":    25,
	"brc20":  10,
	"runes":  10,
	"exotic": 5,
	"ns":     3,
	"ft":     3,
	"local":  2,
	"dkvs":   2,
	"atom":   0, // Atomicals indexing is intentionally disabled for now.
}

func configuredBadgerBlockCacheTotalMB() int {
	for _, key := range []string{"INDEXER_BADGER_BLOCK_CACHE_TOTAL_MB", "INDEXER_DB_CACHE_TOTAL_MB"} {
		raw := os.Getenv(key)
		if raw == "" {
			continue
		}
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			common.Log.Warnf("invalid %s=%q, use default %dMB", key, raw, defaultBadgerBlockCacheTotalMB)
			return defaultBadgerBlockCacheTotalMB
		}
		return value
	}
	return defaultBadgerBlockCacheTotalMB
}

func allocateBadgerBlockCache(totalMB int) map[string]int {
	plan := make(map[string]int, len(badgerBlockCacheWeights))
	if totalMB <= 0 {
		for name := range badgerBlockCacheWeights {
			plan[name] = 0
		}
		return plan
	}

	assigned := 0
	for name, weight := range badgerBlockCacheWeights {
		value := totalMB * weight / 100
		plan[name] = value
		assigned += value
	}
	// Keep the requested process-wide total exact despite integer rounding.
	plan["base"] += totalMB - assigned
	return plan
}

func openDB(filepath string, cacheSizeMB int) (common.KVDB, error) {
	ldb := db.NewKVDBWithCache(filepath, cacheSizeMB)
	if ldb == nil {
		return nil, fmt.Errorf("NewKVDB failed")
	}
	return ldb, nil
}

func (p *IndexerMgr) initDB() (err error) {
	common.Log.Info("InitDB-> start...")

	totalCacheMB := configuredBadgerBlockCacheTotalMB()
	cache := allocateBadgerBlockCache(totalCacheMB)
	common.Log.Infof(
		"Badger process block-cache plan: total=%dMB base=%d nft=%d brc20=%d runes=%d exotic=%d ns=%d ft=%d local=%d dkvs=%d atom=%d",
		totalCacheMB,
		cache["base"], cache["nft"], cache["brc20"], cache["runes"], cache["exotic"],
		cache["ns"], cache["ft"], cache["local"], cache["dkvs"], cache["atom"],
	)

	p.baseDB, err = openDB(p.dbDir+"base", cache["base"])
	if err != nil {
		return err
	}
	p.nftDB, err = openDB(p.dbDir+"nft", cache["nft"])
	if err != nil {
		return err
	}
	p.nsDB, err = openDB(p.dbDir+"ns", cache["ns"])
	if err != nil {
		return err
	}
	p.exoticDB, err = openDB(p.dbDir+"exotic", cache["exotic"])
	if err != nil {
		return err
	}
	p.ftDB, err = openDB(p.dbDir+"ft", cache["ft"])
	if err != nil {
		return err
	}
	p.brc20DB, err = openDB(p.dbDir+"brc20", cache["brc20"])
	if err != nil {
		return err
	}
	p.runesDB, err = openDB(p.dbDir+"runes", cache["runes"])
	if err != nil {
		return err
	}
	p.atomDB, err = openDB(p.dbDir+"atom", cache["atom"])
	if err != nil {
		return err
	}
	p.localDB, err = openDB(p.dbDir+"local", cache["local"])
	if err != nil {
		return err
	}
	p.kvDB, err = openDB(p.dbDir+"dkvs", cache["dkvs"])
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
