package extension

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sat20-labs/indexer/common"
	serverCommon "github.com/sat20-labs/indexer/server/define"
	"github.com/sat20-labs/indexer/share/base_indexer"
)

func getAssetSummary(address string) (*AssetSummary, error) {
	ret := &AssetSummary{
		TotalSatoshis:     0,
		BtcSatoshis:       0,
		AssetSatoshis:     0,
		InscriptionCount:  0,
		RunesCount:        0,
		TokenSummaryList:  []TokenSummary{},
		OrdinalsSummary:   OrdinalsSummary{},
		NameSummaryList:   []NameSummary{},
		ExoticSummaryList: []ExoticSummary{},
	}
	utxoList, err := base_indexer.ShareBaseIndexer.GetUTXOsWithAddress(address)
	if err != nil {
		return nil, err
	}

	for utxoId, v := range utxoList {
		ret.TotalSatoshis += uint64(v)
		utxo := base_indexer.ShareBaseIndexer.GetUtxoById(utxoId)
		if serverCommon.IsExistUtxoInMemPool(utxo) {
			continue
		}
		//Find common utxo (that is, utxo with non-ordinal attributes)
		if base_indexer.ShareBaseIndexer.HasAssetInUtxo(utxo, false) {
			ret.AssetSatoshis += uint64(v)
		}
	}
	ret.BtcSatoshis = ret.TotalSatoshis - ret.AssetSatoshis

	_, inscriptionCount := base_indexer.ShareBaseIndexer.GetNftsWithAddress(address, 0, -1)
	ret.InscriptionCount = uint64(inscriptionCount)
	ret.OrdinalsSummary.Count = uint64(inscriptionCount)

	assetsList := base_indexer.ShareBaseIndexer.GetAssetSummaryInAddress(address)
	for ticker, balance := range assetsList {
		switch ticker.TypeName {
		case common.ASSET_TYPE_NS:
			ret.NameSummaryList = append(ret.NameSummaryList, NameSummary{
				Balance: uint64(balance),
				Count:   uint64(balance),
				Name:    ticker.Name,
			})
			continue
		case common.ASSET_TYPE_FT:
			ret.TokenSummaryList = append(ret.TokenSummaryList, TokenSummary{
				Name:    ticker.Name,
				Balance: uint64(balance),
			})
			continue
		case common.ASSET_TYPE_NFT:
			continue
		case common.ASSET_TYPE_EXOTIC:
			ret.ExoticSummaryList = append(ret.ExoticSummaryList, ExoticSummary{
				Name:    ticker.Name,
				Balance: uint64(balance),
			})
			continue
		}
	}
	return ret, nil
}

