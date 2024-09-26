package ordx

import (
	"github.com/sat20-labs/ordx/share/base_indexer"
	"github.com/gin-gonic/gin"
)

type Service struct {
	handle *Handle
}

func NewService(indexer base_indexer.Indexer) *Service {
	return &Service{
		handle: NewHandle(indexer),
	}
}

func (s *Service) InitRouter(r *gin.Engine, proxy string) {
	// root group
	// 当前网络高度
	r.GET(proxy+"/bestheight", s.handle.getBestHeight)
	r.GET(proxy+"/height/:height", s.handle.getBlockInfo)

	// address
	// 获取某个地址上所有资产和数量的列表
	r.GET(proxy+"/address/summary/:address", s.handle.getBalanceSummaryList)

	// 获取某个地址上某个ticker的utxo数据列表，utxo中不包含其他资产数据
	r.GET(proxy+"/address/utxolist/:address/:ticker", s.handle.getUtxoList)
	// 获取某个地址上某个ticker的utxo数据列表，utxo中包含其他资产数据
	r.GET(proxy+"/address/utxolist2/:address/:ticker", s.handle.getUtxoList2)
	// 获取某个地址上有资产的utxo数据列表
	r.GET(proxy+"/address/utxolist3/:address", s.handle.getUtxoList3)
	// 获取某个地址上某个铭文的铸造历史记录
	r.GET(proxy+"/address/history/:address/:ticker", s.handle.getAddressMintHistory)

	// utxo
	// 获取某个UTXO上所有的资产信息
	r.GET(proxy+"/utxo/assets/:utxo", s.handle.getAssetDetailInfo)
	r.GET(proxy+"/utxo/assetoffset/:utxo", s.handle.getAssetOffset)
	//查询utxo上的资产和数量
	r.GET(proxy+"/utxo/abbrassets/:utxo", s.handle.getAbbrAssetsWithUtxo)
	//获取utxo上的资产类型和对应的seed，seed由聪的属性（资产类型，数量，序号）决定
	r.GET(proxy+"/utxo/seed/:utxo", s.handle.getSeedWithUtxo)
	// for test
	r.GET(proxy+"/utxo/range/:utxo", s.handle.getSatRangeWithUtxo)
	r.POST(proxy+"/utxos/assets", s.handle.getAssetsWithUtxos)

	// range
	// 获取Range上所有的资产信息
	r.GET(proxy+"/range/:start/:size", s.handle.getAssetDetailInfoWithRange)
	r.POST(proxy+"/ranges", s.handle.getAssetDetailInfoWithRanges)

	// inscribe
	// 检查某个ticker是否可以deploy
	r.GET(proxy+"/deploy/:ticker/:address", s.handle.isDeployAllowed)
	r.POST(proxy+"/collection", s.handle.addCollection)

	// ft
	// 所有ticker的数据
	r.GET(proxy+"/tick/status", s.handle.getTickerStatusList)
	// 某个ticker的数据
	r.GET(proxy+"/tick/info/:ticker", s.handle.getTickerStatus)
	// 获取某个ticker的持有人和持有数量列表
	r.GET(proxy+"/tick/holders/:ticker", s.handle.getHolderList)
	// 获取某个铭文的铸造历史记录
	r.GET(proxy+"/tick/history/:ticker", s.handle.getMintHistory)
	// 获取某个ticker已经被拆分的nft列表
	r.GET(proxy+"/splittedInscriptions/:ticker", s.handle.getSplittedInscriptionList)
	r.GET(proxy+"/mint/details/:inscriptionid", s.handle.getMintDetailInfo)
	r.GET(proxy+"/mint/permission/:ticker/:address", s.handle.getMintPermission)
	r.GET(proxy+"/fee/discount/:address", s.handle.getFeeInfo)

	// 名字服务
	r.GET(proxy+"/ns/status", s.handle.getNSStatus)
	r.GET(proxy+"/ns/name/:name", s.handle.getNameInfo)
	r.GET(proxy+"/ns/values/:name/:prefix", s.handle.getNameValues)
	r.GET(proxy+"/ns/routing/:name", s.handle.getNameRouting)
	r.GET(proxy+"/ns/address/:address", s.handle.getNamesWithAddress)
	r.GET(proxy+"/ns/address/:address/:sub", s.handle.getNamesWithAddress)
	r.GET(proxy+"/ns/address/:address/:sub/:filters", s.handle.getNamesWithFilters)
	r.GET(proxy+"/ns/sat/:sat", s.handle.getNamesWithSat)
	r.GET(proxy+"/ns/inscription/:id", s.handle.getNameWithInscriptionId)
	r.POST(proxy+"/ns/check", s.handle.checkNames)

	// nft
	r.GET(proxy+"/nft/status", s.handle.getNftStatus)
	r.GET(proxy+"/nft/nftid/:id", s.handle.getNftInfo)
	r.GET(proxy+"/nft/address/:address", s.handle.getNftsWithAddress)
	r.GET(proxy+"/nft/sat/:sat", s.handle.getNftsWithSat)
	r.GET(proxy+"/nft/inscription/:id", s.handle.getNftWithInscriptionId)

	// // 下面的接口全部删除，跟前端同步，使用上面的接口，保持数据结构一致
	// // 获取铭文列表
	// r.GET(proxy+"/inscription/list", s.handle.getInscriptionList)
	// // 获取地址拥有铭文列表
	// r.GET(proxy+"/inscription/address/:address", s.handle.getAddrInscriptionList)
	// // 获取创世地址拥有铭文列表
	// r.GET(proxy+"/inscription/genesesaddress/:address", s.handle.getGenesesAddrInscriptionList) //
	// // 获取铭文通过id （保留）
	// r.GET(proxy+"/inscription/id/:id", s.handle.getInscriptionWithId)
	// // r.GET(proxy+"/inscription/number/:number", s.handle.getInscriptionWithNumber)
	// // 获取铭文通过sat
	// r.GET(proxy+"/inscription/sat/:sat", s.handle.getInscriptionListWithSat)

	// v1.0 正式版本的接口
	// 对于数据量大的接口，提供翻页拉取数据功能
	//proxy += "/v2"

	// address
	// 获取地址的所有utxo
	// 获取地址的所有资产（返回：资产类别和对应的数量的列表。资产种类：ft，nft，ns，稀有聪...）
	// 获取地址的某类资产（返回：ticker和对应的数量的列表）
	// 获取地址的某个ticker资产的详细信息（返回：utxo对应的资产数量的列表）
	// 获取地址的铸造历史（可以区分资产种类）

	// utxo
	// 获取utxo的range列表
	// 获取utxo的资产列表（返回列表：资产类型-ticker名称-资产数量-Ranges等）
	// 根据utxo生成一个seed

	// sat
	// 获取聪的基础信息，包括绑定的资产种类（nft，ft，ns，exotic）
	// 聪<->InscriptionId<->名字  这三者的相互转换，其中InscriptionId和Ordinals协议对应
	// 获取聪绑定的数据（需要绑定名字，才能获取各种跟名字绑定的数据）

	// ticker
	// 获取支持的资产种类：nft，ft，ns，exotic...
	// 根据资产种类，获取所有ticker的基础信息
	// 获取ticker的实时持有列表，按地址集合
	// 获取ticker的铸造历史，按inscriptionId的时间顺序排序

	//

}
