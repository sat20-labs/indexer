package bitcoin_rpc

import (
	"encoding/hex"
	"fmt"

	"github.com/OLProtocol/go-bitcoind"
	"github.com/sat20-labs/indexer/common"
)

type RESTClient struct {
	Scheme string
	Host   string
	Proxy  string
	Http   HttpClient
}

func NewRESTClient(scheme, host, proxy string, http HttpClient) *RESTClient {
	net := "mainnet"
	if proxy == "" {
		proxy = net
	}

	if scheme == "" {
		scheme = "https"
	}

	return &RESTClient{
		Scheme: scheme,
		Host:   host,
		Proxy:  proxy,
		Http:   http,
	}
}

func (p *RESTClient) GetUrl(path string) *URL {
	return &URL{
		Scheme: p.Scheme,
		Host:   p.Host,
		Path:   p.Proxy + path,
	}
}


type BlockStreamClient struct {
	*RESTClient
}

func NewBlockStreamClient(scheme, host, proxy string, http HttpClient) *BlockStreamClient {
	client := NewRESTClient(scheme, host, proxy, http)
	return &BlockStreamClient{client}
}


func (p *BlockStreamClient) SendTx(signedTxHex string) (string, error) {
	return ShareBitconRpc.SendRawTransaction(signedTxHex, 0)
}

func (p *BlockStreamClient) GetTxHex(txId string) (string, error) {
	url := p.GetUrl("/tx/" + txId + "/hex")
	rsp, err := p.Http.SendGetRequest(url)
	if err != nil {
		common.Log.Errorf("SendGetRequest %v failed. %v", url, err)
		return "", err
	}

	common.Log.Infof("%v response: %s", url, string(rsp))

	return hex.EncodeToString(rsp), nil
}


func (p *BlockStreamClient) GetTx(txid string) (*bitcoind.RawTransaction, error) {
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

func (p *BlockStreamClient) GetRawTx(txid string) (string, error) {
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

func (p *BlockStreamClient) GetTxHeight(txid string) (int64, error) {
	blockHeader, err := GetBlockHeaderWithTx(txid)
	if err != nil {
		return 0, err
	}
	return blockHeader.Height, nil
}


func (p *BlockStreamClient) GetBestBlockHash() (string, error) {
	return ShareBitconRpc.GetBestBlockhash()
}

func (p *BlockStreamClient) GetRawBlock(blockHash string) (string, error) {
	return ShareBitconRpc.GetRawBlock(blockHash)
}

func (p *BlockStreamClient) GetBlockHash(height uint64) (string, error) {
	return ShareBitconRpc.GetBlockHash(height)
}

func (p *BlockStreamClient) GetBlockHeader(blockhash string) (*bitcoind.BlockHeader, error) {
	return ShareBitconRpc.GetBlockheader(blockhash)
}

func (p *BlockStreamClient) GetBlockHeaderWithTx(txid string) (*bitcoind.BlockHeader, error) {
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

func (p *BlockStreamClient) IsExistTxInMemPool(txid string) bool {
	_, err := ShareBitconRpc.GetMemPoolEntry(txid)
	return err == nil
}

// TODO 需要本地维护一个mempool，加快查询速度
func (p *BlockStreamClient) IsExistUtxoInMemPool(utxo string) (bool, error) {
	txid, vout, err := common.ParseUtxo(utxo)
	if err != nil {
		return false, err
	}
	entry, err := ShareBitconRpc.GetUnspendTxOutput(txid, vout, true)
	if err != nil {
		return false, err
	}
	return entry.Confirmations == 0, nil
}

// TODO 需要本地维护一个mempool，加快查询速度
func (p *BlockStreamClient) GetMemPool() ([]string, error) {
	return ShareBitconRpc.GetRawMempool()
}

func (p *BlockStreamClient) GetMemPoolEntry(txId string)  (*bitcoind.MemPoolEntry, error) {
	return ShareBitconRpc.GetMemPoolEntry(txId)
}
