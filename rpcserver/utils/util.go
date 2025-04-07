package utils

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/share/base_indexer"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

// 先实现一个最简单的内存池管理，只根据广播的tx处理数据，后续再实现一个完整的版本 TODO
type MemPool struct {
	mutex sync.RWMutex
	bStop bool
	quit chan bool
	spentUtxoMap   map[string]string  // utxo -> txid
}

var _mp *MemPool

func NewMemPool(stopChan chan bool) *MemPool {
	mp := GetMemPool()
	mp.quit = stopChan
	return mp
}

func GetMemPool() *MemPool {
	if _mp == nil {
		_mp = &MemPool{
			bStop: false,
			spentUtxoMap: make(map[string]string),
		}
	}
	return _mp
}

func (p *MemPool) Stop() {
	p.bStop = true
}

func (p *MemPool) Start() {
	go p.run()
}

func (p *MemPool) run() {
	// 定时删除已经确认的tx
	// 同步mempool

	common.Log.Infof("MemPool start ...")
	tick := func() {
		if p.bStop {
			return
		}

		p.mutex.Lock()
		allTxs := make(map[string]bool)
		for _, v := range p.spentUtxoMap {
			allTxs[v] = true
		}
		confirmed := make(map[string]bool)
		for k := range allTxs {
			if bitcoin_rpc.IsExistTxInMemPool(k) {
				continue
			}
			confirmed[k] = true
		}
		for k := range confirmed {
			delete(p.spentUtxoMap, k)
		}
		p.mutex.Unlock()

		txs, err := bitcoin_rpc.ShareBitconRpc.GetMemPool()
		if err != nil {
			common.Log.Errorf("GetMemPool failed, %v", err)
			return
		}
		for _, tx := range txs {
			txHex, err := bitcoin_rpc.ShareBitconRpc.GetRawTx(tx)
			if err != nil {
				continue
			}
			p.AddTx(txHex)
		}
		common.Log.Infof("MemPool synced, total %d utxo in mempool", len(p.spentUtxoMap))
	}

	tick()

	duration := 3600
	ticker := time.NewTicker(time.Duration(duration) * time.Second)
	quit := p.quit
	for !p.bStop {
		select {
		case <-ticker.C:
			tick()
		case <-quit:
			p.bStop = true
		}
	}

	common.Log.Infof("MemPool exit.")
}

func (p *MemPool) AddTx(txstr string) {
	tx, err := DecodeMsgTx(txstr)
	if err != nil {
		return
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()
	txId := tx.TxID()
	for _, txIn := range tx.TxIn {
		p.spentUtxoMap[txIn.PreviousOutPoint.String()] = txId
	}
}

func (p *MemPool) IsExisting(utxo string) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	_, ok := p.spentUtxoMap[utxo]
	return ok
}


func IsExistingInMemPool(utxo string) bool {
	// isExist, err := bitcoin_rpc.IsExistUtxoInMemPool(utxo)
	// if err != nil {
	// 	common.Log.Errorf("GetUnspendTxOutput %s failed. %v", utxo, err)
	// 	return false
	// }
	// return isExist
	return GetMemPool().IsExisting(utxo)
}

func IsAvailableUtxoId(utxoId uint64) bool {
	return IsAvailableUtxo(base_indexer.ShareBaseIndexer.GetUtxoById(utxoId))
}

func IsAvailableUtxo(utxo string) bool {
	//Find common utxo (that is, utxo with non-ordinal attributes)
	if base_indexer.ShareBaseIndexer.HasAssetInUtxo(utxo, false) {
		return false
	}

	if IsExistingInMemPool(utxo) {
		return false
	}

	return true
}


func DecodeMsgTx(txHex string) (*wire.MsgTx, error) {
	// 1. 将十六进制字符串解码为字节切片
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		return nil, fmt.Errorf("error decoding hex string: %v", err)
	}

	// 2. 创建一个新的 wire.MsgTx 对象
	msgTx := wire.NewMsgTx(wire.TxVersion)

	// 3. 从字节切片中解析交易
	err = msgTx.Deserialize(bytes.NewReader(txBytes))
	if err != nil {
		return nil, fmt.Errorf("error deserializing transaction: %v", err)
	}

	return msgTx, nil
}
