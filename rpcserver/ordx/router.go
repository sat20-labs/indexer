package ordx

import (
	"github.com/gin-gonic/gin"
	"github.com/sat20-labs/indexer/share/base_indexer"
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
	//查询utxo上的资产和数量
	r.GET(proxy+"/utxo/abbrassets/:utxo", s.handle.getAbbrAssetsWithUtxo)
	//获取utxo上的资产类型和对应的seed，seed由聪的属性（资产类型，数量，序号）决定
	r.GET(proxy+"/utxo/seed/:utxo", s.handle.getSeedWithUtxo)
	// for test
	r.POST(proxy+"/utxos/exist", s.handle.getExistingUtxos)


	// inscribe
	// 检查某个ticker是否可以deploy
	r.GET(proxy+"/deploy/:ticker", s.handle.isDeployAllowed)
	r.GET(proxy+"/deploy/mintable/:protocol", s.handle.getMintableTickers)
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

	/////////////////////////////////////////
	// version 2.0 interface for STP

	r.POST(proxy+"/v3/utxos/existing", s.handle.getExistingUtxos)

	// 提供精确资产的数据接口，资产用string类型表示(主网索引器的接口)
	// 获取某个地址上所有资产和数量的列表
	r.GET(proxy+"/v3/address/summary/:address", s.handle.getAssetSummaryV3)
	// 获取某个地址上某个资产的utxo数据列表(utxo包含其他资产), ticker格式：wire.AssetName.String()
	r.GET(proxy+"/v3/address/asset/:address/:ticker", s.handle.getUtxosWithTickerV3)
	// 获取utxo的资产信息
	r.GET(proxy+"/v3/utxo/info/:utxo", s.handle.getUtxoInfoV3)
	r.POST(proxy+"/v3/utxos/info", s.handle.getUtxoInfoListV3)
	r.POST(proxy+"/v3/utxo/unlock", s.handle.unlockOrdinals)
	r.GET(proxy+"/v3/utxos/locked/:address", s.handle.getLockedUtxos)
	// protocol: ordx/runes/brc20
	r.GET(proxy+"/v3/tick/all/:protocol", s.handle.getTickerList)
	r.GET(proxy+"/v3/tick/info/:ticker", s.handle.getTickerInfo)

	// ticker格式：wire.AssetName.String() protocol:f:name
	// 持有者列表
	r.GET(proxy+"/v3/tick/holders/:ticker", s.handle.getHolderListV3)
	// // 铸造历史
	r.GET(proxy+"/v3/tick/history/:ticker", s.handle.getMintHistoryV3)
	// // 某条铸造记录
	// r.GET(proxy+"/v3/mint/details/:ticker/:id", s.handle.getMintDetailInfo)

	// kv记录
	r.POST(proxy+"/kv/nonce", s.handle.getNonce)
	r.GET(proxy+"/kv/get/:pubkey/:key", s.handle.getkv)
	r.POST(proxy+"/kv/put", s.handle.putKVs)
	r.POST(proxy+"/kv/del", s.handle.delKVs)
	// 注册公钥，并返回索引器公钥
	r.POST(proxy+"/kv/register", s.handle.registerPubKey)
}
