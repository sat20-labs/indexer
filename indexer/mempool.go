package indexer

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"

	"sync"
	"time"

	//"github.com/pebbe/zmq4"

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
    lastSyncTime int64
    peer *peer.Peer

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
    go p.fetchMempoolFromRPC()

    netParam := instance.GetChainParam()
    addr := fmt.Sprintf("%s:%s", cfg.Host, netParam.DefaultPort)
    go p.listenP2PTx(addr)

    // zmqTxPort := "38333"
    // zmqBlockPort := "38332"
    // if !instance.IsMainnet() {
    //     zmqTxPort = "58333"
    //     zmqBlockPort = "58332"
    // }

    // zmqTxAddr := fmt.Sprintf("tcp://%s:%s", cfg.Host, zmqTxPort)
    // zmqBlockAddr := fmt.Sprintf("tcp://%s:%s", cfg.Host, zmqBlockPort)

    // go p.listenZMQTx(zmqTxAddr)
    // go p.listenZMQBlock(zmqBlockAddr)
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
    p.lastSyncTime = time.Now().Unix()
    common.Log.Infof("fetchMempoolFromRPC fetch %d tx, %v", len(txIds), time.Since(start).String())
}

// 重新同步池子
func (p *MiniMemPool) resyncMempoolFromRPC() {
    p.mutex.Lock()
    defer p.mutex.Unlock()
    start := time.Now()
    common.Log.Infof("start to fetch all tx from mempool")
    txIds, err := bitcoin_rpc.ShareBitconRpc.GetMemPool()
    if err != nil {
        common.Log.Infof("GetMemPool error: %v", err)
        return
    }
    newMap := make(map[string]bool)
    add := make([]string, 0)
    for _, txId := range txIds {
        newMap[txId] = true
        _, ok := p.txMap[txId]
        if ok {
            continue
        }
        add = append(add, txId)
    }
    del := make([]string, 0)
    for k := range p.txMap {
        _, ok := newMap[k]
        if ok {
            continue
        }
        del = append(del, k)
    }

    common.Log.Infof("resyncMempoolFromRPC, new pool size %d, old pool size %d", len(txIds), len(p.txMap))
    common.Log.Infof("resyncMempoolFromRPC, added %d, deleted %d", len(add), len(del))

    if len(txIds) <= len(add) {
        p.init()
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
        
    } else {
        for _, txId := range del {
            tx, ok := p.txMap[txId]
            if ok {
                p.txConfirmed(tx)
            }
        }

        for _, txId := range add {
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
    }

    p.lastSyncTime = time.Now().Unix()
    common.Log.Infof("resyncMempoolFromRPC completed, new size %d. %v", len(p.txMap), time.Since(start).String())
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


func getTxOutFromRawTx(utxo string) (*common.UtxoInfo, error) {
	parts := strings.Split(utxo, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid utxo %s", utxo)
	}
	vout, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}
	txHex, err := bitcoin_rpc.ShareBitconRpc.GetRawTx(parts[0])
	if err != nil {
		common.Log.Errorf("GetRawTx %s failed, %v", parts[0], err)
		return nil, err
	}
	tx, err := DecodeMsgTx(txHex)
	if err != nil {
		return nil, err
	}
	if vout >= len(tx.TxOut) {
		return nil, fmt.Errorf("invalid index of utxo %s", utxo)
	}
	return &common.UtxoInfo{
		Value: tx.TxOut[vout].Value,
		PkScript: tx.TxOut[vout].PkScript,
	}, nil
}

func (p *MiniMemPool) txBroadcasted(tx *wire.MsgTx) {
    netParam := instance.GetChainParam()
    txId := tx.TxID()
    _, ok := p.txMap[txId]
    if ok {
        common.Log.Debugf("tx %s already in mempool", txId)
        return 
    }
    p.txMap[txId] = tx
    common.Log.Debugf("add tx %s to mempool %d", txId, len(p.txMap))
    for _, txIn := range tx.TxIn {
        if txIn.PreviousOutPoint.Index >= wire.MaxPrevOutIndex {
            continue // coinbase
        }
        spentUtxo := txIn.PreviousOutPoint.String()
        info, err := instance.rpcService.GetUtxoInfo(spentUtxo)
        if err != nil {
            // 可能上个TX也在内存池中
            info, err = getTxOutFromRawTx(spentUtxo)
            if err != nil {
                common.Log.Errorf("GetTxOutFromRawTx %s failed, %v", spentUtxo, err)
                continue
            }
        }
        addr, err := common.PkScriptToAddr(info.PkScript, netParam)
        if err != nil {
            common.Log.Errorf("PkScriptToAddr %s failed, %v", hex.EncodeToString(info.PkScript), err)
            continue
        }
        p.spentUtxoMap[spentUtxo] = addr
        common.Log.Debugf("add utxo %s to spentUtxoMap with %s", spentUtxo, addr)
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
            //common.Log.Errorf("PkScriptToAddr %s failed, %v", hex.EncodeToString(txOut.PkScript), err)
            continue
        }
        p.unConfirmedUtxoMap[unconfirmedUtxo] = addr
        common.Log.Debugf("add utxo %s to unConfirmedUtxoMap with %s", unconfirmedUtxo, addr)
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
        common.Log.Debugf("tx %s removed from mempool %d", txId, len(p.txMap))
    }
    
    netParam := instance.GetChainParam()
    for _, txIn := range tx.TxIn {
        if txIn.PreviousOutPoint.Index >= wire.MaxPrevOutIndex {
            continue // coinbase
        }
        spentUtxo := txIn.PreviousOutPoint.String()
        addr, ok := p.spentUtxoMap[spentUtxo]
        if !ok {
            // 理论上应该能找到，找不到就努力补救
            info, err := getTxOutFromRawTx(spentUtxo)
            if err == nil {
                addr, err = common.PkScriptToAddr(info.PkScript, netParam)
                if err == nil {
                    ok = true
                }
            }
        }

        delete(p.spentUtxoMap, spentUtxo)
        common.Log.Debugf("delete utxo %s from spentUtxoMap", spentUtxo)
        if !ok {
            common.Log.Errorf("can't find utxo %s in spentMap", spentUtxo)
            continue
        }

        user, ok := p.addrUtxoMap[addr]
        if ok {
            common.Log.Debugf("delete utxo %s from address %s SpentUtxo", spentUtxo, addr)
            delete(user.SpentUtxo, spentUtxo)
        }
    }

    for i, txOut := range tx.TxOut {
        if common.IsOpReturn(txOut.PkScript) {
            continue
        }
        unconfirmedUtxo := fmt.Sprintf("%s:%d", txId, i)
        addr, err := common.PkScriptToAddr(txOut.PkScript, netParam)
        if err != nil {
            //common.Log.Errorf("PkScriptToAddr %s failed, %v", hex.EncodeToString(txOut.PkScript), err)
            continue
        }
        delete(p.unConfirmedUtxoMap, unconfirmedUtxo)
        common.Log.Debugf("delete utxo %s from unConfirmedUtxoMap", unconfirmedUtxo)
        user, ok := p.addrUtxoMap[addr]
        if ok {
            common.Log.Debugf("delete utxo %s from address %s unConfirmedUtxoMap", unconfirmedUtxo, addr)
            delete(user.UnconfirmedUtxoMap, unconfirmedUtxo)
        }
    }
}


