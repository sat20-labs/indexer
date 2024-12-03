package ordx

import (
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
	ordx "github.com/sat20-labs/indexer/common"
	serverOrdx "github.com/sat20-labs/indexer/server/define"
	swire "github.com/sat20-labs/satsnet_btcd/wire"
)

type BestHeightResp struct {
	serverOrdx.BaseResp
	Data map[string]int `json:"data" example:"height:100"`
}

type BlockInfoData struct {
	serverOrdx.BaseResp
	Data *ordx.BlockInfo `json:"info"`
}

type StatusListData struct {
	serverOrdx.ListResp
	Height uint64                     `json:"height"`
	Detail []*serverOrdx.TickerStatus `json:"detail"`
}

type StatusListResp struct {
	serverOrdx.BaseResp
	Data *StatusListData `json:"data"`
}

type StatusResp struct {
	serverOrdx.BaseResp
	Data *serverOrdx.TickerStatus `json:"data"`
}

type HolderListData struct {
	serverOrdx.ListResp
	Detail []*serverOrdx.Holder `json:"detail"`
}
type HolderListResp struct {
	serverOrdx.BaseResp
	Data *HolderListData `json:"data"`
}

type MintHistoryData struct {
	serverOrdx.ListResp
	Detail *serverOrdx.MintHistory `json:"detail"`
}
type MintHistoryResp struct {
	serverOrdx.BaseResp
	Data *MintHistoryData `json:"data"`
}

type InscriptionIdListResp struct {
	serverOrdx.BaseResp
	Data []string `json:"data"`
}

type MintDetailInfoResp struct {
	serverOrdx.BaseResp
	Data *serverOrdx.MintDetailInfo `json:"data"`
}

type MintPermissionResp struct {
	serverOrdx.BaseResp
	Data *serverOrdx.MintPermissionInfo `json:"data"`
}

type FeeResp struct {
	serverOrdx.BaseResp
	Data *serverOrdx.FeeInfo `json:"data"`
}

type BalanceSummaryListData struct {
	serverOrdx.ListResp
	Detail []*serverOrdx.BalanceSummary `json:"detail"`
}

type BalanceSummaryListResp struct {
	serverOrdx.BaseResp
	Data *BalanceSummaryListData `json:"data"`
}

type AbbrAssetsWithUtxosResp struct {
	serverOrdx.BaseResp
	Data []*serverOrdx.UtxoAbbrAssets `json:"data"`
}

type UtxoListData struct {
	serverOrdx.ListResp
	Detail []*serverOrdx.TickerAsset `json:"detail"`
}

type UtxoListResp struct {
	serverOrdx.BaseResp
	Data *UtxoListData `json:"data"`
}

type ExistingUtxoResp struct {
	serverOrdx.BaseResp
	ExistingUtxos []string `json:"data"`
}

type OrdInscriptionListData struct {
	serverOrdx.ListResp
	Detail any `json:"detail"`
	// Detail []*OrdInscriptionAsset `json:"detail"`
}

type OrdInscriptionListResp struct {
	serverOrdx.BaseResp
	// Data *OrdInscriptionListData `json:"data"`
	Data any `json:"data"`
}

type OrdInscriptionResp struct {
	serverOrdx.BaseResp
	// Data *OrdInscriptionAsset `json:"data"`
	Data any `json:"data"`
}

type AssetsData struct {
	serverOrdx.ListResp
	Detail *serverOrdx.AssetDetailInfo `json:"detail"`
}

type RangesReq struct {
	Ranges []*ordx.Range `json:"ranges"`
}

type AssetsResp_deprecated struct {
	serverOrdx.BaseResp
	Data *AssetsData `json:"data"`
}

type AssetsResp struct {
	serverOrdx.BaseResp
	Data *TxOutput `json:"data"`
}

type AssetListResp struct {
	serverOrdx.BaseResp
	Data []*serverOrdx.AssetAbbrInfo `json:"data"`
}

type AssetOffsetData struct {
	serverOrdx.ListResp
	AssetOffset []*common.AssetOffsetRange `json:"detail"`
}

type AssetOffsetResp struct {
	serverOrdx.BaseResp
	Data *AssetOffsetData `json:"data"`
}

type SeedsResp struct {
	serverOrdx.BaseResp
	Data []*serverOrdx.Seed `json:"data"`
}

type UtxoInfoResp struct {
	serverOrdx.BaseResp
	Data *serverOrdx.UtxoInfo `json:"data"`
}

type NSStatusData struct {
	Version string                `json:"version"`
	Total   uint64                `json:"total"`
	Start   uint64                `json:"start"`
	Names   []*serverOrdx.NftItem `json:"names"`
}

type NSStatusResp struct {
	serverOrdx.BaseResp
	Data *NSStatusData `json:"data"`
}

type NamePropertiesResp struct {
	serverOrdx.BaseResp
	Data *serverOrdx.OrdinalsName `json:"data"`
}

type NameRoutingResp struct {
	serverOrdx.BaseResp
	Data *serverOrdx.NameRouting `json:"data"`
}

type NameCheckReq struct {
	Names []string `json:"names"`
}

type NameCheckResp struct {
	serverOrdx.BaseResp
	Data []*serverOrdx.NameCheckResult `json:"data"`
}

type AddCollectionReq struct {
	Type   string                      `json:"type"`
	Ticker string                      `json:"ticker"`
	Data   []*serverOrdx.InscriptionId `json:"data"`
}

type AddCollectionResp struct {
	serverOrdx.BaseResp
}

type UtxosReq struct {
	Utxos []string `json:"utxos"`
}

type NftStatusData struct {
	Version string                `json:"version"`
	Total   uint64                `json:"total"`
	Start   uint64                `json:"start"`
	Nfts    []*serverOrdx.NftItem `json:"nfts"`
}

type NftStatusResp struct {
	serverOrdx.BaseResp
	Data *NftStatusData `json:"data"`
}

type NftInfoResp struct {
	serverOrdx.BaseResp
	Data *serverOrdx.NftInfo `json:"data"`
}

type NftsWithAddressData struct {
	serverOrdx.ListResp
	Address string                `json:"address"`
	Amount  int                   `json:"amount"`
	Nfts    []*serverOrdx.NftItem `json:"nfts"`
}

type NftsWithAddressResp struct {
	serverOrdx.BaseResp
	Data *NftsWithAddressData `json:"data"`
}

type NamesWithAddressData struct {
	Address string                     `json:"address"`
	Total   int                        `json:"total"`
	Names   []*serverOrdx.OrdinalsName `json:"names"`
}

type NamesWithAddressResp struct {
	serverOrdx.BaseResp
	Data *NamesWithAddressData `json:"data"`
}

type TxOutput struct {
	OutPoint string      `json:"outpoint"`
	OutValue wire.TxOut  `json:"outvalue"`
	Sats     []*common.Range `json:"rangs"`
	Assets   []*common.AssetInfo_MainNet  `json:"assets"`
}

type AssetSummary struct {
	serverOrdx.ListResp
	Data []*swire.AssetInfo `json:"data"`
}

type AssetSummaryResp struct {
	serverOrdx.BaseResp
	Data *AssetSummary `json:"data"`
}

type UtxosWithAssetResp struct {
	serverOrdx.BaseResp
	Name swire.AssetName
	serverOrdx.ListResp
	Data []*TxOutput `json:"data"`
}

type AssetsWithUtxosResp struct {
	serverOrdx.BaseResp
	Data []*TxOutput `json:"data"`
}
