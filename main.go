package main

import (

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/config"
	"github.com/sat20-labs/indexer/indexer"
	"github.com/sat20-labs/indexer/rpcserver"
	"github.com/sat20-labs/indexer/share/base_indexer"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

func init() {
	config.InitSigInt()
}

func main() {
	yamlcfg := config.InitConfig("")
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

	indexerMgr := indexer.NewIndexerMgr(yamlcfg)
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

	host = conf.ShareRPC.Bitcoin.Host
	port = conf.ShareRPC.Bitcoin.Port
	user = conf.ShareRPC.Bitcoin.User
	pass = conf.ShareRPC.Bitcoin.Password
	
	err := bitcoin_rpc.InitBitconRpc(
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
