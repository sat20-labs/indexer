package base

import (
	"github.com/gin-gonic/gin"
	"github.com/sat20-labs/indexer/share/base_indexer"
)

type Service struct {
	model *Model
	// SP scans can be substantially heavier than normal Base RPC reads.
	// Keep the existing four-request cap without a separate HTTP service.
	spRequests chan struct{}
}

func NewService(i base_indexer.Indexer) *Service {
	return &Service{
		model:      NewModel(i),
		spRequests: make(chan struct{}, 4),
	}
}

func (s *Service) InitRouter(r *gin.Engine, basePath string) {
	// 心跳
	r.GET(basePath+"/health", s.getHealth)
	//查询支持的稀有聪类型
	r.GET(basePath+"/info/satributes", s.getSatributes)
	//获取地址上大于指定value的utxo;如果value=0,获得所有可用的utxo
	r.GET(basePath+"/utxo/address/:address/:value", s.getPlainUtxos)
	//获取地址上获得所有utxo
	r.GET(basePath+"/allutxos/address/:address", s.getAllUtxos)

	// Silent Payments uses the same Service and router as other Base RPCs.
	r.GET(basePath+"/v1/sp/status", s.getSPStatus)
	r.GET(basePath+"/v1/sp/live-snapshot", s.getSPLiveSnapshot)
	r.GET(basePath+"/v1/sp/scan-data", s.getSPScanData)
	r.POST(basePath+"/v1/sp/txids-by-position", s.getSPTxIDsByPosition)
	r.POST(basePath+"/v1/sp/outpoints/query", s.getSPOutpoints)
}
