package main

import (
	"fmt"
	"path/filepath"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/config"
	"github.com/sat20-labs/indexer/indexer"
	"github.com/sat20-labs/indexer/rpcserver"
	"github.com/sat20-labs/indexer/share/base_indexer"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
	"github.com/sirupsen/logrus"
)

type Config struct {
	DBDir           string
	ChainParam      *chaincfg.Params
	MaxIndexHeight  int
	PeriodFlushToDB int
}

func init() {
	config.InitSigInt()
}

func main() {
	yamlcfg := config.InitConfig()
	config.InitLog(yamlcfg)

	common.Log.Info("Starting...")
	defer func() {
		config.ReleaseRes()
		common.Log.Info("shut down")
	}()

	err := InitRpc(yamlcfg)
	if err != nil {
		common.Log.Error(err)
		return
	}

	cfg := GetConfig(yamlcfg)

	indexerMgr := indexer.NewIndexerMgr(cfg.DBDir, cfg.ChainParam, cfg.MaxIndexHeight, cfg.PeriodFlushToDB)
	base_indexer.InitBaseIndexer(indexerMgr)
	indexerMgr.Init()

	_, err = InitRpcService(yamlcfg, indexerMgr)
	if err != nil {
		common.Log.Error(err)
		return
	}

	stopChan := make(chan bool)
	cb := func() {
		common.Log.Info("handle SIGINT for close base indexer")
		stopChan <- true
	}
	config.RegistSigIntFunc(cb)
	common.Log.Info("base indexer start...")
	indexerMgr.StartDaemon(stopChan)

	common.Log.Info("prepare to release resource...")
}


func GetConfig(conf *config.YamlConf) *Config {
	maxIndexHeight := int64(0)
	periodFlushToDB := int(0)
	
	maxIndexHeight = conf.BasicIndex.MaxIndexHeight
	periodFlushToDB = conf.BasicIndex.PeriodFlushToDB
	chain := conf.Chain

	chainParam := &chaincfg.MainNetParams
	switch chain {
	case common.ChainTestnet:
		chainParam = &chaincfg.TestNet3Params
	case common.ChainMainnet:
		chainParam = &chaincfg.MainNetParams
	default:
		chainParam = &chaincfg.MainNetParams
	}
	dbDir := ""
	if conf != nil {
		dbDir = conf.DB.Path
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


func InitRpcService(conf *config.YamlConf, indexerMgr *indexer.IndexerMgr) (*rpcserver.Rpc, error) {
	maxIndexHeight := int64(0)
	addr := ""
	host := ""
	scheme := ""
	proxy := ""
	logPath := ""
	
	maxIndexHeight = conf.BasicIndex.MaxIndexHeight
	rpcService := conf.RPCService
	addr = conf.RPCService.Addr
	host = conf.RPCService.Swagger.Host
	for _, v := range rpcService.Swagger.Schemes {
		scheme += v + ","
	}
	proxy = rpcService.Proxy
	logPath = rpcService.LogPath

	
	chain := conf.Chain
	rpc := rpcserver.NewRpc(indexerMgr, chain)
	if maxIndexHeight <= 0 { // default true. set to false when compiling database.
		err := rpc.Start(addr, host, scheme,
			proxy, logPath, &rpcService.API)
		if err != nil {
			return rpc, err
		}
		common.Log.Info("rpc started")
	}
	return rpc, nil
}


func InitRpc(conf *config.YamlConf) error {
	var host string
	var port int
	var user string
	var pass string
	var dataDir string
	var logLvl logrus.Level
	var logPath string
	var periodFlushToDB int
	
	host = conf.ShareRPC.Bitcoin.Host
	port = conf.ShareRPC.Bitcoin.Port
	user = conf.ShareRPC.Bitcoin.User
	pass = conf.ShareRPC.Bitcoin.Password
	dataDir = conf.DB.Path
	var err error
	logLvl, err = logrus.ParseLevel(conf.Log.Level)
	if err != nil {
		return fmt.Errorf("failed to parse log level: %s", err)
	}
	logPath = conf.Log.Path
	
	chain := conf.Chain
	common.Log.WithFields(logrus.Fields{
		"BitcoinChain":    chain,
		"BitcoinRPCHost":  host,
		"BitcoinRPCPort":  port,
		"BitcoinRPCUser":  user,
		"BitcoinRPCPass":  pass,
		"DataDir":         dataDir,
		"LogLevel":        logLvl,
		"LogPath":         logPath,
		"PeriodFlushToDB": periodFlushToDB,
	}).Info("using configuration")
	err = bitcoin_rpc.InitBitconRpc(
		host,
		port,
		user,
		pass,
		false,
	)
	if err != nil {
		return err
	}
	return nil
}
