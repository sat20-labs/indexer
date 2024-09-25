package extension

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/sat20-labs/ordx/common"
	baseDefine "github.com/sat20-labs/ordx/server/define"
	indexer "github.com/sat20-labs/ordx/share/base_indexer"
)

// func (s *Service) token_list(c *gin.Context) {
// 	resp := &TokenListResp{
// 		BaseResp: baseDefine.BaseResp{
// 			Code: 0,
// 			Msg:  "ok",
// 		},
// 		Data: &TokenListData{
// 			ListResp: baseDefine.ListResp{
// 				Total: 0,
// 				Start: 0,
// 			},
// 			List: make([]*UtxoTokenAsset, 0),
// 		},
// 	}

// 	req := AddressTickerRangeReq{
// 		AddressTickerReq: baseDefine.AddressTickerReq{},
// 		RangeReq:         RangeReq{Cursor: 0, Size: 100},
// 	}
// 	if err := c.ShouldBindQuery(&req); err != nil {
// 		resp.Code = -1
// 		resp.Msg = err.Error()
// 		c.JSON(http.StatusOK, resp)
// 		return
// 	}

// 	curTickerName := common.TickerName{TypeName: common.ASSET_TYPE_FT, Name: req.Ticker}
// 	ftUtxoAssetMap, err := indexer.ShareBaseIndexer.GetAssetUTXOsInAddressWithTick(req.Address, &curTickerName)
// 	if err != nil {
// 		resp.Code = -1
// 		resp.Msg = err.Error()
// 		c.JSON(http.StatusOK, resp)
// 		return
// 	}

// 	ftUtxoAssetSummaryArray := make([]*UtxoAssetSummary, 0)
// 	for utxoId, amount := range ftUtxoAssetMap {
// 		utxo := indexer.ShareBaseIndexer.GetUtxoById(utxoId)
// 		if baseDefine.IsExistUtxoInMemPool(utxo) {
// 			continue
// 		}
// 		ftUtxoAssetSummaryArray = append(ftUtxoAssetSummaryArray, &UtxoAssetSummary{utxoId, amount})
// 	}
// 	sort.Slice(ftUtxoAssetSummaryArray, func(i, j int) bool {
// 		if ftUtxoAssetSummaryArray[i].Amount == ftUtxoAssetSummaryArray[j].Amount {
// 			return ftUtxoAssetSummaryArray[i].UtxoId < ftUtxoAssetSummaryArray[j].UtxoId
// 		} else {
// 			return ftUtxoAssetSummaryArray[i].Amount > ftUtxoAssetSummaryArray[j].Amount
// 		}
// 	})

// 	total := len(ftUtxoAssetSummaryArray)
// 	if total < req.Cursor {
// 		resp.Code = -1
// 		resp.Msg = "start out of range"
// 		c.JSON(http.StatusOK, resp)
// 		return
// 	}
// 	if total < req.Cursor+req.Size {
// 		req.Size = total - req.Cursor
// 	}
// 	end := req.Cursor + req.Size
// 	limitFtUtxoAssetSummaryArray := ftUtxoAssetSummaryArray[req.Cursor:end]

// 	for _, ftUtxoAssetSummary := range limitFtUtxoAssetSummaryArray {
// 		_, rangeList, err := indexer.ShareBaseIndexer.GetOrdinalsWithUtxoId(ftUtxoAssetSummary.UtxoId)
// 		if err != nil {
// 			resp.Code = -1
// 			resp.Msg = err.Error()
// 			c.JSON(http.StatusOK, resp)
// 			return
// 		}

// 		utxoAssetMap := indexer.ShareBaseIndexer.GetAssetsWithUtxo(ftUtxoAssetSummary.UtxoId)
// 		for tickerName, mintInfo := range utxoAssetMap {
// 			if tickerName.TypeName != curTickerName.TypeName || (curTickerName.Name != "" && tickerName.Name != curTickerName.Name) {
// 				continue
// 			}
// 			utxoTokenAsset := &UtxoTokenAsset{
// 				Ticker: curTickerName.Name,
// 				Utxo:   indexer.ShareBaseIndexer.GetUtxoById(ftUtxoAssetSummary.UtxoId),
// 				Amount: common.GetOrdinalsSize(rangeList),
// 			}
// 			for inscriptionId, ranges := range mintInfo {
// 				tokenAsset := TokenAsset{
// 					InscriptionID:  inscriptionId,
// 					InscriptionNum: uint64(common.INVALID_INSCRIPTION_NUM),
// 					AssetAmount:    common.GetOrdinalsSize(ranges),
// 					Ranges:         ranges,
// 				}
// 				utxoTokenAsset.AssetList = append(utxoTokenAsset.AssetList, &tokenAsset)
// 				utxoTokenAsset.AssetAmount += tokenAsset.AssetAmount
// 			}
// 			sort.Slice(utxoTokenAsset.AssetList, func(i, j int) bool {
// 				return utxoTokenAsset.AssetList[i].InscriptionID < utxoTokenAsset.AssetList[j].InscriptionID
// 			})
// 			resp.Data.List = append(resp.Data.List, utxoTokenAsset)
// 		}
// 	}

