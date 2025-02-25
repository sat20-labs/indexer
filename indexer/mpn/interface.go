package mpn

import (
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/config"
)

const (
	// blockDbNamePrefix is the prefix for the block database name.  The
	// database type is appended to this value to form the full block
	// database name.
	blockDbNamePrefix = "blocks"
)

var (
	_cfg *mpnconfig
)

// StartMPN is the real main function for mpn.  It is necessary to work around
// the fact that deferred functions do not run when os.Exit() is called.  The
// optional serverChan parameter is mainly used by the service code to be
// notified with the MemPoolNode once it is setup so it can gracefully stop it when
// requested from the service control manager.
func StartMPN(yamlCfg *config.YamlConf, db *badger.DB, interrupt <-chan struct{}) (*MemPoolNode, error) {
	// Load configuration and parse command line.  This function also
	// initializes logging and configures it accordingly.
	tcfg, err := loadConfig(yamlCfg)
	if err != nil {
		return nil, err
	}
	_cfg = tcfg

	// Show version at startup.
	common.Log.Infof("Version %s", version())

	// Check if the database had previously been pruned.  If it had been, it's
	// not possible to newly generate the tx index and addr index.
	//var beenPruned bool
	// db.View(func(dbTx database.Tx) error {
	// 	beenPruned, err = dbTx.BeenPruned()
	// 	return err
	// })
	// if err != nil {
	// 	common.Log.Errorf("%v", err)
	// 	return nil, err
	// }
	// if beenPruned && _cfg.Prune == 0 {
	// 	err = fmt.Errorf("--prune cannot be disabled as the node has been "+
	// 		"previously pruned. You must delete the files in the datadir: \"%s\" "+
	// 		"and sync from the beginning to disable pruning", _cfg.DataDir)
	// 	common.Log.Errorf("%v", err)
	// 	return nil, err
	// }
	// if beenPruned && _cfg.TxIndex {
	// 	err = fmt.Errorf("--txindex cannot be enabled as the node has been "+
	// 		"previously pruned. You must delete the files in the datadir: \"%s\" "+
	// 		"and sync from the beginning to enable the desired index", _cfg.DataDir)
	// 	common.Log.Errorf("%v", err)
	// 	return nil, err
	// }
	// if beenPruned && _cfg.AddrIndex {
	// 	err = fmt.Errorf("--addrindex cannot be enabled as the node has been "+
	// 		"previously pruned. You must delete the files in the datadir: \"%s\" "+
	// 		"and sync from the beginning to enable the desired index", _cfg.DataDir)
	// 	common.Log.Errorf("%v", err)
	// 	return nil, err
	// }
	// If we've previously been pruned and the cfindex isn't present, it means that the
	// user wants to enable the cfindex after the node has already synced up and been
	// pruned.
	// if beenPruned && !indexers.CfIndexInitialized(db) && !_cfg.NoCFilters {
	// 	err = fmt.Errorf("compact filters cannot be enabled as the node has been "+
	// 		"previously pruned. You must delete the files in the datadir: \"%s\" "+
	// 		"and sync from the beginning to enable the desired index. You may "+
	// 		"use the --nocfilters flag to start the node up without the compact "+
	// 		"filters", _cfg.DataDir)
	// 	common.Log.Errorf("%v", err)
	// 	return nil, err
	// }
	// // If the user wants to disable the cfindex and is pruned or has enabled pruning, force
	// // the user to either drop the cfindex manually or restart the node without the --nocfilters
	// // flag.
	// if (beenPruned || _cfg.Prune != 0) && indexers.CfIndexInitialized(db) && _cfg.NoCFilters {
	// 	err = fmt.Errorf("--nocfilters flag was given but the compact filters have " +
	// 		"previously been enabled on this node and the index data currently " +
	// 		"exists in the database. The node has also been previously pruned and " +
	// 		"the database would be left in an inconsistent state if the compact " +
	// 		"filters don't get indexed now. To disable compact filters, please drop the " +
	// 		"index completely with the --dropcfindex flag and restart the node. " +
	// 		"To keep the compact filters, restart the node without the --nocfilters " +
	// 		"flag")
	// 	common.Log.Errorf("%v", err)
	// 	return nil, err
	// }

	// Enforce removal of txindex and addrindex if user requested pruning.
	// This is to require explicit action from the user before removing
	// indexes that won't be useful when block files are pruned.
	//
	// NOTE: The order is important here because dropping the tx index also
	// drops the address index since it relies on it.  We explicitly make the
	// user drop both indexes if --addrindex was enabled previously.
	// if _cfg.Prune != 0 && indexers.AddrIndexInitialized(db) {
	// 	err = fmt.Errorf("--prune flag may not be given when the address index " +
	// 		"has been initialized. Please drop the address index with the " +
	// 		"--dropaddrindex flag before enabling pruning")
	// 	common.Log.Errorf("%v", err)
	// 	return nil, err
	// }
	// if _cfg.Prune != 0 && indexers.TxIndexInitialized(db) {
	// 	err = fmt.Errorf("--prune flag may not be given when the transaction index " +
	// 		"has been initialized. Please drop the transaction index with the " +
	// 		"--droptxindex flag before enabling pruning")
	// 	common.Log.Errorf("%v", err)
	// 	return nil, err
	// }

	err = os.MkdirAll(_cfg.DataDir, 0700)
	if err != nil {
		return nil, err
	}

	// Create MemPoolNode and start it.
	MemPoolNode, err := newServer(_cfg.Listeners, _cfg.AgentBlacklist,
		_cfg.AgentWhitelist, db, activeNetParams.Params, interrupt)
	if err != nil {
		// TODO: this logging could do with some beautifying.
		common.Log.Errorf("Unable to start MemPoolNode on %v: %v",
			_cfg.Listeners, err)
		return nil, err
	}
	MemPoolNode.Start()

	return MemPoolNode, nil
}

func StopMPN(mpn *MemPoolNode) {
	common.Log.Infof("Gracefully shutting down the MemPoolNode...")
	mpn.Stop()
	mpn.WaitForShutdown()
	common.Log.Infof("Server shutdown complete")
}


// dbPath returns the path to the block database given a database type.
func blockDbPath(dbType string) string {
	// The database name is based on the database type.
	dbName := blockDbNamePrefix + "_" + dbType
	if dbType == "sqlite" {
		dbName = dbName + ".db"
	}
	dbPath := filepath.Join(_cfg.DataDir, dbName)
	return dbPath
}
