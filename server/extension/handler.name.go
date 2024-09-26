package extension

import (
	"net/http"
	"sort"

	"github.com/sat20-labs/indexer/common"

	"github.com/gin-gonic/gin"
	serverDefine "github.com/sat20-labs/indexer/server/define"
	indexer "github.com/sat20-labs/indexer/share/base_indexer"
)

func (s *Service) name_list(c *gin.Context) {
	resp := &OrdinalsNameListResp{
		BaseResp: serverDefine.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: &OrdinalsNameListData{
			ListResp: serverDefine.ListResp{
				Total: 0,
				Start: 0,
			},
			List: make([]*OrdinalsName, 0),
		},
	}

	req := AddressRangeReq{
		AddressReq: serverDefine.AddressReq{},
		RangeReq:   RangeReq{Cursor: 0, Size: 100},
	}

	c.ShouldBindQuery(&req)
	// if err := c.ShouldBindQuery(&req); err != nil {
	// resp.Code = -1
	// resp.Msg = err.Error()
	// c.JSON(http.StatusOK, resp)
	// return
	// }

	if req.Address != "" {
		ordinalsNameList := make([]*OrdinalsName, 0)
		nameInfoList, total := indexer.ShareBaseIndexer.GetNamesWithAddress(req.Address, req.Cursor, req.Size)
		for _, nameInfo := range nameInfoList {
			preview, _ := getOrdContentUrl(nameInfo.Base.InscriptionId)
			ordinalsName := OrdinalsName{
				InscriptionNumber:  nameInfo.Id,
				Name:               nameInfo.Name,
				Sat:                nameInfo.Base.Sat,
				Address:            nameInfo.OwnerAddress,
				InscriptionId:      nameInfo.Base.InscriptionId,
				Utxo:               nameInfo.Utxo,
				BlockHeight:        int64(nameInfo.Base.BlockHeight),
				BlockTimestamp:     nameInfo.Base.BlockTime,
				InscriptionAddress: indexer.ShareBaseIndexer.GetAddressById(nameInfo.Base.InscriptionAddress),
				Preview:            preview,
				KVs:                nameInfo.KVs,
			}

			_, rngs, err := indexer.ShareBaseIndexer.GetOrdinalsWithUtxo(nameInfo.Utxo)
			if err == nil {
				ordinalsName.Value = common.GetOrdinalsSize(rngs)
			}
			ordinalsNameList = append(ordinalsNameList, &ordinalsName)
		}
		sort.Slice(ordinalsNameList, func(i, j int) bool {
			return ordinalsNameList[i].Name < ordinalsNameList[j].Name
		})
		resp.Data = &OrdinalsNameListData{
			ListResp: serverDefine.ListResp{
				Total: uint64(total),
				Start: int64(req.Cursor),
			},
			List: ordinalsNameList,
		}
		c.JSON(http.StatusOK, resp)
	} else {
		ordinalsNameList := make([]*OrdinalsName, 0)
		nameList := indexer.ShareBaseIndexer.GetNames(req.Cursor, req.Size)
		for _, name := range nameList {
			nameInfo := indexer.ShareBaseIndexer.GetNameInfo(name)
			preview, _ := getOrdContentUrl(nameInfo.Base.InscriptionId)
			ordinalsName := OrdinalsName{
				InscriptionNumber:  nameInfo.Id,
				Name:               nameInfo.Name,
				Sat:                nameInfo.Base.Sat,
				Address:            nameInfo.OwnerAddress,
				InscriptionId:      nameInfo.Base.InscriptionId,
				Utxo:               nameInfo.Utxo,
				BlockHeight:        int64(nameInfo.Base.BlockHeight),
				BlockTimestamp:     nameInfo.Base.BlockTime,
				InscriptionAddress: indexer.ShareBaseIndexer.GetAddressById(nameInfo.Base.InscriptionAddress),
				Preview:            preview,
				KVs:                nameInfo.KVs,
			}

			_, rngs, err := indexer.ShareBaseIndexer.GetOrdinalsWithUtxo(nameInfo.Utxo)
			if err == nil {
				ordinalsName.Value = common.GetOrdinalsSize(rngs)
			}
			ordinalsNameList = append(ordinalsNameList, &ordinalsName)
		}
		sort.Slice(ordinalsNameList, func(i, j int) bool {
			return ordinalsNameList[i].Name < ordinalsNameList[j].Name
		})

		total := indexer.ShareBaseIndexer.GetNSStatus().NameCount
		resp.Data = &OrdinalsNameListData{
			ListResp: serverDefine.ListResp{
				Total: total,
				Start: int64(req.Cursor),
			},
			List: ordinalsNameList,
		}
		c.JSON(http.StatusOK, resp)
	}
}
