package extension

import (
	"bytes"
	"encoding/hex"
	"math"
	"net/http"
	"strconv"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/mempool"
	"github.com/btcsuite/btcd/txscript"
	"github.com/gin-gonic/gin"
	"github.com/sat20-labs/indexer/common"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
	"github.com/sat20-labs/indexer/share/base_indexer"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

func (s *Service) tx_decodePsbt(c *gin.Context) {
	resp := &TxDecode2Resp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}
	var req TxDecode2Req
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	// handle psbt
	psbtBytes, err := hex.DecodeString(req.PsbtHex)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	pb, err := psbt.NewFromRawBytes(
		bytes.NewReader(psbtBytes), false,
	)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	fee, err := pb.GetTxFee()
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	vsize := int64(0)
	feeRateStr := ""
	isIncompletePSBT := false
	msgTx, err := psbt.Extract(pb)
	if err == nil {
		var buf bytes.Buffer
		err = msgTx.Serialize(&buf)
		if err != nil {
			resp.Code = -1
			resp.Msg = err.Error()
			c.JSON(http.StatusOK, resp)
			return
		}
		tx, err := btcutil.NewTxFromBytes(buf.Bytes())
		if err != nil {
			resp.Code = -1
			resp.Msg = err.Error()
			c.JSON(http.StatusOK, resp)
			return
		}
		vsize = mempool.GetTxVirtualSize(tx)
	} else if err == psbt.ErrIncompletePSBT {
		isIncompletePSBT = true
		vsize = int64(pb.UnsignedTx.SerializeSize())
	} else {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	feeRate := int64(math.Round(float64(fee) / float64(vsize)))
	shouldWarnFeeRate := true
	estimateSmartFeeResult, err := bitcoin_rpc.ShareBitconRpc.EstimateSmartFee(3)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	averageFeeRate := int64(math.Round(estimateSmartFeeResult.FeeRate * 100000))
	if feeRate > averageFeeRate {
		shouldWarnFeeRate = true
	}

	recommendedFeeRate := int64(1)
	switch s.chain {
	case common.ChainTestnet:
		recommendedFeeRate = 1
	case common.ChainTestnet4:
		recommendedFeeRate = 1
	case common.ChainMainnet:
		recommendedFeeRate = feeRate
	}

	feeRateStr = strconv.FormatInt(feeRate, 10) + ".0"
	if isIncompletePSBT {
		feeRateStr = "≈" + feeRateStr
	}

	resp.Data = &TxDecode2Data{
		InputInfos:  make([]InputInfo, 0),
		OutputInfos: make([]OutputInfo, 0),
		Features: &TxDecode2Features{
			Rbf: common.SignalsReplacement(pb.UnsignedTx),
		},
		Inscriptions:       make(map[string]*Inscription),
		FeeRate:            feeRateStr,
		Fee:                int64(fee),
		Risks:              make([]Risk, 0),
		IsScammer:          false,
		RecommendedFeeRate: recommendedFeeRate,
		ShouldWarnFeeRate:  shouldWarnFeeRate,
	}

	chain := base_indexer.ShareBaseIndexer.GetChainParam().Name

	// handle inputs
	inputRanges := make([]*common.Range, 0)
	for index, unsingedTxIn := range pb.UnsignedTx.TxIn {
		if pb.Inputs[index].WitnessUtxo.PkScript[0] == txscript.OP_RETURN {
			continue
		}
		address, err := common.PkScriptToAddr(pb.Inputs[index].WitnessUtxo.PkScript, chain)
		if err != nil {
			resp.Code = -1
			resp.Msg = err.Error()
			c.JSON(http.StatusOK, resp)
			return
		}

		txid := unsingedTxIn.PreviousOutPoint.Hash.String()
		voutIndex := int(unsingedTxIn.PreviousOutPoint.Index)

		inputInfo := InputInfo{
			Txid:         txid,
			Vout:         voutIndex,
			Address:      address,
			Value:        pb.Inputs[index].WitnessUtxo.Value,
			Inscriptions: make([]Inscription, 0),
			// Atomicals:    make([]Atomical, 0),
			Runes: make([]RuneBalance, 0),
		}

		utxo := txid + ":" + strconv.Itoa(voutIndex)
		utxoId, rngs, err := base_indexer.ShareBaseIndexer.GetOrdinalsWithUtxo(utxo)
		if err != nil {
			resp.Code = -1
			resp.Msg = err.Error()
			c.JSON(http.StatusOK, resp)
			return
		}
		inputRanges = append(inputRanges, rngs...)
		assets := base_indexer.ShareBaseIndexer.GetAssetsWithUtxo(utxoId)
		for ticker, mintinfo := range assets {
			if ticker.Type == common.ASSET_TYPE_EXOTIC {
				// TODO 稀有聪需要有所表示出来
				continue
			}
			for id := range mintinfo {
				// TODO ordx资产，同一个inscriptionID，有多个资产
				nft := base_indexer.ShareBaseIndexer.GetNftInfoWithInscriptionId(id)
				inscription := newInscription(nft)
				resp.Data.Inscriptions[inscription.Id] = inscription
				inputInfo.Inscriptions = append(inputInfo.Inscriptions, *inscription)
			}
		}

		resp.Data.InputInfos = append(resp.Data.InputInfos, inputInfo)
	}

	// handle outputs
	for _, unSignedTxOut := range pb.UnsignedTx.TxOut {
		if unSignedTxOut.PkScript[0] == txscript.OP_RETURN {
			continue
		}
		address, err := common.PkScriptToAddr(unSignedTxOut.PkScript, chain)
		if err != nil {
			resp.Code = -1
			resp.Msg = err.Error()
			c.JSON(http.StatusOK, resp)
			return
		}
		outputInfo := OutputInfo{
			Address:      address,
			Value:        unSignedTxOut.Value,
			Inscriptions: make([]Inscription, 0),
			// Atomicals:    make([]Atomical, 0),
			Runes: make([]RuneBalance, 0),
		}
		// psbt
		if !isIncompletePSBT && len(inputRanges) > 0 && common.GetOrdinalsSize(inputRanges) >= unSignedTxOut.Value {
			transferred, remaining := common.TransferRanges(inputRanges, unSignedTxOut.Value)
			inputRanges = remaining

			assets := base_indexer.ShareBaseIndexer.GetAssetsWithRanges(transferred)
			for ticker, mintinfo := range assets {
				if ticker == common.ASSET_TYPE_EXOTIC {
					// TODO 稀有聪需要有所表示出来
					continue
				}
				for id, v := range mintinfo {
					nft := base_indexer.ShareBaseIndexer.GetNftInfoWithInscriptionId(id)
					inscription := newInscription(nft)
					inscription.OutputValue = uint64(common.GetOrdinalsSize(v))
					outputInfo.Inscriptions = append(outputInfo.Inscriptions, *inscription)
				}
			}
		}

		resp.Data.OutputInfos = append(resp.Data.OutputInfos, outputInfo)
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Service) tx_broadcast(c *gin.Context) {
	resp := &TxBroadcastResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: "",
	}
	var req TxBroadcastReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	txid, err := bitcoin_rpc.ShareBitconRpc.SendRawTransaction(req.Rawtx, 0)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = txid
	c.JSON(http.StatusOK, resp)
}
