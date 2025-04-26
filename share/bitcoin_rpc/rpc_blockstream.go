package bitcoin_rpc

import (
	"encoding/hex"

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


func (p *BlockStreamClient) TestTx(signedTxHex string) (*bitcoind.TransactionTestResult, error) {
	return nil, nil
}

func (p *BlockStreamClient) SendTx(signedTxHex string) (string, error) {
	return "", nil
}

func (p *BlockStreamClient) GetTx(txid string) (*bitcoind.RawTransaction, error) {
	return nil, nil
}

func (p *BlockStreamClient) GetRawTx(txId string) (string, error) {
	url := p.GetUrl("/tx/" + txId + "/hex")
	rsp, err := p.Http.SendGetRequest(url)
	if err != nil {
		common.Log.Errorf("SendGetRequest %v failed. %v", url, err)
		return "", err
	}

	common.Log.Infof("%v response: %s", url, string(rsp))

	return hex.EncodeToString(rsp), nil
}

func (p *BlockStreamClient) GetBlockCount() (uint64, error) {
	return 0, nil
}

func (p *BlockStreamClient) GetBestBlockHash() (string, error) {
	return "", nil
}

func (p *BlockStreamClient) GetRawBlock(blockHash string) (string, error) {
	return "", nil
}

func (p *BlockStreamClient) GetBlockHash(height uint64) (string, error) {
	return "", nil
}

func (p *BlockStreamClient) GetBlockHeader(blockhash string) (*bitcoind.BlockHeader, error) {
	return nil, nil
}


// TODO 需要本地维护一个mempool，加快查询速度
func (p *BlockStreamClient) GetMemPool() ([]string, error) {
	return nil, nil
}

func (p *BlockStreamClient) GetMemPoolEntry(txId string)  (*bitcoind.MemPoolEntry, error) {
	return nil, nil
}

func (p *BlockStreamClient) EstimateSmartFeeWithMode(minconf int, mode string) (*bitcoind.EstimateSmartFeeResult, error) {
	
	return nil, nil
}
