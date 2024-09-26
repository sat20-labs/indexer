package extension

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/sat20-labs/indexer/indexer/exotic"
	baseDefine "github.com/sat20-labs/indexer/server/define"
	indexer "github.com/sat20-labs/indexer/share/base_indexer"
)

func (s *Service) raresat_list(c *gin.Context) {
	resp := &RareSatListResp{
		BaseResp: baseDefine.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: &RareSatListData{
			ListResp: baseDefine.ListResp{
				Total: 0,
				Start: 0,
			},
			List: make([]*baseDefine.ExoticSatRangeUtxo, 0),
		},
	}

	req := AddressRangeReq{
		AddressReq: baseDefine.AddressReq{},
		RangeReq:   RangeReq{Cursor: 0, Size: 100},
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	utxoList, err := indexer.ShareBaseIndexer.GetUTXOsWithAddress(req.Address)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	satributeSatList := make([]*baseDefine.ExoticSatRangeUtxo, 0)
	for utxoId, value := range utxoList {
		utxo, res, err := indexer.ShareBaseIndexer.GetOrdinalsWithUtxoId(utxoId)
		if err != nil {
			resp.Code = -1
			resp.Msg = err.Error()
			c.JSON(http.StatusOK, resp)
			return
		}

		if indexer.ShareBaseIndexer.HasAssetInUtxo(utxo, true) {
			continue
		}

		// Caluclate the offset for each range
		var satList []baseDefine.SatDetailInfo
		sr := indexer.ShareBaseIndexer.GetExoticsWithRanges(res)
		for _, r := range sr {
			exoticSat := exotic.Sat(r.Range.Start)
			sat := baseDefine.SatDetailInfo{
				SatributeRange: baseDefine.SatributeRange{
					SatRange: baseDefine.SatRange{
						Start:  r.Range.Start,
						Size:   r.Range.Size,
						Offset: r.Offset,
					},
					Satributes: r.Satributes,
				},
				Block: int(exoticSat.Height()),
				// Time:  0, //暂时不显示，需要获取Block的时间。
			}
			satList = append(satList, sat)
		}
		if len(satList) == 0 {
			continue
		}
		satributeSatList = append(satributeSatList, &baseDefine.ExoticSatRangeUtxo{
			Utxo:  utxo,
			Value: value,
			Sats:  satList,
		})
	}
	total := len(satributeSatList)
	if total > 0 {
		if req.Cursor >= total {
			resp.Code = -1
			resp.Msg = "cursor out of range"
			c.JSON(http.StatusOK, resp)
			return
		}
		end := total
		if req.Size > 0 && req.Cursor+req.Size < total {
			end = req.Cursor + req.Size
		}
		satributeSatList = satributeSatList[req.Cursor:end]
	}

	sort.Slice(satributeSatList, func(i, j int) bool {
		return satributeSatList[i].Value > satributeSatList[j].Value
	})

	resp.Data = &RareSatListData{
		ListResp: baseDefine.ListResp{
			Total: uint64(total),
			Start: int64(req.Cursor),
		},
		List: satributeSatList,
	}
	c.JSON(http.StatusOK, resp)
}