func (s *Service) address_assetsSummary(c *gin.Context) {
	resp := &AssetsSummaryResp{
		BaseResp: serverCommon.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: &AssetSummary{
			TotalSatoshis:     0,
			BtcSatoshis:       0,
			AssetSatoshis:     0,
			InscriptionCount:  0,
			RunesCount:        0,
			TokenSummaryList:  []TokenSummary{},
			OrdinalsSummary:   OrdinalsSummary{},
			NameSummaryList:   []NameSummary{},
			ExoticSummaryList: []ExoticSummary{},
		},
	}

	req := serverCommon.AddressReq{}
	if err := c.ShouldBindQuery(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	data, err := getAssetSummary(req.Address)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data = data
	c.JSON(http.StatusOK, resp)
}

func (s *Service) address_AssetSummaryList(c *gin.Context) {
	resp := &MultiAddressAssetsResp{
		BaseResp: serverCommon.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: []*AssetSummary{},
	}

	req := AddressListReq{}
	if err := c.ShouldBindQuery(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	addressList := strings.Split(req.AddressList, ",")
	for _, address := range addressList {
		assetSummary, err := getAssetSummary(address)
		if err != nil {
			resp.Code = -1
			resp.Msg = err.Error()
			c.JSON(http.StatusOK, resp)
			return
		}
		resp.Data = append(resp.Data, assetSummary)
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Service) address_balance(c *gin.Context) {
	resp := &BalanceResp{
		BaseResp: serverCommon.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	req := serverCommon.AddressReq{}
	if err := c.ShouldBindQuery(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	confirmAmount := uint64(0)
	pendingAmount := uint64(0)
	confirmInscriptionAmount := uint64(0)
	pendingInscriptionAmount := uint64(0)

	utxoList, err := base_indexer.ShareBaseIndexer.GetUTXOsWithAddress(req.Address)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	for utxoId, v := range utxoList {
		utxo := base_indexer.ShareBaseIndexer.GetUtxoById(utxoId)
		if serverCommon.IsExistUtxoInMemPool(utxo) {
			pendingAmount += uint64(v)
			if base_indexer.ShareBaseIndexer.HasAssetInUtxo(utxo, false) {
				pendingInscriptionAmount += uint64(v)
			}
		} else {
			confirmAmount += uint64(v)
			if base_indexer.ShareBaseIndexer.HasAssetInUtxo(utxo, false) {
				confirmInscriptionAmount += uint64(v)
			}
		}
	}

	amount := uint64(0)
	availableBtcAmount := uint64(0)
	pendingBtcAmount := pendingAmount - pendingInscriptionAmount
	btcAmount := uint64(0)
	inscriptionAmount := uint64(0)

	amount = confirmAmount + pendingAmount
	inscriptionAmount = confirmInscriptionAmount + pendingInscriptionAmount
	availableBtcAmount = confirmAmount - confirmInscriptionAmount
	btcAmount = amount

	const prec = 8
	resp.Data = &Balance{
		ConfirmAmount:            strconv.FormatFloat(float64(confirmAmount)/BtcBitLen, 'f', prec, 64),
		PendingAmount:            strconv.FormatFloat(float64(pendingAmount)/BtcBitLen, 'f', prec, 64),
		Amount:                   strconv.FormatFloat(float64(amount)/BtcBitLen, 'f', prec, 64),
		ConfirmBtcAmount:         strconv.FormatFloat(float64(availableBtcAmount)/BtcBitLen, 'f', prec, 64),
		PendingBtcAmount:         strconv.FormatFloat(float64(pendingBtcAmount)/BtcBitLen, 'f', prec, 64),
		BtcAmount:                strconv.FormatFloat(float64(btcAmount)/BtcBitLen, 'f', prec, 64),
		ConfirmInscriptionAmount: strconv.FormatFloat(float64(confirmInscriptionAmount)/BtcBitLen, 'f', prec, 64),
		PendingInscriptionAmount: strconv.FormatFloat(float64(pendingInscriptionAmount)/BtcBitLen, 'f', prec, 64),
		InscriptionAmount:        strconv.FormatFloat(float64(inscriptionAmount)/BtcBitLen, 'f', prec, 64),
		UsdValue:                 "0",
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Service) address_findGroupAssetList(c *gin.Context) {
	resp := &AddressFindGroupAssetsResp{
		BaseResp: serverCommon.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}

	var req AddressFindGroupAssetsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Service) address_UnavailableUtxoList(c *gin.Context) {
	resp := &AddressUtxoResp{
		BaseResp: serverCommon.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: make([]*Utxo, 0),
	}
	req := serverCommon.AddressReq{}
	if err := c.ShouldBindQuery(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	utxomap := base_indexer.ShareBaseIndexer.GetAssetUTXOsInAddress(req.Address)
	for _, utxos := range utxomap {
		for _, utxoId := range utxos {
			utxo := newUtxoDataWithId(utxoId, req.Address, false)
			if utxo != nil {
				resp.Data = append(resp.Data, utxo)
			}
		}
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Service) address_BTCUtxoList(c *gin.Context) {
	resp := &AddressUtxoResp{
		BaseResp: serverCommon.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: make([]*Utxo, 0),
	}

	req := serverCommon.AddressReq{}
	if err := c.ShouldBindQuery(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	utxoList, err := base_indexer.ShareBaseIndexer.GetUTXOsWithAddress(req.Address)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	for utxoId := range utxoList {
		if serverCommon.IsAvailableUtxoId(utxoId) {
			resp.Data = append(resp.Data, newUtxoDataWithId(utxoId, req.Address, true))
		}
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Service) address_inscriptionList(c *gin.Context) {
	resp := &AddressInscriptionResp{
		BaseResp: serverCommon.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: &AddressInscriptionData{
			ListResp: serverCommon.ListResp{
				Total: 0,
				Start: 0,
			},
			InscriptionList: make([]*Inscription, 0),
		},
	}

	req := AddressRangeReq{
		AddressReq: serverCommon.AddressReq{},
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
			resp.Data.InscriptionList = append(resp.Data.InscriptionList, inscription)
		}
	}

	resp.Data.Total = uint64(total)
	resp.Data.Start = int64(req.RangeReq.Cursor)

	c.JSON(http.StatusOK, resp)
}

func (s *Service) address_domainInfo(c *gin.Context) {
	resp := &AddressSearchResp{
		BaseResp: serverCommon.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: make([]*Inscription, 0),
	}

	domain := c.Query("domain")
	common.Log.Debugf("domain: %v", domain)
	c.JSON(http.StatusOK, resp)
}
