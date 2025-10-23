package bitcoind

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/wire"
	"github.com/gin-gonic/gin"
	"github.com/sat20-labs/indexer/common"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
	"github.com/sat20-labs/indexer/share/base_indexer"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

// @Summary send Raw Transaction
// @Description send Raw Transaction
// @Tags ordx.btc
// @Produce json
// @Param signedTxHex body string true "Signed transaction hex"
// @Param maxfeerate body number false "Reject transactions whose fee rate is higher than the specified value, expressed in BTC/kB.default:"0.01"
// @Security Bearer
// @Success 200 {object} rpcwire.SendRawTxResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /btc/tx [post]
func (s *Service) sendRawTx(c *gin.Context) {
	resp := &rpcwire.SendRawTxResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: "",
	}
	var req rpcwire.SendRawTxReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	txid, err := bitcoin_rpc.ShareBitconRpc.SendTx(req.SignedTxHex)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = strings.Trim(txid, "\"")
	c.JSON(http.StatusOK, resp)
}

func (s *Service) sendRawTxs(c *gin.Context) {
	resp := &rpcwire.SendRawTxsResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}
	var req rpcwire.SendRawTxsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	result, err := bitcoin_rpc.ShareBitconRpc.TestTx(req.SignedTxHex)
	if err != nil {
	    resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	var reject bool
	for _, r := range result {
		if r.Allowed {
			resp.Data = append(resp.Data, r.TxId)
		} else {
			resp.Data = append(resp.Data, r.RejectReason)
			reject = true
		}
	}
	if reject {
		resp.Code = -1
		resp.Msg = "reject"
		c.JSON(http.StatusOK, resp)
		return
	}

	for i, signedTx := range req.SignedTxHex {
		_, err := bitcoin_rpc.ShareBitconRpc.SendTx(signedTx)
		if err != nil {
			resp.Code = -i
			resp.Msg = err.Error()
			c.JSON(http.StatusOK, resp)
			return
		}
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Service) testRawTx(c *gin.Context) {
	resp := &rpcwire.TestRawTxResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
	}
	var req rpcwire.TestRawTxReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	result, err := bitcoin_rpc.ShareBitconRpc.TestTx(req.SignedTxs)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	for _, r := range result {
		resp.Data = append(resp.Data, &rpcwire.TxTestResult{
			TxId: r.TxId,
			Allowed: r.Allowed,
			RejectReason: r.RejectReason,
		})
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary get raw block with blockhash
// @Description get raw block with blockhash
// @Tags ordx.btc
// @Produce json
// @Param blockHash path string true "blockHash"
// @Security Bearer
// @Success 200 {object} rpcwire.RawBlockResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /btc/block/{blockhash} [get]
func (s *Service) getRawBlock(c *gin.Context) {
	resp := &rpcwire.RawBlockResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: "",
	}
	blockHash := c.Param("blockhash")
	data, err := bitcoin_rpc.ShareBitconRpc.GetRawBlock(blockHash)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = data
	c.JSON(http.StatusOK, resp)
}

// @Summary get block hash with height
// @Description get block hash with height
// @Tags ordx.btc
// @Produce json
// @Param height path string true "height"
// @Security Bearer
// @Success 200 {object} rpcwire.BlockHashResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /btc/block/blockhash/{height} [get]
func (s *Service) getBlockHash(c *gin.Context) {
	resp := &rpcwire.BlockHashResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: "",
	}
	height, err := strconv.ParseUint(c.Param("height"), 10, 64)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	data, err := bitcoin_rpc.ShareBitconRpc.GetBlockHash(height)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = data
	c.JSON(http.StatusOK, resp)
}

