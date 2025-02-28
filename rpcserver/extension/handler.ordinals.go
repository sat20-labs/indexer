package extension

import (
	"net/http"

	"github.com/gin-gonic/gin"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
	"github.com/sat20-labs/indexer/share/base_indexer"
)

func (s *Service) ordinals_inscriptionList(c *gin.Context) {
	resp := &OrdinalsInscriptionListResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: &OrdinalsInscriptionList{
			ListResp: rpcwire.ListResp{
				Total: 0,
				Start: 0,
			},
			List: make([]*Inscription, 0),
		},
	}

	req := AddressRangeReq{
		AddressReq: rpcwire.AddressReq{},
		RangeReq:   RangeReq{Cursor: 0, Size: 100},
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	nftList, total := base_indexer.ShareBaseIndexer.GetNftsWithAddress(req.Address, req.Cursor, req.Size)
	for _, nft := range nftList {
		inscription := newInscription(nft)
		if inscription != nil {
			resp.Data.List = append(resp.Data.List, inscription)
		}
	}

	resp.Data.ListResp.Start = int64(req.Cursor)
	resp.Data.ListResp.Total = uint64(total)
	c.JSON(http.StatusOK, resp)
}