// // 通过ZMQ监听交易广播，代替P2P监听
// func (p *MiniMemPool) listenZMQTx(zmqAddr string) {
//     subscriber, err := zmq4.NewSocket(zmq4.SUB)
//     if err != nil {
//         common.Log.Errorf("ZMQ NewSocket error: %v", err)
//         return
//     }
//     defer subscriber.Close()

//     err = subscriber.Connect(zmqAddr)
//     if err != nil {
//         common.Log.Errorf("ZMQ Connect error: %v", err)
//         return
//     }
//     // 订阅所有消息
//     err = subscriber.SetSubscribe("")
//     if err != nil {
//         common.Log.Errorf("ZMQ SetSubscribe error: %v", err)
//         return
//     }

//     common.Log.Infof("ZMQ listening for raw tx at %s", zmqAddr)
//     for {
//         msg, err := subscriber.RecvMessage(0)
//         if err != nil {
//             common.Log.Errorf("ZMQ RecvMessage error: %v", err)
//             time.Sleep(time.Second)
//             continue
//         }
//         if len(msg) < 2 {
//             continue
//         }
//         topic := msg[0]
//         raw := msg[1]
//         if topic != "rawtx" {
//             continue
//         }
//         txBytes := []byte(raw)
//         // ZMQ 传递的是二进制，需要解码
//         tx := wire.NewMsgTx(wire.TxVersion)
//         err = tx.Deserialize(bytes.NewReader(txBytes))
//         if err != nil {
//             common.Log.Errorf("ZMQ tx Deserialize error: %v", err)
//             continue
//         }
//         p.mutex.Lock()
//         common.Log.Infof("ZMQ OnTx %s", tx.TxID())
//         p.txBroadcasted(tx)
//         p.mutex.Unlock()
//     }
// }

