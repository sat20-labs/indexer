package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/sirupsen/logrus"
)

type YamlConf struct {
	Chain      string     `yaml:"chain"`
	DB         DB         `yaml:"db"`
	ShareRPC   ShareRPC   `yaml:"share_rpc"`
	Log        Log        `yaml:"log"`
	BasicIndex BasicIndex `yaml:"basic_index"`
	RPCService RPCService `yaml:"rpc_service"`
	PubKey	   string     `yaml:"pubkey"`
}

type DB struct {
	Path string `yaml:"path"`
}

type ShareRPC struct {
	Bitcoin Bitcoin `yaml:"bitcoin"`
}

type Bitcoin struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type Log struct {
	Level string `yaml:"level"`
	Path  string `yaml:"path"`
}

type BasicIndex struct {
	MaxIndexHeight  int64 `yaml:"max_index_height"`
	PeriodFlushToDB int   `yaml:"period_flush_to_db"`
}

type MPNConfig struct {
	AddCheckpoints      []string      `yaml:"addcheckpoint" description:"Add a custom checkpoint.  Format: '<height>:<hash>'"`
	AddPeers            []string      `yaml:"addpeer" description:"Add a peer to connect with at startup"`
	AddrIndex           bool          `yaml:"addrindex" description:"Maintain a full address-based transaction index which makes the searchrawtransactions RPC available"`
	AgentBlacklist      []string      `yaml:"agentblacklist" description:"A comma separated list of user-agent substrings which will cause btcd to reject any peers whose user-agent contains any of the blacklisted substrings."`
	AgentWhitelist      []string      `yaml:"agentwhitelist" description:"A comma separated list of user-agent substrings which will cause btcd to require all peers' user-agents to contain one of the whitelisted substrings. The blacklist is applied before the whitelist, and an empty whitelist will allow all agents that do not fail the blacklist."`
	BanDuration         time.Duration `yaml:"banduration" description:"How long to ban misbehaving peers.  Valid time units are {s, m, h}.  Minimum 1 second"`
	BanThreshold        uint32        `yaml:"banthreshold" description:"Maximum allowed ban score before disconnecting and banning misbehaving peers."`
	BlockMaxSize        uint32        `yaml:"blockmaxsize" description:"Maximum block size in bytes to be used when creating a block"`
	BlockMinSize        uint32        `yaml:"blockminsize" description:"Minimum block size in bytes to be used when creating a block"`
	BlockMaxWeight      uint32        `yaml:"blockmaxweight" description:"Maximum block weight to be used when creating a block"`
	BlockMinWeight      uint32        `yaml:"blockminweight" description:"Minimum block weight to be used when creating a block"`
	BlockPrioritySize   uint32        `yaml:"blockprioritysize" description:"Size in bytes for high-priority/low-fee transactions when creating a block"`
	BlocksOnly          bool          `yaml:"blocksonly" description:"Do not accept transactions from remote peers."`
	ConfigFile          string        `yaml:"configfile" description:"Path to configuration file"`
	ConnectPeers        []string      `yaml:"connect" description:"Connect only to the specified peers at startup"`
	CPUProfile          string        `yaml:"cpuprofile" description:"Write CPU profile to the specified file"`
	MemoryProfile       string        `yaml:"memprofile" description:"Write memory profile to the specified file"`
	TraceProfile        string        `yaml:"traceprofile" description:"Write execution trace to the specified file"`
	DataDir             string        `yaml:"datadir" description:"Directory to store data"`
	DbType              string        `yaml:"dbtype" description:"Database backend to use for the Block Chain"`
	DebugLevel          string        `yaml:"debuglevel" description:"Logging level for all subsystems {trace, debug, info, warn, error, critical} -- You may also specify <subsystem>=<level>,<subsystem2>=<level>,... to set the log level for individual subsystems -- Use show to list available subsystems"`
	DropAddrIndex       bool          `yaml:"dropaddrindex" description:"Deletes the address-based transaction index from the database on start up and then exits."`
	DropCfIndex         bool          `yaml:"dropcfindex" description:"Deletes the index used for committed filtering (CF) support from the database on start up and then exits."`
	DropTxIndex         bool          `yaml:"droptxindex" description:"Deletes the hash-based transaction index from the database on start up and then exits."`
	ExternalIPs         []string      `yaml:"externalip" description:"Add an ip to the list of local addresses we claim to listen on to peers"`
	FreeTxRelayLimit    float64       `yaml:"limitfreerelay" description:"Limit relay of transactions with no transaction fee to the given amount in thousands of bytes per minute"`
	Listeners           []string      `yaml:"listen" description:"Add an interface/port to listen for connections (default all interfaces port: 8333, testnet: 18333)"`
	LogDir              string        `yaml:"logdir" description:"Directory to log output."`
	MaxOrphanTxs        int           `yaml:"maxorphantx" description:"Max number of orphan transactions to keep in memory"`
	MaxPeers            int           `yaml:"maxpeers" description:"Max number of inbound and outbound peers"`
	MinRelayTxFee       float64       `yaml:"minrelaytxfee" description:"The minimum transaction fee in BTC/kB to be considered a non-zero fee."`
	DisableBanning      bool          `yaml:"nobanning" description:"Disable banning of misbehaving peers"`
	NoCFilters          bool          `yaml:"nocfilters" description:"Disable committed filtering (CF) support"`
	DisableCheckpoints  bool          `yaml:"nocheckpoints" description:"Disable built-in checkpoints.  Don't do this unless you know what you're doing."`
	DisableDNSSeed      bool          `yaml:"nodnsseed" description:"Disable DNS seeding for peers"`
	DisableListen       bool          `yaml:"nolisten" description:"Disable listening for incoming connections -- NOTE: Listening is automatically disabled if the --connect or --proxy options are used without also specifying listen interfaces via --listen"`
	NoOnion             bool          `yaml:"noonion" description:"Disable connecting to tor hidden services"`
	NoPeerBloomFilters  bool          `yaml:"nopeerbloomfilters" description:"Disable bloom filtering support"`
	NoRelayPriority     bool          `yaml:"norelaypriority" description:"Do not require free or low-fee transactions to have high priority for relaying"`
	NoWinService        bool          `yaml:"nowinservice" description:"Do not start as a background service on Windows -- NOTE: This flag only works on the command line, not in the config file"`
	DisableStallHandler bool          `yaml:"nostalldetect" description:"Disables the stall handler system for each peer, useful in simnet/regtest integration tests frameworks"`
	OnionProxy          string        `yaml:"onion" description:"Connect to tor hidden services via SOCKS5 proxy (eg. 127.0.0.1:9050)"`
	OnionProxyPass      string        `yaml:"onionpass" default-mask:"-" description:"Password for onion proxy server"`
	OnionProxyUser      string        `yaml:"onionuser" description:"Username for onion proxy server"`
	Profile             string        `yaml:"profile" description:"Enable HTTP profiling on given port -- NOTE port must be between 1024 and 65536"`
	Proxy               string        `yaml:"proxy" description:"Connect via SOCKS5 proxy (eg. 127.0.0.1:9050)"`
	ProxyPass           string        `yaml:"proxypass" default-mask:"-" description:"Password for proxy server"`
	ProxyUser           string        `yaml:"proxyuser" description:"Username for proxy server"`
	Prune               uint64        `yaml:"prune" description:"Prune already validated blocks from the database. Must specify a target size in MiB (minimum value of 1536, default value of 0 will disable pruning)"`
	RegressionTest      bool          `yaml:"regtest" description:"Use the regression test network"`
	RejectNonStd        bool          `yaml:"rejectnonstd" description:"Reject non-standard transactions regardless of the default settings for the active network."`
	RejectReplacement   bool          `yaml:"rejectreplacement" description:"Reject transactions that attempt to replace existing transactions within the mempool through the Replace-By-Fee (RBF) signaling policy."`
	RelayNonStd         bool          `yaml:"relaynonstd" description:"Relay non-standard transactions regardless of the default settings for the active network."`
	SigCacheMaxSize     uint          `yaml:"sigcachemaxsize" description:"The maximum number of entries in the signature verification cache"`
	SimNet              bool          `yaml:"simnet" description:"Use the simulation test network"`
	SigNet              bool          `yaml:"signet" description:"Use the signet test network"`
	SigNetChallenge     string        `yaml:"signetchallenge" description:"Connect to a custom signet network defined by this challenge instead of using the global default signet test network -- Can be specified multiple times"`
	SigNetSeedNode      []string      `yaml:"signetseednode" description:"Specify a seed node for the signet network instead of using the global default signet network seed nodes"`
	TestNet4            bool          `yaml:"testnet" description:"Use the test network"`
	TorIsolation        bool          `yaml:"torisolation" description:"Enable Tor stream isolation by randomizing user credentials for each connection."`
	TrickleInterval     time.Duration `yaml:"trickleinterval" description:"Minimum time between attempts to send new inventory to a connected peer"`
	UtxoCacheMaxSizeMiB uint          `yaml:"utxocachemaxsize" description:"The maximum size in MiB of the UTXO cache"`
	TxIndex             bool          `yaml:"txindex" description:"Maintain a full hash-based transaction index which makes all transactions available via the getrawtransaction RPC"`
	UserAgentComments   []string      `yaml:"uacomment" description:"Comment to add to the user agent -- See BIP 14 for more information."`
	Upnp                bool          `yaml:"upnp" description:"Use UPnP to map our listening port outside of NAT"`
	ShowVersion         bool          `yaml:"version" description:"Display version information and exit"`
	Whitelists          []string      `yaml:"whitelist" description:"Add an IP network or IP that will not be banned. (eg. 192.168.1.0/24 or ::1)"`
}