// @Summary get tx with txid
// @Description get tx with txid
// @Tags ordx.btc
// @Produce json
// @Param txid path string true "txid"
// @Security Bearer
// @Success 200 {object} rpcwire.TxResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /btc/tx/{txid} [get]
func (s *Service) getTxInfo(c *gin.Context) {
	resp := &rpcwire.TxResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}
	txid := c.Param("txid")
	tx, err := bitcoin_rpc.ShareBitconRpc.GetTx(txid)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	var blockHeight int64
	if tx.BlockHash == "" {
		// try to read utxoId 
		for _, txOut := range tx.Vout {
			utxo := fmt.Sprintf("%s:%d", txid, txOut.N)
			id := base_indexer.ShareBaseIndexer.GetUtxoId(utxo)
			if id != common.INVALID_ID {
				height, _, _ := common.FromUtxoId(id)
				blockHeight = int64(height)
				break
			}
		}
	} else {
		blockHeader, err := bitcoin_rpc.ShareBitconRpc.GetBlockHeader(tx.BlockHash)
		if err != nil {
			mt, err := bitcoin_rpc.ShareBitconRpc.GetMemPoolEntry(tx.Txid)
			if err != nil {
				resp.Code = -1
				resp.Msg = err.Error()
				c.JSON(http.StatusOK, resp)
				return
			}
			blockHeight = int64(mt.Height)
		} else {
			blockHeight = (blockHeader.Height)
		}
	}

	
	txInfo := &rpcwire.TxInfo{
		TxID:          tx.Txid,
		Version:       tx.Version,
		Confirmations: tx.Confirmations,
		BlockHeight:   blockHeight,
		BlockTime:     tx.Blocktime,
		Vins:          make([]rpcwire.Vin, 0),
		Vouts:         make([]rpcwire.Vout, 0),
	}

	for _, vin := range tx.Vin {
		address := ""
		value := float64(0)
		utxo := ""
		if vin.Vout >= 0 {
			rawTx, err := bitcoin_rpc.ShareBitconRpc.GetTx(vin.Txid)
			if err != nil {
				resp.Code = -1
				resp.Msg = err.Error()
				c.JSON(http.StatusOK, resp)
				return
			}

			if len(rawTx.Vout) > vin.Vout {
				vout := rawTx.Vout[vin.Vout]
				address = vout.ScriptPubKey.Address
				value = vout.Value * 1e8
			} else {
				resp.Code = -1
				resp.Msg = "vout not found"
				c.JSON(http.StatusOK, resp)
				return
			}
			utxo = fmt.Sprintf("%s:%d", vin.Txid, vin.Vout)
		} else {
			out := wire.OutPoint{}
			utxo = out.String()
		}

		txInfo.Vins = append(txInfo.Vins, rpcwire.Vin{
			Utxo:     utxo,
			Sequence: vin.Sequence,
			Address:  address,
			Value:    int64(value),
		})
	}

	for _, vout := range tx.Vout {
		txInfo.Vouts = append(txInfo.Vouts, rpcwire.Vout{
			Address: vout.ScriptPubKey.Address,
			Value:   int64(vout.Value * 1e8),
		})
	}

	resp.Data = txInfo
	c.JSON(http.StatusOK, resp)
}

