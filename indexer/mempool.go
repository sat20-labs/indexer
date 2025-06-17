package indexer

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"

	"sync"
	"time"

    "github.com/pebbe/zmq4"

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
     p.running = false
}

// indexer同步到最高区块，再启动
func (p *MiniMemPool) Start(cfg *config.Bitcoin) {
    if p.running {
        return
    }

    if instance.GetSyncHeight() != instance.GetChainTip() {
        return
    }

    p.running = true
    // 定时每10小时同步一次mempool
    go func() {
        for {
            p.fetchMempoolFromRPC()
            time.Sleep(10 * time.Hour)
        }
    }()

    // netParam := instance.GetChainParam()
    // addr := fmt.Sprintf("%s:%s", cfg.Host, netParam.DefaultPort)
    // go p.listenP2PTx(addr)

    zmqTxPort := "38333"
    zmqBlockPort := "38332"
    if !instance.IsMainnet() {
        zmqTxPort = "58333"
        zmqBlockPort = "58332"
    }

    zmqTxAddr := fmt.Sprintf("tcp://%s:%s", cfg.Host, zmqTxPort)
    zmqBlockAddr := fmt.Sprintf("tcp://%s:%s", cfg.Host, zmqBlockPort)

    go p.listenZMQTx(zmqTxAddr)
    go p.listenZMQBlock(zmqBlockAddr)
}