func GetBaseDir() string {
	execPath, err := os.Executable()
	if err != nil {
		return "./."
	}
	execPath = filepath.Dir(execPath)
	// if strings.Contains(execPath, "/cli") {
	// 	execPath, _ = strings.CutSuffix(execPath, "/cli")
	// }
	return execPath
}

func InitConfig(configFile string) *YamlConf {
	if configFile == "" {
		for i, item := range os.Args {
			if item == "-env" {
				if i < len(os.Args) {
					configFile = os.Args[i+1]
					break
				}
			}
		}
		if configFile == "" {
			configFile = "./.env"
		}
	}
	if !strings.HasPrefix(configFile, "/") {
		configFile = filepath.Join(GetBaseDir(), configFile)
	}

	fmt.Printf("config file: %s\n", configFile)

	cfg, err := LoadYamlConf(configFile)
	if err != nil {
		return nil
	}
	return cfg
}

func LoadYamlConf(cfgPath string) (*YamlConf, error) {
	confFile, err := os.Open(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cfg: %s, error: %s", cfgPath, err)
	}
	defer confFile.Close()

	ret := &YamlConf{}
	decoder := yaml.NewDecoder(confFile)
	err = decoder.Decode(ret)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cfg: %s, error: %s", cfgPath, err)
	}

	_, err = logrus.ParseLevel(ret.Log.Level)
	if err != nil {
		ret.Log.Level = "info"
	}

	if ret.Log.Path == "" {
		ret.Log.Path = "log"
	}
	ret.Log.Path = filepath.FromSlash(ret.Log.Path)
	if ret.Log.Path[len(ret.Log.Path)-1] != filepath.Separator {
		ret.Log.Path += string(filepath.Separator)
	}

	if ret.BasicIndex.PeriodFlushToDB <= 0 {
		ret.BasicIndex.PeriodFlushToDB = 12
	}

	if ret.BasicIndex.MaxIndexHeight <= 0 {
		ret.BasicIndex.MaxIndexHeight = -2
	}

	if ret.DB.Path == "" {
		ret.DB.Path = "db"
	}
	ret.DB.Path = filepath.FromSlash(ret.DB.Path)
	if ret.DB.Path[len(ret.DB.Path)-1] != filepath.Separator {
		ret.DB.Path += string(filepath.Separator)
	}

	rpcService := ret.RPCService
	if rpcService.Addr == "" {
		rpcService.Addr = "0.0.0.0:80"
	}

	if rpcService.Proxy == "" {
		rpcService.Proxy = "/"
	}
	if rpcService.Proxy[0] != '/' {
		rpcService.Proxy = "/" + rpcService.Proxy
	}

	if rpcService.LogPath == "" {
		rpcService.LogPath = "log"
	}

	if rpcService.Swagger.Host == "" {
		rpcService.Swagger.Host = "127.0.0.1"
	}

	if len(rpcService.Swagger.Schemes) == 0 {
		rpcService.Swagger.Schemes = []string{"http"}
	}


	
	return ret, nil
}
