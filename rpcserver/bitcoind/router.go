package bitcoind

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/config"
	"github.com/sat20-labs/indexer/share/btclucky"
)

type Service struct {
	btcLucky *btclucky.TemplateService
}

func NewService(cfg *config.YamlConf, localDB common.KVDB) *Service {
	s := &Service{}
	if cfg == nil {
		return s
	}
	network := cfg.Chain
	if network == "" {
		network = "mainnet"
	}
	svc, err := btclucky.NewTemplateService(btclucky.BTCLuckyTemplateServiceConfig{
		Enabled:       true,
		Backend:       "bitcoin-core",
		RPCConnect:    fmt.Sprintf("%s:%d", cfg.ShareRPC.Bitcoin.Host, cfg.ShareRPC.Bitcoin.Port),
		RPCUser:       cfg.ShareRPC.Bitcoin.User,
		RPCPass:       cfg.ShareRPC.Bitcoin.Password,
		RPCDisableTLS: true,
		Network:       network,
		CacheLimit:    16,
		FoundBlocksDB: localDB,
	})
	if err != nil {
		common.Log.Warnf("BTC lucky template service disabled: %v", err)
		return s
	}
	if err := svc.Start(); err != nil {
		common.Log.Warnf("BTC lucky template service not ready: %v", err)
	}
	s.btcLucky = svc
	return s
}

func (s *Service) BTCLuckyTemplateService() *btclucky.TemplateService {
	if s == nil {
		return nil
	}
	return s.btcLucky
}

func (s *Service) InitRouter(r *gin.Engine, basePath string) {
	r.POST(basePath+"/v3/bitcoin/utxos/by-scripts", s.getBitcoinUTXOsByScripts)
	r.POST(basePath+"/v3/bitcoin/utxos/status", s.getBitcoinUTXOStatuses)
	r.POST(basePath+"/v3/bitcoin/tx/status/batch", s.getBitcoinTxStatuses)
	r.POST(basePath+"/v3/bitcoin/rawtx/batch", s.getBitcoinRawTxs)
	r.POST(basePath+"/v3/bitcoin/outspends/batch", s.getBitcoinOutspends)
	r.POST(basePath+"/v3/bitcoin/tx/broadcast", s.broadcastBitcoinTx)
	r.GET(basePath+"/v3/bitcoin/tip", s.getBitcoinTip)
	r.GET(basePath+"/v3/bitcoin/block-header/:height", s.getBitcoinBlockHeader)
	r.GET(basePath+"/v3/bitcoin/fee-rate", s.getBitcoinFeeRate)

	//broadcast raw tx => blockstream api: POST /tx
	r.POST(basePath+"/btc/tx", s.sendRawTx)
	r.POST(basePath+"/btc/txs", s.sendRawTxs)
	r.POST(basePath+"/btc/tx/test", s.testRawTx)
	r.GET(basePath+"/btc/tx/:txid", s.getTxInfo)
	r.GET(basePath+"/btc/tx/simpleinfo/:txid", s.getTxSimpleInfo)
	r.GET(basePath+"/btc/rawtx/:txid", s.getRawTx)
	r.GET(basePath+"/btc/block/:blockhash", s.getRawBlock)
	r.GET(basePath+"/btc/block/blockhash/:height", s.getBlockHash)
	r.GET(basePath+"/btc/block/bestblockheight", s.getBestBlockHeight)
	r.GET(basePath+"/btc/fee/summary", s.feeSummary)
	r.POST(basePath+"/btc/lucky/job", s.getBTCLuckyJob)
	r.POST(basePath+"/btc/lucky/submit", s.submitBTCLuckySolution)
	r.GET(basePath+"/btc/lucky/info", s.getBTCLuckyInfo)
}
