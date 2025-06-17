package indexer

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"

	"sync"
	"time"

	"github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/config"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

// 一个简化的内存池数据同步线程，启动时先从节点拉取所有mempool数据，然后通过p2p协议实时同步TX

type UserUtxoInMempool struct {
    SpentUtxo           map[string]bool
    UnconfirmedUtxoMap  map[string]bool
}

type MiniMemPool struct {
    txMap    map[string]*wire.MsgTx          // 内存池中所有tx
    spentUtxoMap  map[string]string          // utxo->address，内存池中所有tx的输入utxo
    unConfirmedUtxoMap  map[string]string    // utxo->address，内存池中所有tx的输出utxo
    addrUtxoMap  map[string]*UserUtxoInMempool   // key:address，内存池中所有tx的输入和输出的所属地址
    running  bool

    mutex   sync.RWMutex
}

func NewMiniMemPool() *MiniMemPool {
    r := &MiniMemPool{}
    r.init()
    return r
}

func (p *MiniMemPool) init() {
    p.txMap = make(map[string]*wire.MsgTx)
    p.spentUtxoMap = make(map[string]string)
    p.unConfirmedUtxoMap = make(map[string]string)
    p.addrUtxoMap = make(map[string]*UserUtxoInMempool)
}

// indexer同步到最高区块，再启动
func (p *MiniMemPool) Start(cfg *config.Bitcoin) {
    if p.running {
        return
    }
    p.running = true
    // 1. 启动时通过RPC拉取所有mempool已有数据
    //for _, rpcAddr := range rpcNodes {
        go p.fetchMempoolFromRPC()
    //}

    netParam := instance.GetChainParam()
    addr := fmt.Sprintf("%s:%s", cfg.Host, netParam.DefaultPort)

    // 2. 监听多个P2P节点，实时同步新交易
    //for _, p2pAddr := range p2pNodes {
        go p.listenP2PTx(addr)
    //}
}

