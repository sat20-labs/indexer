package config

import (
	"os"
	"os/signal"
	"github.com/sat20-labs/indexer/common"
)

var (
	SigInt         chan os.Signal
	sigIntFuncList = []func(){}
)

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
