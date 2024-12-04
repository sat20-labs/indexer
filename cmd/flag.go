package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/config"
	"gopkg.in/yaml.v2"
)

func ParseCmdParams() {
	init := flag.String("init", "", "generate config file in current dir")
	//env := flag.String("env", ".env", "env config file, default ./.env")
	dbgc := flag.String("dbgc", "", "gc database log")
	help := flag.Bool("help", false, "show help.")
	flag.Parse()

	if *help {
		common.Log.Info("ordx server help:")
		common.Log.Info("Usage: 'ordx-server -init testnet' or 'ordx-server -init mainnet'")
		common.Log.Info("Usage: 'ordx-server -env default.yaml'")
		common.Log.Info("Usage: 'ordx-server -env .env'")
		common.Log.Info("Usage: 'ordx-server -dbgc ./db/mainnet'")
		common.Log.Info("Options:")
		common.Log.Info("  run service ->")
		common.Log.Info("    -init: init config file in current dir, default 'testnet'")
		common.Log.Info("    -env: config file, default ./.env")
		common.Log.Info("  run tool ->")
		common.Log.Info("    -dbgc: gc database log, ex: ordx-server -dbgc ./db/mainnet")
		os.Exit(0)
	}

	if *init != "" {
		err := generateDefaultCfg(*init)
		if err != nil {
			common.Log.Fatal(err)
		}
		os.Exit(0)
	}

	if *dbgc != "" {
		err := dbLogGC(*dbgc, 0.5)
		if err != nil {
			common.Log.Fatal(err)
		}
		os.Exit(0)
	}

}

func generateDefaultCfg(chain string) error {
	cfg, err := NewDefaultYamlConf(chain)
	if err != nil {
		return err
	}
	cfgPath, err := os.Getwd()
	if err != nil {
		return err
	}

	err = SaveYamlConf(cfg, cfgPath+"/default.yaml")
	if err != nil {
		return err
	}
	return nil
}

func NewDefaultYamlConf(chain string) (*config.YamlConf, error) {
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
	ret := &config.YamlConf{
		Chain: chain,
		DB: config.DB{
			Path: "db",
		},
		ShareRPC: config.ShareRPC{
			Bitcoin: config.Bitcoin{
				Host:     "host",
				Port:     bitcoinPort,
				User:     "user",
				Password: "password",
			},
		},
		Log: config.Log{
			Level: "error",
			Path:  "log",
		},
		BasicIndex: config.BasicIndex{
			MaxIndexHeight:  0,
			PeriodFlushToDB: 12,
		},
		RPCService: config.RPCService{
			Addr:  "0.0.0.0:80",
			Proxy: chain,
			Swagger: config.Swagger{
				Host:    "127.0.0.0",
				Schemes: []string{"http"},
			},
			API: config.API{
				APIKeyList:      make(map[string]*config.APIKey),
				NoLimitApiList:  []string{"/health"},
				NoLimitHostList: []string{},
			},
		},
	}

	return ret, nil
}

func SaveYamlConf(config *config.YamlConf, filePath string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

