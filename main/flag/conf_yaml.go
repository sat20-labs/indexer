package flag

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/sat20-labs/indexer/main/conf"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
	"github.com/sirupsen/logrus"
)

func LoadYamlConf(cfgPath string) (*conf.YamlConf, error) {
	confFile, err := os.Open(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cfg: %s, error: %s", cfgPath, err)
	}
	defer confFile.Close()

	ret := &conf.YamlConf{}
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

	rpcService, err := rpcwire.ParseRpcService(ret.RPCService)
	if err != nil {
		return nil, err
	}
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

	ret.RPCService = rpcService

	return ret, nil
}

func NewDefaultYamlConf(chain string) (*conf.YamlConf, error) {
	bitcoinPort := 18332
	switch chain {
	case "mainnet":
		bitcoinPort = 8332
	case "testnet":
		bitcoinPort = 18332
	case "testnet4":
		bitcoinPort = 28332
	default:
		return nil, fmt.Errorf("unsupported chain: %s", chain)
	}
	ret := &conf.YamlConf{
		Chain: chain,
		DB: conf.DB{
			Path: "db",
		},
		ShareRPC: conf.ShareRPC{
			Bitcoin: conf.Bitcoin{
				Host:     "host",
				Port:     bitcoinPort,
				User:     "user",
				Password: "password",
			},
		},
		Log: conf.Log{
			Level: "error",
			Path:  "log",
		},
		BasicIndex: conf.BasicIndex{
			MaxIndexHeight:  0,
			PeriodFlushToDB: 12,
		},
		RPCService: rpcwire.RPCService{
			Addr:  "0.0.0.0:80",
			Proxy: chain,
			Swagger: rpcwire.Swagger{
				Host:    "127.0.0.0",
				Schemes: []string{"http"},
			},
			API: rpcwire.API{
				APIKeyList:      make(map[string]*rpcwire.APIKey),
				NoLimitApiList:  []string{"/health"},
				NoLimitHostList: []string{},
			},
		},
	}

	return ret, nil
}

func SaveYamlConf(conf *conf.YamlConf, filePath string) error {
	data, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}
