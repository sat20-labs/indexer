package bitcoind

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/wire"
	"github.com/gin-gonic/gin"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

// @Summary send Raw Transaction
// @Description send Raw Transaction
// @Tags ordx.btc
// @Produce json
// @Param signedTxHex body string true "Signed transaction hex"
// @Param maxfeerate body number false "Reject transactions whose fee rate is higher than the specified value, expressed in BTC/kB.default:"0.01"
// @Security Bearer
// @Success 200 {object} SendRawTxResp "Successful response"
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

	txid, err := bitcoin_rpc.ShareBitconRpc.SendRawTransaction(req.SignedTxHex, req.Maxfeerate)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = strings.Trim(txid, "\"")
	c.JSON(http.StatusOK, resp)
}

// @Summary get raw block with blockhash
// @Description get raw block with blockhash
// @Tags ordx.btc
// @Produce json
// @Param blockHash path string true "blockHash"
// @Security Bearer
// @Success 200 {object} RawBlockResp "Successful response"
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
// @Success 200 {object} BlockHashResp "Successful response"
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
// @Success 200 {object} TxResp "Successful response"
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
	tx, err := bitcoin_rpc.GetTx(txid)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	blockHeight, err := bitcoin_rpc.GetTxHeight(tx.Txid)
	if err != nil {
		mt, err := bitcoin_rpc.ShareBitconRpc.GetMemPoolEntry(tx.Txid)
		if err != nil {
			resp.Code = -1
			resp.Msg = err.Error()
			c.JSON(http.StatusOK, resp)
			return
		}
		blockHeight = int64(mt.Height)
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
			rawTx, err := bitcoin_rpc.GetTx(vin.Txid)
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
	tx, err := bitcoin_rpc.GetTx(txid)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	blockHeight, err := bitcoin_rpc.GetTxHeight(tx.Txid)
	if err != nil {
		mt, err := bitcoin_rpc.ShareBitconRpc.GetMemPoolEntry(tx.Txid)
		if err != nil {
			resp.Code = -1
			resp.Msg = err.Error()
			c.JSON(http.StatusOK, resp)
			return
		}
		blockHeight = int64(mt.Height)
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
// @Success 200 {object} TxResp "Successful response"
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
	rawtx, err := bitcoin_rpc.GetRawTx(txid)
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
// @Success 200 {object} BestBlockHeightResp "Successful response"
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

	blockhash, err := bitcoin_rpc.ShareBitconRpc.GetBestBlockhash()
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	header, err := bitcoin_rpc.ShareBitconRpc.GetBlockheader(blockhash)
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Data = header.Height
	c.JSON(http.StatusOK, resp)
}
