package bitcoin_rpc

import (
	"fmt"
	"strings"

	"github.com/OLProtocol/go-bitcoind"
)


func InitBitconRpc(host string, port int, user, passwd string, useSSL bool) error {
	
	if strings.Contains(host, "blockstream") {

	} else {
		rpc, err := bitcoind.New(
			host,
			port,
			user,
			passwd,
			useSSL,
			120, // server timeout is 1 hour for debug
		)
		if err != nil {
			return err
		}
		ShareBitconRpc = &BitcoindRPC{
			bitcoind: rpc,
		}
	}
	return nil
}

type BitcoindRPC struct {
	bitcoind *bitcoind.Bitcoind
}


func (p *BitcoindRPC) TestTx(signedTxHex string) (*bitcoind.TransactionTestResult, error) {
	resp, err := p.bitcoind.TestMempoolAccept(signedTxHex)
	if err != nil {
		return nil, err
	}
	ret, ok := resp.(bitcoind.TransactionTestResult)
	if !ok {
		return nil, fmt.Errorf("invalid TransactionTestResult type")
	}
	return &ret, nil
}


func (p *BitcoindRPC) SendTx(signedTxHex string) (string, error) {
	return p.bitcoind.SendRawTransaction(signedTxHex, 0)
}

func (p *BitcoindRPC) GetTx(txid string) (*bitcoind.RawTransaction, error) {
	resp, err := p.bitcoind.GetRawTransaction(txid, true)
	if err != nil {
		return nil, err
	}
	ret, ok := resp.(bitcoind.RawTransaction)
	if !ok {
		return nil, fmt.Errorf("invalid RawTransaction type")
	}
	return &ret, nil
}

func (p *BitcoindRPC) GetRawTx(txid string) (string, error) {
	resp, err := p.bitcoind.GetRawTransaction(txid, false)
	if err != nil {
		return "", err
	}
	ret, ok := resp.(string)
	if !ok {
		return "", fmt.Errorf("invalid string type")
	}
	return ret, nil
}

func (p *BitcoindRPC) GetBlockCount() (uint64, error) {
	return p.bitcoind.GetBlockCount()
}

func (p *BitcoindRPC) GetBestBlockHash() (string, error) {
	return p.bitcoind.GetBestBlockhash()
}

func (p *BitcoindRPC) GetRawBlock(blockHash string) (string, error) {
	return p.bitcoind.GetRawBlock(blockHash)
}

func (p *BitcoindRPC) GetBlockHash(height uint64) (string, error) {
	return p.bitcoind.GetBlockHash(height)
}

func (p *BitcoindRPC) GetBlockHeader(blockhash string) (*bitcoind.BlockHeader, error) {
	return p.bitcoind.GetBlockheader(blockhash)
}

func IsExistTxInMemPool(txid string) bool {
	_, err := ShareBitconRpc.GetMemPoolEntry(txid)
	return err == nil
}

// TODO 需要本地维护一个mempool，加快查询速度
func (p *BitcoindRPC) GetMemPool() ([]string, error) {
	return p.bitcoind.GetRawMempool()
}

func (p *BitcoindRPC) GetMemPoolEntry(txId string)  (*bitcoind.MemPoolEntry, error) {
	return p.bitcoind.GetMemPoolEntry(txId)
}

func (p *BitcoindRPC) EstimateSmartFeeWithMode(minconf int, mode string) (*bitcoind.EstimateSmartFeeResult, error) {
	ret, err := p.bitcoind.EstimateSmartFeeWithMode(minconf, mode)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}


// 提供一些接口，可以快速同步mempool中的数据，并将数据保存在本地kv数据库
// 1. 启动一个线程，或者一个被动的监听接口，监控内存池的新增tx的信息，
//    需要先获取mempool中所有tx（仅在初始化时调用），并且按照utxo为索引保存在数据库，
//    输入的utxo的spent为true，输出的utxo的spent为false
//    一个utxo很可能在生成后就马上被花费，所以生成时spent为false，被花费时设置为true
//    在上面的基础上，快速获取增量的tx（一般5s调用一次，期望10ms内完成操作）
// 2. 查询接口，查询一个utxo是否已经被花费，数据库查询，代替 IsExistUtxoInMemPool
// 3. 删除接口，删除一个UTXO（该utxo作为输入的tx所在block已经完成）
// 4. 以后可能会有很多基于内存池的操作，比如检查下内存池都是什么类型的tx，是否可以做RBF等等
