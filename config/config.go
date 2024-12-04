package config

import (
	"fmt"
	"os"
	"path/filepath"

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

func InitConfig() *YamlConf {
	cfgFile := GetBaseDir()+"/.env"
	cfg, err := LoadYamlConf(cfgFile)
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

