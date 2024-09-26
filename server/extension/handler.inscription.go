package extension

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sat20-labs/indexer/common"
	serverCommon "github.com/sat20-labs/indexer/server/define"
	"github.com/sat20-labs/indexer/share/base_indexer"
)

func (s *Service) inscription_utxo(c *gin.Context) {
	resp := &InscriptionUtxoResp{
		BaseResp: serverCommon.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	req := InscriptionIdReq{}
	if err := c.ShouldBindQuery(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	nft := base_indexer.ShareBaseIndexer.GetNftInfoWithInscriptionId(req.InscriptionId)
	if nft == nil {
		resp.Code = -1
		resp.Msg = "can't find inscription"
		c.JSON(http.StatusOK, resp)
		return
	}
	address := base_indexer.ShareBaseIndexer.GetAddressById(nft.OwnerAddressId)
	utxo, rngs, err := base_indexer.ShareBaseIndexer.GetOrdinalsWithUtxoId(nft.UtxoId)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	txid, voutIndex, err := common.ParseUtxo(utxo)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	inscriptionList, err := getInsctiptionList(utxo)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = &Utxo{
		Txid:        txid,
		Vout:        voutIndex,
		Satoshis:    uint64(common.GetOrdinalsSize(rngs)),
		ScriptPk:    GetScriptPK(address),
		AddressType: P2TR,
		// Inscriptions: []*Inscription{newAbbrInscription(nft)},
		Inscriptions: inscriptionList,
		Runes:        make([]*Rune, 0),
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Service) inscription_utxoDetail(c *gin.Context) {
	resp := &InscriptionUtxoDetailResp{
		BaseResp: serverCommon.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	req := InscriptionIdReq{}
	if err := c.ShouldBindQuery(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	nft := base_indexer.ShareBaseIndexer.GetNftInfoWithInscriptionId(req.InscriptionId)
	if nft == nil {
		resp.Code = -1
		resp.Msg = "can't find inscription"
		c.JSON(http.StatusOK, resp)
		return
	}
	address := base_indexer.ShareBaseIndexer.GetAddressById(nft.OwnerAddressId)
	utxo, rngs, err := base_indexer.ShareBaseIndexer.GetOrdinalsWithUtxoId(nft.UtxoId)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	txid, voutIndex, err := common.ParseUtxo(utxo)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	inscriptionList, err := getInsctiptionList(utxo)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = &UtxoDetail{
		Txid:        txid,
		Vout:        voutIndex,
		Satoshis:    uint64(common.GetOrdinalsSize(rngs)),
		ScriptPk:    (GetScriptPK(address)),
		AddressType: P2TR,
		// Inscriptions: []*Inscription{newInscription(nft)},
		Inscriptions: inscriptionList,
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Service) inscription_utxoList(c *gin.Context) {
	resp := &InscriptionUtxoListResp{
		BaseResp: serverCommon.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: make([]*Utxo, 0),
	}
	var req InscriptionIdListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	for _, inscriptionId := range req.InscriptionIdList {
		resp.Data = append(resp.Data, newUtxoDataWithInscription(inscriptionId))
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Service) inscription_info(c *gin.Context) {
	resp := &InscriptionInfoResp{
		BaseResp: serverCommon.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	req := InscriptionIdReq{}
	if err := c.ShouldBindQuery(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	nft := base_indexer.ShareBaseIndexer.GetNftInfoWithInscriptionId(req.InscriptionId)
	if nft == nil {
		resp.Code = -1
		resp.Msg = "can't find inscription"
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = newInscription(nft)
	c.JSON(http.StatusOK, resp)
}
