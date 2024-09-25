package flag

import (
	"fmt"
	"os"
	"path/filepath"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/ordx/common"
)

func dbLogGC(dbDir string, discardRatio float64) error {
	if !filepath.IsAbs(dbDir) {
		dbDir = filepath.Clean(dbDir) + string(filepath.Separator)
	}

	_, err := os.Stat(dbDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("dbLogGC-> db directory isn't exist: %v", dbDir)
	} else if err != nil {
		return err
	}

	opts := badger.DefaultOptions(dbDir).
		WithLoggingLevel(badger.WARNING).
		WithSyncWrites(true)
	db, err := badger.Open(opts)
	if err != nil {
		return fmt.Errorf("dbLogGC-> open db error: %v", err)
	}
	defer db.Close()

	lastDbSize := int64(0)
	filepath.Walk(dbDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			lastDbSize += info.Size()
		}
		return nil
	})
	common.Log.Infof("dbLogGC-> RunValueLogGC start, db dir: %v, DB size: %d MB, discardRatio: %v", dbDir, lastDbSize/(1024*1024), discardRatio)

	gcCount := 0
	for {
		err = db.RunValueLogGC(discardRatio)
		if err == badger.ErrNoRewrite {
			break
		} else if err != nil {
			return err
		}
		gcCount++
	}
	err = db.Sync()
	if err != nil {
		return err
	}
	dirSize := int64(0)
	err = filepath.Walk(dbDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			dirSize += info.Size()
		}
		return nil
	})
	if err != nil {
		return err
	}
	common.Log.Infof("dbLogGC-> RunValueLogGC count: %v, DB size after GC: %d MB", gcCount, dirSize/(1024*1024))
	return nil
}