// 通过RPC拉取mempool
func (p *MiniMemPool) fetchMempoolFromRPC() {
    p.mutex.Lock()
    defer p.mutex.Unlock()
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
        common.Log.Infof("tx %s already in mempool", txId)
        return 
    }
    p.txMap[txId] = tx
    common.Log.Infof("add tx %s to mempool %d", txId, len(p.txMap))
    for _, txIn := range tx.TxIn {
        if txIn.PreviousOutPoint.Index >= wire.MaxPrevOutIndex {
            continue // coinbase
        }
        spentUtxo := txIn.PreviousOutPoint.String()
        info, err := instance.rpcService.GetUtxoInfo(spentUtxo)
        if err != nil {
            common.Log.Errorf("GetUtxoInfo %s failed, %v", spentUtxo, err)
            continue
        }
        addr, err := common.PkScriptToAddr(info.PkScript, netParam)
        if err != nil {
            common.Log.Errorf("PkScriptToAddr %s failed, %v", hex.EncodeToString(info.PkScript), err)
            continue
        }
        p.spentUtxoMap[spentUtxo] = addr
        common.Log.Infof("add utxo %s to spentUtxoMap with %s", spentUtxo, addr)
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
        common.Log.Infof("add utxo %s to unConfirmedUtxoMap with %s", unconfirmedUtxo, addr)
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
    if ok {
        delete(p.txMap, txId)
        common.Log.Infof("tx %s removed from mempool %d", txId, len(p.txMap))
    }
    
    for _, txIn := range tx.TxIn {
        if txIn.PreviousOutPoint.Index >= wire.MaxPrevOutIndex {
            continue // coinbase
        }
        spentUtxo := txIn.PreviousOutPoint.String()
        addr, ok := p.spentUtxoMap[spentUtxo]
        delete(p.spentUtxoMap, spentUtxo)
        common.Log.Infof("delete utxo %s from spentUtxoMap", spentUtxo)
        if !ok {
            common.Log.Infof("can't find utxo %s in spentMap", spentUtxo)
            continue
        }

        user, ok := p.addrUtxoMap[addr]
        if ok {
            common.Log.Infof("delete utxo %s from address %s SpentUtxo", spentUtxo, addr)
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
        common.Log.Infof("delete utxo %s from unConfirmedUtxoMap", unconfirmedUtxo)
        if !ok {
            common.Log.Infof("can't find utxo %s in unConfirmedUtxoMap", unconfirmedUtxo)
            continue
        }

        user, ok := p.addrUtxoMap[addr]
        if ok {
            common.Log.Infof("delete utxo %s from address %s unConfirmedUtxoMap", unconfirmedUtxo, addr)
            delete(user.UnconfirmedUtxoMap, unconfirmedUtxo)
        }
    }
}


// 通过ZMQ监听交易广播，代替P2P监听
func (p *MiniMemPool) listenZMQTx(zmqAddr string) {
    subscriber, err := zmq4.NewSocket(zmq4.SUB)
    if err != nil {
        common.Log.Errorf("ZMQ NewSocket error: %v", err)
        return
    }
    defer subscriber.Close()

    err = subscriber.Connect(zmqAddr)
    if err != nil {
        common.Log.Errorf("ZMQ Connect error: %v", err)
        return
    }
    // 订阅所有消息
    err = subscriber.SetSubscribe("")
    if err != nil {
        common.Log.Errorf("ZMQ SetSubscribe error: %v", err)
        return
    }

    common.Log.Infof("ZMQ listening for raw tx at %s", zmqAddr)
    for {
        msg, err := subscriber.RecvMessage(0)
        if err != nil {
            common.Log.Errorf("ZMQ RecvMessage error: %v", err)
            time.Sleep(time.Second)
            continue
        }
        if len(msg) < 2 {
            continue
        }
        topic := msg[0]
        raw := msg[1]
        if topic != "rawtx" {
            continue
        }
        txBytes := []byte(raw)
        // ZMQ 传递的是二进制，需要解码
        tx := wire.NewMsgTx(wire.TxVersion)
        err = tx.Deserialize(bytes.NewReader(txBytes))
        if err != nil {
            common.Log.Errorf("ZMQ tx Deserialize error: %v", err)
            continue
        }
        p.mutex.Lock()
        common.Log.Infof("ZMQ OnTx %s", tx.TxID())
        p.txBroadcasted(tx)
        p.mutex.Unlock()
    }
}

func (p *MiniMemPool) listenZMQBlock(zmqAddr string) {
    subscriber, err := zmq4.NewSocket(zmq4.SUB)
    if err != nil {
        common.Log.Errorf("ZMQ NewSocket error: %v", err)
        return
    }
    defer subscriber.Close()

    err = subscriber.Connect(zmqAddr)
    if err != nil {
        common.Log.Errorf("ZMQ Connect error: %v", err)
        return
    }
    err = subscriber.SetSubscribe("")
    if err != nil {
        common.Log.Errorf("ZMQ SetSubscribe error: %v", err)
        return
    }

    common.Log.Infof("ZMQ listening for raw block at %s", zmqAddr)
    for {
        msg, err := subscriber.RecvMessage(0)
        if err != nil {
            common.Log.Errorf("ZMQ RecvMessage error: %v", err)
            time.Sleep(time.Second)
            continue
        }
        if len(msg) < 2 {
            continue
        }
        topic := msg[0]
        raw := msg[1]
        if topic != "rawblock" {
            continue
        }
        blockBytes := []byte(raw)
        block := wire.NewMsgBlock(&wire.BlockHeader{})
        err = block.Deserialize(bytes.NewReader(blockBytes))
        if err != nil {
            common.Log.Errorf("ZMQ block Deserialize error: %v", err)
            continue
        }
        common.Log.Infof("ZMQ OnBlock %s", block.BlockHash().String())
        p.ProcessBlock(block)
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
                OnInv: func(peer *peer.Peer, msg *wire.MsgInv) {
                    common.Log.Infof("OnInv: %v", msg.InvList)
                    var getDataMsg wire.MsgGetData
                    for _, inv := range msg.InvList {
                        if inv.Type == wire.InvTypeTx || inv.Type == wire.InvTypeBlock {
                            getDataMsg.AddInvVect(inv)
                        }
                    }
                    if len(getDataMsg.InvList) > 0 {
                        peer.QueueMessage(&getDataMsg, nil)
                    }
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
