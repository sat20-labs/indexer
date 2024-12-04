package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sat20-labs/indexer/common"
	"github.com/sirupsen/logrus"
)



func InitLog(conf *YamlConf) error {
	var writers []io.Writer
	logPath := ""
	var lvl logrus.Level
	if conf != nil {
		logPath = conf.Log.Path
		var err error
		lvl, err = logrus.ParseLevel(conf.Log.Level)
		if err != nil {
			return fmt.Errorf("failed to parse log level: %s", err)
		}
	} else {
		return fmt.Errorf("cfg is not set")
	}
	
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
	
	writers = append(writers, os.Stdout)
	common.Log.SetOutput(io.MultiWriter(writers...))
	common.Log.SetLevel(lvl)
	return nil
}
