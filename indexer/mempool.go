package indexer

import (
	"fmt"
	"log"
	"net"

	"sync"
	"time"

	"github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/config"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

// 一个简化的内存池数据同步线程，启动时先从节点拉取所有mempool数据，然后通过p2p协议实时同步TX

// 用于去重
var (
    mempoolTxs sync.Map // map[string]struct{}
)

func StartMempoolListener(cfg *config.Bitcoin) {
    // 1. 启动时通过RPC拉取所有mempool已有数据
    //for _, rpcAddr := range rpcNodes {
        go fetchMempoolFromRPC()
    //}

    // 2. 监听多个P2P节点，实时同步新交易
    //for _, p2pAddr := range p2pNodes {
        go listenP2PTx(cfg.Host)
    //}

    // 阻塞主线程
    select {}
}

// 通过RPC拉取mempool
func fetchMempoolFromRPC() {
    

    txids, err := bitcoin_rpc.ShareBitconRpc.GetMemPool()
    if err != nil {
        log.Printf("GetMemPool error: %v", err)
        return
    }
    for _, txid := range txids {
        if _, loaded := mempoolTxs.LoadOrStore(txid, struct{}{}); !loaded {
            fmt.Printf("[RPC] Got mempool tx: %s\n", txid)
        }
    }
}

func listenP2PTx(addr string) {
    for {
        cfg := &peer.Config{
			UserAgentName:    "MempoolSync",
			UserAgentVersion: "0.1",
			ChainParams:      instance.GetChainParam(),
			Listeners: peer.MessageListeners{
				OnTx: func(p *peer.Peer, msg *wire.MsgTx) {
					txid := msg.TxHash().String()
					if _, loaded := mempoolTxs.LoadOrStore(txid, struct{}{}); !loaded {
						fmt.Printf("[P2P] Got new tx: %s\n", txid)
					}
				},
			},
		}
        p, err := peer.NewOutboundPeer(cfg, addr)
        if err != nil {
            log.Printf("NewOutboundPeer error: %v", err)
            time.Sleep(time.Second * 5)
            continue
        }
        conn, err := net.Dial("tcp", addr)
        if err != nil {
            log.Printf("Dial P2P error: %v", err)
            time.Sleep(time.Second * 5)
            continue
        }
        p.AssociateConnection(conn)
        log.Printf("Connected to P2P node: %s", addr)

        // 等待断开
        for p.Connected() {
            time.Sleep(3*time.Second)
        }
        log.Printf("Disconnected from P2P node: %s, will reconnect...", addr)
        time.Sleep(time.Second * 5)
    }
}
