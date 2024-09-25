package main

import (
	"path/filepath"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/ordx/common"
	"github.com/sat20-labs/ordx/indexer"
	mainCommon "github.com/sat20-labs/ordx/main/common"
	"github.com/sat20-labs/ordx/main/flag"
	"github.com/sat20-labs/ordx/main/g"
	"github.com/sat20-labs/ordx/share/base_indexer"
)

type Config struct {
	DBDir           string
	ChainParam      *chaincfg.Params
	MaxIndexHeight  int
	PeriodFlushToDB int
}

func init() {
	flag.ParseCmdParams()
	g.InitSigInt()
}

func main() {
	common.Log.Info("Starting...")
	defer func() {
		g.ReleaseRes()
		common.Log.Info("shut down")
	}()

	err := g.InitRpc()
	if err != nil {
		common.Log.Error(err)
		return
	}

	cfg := GetConfig()

	indexerMgr := indexer.NewIndexerMgr(cfg.DBDir, cfg.ChainParam, cfg.MaxIndexHeight)
	base_indexer.InitBaseIndexer(indexerMgr)
	indexerMgr.Init()

	if cfg.PeriodFlushToDB != 0 {
		indexerMgr.WithPeriodFlushToDB(cfg.PeriodFlushToDB)
	}

	_, err = g.InitRpcService(indexerMgr)
	if err != nil {
		common.Log.Error(err)
		return
	}

	stopChan := make(chan bool)
	cb := func() {
		common.Log.Info("handle SIGINT for close base indexer")
		stopChan <- true
	}
	g.RegistSigIntFunc(cb)
	common.Log.Info("base indexer start...")
	indexerMgr.StartDaemon(stopChan)

	common.Log.Info("prepare to release resource...")
}

func GetConfig() *Config {
	maxIndexHeight := int64(0)
	periodFlushToDB := int(0)
	if mainCommon.YamlCfg != nil {
		maxIndexHeight = mainCommon.YamlCfg.BasicIndex.MaxIndexHeight
		periodFlushToDB = mainCommon.YamlCfg.BasicIndex.PeriodFlushToDB
	}
	chain := mainCommon.GetChain()

	chainParam := &chaincfg.MainNetParams
	switch chain {
	case common.ChainTestnet:
		chainParam = &chaincfg.TestNet3Params
	case common.ChainTestnet4:
		chainParam = &chaincfg.TestNet3Params
		chainParam.Name = common.ChainTestnet4
	case common.ChainMainnet:
		chainParam = &chaincfg.MainNetParams
	default:
		chainParam = &chaincfg.MainNetParams
	}
	dbDir := ""
	if mainCommon.YamlCfg != nil {
		dbDir = mainCommon.YamlCfg.DB.Path
	} else {
		dbDir = "./"
	}
	if !filepath.IsAbs(dbDir) {
		dbDir = filepath.Clean(dbDir) + string(filepath.Separator)
	}

	return &Config{
		DBDir:           dbDir,
		ChainParam:      chainParam,
		MaxIndexHeight:  int(maxIndexHeight),
		PeriodFlushToDB: periodFlushToDB,
	}
}
