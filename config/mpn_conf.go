package config

import (
	"path/filepath"
	"time"
)

const (
	defaultConfigFilename        = "mpn.conf"
	defaultDataDirname           = "data"
	defaultLogLevel              = "info"
	defaultLogDirname            = "logs"
	defaultLogFilename           = "mpn.log"
	defaultMaxPeers              = 125
	defaultBanDuration           = time.Hour * 24
	defaultBanThreshold          = 100
	defaultConnectTimeout        = time.Second * 30
	defaultMaxRPCClients         = 10
	defaultMaxRPCWebsockets      = 25
	defaultMaxRPCConcurrentReqs  = 20
	defaultDbType                = "ffldb"
	defaultFreeTxRelayLimit      = 15.0
	defaultTrickleInterval       = 10 * time.Second //peer.DefaultTrickleInterval
	defaultBlockMinSize          = 0
	defaultBlockMaxSize          = 750000
	defaultBlockMinWeight        = 0
	defaultBlockMaxWeight        = 3000000
	blockMaxSizeMin              = 1000
	blockMaxSizeMax              = 1000000 - 1000 //blockchain.MaxBlockBaseSize - 1000
	blockMaxWeightMin            = 4000
	blockMaxWeightMax            = 4000000 - 40000 // blockchain.MaxBlockWeight - 4000
	defaultGenerate              = false
	defaultMaxOrphanTransactions = 100
	defaultMaxOrphanTxSize       = 100000
	defaultSigCacheMaxSize       = 100000
	defaultUtxoCacheMaxSizeMiB   = 250
	sampleConfigFilename         = "sample.conf"
	defaultTxIndex               = false
	defaultAddrIndex             = false
	pruneMinSize                 = 1536
)

var (
	defaultHomeDir    = "./" //btcutil.AppDataDir("./", false)
	defaultConfigFile = filepath.Join(defaultHomeDir, defaultConfigFilename)
	defaultDataDir    = filepath.Join(defaultHomeDir, defaultDataDirname)
	knownDbTypes      = []string{"ffldb"} //database.SupportedDrivers()
	defaultLogDir     = filepath.Join(defaultHomeDir, defaultLogDirname)
)

func getDefaultMPNConfig() *MPNConfig {
	return &MPNConfig{
		ConfigFile:   defaultConfigFile,
		DebugLevel:   defaultLogLevel,
		MaxPeers:     defaultMaxPeers,
		BanDuration:  defaultBanDuration,
		BanThreshold: defaultBanThreshold,

		DataDir: defaultDataDir,
		DbType:  defaultDbType,

		MinRelayTxFee:       0.00001, //mempool.DefaultMinRelayTxFee.ToBTC(),
		FreeTxRelayLimit:    defaultFreeTxRelayLimit,
		TrickleInterval:     defaultTrickleInterval,
		BlockMinSize:        defaultBlockMinSize,
		BlockMaxSize:        defaultBlockMaxSize,
		BlockMinWeight:      defaultBlockMinWeight,
		BlockMaxWeight:      defaultBlockMaxWeight,
		BlockPrioritySize:   50000, //mempool.DefaultBlockPrioritySize,
		MaxOrphanTxs:        defaultMaxOrphanTransactions,
		SigCacheMaxSize:     defaultSigCacheMaxSize,
		UtxoCacheMaxSizeMiB: defaultUtxoCacheMaxSizeMiB,
		TxIndex:             defaultTxIndex,
		AddrIndex:           defaultAddrIndex,
	}
}
