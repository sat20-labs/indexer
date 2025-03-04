package common

import (
	"bytes"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
)

var Log = NewLogger()

func init() {
	logrus.SetReportCaller(true)
	Log.SetLevel(logrus.InfoLevel)
}

func NewLogger() *logrus.Logger {
	log := logrus.New()
	log.SetLevel(logrus.TraceLevel)
	log.SetFormatter(&CustomTextFormatter{})
	return log
}

func DisabledLog() {
	Log.SetOutput(io.Discard)
}

// 创建一个带模块名的日志实例
func GetLoggerEntry(module string) *logrus.Entry {
	return Log.WithField("module", module)
}

func GetDisabledLogEntry() *logrus.Entry {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger.WithField("", "")
}

type CustomTextFormatter struct{}

func (f *CustomTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b bytes.Buffer

	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	b.WriteString(fmt.Sprintf("%s ", timestamp))
	b.WriteString(fmt.Sprintf("[%s] ", entry.Level.String()))
	moduleName, ok := entry.Data["module"].(string)
	if !ok {
		moduleName = "default"
	}
	b.WriteString(fmt.Sprintf("%s: ", moduleName))
	b.WriteString(entry.Message)
	b.WriteByte('\n')

	return b.Bytes(), nil
}