// 通过RPC拉取mempool
func (p *MiniMemPool) fetchMempoolFromRPC() {
    p.mutex.Lock()
    defer p.mutex.RLock()
    start := time.Now()
    common.Log.Infof("start to fetch all tx from mempool")
    txIds, err := bitcoin_rpc.ShareBitconRpc.GetMemPool()
    if err != nil {
        common.Log.Infof("GetMemPool error: %v", err)
        return
    }
    for _, txId := range txIds {
        txHex, err := bitcoin_rpc.ShareBitconRpc.GetRawTx(txId)
        if err != nil {
            common.Log.Errorf("GetRawTx %s failed, %v", txId, err)
            continue
        }
        tx, err := DecodeMsgTx(txHex)
        if err != nil {
            common.Log.Errorf("DecodeMsgTx %s failed, %v", txId, err)
            continue
        }
        p.txBroadcasted(tx)
    }
    common.Log.Infof("fetchMempoolFromRPC fetch %d tx, %v", len(txIds), time.Since(start).String())
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


func (p *MiniMemPool) txBroadcasted(tx *wire.MsgTx) {
    netParam := instance.GetChainParam()
    txId := tx.TxID()
    _, ok := p.txMap[txId]
    if ok {
        return 
    }
    p.txMap[txId] = tx
    for _, txIn := range tx.TxIn {
        if txIn.PreviousOutPoint.Index >= wire.MaxPrevOutIndex {
            continue // coinbase
        }
        spentUtxo := txIn.PreviousOutPoint.String()
        info, err := instance.rpcService.GetUtxoInfo(txIn.PreviousOutPoint.String())
        if err != nil {
            common.Log.Errorf("GetUtxoInfo %s failed, %v", txIn.PreviousOutPoint.String(), err)
            continue
        }
        addr, err := common.PkScriptToAddr(info.PkScript, netParam)
        if err != nil {
            common.Log.Errorf("PkScriptToAddr %s failed, %v", hex.EncodeToString(info.PkScript), err)
            continue
        }
        p.spentUtxoMap[spentUtxo] = addr
        user, ok := p.addrUtxoMap[addr]
        if !ok {
            user = &UserUtxoInMempool{
                SpentUtxo: make(map[string]bool),
                UnconfirmedUtxoMap: make(map[string]bool),
            }
            p.addrUtxoMap[addr] = user
        }
        user.SpentUtxo[spentUtxo] = true
    }

    for i, txOut := range tx.TxOut {
        if common.IsOpReturn(txOut.PkScript) {
            continue
        }
        unconfirmedUtxo := fmt.Sprintf("%s:%d", txId, i)
        addr, err := common.PkScriptToAddr(txOut.PkScript, netParam)
        if err != nil {
            common.Log.Errorf("PkScriptToAddr %s failed, %v", hex.EncodeToString(txOut.PkScript), err)
            continue
        }
        p.unConfirmedUtxoMap[unconfirmedUtxo] = addr
        user, ok := p.addrUtxoMap[addr]
        if !ok {
            user = &UserUtxoInMempool{
                SpentUtxo: make(map[string]bool),
                UnconfirmedUtxoMap: make(map[string]bool),
            }
            p.addrUtxoMap[addr] = user
        }
        user.UnconfirmedUtxoMap[unconfirmedUtxo] = true
    }
}


func (p *MiniMemPool) txConfirmed(tx *wire.MsgTx) {
    txId := tx.TxID()
    _, ok := p.txMap[txId]
    if !ok {
        return 
    }
    delete(p.txMap, txId)
    for _, txIn := range tx.TxIn {
        if txIn.PreviousOutPoint.Index >= wire.MaxPrevOutIndex {
            continue // coinbase
        }
        spentUtxo := txIn.PreviousOutPoint.String()
        addr, ok := p.spentUtxoMap[spentUtxo]
        delete(p.spentUtxoMap, spentUtxo)
        if !ok {
            continue
        }

        user, ok := p.addrUtxoMap[addr]
        if ok {
            delete(user.SpentUtxo, spentUtxo)
        }
    }

    for i, txOut := range tx.TxOut {
        if common.IsOpReturn(txOut.PkScript) {
            continue
        }
        unconfirmedUtxo := fmt.Sprintf("%s:%d", txId, i)
        p.unConfirmedUtxoMap[unconfirmedUtxo] = txId
        addr, ok := p.unConfirmedUtxoMap[unconfirmedUtxo]
        delete(p.unConfirmedUtxoMap, unconfirmedUtxo)
        if !ok {
            continue
        }

        user, ok := p.addrUtxoMap[addr]
        if !ok {
            delete(user.UnconfirmedUtxoMap, unconfirmedUtxo)
        }
    }
}

// 接受p2p的消息
func (p *MiniMemPool) listenP2PTx(addr string) {
    for {
        cfg := &peer.Config{
			UserAgentName:    "MempoolSync",
			UserAgentVersion: "0.1",
			ChainParams:      instance.GetChainParam(),
			Listeners: peer.MessageListeners{
				OnTx: func(_ *peer.Peer, msg *wire.MsgTx) {
                    p.mutex.Lock()
                    defer p.mutex.Unlock()
                    common.Log.Infof("OnTx %s", msg.TxID())
					p.txBroadcasted(msg)
				},
                OnBlock: func(_ *peer.Peer, msg *wire.MsgBlock, buf []byte) {
                    common.Log.Infof("OnBlock %s", msg.BlockHash().String())
                    // 需要检查当前区块是不是tip
                    p.ProcessBlock(msg)
                },
			},
		}
        p, err := peer.NewOutboundPeer(cfg, addr)
        if err != nil {
            common.Log.Infof("NewOutboundPeer error: %v", err)
            time.Sleep(time.Second * 5)
            continue
        }
        conn, err := net.Dial("tcp", addr)
        if err != nil {
            common.Log.Infof("Dial P2P error: %v", err)
            time.Sleep(time.Second * 5)
            continue
        }
        p.AssociateConnection(conn)
        common.Log.Infof("Connected to P2P node: %s", addr)

        // 等待断开
        for p.Connected() {
            time.Sleep(3*time.Second)
        }
        common.Log.Infof("Disconnected from P2P node: %s, will reconnect...", addr)
        time.Sleep(time.Second * 5)
    }
}


// 处理已经确认的tx
func (p *MiniMemPool) ProcessBlock(msg *wire.MsgBlock) {
    p.mutex.Lock()
    defer p.mutex.Unlock()

    for _, tx := range msg.Transactions {
        p.txConfirmed(tx)
    }
}

// 处理回滚
func (p *MiniMemPool) ProcessReorg() {
    // 清空所有数据
    p.mutex.Lock()
    defer p.mutex.Unlock()
    
    p.init()
    p.fetchMempoolFromRPC()
    
    // 重新读内存池数据
}

func (p *MiniMemPool) CheckUtxoSpent(utxos []string) []string {
    p.mutex.RLock()
    defer p.mutex.RUnlock()

    result := make([]string, 0)
    for _, utxo := range utxos {
        _, ok := p.spentUtxoMap[utxo]
        if ok {
            continue
        }
        result = append(result, utxo)
    }
    return result
}


func (p *MiniMemPool) GetSpentUtxoByAddress(address string) []string {
    p.mutex.RLock()
    defer p.mutex.RUnlock()

    addrUtxo, ok := p.addrUtxoMap[address]
    if !ok {
        return nil
    }

    result := make([]string, 0)
    for k := range addrUtxo.SpentUtxo {
        result = append(result, k)
    }
    return result
}


func (p *MiniMemPool) GetUnconfirmedUtxoByAddress(address string) []string {
    p.mutex.RLock()
    defer p.mutex.RUnlock()

    addrUtxo, ok := p.addrUtxoMap[address]
    if !ok {
        return nil
    }

    result := make([]string, 0)
    for k := range addrUtxo.UnconfirmedUtxoMap {
        result = append(result, k)
    }
    return result
}
