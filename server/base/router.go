package base

import (
	"github.com/gin-gonic/gin"
	"github.com/sat20-labs/ordx/share/base_indexer"
)

type Service struct {
	model *Model
}

func NewService(i base_indexer.Indexer) *Service {
	return &Service{
		model: NewModel(i),
	}
}

func (s *Service) InitRouter(r *gin.Engine, basePath string) {
	// 心跳
	r.GET(basePath+"/health", s.getHealth)
	// 获取聪的属性
	r.GET(basePath+"/sat/:sat", s.getSatInfo)
	// 查找钱包中是否存在某些聪
	r.POST(basePath+"/sat/FindSatsInAddress", s.findSatsInAddress)
	// 全局查找，看看聪在哪里，花的时间很长，一个小时左右
	r.GET(basePath+"/sat/FindSat/:sat", s.findSat)
	//查询支持的稀有聪类型
	r.GET(basePath+"/info/satributes", s.getSatributes)
	//获取地址上大于指定value的utxo;如果value=0,获得所有可用的utxo
	r.GET(basePath+"/utxo/address/:address/:value", s.getPlainUtxos)
	//获取地址上获得所有utxo
	r.GET(basePath+"/allutxos/address/:address", s.getAllUtxos)
	//获取地址上有某种类型稀有聪的utxo
	r.GET(basePath+"/exotic/address/:address/:type", s.getExoticUtxosWithType)
	//获取地址上的稀有聪
	r.GET(basePath+"/exotic/address/:address", s.getExoticUtxos)
	//查询utxo上所有聪的数量和属性
	r.GET(basePath+"/exotic/utxo/:utxo", s.getExoticWithUtxo)
}
