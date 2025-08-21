package mpn

import (
	"os"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/config"
	localCommon "github.com/sat20-labs/indexer/indexer/mpn/common"
)

var (
	_cfg *mpnconfig
)

// StartMPN is the real main function for mpn.  It is necessary to work around
// the fact that deferred functions do not run when os.Exit() is called.  The
// optional serverChan parameter is mainly used by the service code to be
// notified with the MemPoolNode once it is setup so it can gracefully stop it when
// requested from the service control manager.
func StartMPN(yamlCfg *config.YamlConf, db common.KVDB, indexManager localCommon.IndexManager,
	interrupt <-chan struct{}) (*MemPoolNode, error) {
	// Load configuration and parse command line.  This function also
	// initializes logging and configures it accordingly.
	tcfg, err := loadConfig(yamlCfg)
	if err != nil {
		return nil, err
	}
	_cfg = tcfg

	// Show version at startup.
	common.Log.Infof("Version %s", version())

	err = os.MkdirAll(_cfg.DataDir, 0700)
	if err != nil {
		return nil, err
	}

	// Create MemPoolNode and start it.
	MemPoolNode, err := newServer(_cfg.Listeners, _cfg.AgentBlacklist,
		_cfg.AgentWhitelist, db, activeNetParams.Params, indexManager, interrupt)
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
