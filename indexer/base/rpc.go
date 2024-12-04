package base

import (
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

// 带了延时，仅用于跑数据

func getBlockCount() (uint64, error) {
	h, err := bitcoin_rpc.ShareBitconRpc.GetBlockCount()
	if err != nil {
		n := 1
		for n < 10 {
			common.Log.Infof("GetBlockCount failed. try again ...")
			time.Sleep(time.Duration(n) * time.Second)
			n++
			h, err = bitcoin_rpc.ShareBitconRpc.GetBlockCount()
			if err == nil {
				break
			}
		}
	}

	return h, err
}

func getBlockHash(height uint64) (string, error) {
	h, err := bitcoin_rpc.ShareBitconRpc.GetBlockHash(height)
	if err != nil {
		n := 1
		for n < 10 {
			common.Log.Infof("GetBlockHash failed. try again ...")
			time.Sleep(time.Duration(n) * time.Second)
			n++
			h, err = bitcoin_rpc.ShareBitconRpc.GetBlockHash(height)
			if err == nil {
				break
			}
		}
	}
	return h, err
}

func getRawBlock(blockHash string) (string, error) {
	h, err := bitcoin_rpc.ShareBitconRpc.GetRawBlock(blockHash)
	if err != nil {
		n := 1
		for n < 10 {
			common.Log.Infof("GetRawBlock failed %v. try again ...", err)
			time.Sleep(time.Duration(n) * time.Second)
			n++
			h, err = bitcoin_rpc.ShareBitconRpc.GetRawBlock(blockHash)
			if err == nil {
				break
			}
		}
	}
	return h, err
}
