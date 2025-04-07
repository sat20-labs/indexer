package bitcoin_rpc

import (
	"fmt"

	"github.com/OLProtocol/go-bitcoind"
)

var ShareBitconRpc *bitcoind.Bitcoind

func InitBitconRpc(host string, port int, user, passwd string, useSSL bool) error {
	var err error
	ShareBitconRpc, err = bitcoind.New(
		host,
		port,
		user,
		passwd,
		useSSL,
		3600, // server timeout is 1 hour for debug
	)
	return err
}

func SendTx(signedTxHex string) (string, error) {
	return ShareBitconRpc.SendRawTransaction(signedTxHex, 0)
}

func GetTx(txid string) (*bitcoind.RawTransaction, error) {
	resp, err := ShareBitconRpc.GetRawTransaction(txid, true)
	if err != nil {
		return nil, err
	}
	ret, ok := resp.(bitcoind.RawTransaction)
	if !ok {
		return nil, fmt.Errorf("invalid RawTransaction type")
	}
	return &ret, nil
}

func GetRawTx(txid string) (string, error) {
	resp, err := ShareBitconRpc.GetRawTransaction(txid, false)
	if err != nil {
		return "", err
	}
	ret, ok := resp.(string)
	if !ok {
		return "", fmt.Errorf("invalid string type")
	}
	return ret, nil
}

func GetTxHeight(txid string) (int64, error) {
	blockHeader, err := GetBlockHeaderWithTx(txid)
	if err != nil {
		return 0, err
	}
	return blockHeader.Height, nil
}

func GetBestBlockHash() (string, error) {
	return ShareBitconRpc.GetBestBlockhash()
}

func GetRawBlock(blockHash string) (string, error) {
	return ShareBitconRpc.GetRawBlock(blockHash)
}

func GetBlockHash(height uint64) (string, error) {
	return ShareBitconRpc.GetBlockHash(height)
}

func GetBlockHeader(blockhash string) (*bitcoind.BlockHeader, error) {
	return ShareBitconRpc.GetBlockheader(blockhash)
}

func GetBlockHeaderWithTx(txid string) (*bitcoind.BlockHeader, error) {
	rawTx, err := GetTx(txid)
	if err != nil {
		return nil, err
	}
	blockHeader, err := ShareBitconRpc.GetBlockheader(rawTx.BlockHash)
	if err != nil {
		return nil, err
	}
	return blockHeader, nil
}

func IsExistTxInMemPool(txid string) bool {
	_, err := ShareBitconRpc.GetMemPoolEntry(txid)
	return err == nil
}


// TODO 需要本地维护一个mempool，加快查询速度
func GetMemPool() ([]string, error) {
	return ShareBitconRpc.GetRawMempool()
}

func GetMemPoolEntry(txId string)  (*bitcoind.MemPoolEntry, error) {
	return ShareBitconRpc.GetMemPoolEntry(txId)
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
