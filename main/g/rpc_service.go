package g

import (
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer"
	mainCommon "github.com/sat20-labs/indexer/main/common"
	"github.com/sat20-labs/indexer/rpcserver"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
)

func InitRpcService(indexerMgr *indexer.IndexerMgr) (*server.Rpc, error) {
	maxIndexHeight := int64(0)
	addr := ""
	host := ""
	scheme := ""
	proxy := ""
	logPath := ""
	var apiCfgData any
	if mainCommon.YamlCfg != nil {
		maxIndexHeight = mainCommon.YamlCfg.BasicIndex.MaxIndexHeight
		rpcService, err := rpcwire.ParseRpcService(mainCommon.YamlCfg.RPCService)
		if err != nil {
			return nil, err
		}
		addr = rpcService.Addr
		host = rpcService.Swagger.Host
		for _, v := range rpcService.Swagger.Schemes {
			scheme += v + ","
		}
		proxy = rpcService.Proxy
		logPath = rpcService.LogPath
		if len(rpcService.API.APIKeyList) > 0 || len(rpcService.API.NoLimitApiList) > 0 || len(rpcService.API.NoLimitHostList) > 0 {
			apiCfgData = rpcService.API
		}
	}
	chain := mainCommon.GetChain()
	rpc := server.NewRpc(indexerMgr, chain)
	if maxIndexHeight <= 0 { // default true. set to false when compiling database.
		err := rpc.Start(addr, host, scheme,
			proxy, logPath, apiCfgData)
		if err != nil {
			return rpc, err
		}
		common.Log.Info("rpc started")
	}
	return rpc, nil
}
