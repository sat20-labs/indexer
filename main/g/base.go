package g

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sat20-labs/indexer/common"
	mainCommon "github.com/sat20-labs/indexer/main/common"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
	"github.com/sirupsen/logrus"
)

var (
	SigInt         chan os.Signal
	sigIntFuncList = []func(){}
)

func InitLog() error {
	var writers []io.Writer
	logPath := ""
	var lvl logrus.Level
	if mainCommon.YamlCfg != nil {
		logPath = mainCommon.YamlCfg.Log.Path
		var err error
		lvl, err = logrus.ParseLevel(mainCommon.YamlCfg.Log.Level)
		if err != nil {
			return fmt.Errorf("failed to parse log level: %s", err)
		}
	} else {
		return fmt.Errorf("cfg is not set")
	}
	if logPath != "" {
		exePath, _ := os.Executable()
		executableName := filepath.Base(exePath)
		if strings.Contains(executableName, "debug") {
			executableName = "debug"
		}
		fileHook, err := rotatelogs.New(
			logPath+"/"+executableName+".%Y%m%d%H%M.log",
			rotatelogs.WithLinkName(logPath+"/"+executableName+".log"),
			rotatelogs.WithMaxAge(30*24*time.Hour),
			rotatelogs.WithRotationTime(24*time.Hour),
		)
		if err != nil {
			return fmt.Errorf("failed to create RotateFile hook, error: %s", err)
		}
		writers = append(writers, fileHook)
	}
	writers = append(writers, os.Stdout)
	common.Log.SetOutput(io.MultiWriter(writers...))
	common.Log.SetLevel(lvl)
	return nil
}

func InitRpc() error {
	var host string
	var port int
	var user string
	var pass string
	var dataDir string
	var logLvl logrus.Level
	var logPath string
	var periodFlushToDB int
	if mainCommon.YamlCfg != nil {
		host = mainCommon.YamlCfg.ShareRPC.Bitcoin.Host
		port = mainCommon.YamlCfg.ShareRPC.Bitcoin.Port
		user = mainCommon.YamlCfg.ShareRPC.Bitcoin.User
		pass = mainCommon.YamlCfg.ShareRPC.Bitcoin.Password
		dataDir = mainCommon.YamlCfg.DB.Path
		var err error
		logLvl, err = logrus.ParseLevel(mainCommon.YamlCfg.Log.Level)
		if err != nil {
			return fmt.Errorf("failed to parse log level: %s", err)
		}
		logPath = mainCommon.YamlCfg.Log.Path
	} else {
		// test env
		host = "192.168.1.102"
		port = 28332
		user = "jacky"
		pass = "123456"
		dataDir = "./db/testnet4"
		logLvl = logrus.DebugLevel
		logPath = "./log/testnet4/"
		periodFlushToDB = 20
	}
	chain := mainCommon.GetChain()
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

func InitSigInt() {
	count := 0
	SigInt = make(chan os.Signal, 100)
	signal.Notify(SigInt, os.Interrupt)
	go func() {
		for {
			<-SigInt
			count++
			common.Log.Infof("Received SIGINT (CTRL+C), count %d, 3 times will close db and force exit", count)
			if count >= 3 {
				ReleaseRes()
				os.Exit(1)
			} else if count == 1 {
				for index := range sigIntFuncList {
					go sigIntFuncList[index]()
				}
			}
		}
	}()
}

func RegistSigIntFunc(callback func()) {
	sigIntFuncList = append(sigIntFuncList, callback)
}

func ReleaseRes() {
}
