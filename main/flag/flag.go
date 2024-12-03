package flag

import (
	"flag"
	"os"

	"github.com/sat20-labs/indexer/common"
	mainCommon "github.com/sat20-labs/indexer/main/common"
	"github.com/sat20-labs/indexer/main/g"
)

func ParseCmdParams() {
	init := flag.String("init", "", "generate config file in current dir")
	env := flag.String("env", ".env", "env config file, default ./.env")
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

	err := InitConf(*env)
	if err != nil {
		common.Log.Fatal(err)
	}
	err = g.InitLog()
	if err != nil {
		common.Log.Fatal(err)
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

func InitConf(cfgPath string) error {
	var err error
	mainCommon.YamlCfg, err = LoadYamlConf(cfgPath)
	if err == nil {
		return nil
	}
	return err
}