// func (p *MiniMemPool) listenZMQBlock(zmqAddr string) {
//     subscriber, err := zmq4.NewSocket(zmq4.SUB)
//     if err != nil {
//         common.Log.Errorf("ZMQ NewSocket error: %v", err)
//         return
//     }
//     defer subscriber.Close()

//     err = subscriber.Connect(zmqAddr)
//     if err != nil {
//         common.Log.Errorf("ZMQ Connect error: %v", err)
//         return
//     }
//     err = subscriber.SetSubscribe("")
//     if err != nil {
//         common.Log.Errorf("ZMQ SetSubscribe error: %v", err)
//         return
//     }

//     common.Log.Infof("ZMQ listening for raw block at %s", zmqAddr)
//     for {
//         msg, err := subscriber.RecvMessage(0)
//         if err != nil {
//             common.Log.Errorf("ZMQ RecvMessage error: %v", err)
//             time.Sleep(time.Second)
//             continue
//         }
//         if len(msg) < 2 {
//             continue
//         }
//         topic := msg[0]
//         raw := msg[1]
//         if topic != "rawblock" {
//             continue
//         }
//         blockBytes := []byte(raw)
//         block := wire.NewMsgBlock(&wire.BlockHeader{})
//         err = block.Deserialize(bytes.NewReader(blockBytes))
//         if err != nil {
//             common.Log.Errorf("ZMQ block Deserialize error: %v", err)
//             continue
//         }
//         common.Log.Infof("ZMQ OnBlock %s", block.BlockHash().String())
//         p.ProcessBlock(block)
//     }
// }

