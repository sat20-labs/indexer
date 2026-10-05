package indexer

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/config"
	"github.com/sat20-labs/indexer/indexer/db"
)

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

func configuredBadgerBlockCacheTotalMB(configuredMB int) int {
	for _, key := range []string{"INDEXER_BADGER_BLOCK_CACHE_TOTAL_MB", "INDEXER_DB_CACHE_TOTAL_MB"} {
		raw := os.Getenv(key)
		if raw == "" {
			continue
		}
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			fallback := configuredMB
			if fallback <= 0 {
				fallback = config.DefaultBadgerBlockCacheTotalMB
			}
			common.Log.Warnf("invalid %s=%q, use configured/default %dMB", key, raw, fallback)
			return fallback
		}
		return value
	}
	if configuredMB > 0 {
		return configuredMB
	}
	return config.DefaultBadgerBlockCacheTotalMB
}

func configuredBadgerIndexCacheTotalMB(configuredMB int) int {
	if raw := os.Getenv("INDEXER_BADGER_INDEX_CACHE_TOTAL_MB"); raw != "" {
		value, err := strconv.Atoi(raw)
		// Badger IndexCacheSize=0 means unbounded/all indices in memory, not
		// disabled. Reject non-positive values so memory remains bounded.
		if err != nil || value <= 0 {
			common.Log.Warnf("invalid INDEXER_BADGER_INDEX_CACHE_TOTAL_MB=%q, use configured/default %dMB", raw, configuredMB)
		} else {
			return value
		}
	}
	if configuredMB > 0 {
		return configuredMB
	}
	return config.DefaultBadgerIndexCacheTotalMB
}

func configuredBadgerNumCompactors(configured int) int {
	if raw := os.Getenv("INDEXER_BADGER_NUM_COMPACTORS"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			common.Log.Warnf("invalid INDEXER_BADGER_NUM_COMPACTORS=%q, use configured/default %d", raw, configured)
		} else {
			return value
		}
	}
	if configured > 0 {
		return configured
	}
	return config.DefaultBadgerNumCompactors
}

func allocateBadgerIndexCache(totalMB int) map[string]int {
	if totalMB <= 0 {
		totalMB = config.DefaultBadgerIndexCacheTotalMB
	}
	plan := allocateBadgerBlockCache(totalMB)
	// Badger IndexCacheSize=0 is unbounded. Atom is disabled but its DB is
	// still opened, so give it a minimal bounded cache and take that MiB from
	// Base to keep the process-wide total exact.
	if plan["atom"] == 0 {
		plan["atom"] = 1
		if plan["base"] > 1 {
			plan["base"]--
		}
	}
	return plan
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

func openDB(filepath string, blockCacheMB, indexCacheMB, numCompactors int) (common.KVDB, error) {
	ldb := db.NewKVDBWithOptions(filepath, db.OpenOptions{
		BlockCacheMB:  blockCacheMB,
		IndexCacheMB:  indexCacheMB,
		NumCompactors: numCompactors,
	})
	if ldb == nil {
		return nil, fmt.Errorf("NewKVDB failed")
	}
	return ldb, nil
}

func (p *IndexerMgr) initDB() (err error) {
	common.Log.Info("InitDB-> start...")

	blockTotalMB := configuredBadgerBlockCacheTotalMB(p.cfg.DB.BadgerBlockCacheTotalMB)
	indexTotalMB := configuredBadgerIndexCacheTotalMB(p.cfg.DB.BadgerIndexCacheTotalMB)
	numCompactors := configuredBadgerNumCompactors(p.cfg.DB.BadgerNumCompactors)
	blockCache := allocateBadgerBlockCache(blockTotalMB)
	indexCache := allocateBadgerIndexCache(indexTotalMB)
	common.Log.Infof(
		"Badger process cache plan: block_total=%dMB index_total=%dMB compactors=%d; base=%d/%d nft=%d/%d brc20=%d/%d runes=%d/%d exotic=%d/%d ns=%d/%d ft=%d/%d local=%d/%d dkvs=%d/%d atom=%d/%d",
		blockTotalMB, indexTotalMB, numCompactors,
		blockCache["base"], indexCache["base"], blockCache["nft"], indexCache["nft"],
		blockCache["brc20"], indexCache["brc20"], blockCache["runes"], indexCache["runes"],
		blockCache["exotic"], indexCache["exotic"], blockCache["ns"], indexCache["ns"],
		blockCache["ft"], indexCache["ft"], blockCache["local"], indexCache["local"],
		blockCache["dkvs"], indexCache["dkvs"], blockCache["atom"], indexCache["atom"],
	)

	p.baseDB, err = openDB(p.dbDir+"base", blockCache["base"], indexCache["base"], numCompactors)
	if err != nil {
		return err
	}
	p.nftDB, err = openDB(p.dbDir+"nft", blockCache["nft"], indexCache["nft"], numCompactors)
	if err != nil {
		return err
	}
	p.nsDB, err = openDB(p.dbDir+"ns", blockCache["ns"], indexCache["ns"], numCompactors)
	if err != nil {
		return err
	}
	p.exoticDB, err = openDB(p.dbDir+"exotic", blockCache["exotic"], indexCache["exotic"], numCompactors)
	if err != nil {
		return err
	}
	p.ftDB, err = openDB(p.dbDir+"ft", blockCache["ft"], indexCache["ft"], numCompactors)
	if err != nil {
		return err
	}
	p.brc20DB, err = openDB(p.dbDir+"brc20", blockCache["brc20"], indexCache["brc20"], numCompactors)
	if err != nil {
		return err
	}
	p.runesDB, err = openDB(p.dbDir+"runes", blockCache["runes"], indexCache["runes"], numCompactors)
	if err != nil {
		return err
	}
	p.atomDB, err = openDB(p.dbDir+"atom", blockCache["atom"], indexCache["atom"], numCompactors)
	if err != nil {
		return err
	}
	p.localDB, err = openDB(p.dbDir+"local", blockCache["local"], indexCache["local"], numCompactors)
	if err != nil {
		return err
	}
	p.kvDB, err = openDB(p.dbDir+"dkvs", blockCache["dkvs"], indexCache["dkvs"], numCompactors)
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