// 	resp.Data.ListResp.Total = uint64(total)
// 	resp.Data.ListResp.Start = int64(req.Cursor)
// 	c.JSON(http.StatusOK, resp)
// }

func (s *Service) token_list(c *gin.Context) {
	resp := &TokenListResp{
		BaseResp: baseDefine.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: &TokenListData{
			ListResp: baseDefine.ListResp{
				Total: 0,
				Start: 0,
			},
			List: make([]*UtxoTokenAsset, 0),
		},
	}

	req := AddressTickerRangeReq{
		AddressTickerReq: baseDefine.AddressTickerReq{},
		RangeReq:         RangeReq{Cursor: 0, Size: 100},
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	curTickerName := common.TickerName{TypeName: common.ASSET_TYPE_FT, Name: req.Ticker}
	ftUtxoAssetMap, err := indexer.ShareBaseIndexer.GetAssetUTXOsInAddressWithTick(req.Address, &curTickerName)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	ftUtxoAssetSummaryArray := make([]*UtxoAssetSummary, 0)
	for utxoId, amount := range ftUtxoAssetMap {
		utxo := indexer.ShareBaseIndexer.GetUtxoById(utxoId)
		if baseDefine.IsExistUtxoInMemPool(utxo) {
			continue
		}
		ftUtxoAssetSummaryArray = append(ftUtxoAssetSummaryArray, &UtxoAssetSummary{utxoId, amount})
	}
	sort.Slice(ftUtxoAssetSummaryArray, func(i, j int) bool {
		if ftUtxoAssetSummaryArray[i].Amount == ftUtxoAssetSummaryArray[j].Amount {
			return ftUtxoAssetSummaryArray[i].UtxoId < ftUtxoAssetSummaryArray[j].UtxoId
		} else {
			return ftUtxoAssetSummaryArray[i].Amount > ftUtxoAssetSummaryArray[j].Amount
		}
	})

	for _, ftUtxoAssetSummary := range ftUtxoAssetSummaryArray {
		_, rangeList, err := indexer.ShareBaseIndexer.GetOrdinalsWithUtxoId(ftUtxoAssetSummary.UtxoId)
		if err != nil {
			resp.Code = -1
			resp.Msg = err.Error()
			c.JSON(http.StatusOK, resp)
			return
		}

		utxoAssetMap := indexer.ShareBaseIndexer.GetAssetsWithUtxo(ftUtxoAssetSummary.UtxoId)
		for tickerName, mintInfo := range utxoAssetMap {
			if tickerName.TypeName != curTickerName.TypeName || (curTickerName.Name != "" && tickerName.Name != curTickerName.Name) {
				continue
			}
			tickerName := curTickerName.Name
			utxo := indexer.ShareBaseIndexer.GetUtxoById(ftUtxoAssetSummary.UtxoId)
			amount := common.GetOrdinalsSize(rangeList)
			for inscriptionId, ranges := range mintInfo {
				utxoTokenAsset := &UtxoTokenAsset{
					Ticker:         tickerName,
					Utxo:           utxo,
					Amount:         amount,
					InscriptionID:  inscriptionId,
					InscriptionNum: uint64(common.INVALID_INSCRIPTION_NUM),
					AssetAmount:    common.GetOrdinalsSize(ranges),
					Ranges:         ranges,
				}
				resp.Data.List = append(resp.Data.List, utxoTokenAsset)
			}
		}
	}

	total := len(resp.Data.List)
	if total < req.Cursor {
		resp.Code = -1
		resp.Msg = "start out of range"
		c.JSON(http.StatusOK, resp)
		return
	}
	if total < req.Cursor+req.Size {
		req.Size = total - req.Cursor
	}
	end := req.Cursor + req.Size
	resp.Data.List = resp.Data.List[req.Cursor:end]

	resp.Data.ListResp.Total = uint64(total)
	resp.Data.ListResp.Start = int64(req.Cursor)
	sort.Slice(resp.Data.List, func(i, j int) bool {
		return resp.Data.List[i].InscriptionID < resp.Data.List[j].InscriptionID
	})
	for _, utxoTokenAsset := range resp.Data.List {
		nft := indexer.ShareBaseIndexer.GetNftInfoWithInscriptionId(utxoTokenAsset.InscriptionID)
		inscription := newInscription(nft)
		if inscription == nil {
			continue
		}

		utxoTokenAsset.InscriptionNum = uint64(nft.Base.Id)
		utxoTokenAsset.Address = inscription.Address
		utxoTokenAsset.OutputValue = inscription.OutputValue
		utxoTokenAsset.Preview = inscription.Preview
		utxoTokenAsset.Content = inscription.Content
		utxoTokenAsset.ContentType = inscription.ContentType
		utxoTokenAsset.ContentLength = inscription.ContentLength
		utxoTokenAsset.Timestamp = inscription.Timestamp
		utxoTokenAsset.GenesisTransaction = inscription.GenesisTransaction
		utxoTokenAsset.Location = inscription.Location
		utxoTokenAsset.Output = inscription.Output
		utxoTokenAsset.Offset = inscription.Offset
		utxoTokenAsset.ContentBody = inscription.ContentBody
		utxoTokenAsset.Height = inscription.Height
		utxoTokenAsset.Confirmation = inscription.Confirmation
	}
	c.JSON(http.StatusOK, resp)
}