// 接受p2p的消息
func (p *MiniMemPool) listenP2PTx(addr string) {
    if p.peer != nil && p.peer.Connected() {
        return
    }
    for {
        cfg := &peer.Config{
			UserAgentName:    "MempoolSync",
			UserAgentVersion: "0.1",
			ChainParams:      instance.GetChainParam(),
			Listeners: peer.MessageListeners{
				OnTx: func(_ *peer.Peer, msg *wire.MsgTx) {
                    p.mutex.Lock()
                    defer p.mutex.Unlock()
                    if !p.running {
                        return
                    }
                    common.Log.Debugf("OnTx %s", msg.TxID())
					p.txBroadcasted(msg)
				},
                OnBlock: func(_ *peer.Peer, msg *wire.MsgBlock, buf []byte) {
                    if !p.running {
                        return
                    }
                    common.Log.Infof("OnBlock %s", msg.BlockHash().String())
                    // 需要检查当前区块是不是tip
                    p.ProcessBlock(msg)
                },
                OnInv: func(peer *peer.Peer, msg *wire.MsgInv) {
                    if !p.running {
                        return
                    }
                    common.Log.Debugf("OnInv: %v", msg.InvList)
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
        outBoundPeer, err := peer.NewOutboundPeer(cfg, addr)
        if err != nil {
            common.Log.Errorf("NewOutboundPeer error: %v", err)
            time.Sleep(time.Second * 5)
            continue
        }
        conn, err := net.Dial("tcp", addr)
        if err != nil {
            common.Log.Errorf("Dial P2P error: %v", err)
            time.Sleep(time.Second * 5)
            continue
        }
        outBoundPeer.AssociateConnection(conn)
        common.Log.Infof("Connected to P2P node: %s", addr)

        p.peer = outBoundPeer

        // 等待断开
        for outBoundPeer.Connected() {
            time.Sleep(3*time.Second)
        }
        common.Log.Warningf("Disconnected from P2P node: %s, will reconnect...", addr)
        time.Sleep(time.Second * 5)
    }
}


// 处理已经确认的tx
func (p *MiniMemPool) ProcessBlock(msg *wire.MsgBlock) {
    p.mutex.Lock()
    defer p.mutex.Unlock()

    start := time.Now()
    for _, tx := range msg.Transactions {
        p.txConfirmed(tx)
    }
     common.Log.Infof("ProcessBlock completed, new size %d. %v", len(p.txMap), time.Since(start).String())

    if time.Now().Unix() - p.lastSyncTime >= 36000 {
        go p.resyncMempoolFromRPC()
    }
}

// 处理回滚
func (p *MiniMemPool) ProcessReorg() {
    // 清空所有数据
    p.mutex.Lock()
    p.init()
    p.running = false
    p.mutex.Unlock()
    // 等主线程通过Start()重新启动
    //p.fetchMempoolFromRPC()
    common.Log.Infof("ProcessReorg, reset mempool")
    
    // 重新读内存池数据
}

// 返回没有被花费的utxo
func (p *MiniMemPool) RemoveSpentUtxo(utxos []string) []string {
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

// 返回没有被花费的utxo
func (p *MiniMemPool) IsSpent(utxo string) bool {
    p.mutex.RLock()
    defer p.mutex.RUnlock()

    _, ok := p.spentUtxoMap[utxo]
    return ok
}

// 返回内存池中的该地址的被花费的utxo
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

// 返回内存池中属于该地址的还没确认的utxo
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

// 直接从p2p协议同步数据，但比较慢
// syncedBlocks map[chainhash.Hash]int // 已同步区块
// lastBlockHash *chainhash.Hash        // 最新同步到的区块hash
// syncMutex sync.Mutex                 // 区块同步相关锁
//  OnBlock: func(_ *peer.Peer, msg *wire.MsgBlock, buf []byte) {
//     if !p.running {
//         return
//     }
//     // common.Log.Infof("OnBlock %s", msg.BlockHash().String())
//     // p.ProcessBlock(msg)

//     blockHash := msg.BlockHash()
//     prevHash := msg.Header.PrevBlock
//     p.syncMutex.Lock()
//     prevHeight, ok := p.syncedBlocks[prevHash]
//     var height int
//     if ok {
//         height = prevHeight + 1
//     } else {
//         // 如果找不到，可能是断点同步或重组，可以特殊处理
//         height = -1 // 或者查找其他来源
//     }
//     p.syncedBlocks[blockHash] = height
//     common.Log.Infof("OnBlock %s, height=%d", blockHash.String(), height)
//     p.syncMutex.Unlock()
    
//     p.lastBlockHash = &blockHash
//     p.ProcessBlock(msg)
//     common.Log.Infof("Synced block %s", blockHash.String())
//     // 主动请求下一个区块
//     p.sendGetBlocks(&chainhash.Hash{})
    
    
// },
// OnInv: func(peer *peer.Peer, msg *wire.MsgInv) {
//     if !p.running {
//         return
//     }
//     //common.Log.Debugf("OnInv: %v", msg.InvList)
//     var getDataMsg wire.MsgGetData
//     for _, inv := range msg.InvList {
//         if inv.Type == wire.InvTypeTx {
//             getDataMsg.AddInvVect(inv)
//         }

//         if inv.Type == wire.InvTypeBlock {
//             // 只请求未同步过的区块
//             _, ok := p.syncedBlocks[inv.Hash]
//             if !ok {
//                 getDataMsg.AddInvVect(inv)
//             }
//         }
//     }
//     if len(getDataMsg.InvList) > 0 {
//         peer.QueueMessage(&getDataMsg, nil)
//     }
// },

// func (p *MiniMemPool) StartBlockSyncFromGenesis() {
//     genesisHash := instance.GetChainParam().GenesisHash
//     go p.syncBlocksFromHash(genesisHash)
// }

// func (p *MiniMemPool) StartBlockSyncFromHash(startHash *chainhash.Hash) {
//     go p.syncBlocksFromHash(startHash)
// }

// func (p *MiniMemPool) syncBlocksFromHash(startHash *chainhash.Hash) {
//     p.syncMutex.Lock()
//     p.lastBlockHash = startHash
//     p.syncMutex.Unlock()
//     p.sendGetBlocks(&chainhash.Hash{}) // hashStop为零，表示同步到tip
// }

// func (p *MiniMemPool) sendGetBlocks(hashStop *chainhash.Hash) {
//     if p.peer == nil || !p.peer.Connected() {
//         common.Log.Errorf("P2P peer not connected")
//         return
//     }
//     getBlocksMsg := wire.NewMsgGetBlocks(hashStop)
//     // 你还需要设置 BlockLocatorHashes（区块定位器），否则对方不知道你从哪里开始同步
//     // 例如只同步从某个起点hash开始：
//     if p.lastBlockHash != nil {
//         getBlocksMsg.BlockLocatorHashes = append(getBlocksMsg.BlockLocatorHashes, p.lastBlockHash)
//     } else {
//         // 创世区块
//         genesisHash := instance.GetChainParam().GenesisHash
//         getBlocksMsg.BlockLocatorHashes = append(getBlocksMsg.BlockLocatorHashes, genesisHash)
//     }
//     p.peer.QueueMessage(getBlocksMsg, nil)
//     common.Log.Infof("Sent getblocks, locator=%s, stop=%s", getBlocksMsg.BlockLocatorHashes[0].String(), hashStop.String())
// }
