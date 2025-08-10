package main

import (
	"fmt"
	"os"
	"path/filepath"

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

	
	return nil
}