func (s *Service) getTxSimpleInfo(c *gin.Context) {
	resp := &rpcwire.TxSimpleInfoResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}
	txid := c.Param("txid")
	tx, err := bitcoin_rpc.ShareBitconRpc.GetTx(txid)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	var blockHeight int64
	if tx.BlockHash == "" {
		// try to read utxoId 
		for _, txOut := range tx.Vout {
			utxo := fmt.Sprintf("%s:%d", txid, txOut.N)
			id := base_indexer.ShareBaseIndexer.GetUtxoId(utxo)
			if id != common.INVALID_ID {
				height, _, _ := common.FromUtxoId(id)
				blockHeight = int64(height)
				break
			}
		}
	} else {
		blockHeader, err := bitcoin_rpc.ShareBitconRpc.GetBlockHeader(tx.BlockHash)
		if err != nil {
			mt, err := bitcoin_rpc.ShareBitconRpc.GetMemPoolEntry(tx.Txid)
			if err != nil {
				resp.Code = -1
				resp.Msg = err.Error()
				c.JSON(http.StatusOK, resp)
				return
			}
			blockHeight = int64(mt.Height)
		} else {
			blockHeight = (blockHeader.Height)
		}
	}

	if tx.Confirmations == 1 {
		// 需要确保这个tx已经被索引器解析，
		if blockHeight > int64(base_indexer.ShareBaseIndexer.GetSyncHeight()) {
			resp.Code = -1
			resp.Msg = "tx is not be indexed yet"
			c.JSON(http.StatusOK, resp)
			return
		}
	}

	txInfo := &rpcwire.TxSimpleInfo{
		TxID:          tx.Txid,
		Version:       tx.Version,
		Confirmations: tx.Confirmations,
		BlockHeight:   blockHeight,
		BlockTime:     tx.Blocktime,
	}

	resp.Data = txInfo
	c.JSON(http.StatusOK, resp)
}

// @Summary get raw tx with txid
// @Description get raw tx with txid
// @Tags ordx.btc
// @Produce json
// @Param txid path string true "txid"
// @Security Bearer
// @Success 200 {object} rpcwire.TxResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /btc/rawtx/{txid} [get]
func (s *Service) getRawTx(c *gin.Context) {
	resp := &rpcwire.TxResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: nil,
	}
	txid := c.Param("txid")
	rawtx, err := bitcoin_rpc.ShareBitconRpc.GetRawTx(txid)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = rawtx
	c.JSON(http.StatusOK, resp)
}

// @Summary get best block height
// @Description get best block height
// @Tags ordx.btc
// @Produce json
// @Security Bearer
// @Success 200 {object} rpcwire.BestBlockHeightResp "Successful response"
// @Failure 401 "Invalid API Key"
// @Router /btc/block/bestblockheight [get]
func (s *Service) getBestBlockHeight(c *gin.Context) {
	resp := &rpcwire.BestBlockHeightResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: -1,
	}

	blockhash, err := bitcoin_rpc.ShareBitconRpc.GetBestBlockHash()
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	header, err := bitcoin_rpc.ShareBitconRpc.GetBlockHeader(blockhash)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = header.Height
	c.JSON(http.StatusOK, resp)
}


func (s *Service) feeSummary(c *gin.Context) {
	resp := &rpcwire.FeeSummaryResp{
		BaseResp: rpcwire.BaseResp{
			Code: 0,
			Msg:  "ok",
		},
		Data: &rpcwire.FeeSummaryList{
			List: []*rpcwire.FeeSummary{
				{
					Title:   "Slow",
					Desc:    "About 1 hours",
					FeeRate: "20",
				},
				{
					Title:   "Normal",
					Desc:    "About 30 minutes",
					FeeRate: "50",
				},
				{
					Title:   "Fast",
					Desc:    "About 10 minutes",
					FeeRate: "100",
				},
			},
		},
	}

	ret, err := bitcoin_rpc.ShareBitconRpc.EstimateSmartFeeWithMode(6, "ECONOMICAL")
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	// BTC/kb -> sat/vb
	resp.Data.List[0].FeeRate = strconv.FormatFloat((ret.FeeRate * 100000), 'f', 2, 64)

	ret, err = bitcoin_rpc.ShareBitconRpc.EstimateSmartFeeWithMode(3, "ECONOMICAL")
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data.List[1].FeeRate = strconv.FormatFloat((ret.FeeRate * 100000), 'f', 2, 64)

	ret, err = bitcoin_rpc.ShareBitconRpc.EstimateSmartFeeWithMode(1, "CONSERVATIVE")
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Data.List[2].FeeRate = strconv.FormatFloat((ret.FeeRate * 100000), 'f', 2, 64)
	c.JSON(http.StatusOK, resp)
}

