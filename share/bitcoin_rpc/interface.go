package bitcoin_rpc

import "github.com/OLProtocol/go-bitcoind"

type BitcoinRPC interface {
	TestTx(signedTxHex string) (*bitcoind.TransactionTestResult, error)
	SendTx(signedTxHex string) (string, error)

	GetTx(txid string) (*bitcoind.RawTransaction, error)
	GetRawTx(txid string) (string, error)
	
	GetBlockCount() (uint64, error)
	GetBestBlockHash() (string, error)
	GetBlockHash(height uint64) (string, error)
	GetRawBlock(blockHash string) (string, error)
	GetBlockHeader(blockhash string) (*bitcoind.BlockHeader, error)

	GetMemPoolEntry(txid string) (*bitcoind.MemPoolEntry, error)
	GetMemPool() (txId []string, err error)

	EstimateSmartFeeWithMode(minconf int, mode string) (*bitcoind.EstimateSmartFeeResult, error)
}

var ShareBitconRpc BitcoinRPC