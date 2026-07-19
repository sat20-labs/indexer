package bitcoind

import (
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/gin-gonic/gin"
	"github.com/sat20-labs/indexer/common"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
	"github.com/sat20-labs/indexer/share/base_indexer"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

const maxBitcoinEvidenceBatch = 500

func evidenceOK() rpcwire.BaseResp {
	return rpcwire.BaseResp{Code: 0, Msg: "ok"}
}

func evidenceError(c *gin.Context, err error) {
	c.JSON(http.StatusOK, &rpcwire.TxResp{
		BaseResp: rpcwire.BaseResp{Code: -1, Msg: err.Error()},
	})
}

func validateEvidenceBatch(size int) error {
	if size == 0 {
		return fmt.Errorf("empty Bitcoin evidence batch")
	}
	if size > maxBitcoinEvidenceBatch {
		return fmt.Errorf("Bitcoin evidence batch exceeds %d items", maxBitcoinEvidenceBatch)
	}
	return nil
}

func requireBitcoinEvidenceBackend() error {
	if bitcoin_rpc.ShareBitconRpc == nil {
		return fmt.Errorf("Bitcoin evidence backend is unavailable")
	}
	return nil
}

func parseEvidenceOutpoint(outpoint string) (string, uint32, error) {
	separator := strings.LastIndexByte(outpoint, ':')
	if separator <= 0 || separator == len(outpoint)-1 {
		return "", 0, fmt.Errorf("invalid outpoint %q", outpoint)
	}
	txid := outpoint[:separator]
	if len(txid) != chainhash.MaxHashStringSize {
		return "", 0, fmt.Errorf("invalid outpoint txid %q", txid)
	}
	if _, err := chainhash.NewHashFromStr(txid); err != nil {
		return "", 0, fmt.Errorf("invalid outpoint txid %q: %w", txid, err)
	}
	index, err := strconv.ParseUint(outpoint[separator+1:], 10, 32)
	if err != nil {
		return "", 0, fmt.Errorf("invalid outpoint index %q: %w", outpoint[separator+1:], err)
	}
	return txid, uint32(index), nil
}

func bitcoinValueSats(value float64) int64 {
	return int64(math.Round(value * 1e8))
}

func getBitcoinUTXOStatus(outpoint string) *rpcwire.BitcoinUTXOStatus {
	status := &rpcwire.BitcoinUTXOStatus{Outpoint: outpoint}
	if err := requireBitcoinEvidenceBackend(); err != nil {
		status.Error = err.Error()
		return status
	}
	txid, vout, err := parseEvidenceOutpoint(outpoint)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	tx, err := bitcoin_rpc.ShareBitconRpc.GetTx(txid)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	if int(vout) >= len(tx.Vout) {
		status.Error = fmt.Sprintf("output index %d is outside transaction %s", vout, txid)
		return status
	}
	origin := tx.Vout[vout]
	status.Exists = true
	status.Value = bitcoinValueSats(origin.Value)
	status.PkScript = origin.ScriptPubKey.Hex
	status.Confirmations = int64(tx.Confirmations)
	status.BlockHash = tx.BlockHash

	unspent, err := bitcoin_rpc.ShareBitconRpc.GetTxOut(txid, vout, true)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	status.Unspent = unspent != nil
	if unspent != nil {
		status.Value = bitcoinValueSats(unspent.Value)
		status.PkScript = unspent.ScriptPubKey.Hex
		status.Confirmations = int64(unspent.Confirmations)
		if status.BlockHash == "" {
			status.BlockHash = unspent.Bestblock
		}
	}
	return status
}

func (s *Service) getBitcoinUTXOsByScripts(c *gin.Context) {
	var req rpcwire.BitcoinScriptsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		evidenceError(c, err)
		return
	}
	if err := validateEvidenceBatch(len(req.Scripts)); err != nil {
		evidenceError(c, err)
		return
	}
	if base_indexer.ShareBaseIndexer == nil {
		evidenceError(c, fmt.Errorf("Bitcoin evidence backend is unavailable"))
		return
	}
	if err := requireBitcoinEvidenceBackend(); err != nil {
		evidenceError(c, err)
		return
	}

	data := make([]*rpcwire.BitcoinScriptUTXOs, 0, len(req.Scripts))
	for _, scriptHex := range req.Scripts {
		item := &rpcwire.BitcoinScriptUTXOs{Script: scriptHex, UTXOs: make([]*rpcwire.BitcoinUTXO, 0)}
		script, err := hex.DecodeString(scriptHex)
		if err != nil {
			item.Error = err.Error()
			data = append(data, item)
			continue
		}
		address, err := common.PkScriptToAddr(script, base_indexer.ShareBaseIndexer.GetChainParam())
		if err != nil {
			item.Error = err.Error()
			data = append(data, item)
			continue
		}
		utxos, err := base_indexer.ShareBaseIndexer.GetUTXOsWithAddress(address)
		if err != nil {
			item.Error = err.Error()
			data = append(data, item)
			continue
		}
		for id := range utxos {
			outpoint := base_indexer.ShareBaseIndexer.GetUtxoById(id)
			txid, vout, err := parseEvidenceOutpoint(outpoint)
			if err != nil {
				continue
			}
			out, err := bitcoin_rpc.ShareBitconRpc.GetTxOut(txid, vout, true)
			if err != nil || out == nil || !strings.EqualFold(out.ScriptPubKey.Hex, scriptHex) {
				continue
			}
			item.UTXOs = append(item.UTXOs, &rpcwire.BitcoinUTXO{
				Outpoint:      outpoint,
				Value:         bitcoinValueSats(out.Value),
				PkScript:      out.ScriptPubKey.Hex,
				Confirmations: int64(out.Confirmations),
			})
		}
		sort.Slice(item.UTXOs, func(i, j int) bool { return item.UTXOs[i].Outpoint < item.UTXOs[j].Outpoint })
		data = append(data, item)
	}
	c.JSON(http.StatusOK, &rpcwire.BitcoinUTXOsByScriptsResp{BaseResp: evidenceOK(), Data: data})
}

func (s *Service) getBitcoinUTXOStatuses(c *gin.Context) {
	var req rpcwire.BitcoinOutpointsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		evidenceError(c, err)
		return
	}
	if err := validateEvidenceBatch(len(req.Outpoints)); err != nil {
		evidenceError(c, err)
		return
	}
	data := make([]*rpcwire.BitcoinUTXOStatus, 0, len(req.Outpoints))
	for _, outpoint := range req.Outpoints {
		data = append(data, getBitcoinUTXOStatus(outpoint))
	}
	c.JSON(http.StatusOK, &rpcwire.BitcoinUTXOStatusResp{BaseResp: evidenceOK(), Data: data})
}

func getBitcoinTxStatus(txid string) *rpcwire.BitcoinTxStatus {
	status := &rpcwire.BitcoinTxStatus{TxID: txid}
	if err := requireBitcoinEvidenceBackend(); err != nil {
		status.Error = err.Error()
		return status
	}
	if _, err := chainhash.NewHashFromStr(txid); err != nil {
		status.Error = err.Error()
		return status
	}
	tx, err := bitcoin_rpc.ShareBitconRpc.GetTx(txid)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	status.Exists = true
	status.Confirmations = int64(tx.Confirmations)
	status.BlockHash = tx.BlockHash
	status.BlockTime = tx.Blocktime
	status.Confirmed = tx.BlockHash != "" && tx.Confirmations > 0
	status.InMempool = !status.Confirmed
	if status.Confirmed {
		header, err := bitcoin_rpc.ShareBitconRpc.GetBlockHeader(tx.BlockHash)
		if err != nil {
			status.Error = err.Error()
		} else {
			status.BlockHeight = header.Height
		}
	}
	return status
}

func (s *Service) getBitcoinTxStatuses(c *gin.Context) {
	var req rpcwire.BitcoinTxIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		evidenceError(c, err)
		return
	}
	if err := validateEvidenceBatch(len(req.TxIDs)); err != nil {
		evidenceError(c, err)
		return
	}
	data := make([]*rpcwire.BitcoinTxStatus, 0, len(req.TxIDs))
	for _, txid := range req.TxIDs {
		data = append(data, getBitcoinTxStatus(txid))
	}
	c.JSON(http.StatusOK, &rpcwire.BitcoinTxStatusResp{BaseResp: evidenceOK(), Data: data})
}

func (s *Service) getBitcoinRawTxs(c *gin.Context) {
	var req rpcwire.BitcoinTxIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		evidenceError(c, err)
		return
	}
	if err := validateEvidenceBatch(len(req.TxIDs)); err != nil {
		evidenceError(c, err)
		return
	}
	if err := requireBitcoinEvidenceBackend(); err != nil {
		evidenceError(c, err)
		return
	}
	data := make([]*rpcwire.BitcoinRawTx, 0, len(req.TxIDs))
	for _, txid := range req.TxIDs {
		item := &rpcwire.BitcoinRawTx{TxID: txid}
		raw, err := bitcoin_rpc.ShareBitconRpc.GetRawTx(txid)
		if err != nil {
			item.Error = err.Error()
		} else {
			item.RawTx = raw
		}
		data = append(data, item)
	}
	c.JSON(http.StatusOK, &rpcwire.BitcoinRawTxResp{BaseResp: evidenceOK(), Data: data})
}

func (s *Service) getBitcoinOutspends(c *gin.Context) {
	var req rpcwire.BitcoinOutpointsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		evidenceError(c, err)
		return
	}
	if err := validateEvidenceBatch(len(req.Outpoints)); err != nil {
		evidenceError(c, err)
		return
	}
	data := make([]*rpcwire.BitcoinOutspend, 0, len(req.Outpoints))
	for _, outpoint := range req.Outpoints {
		status := getBitcoinUTXOStatus(outpoint)
		data = append(data, &rpcwire.BitcoinOutspend{
			Outpoint: outpoint,
			Exists:   status.Exists,
			Spent:    status.Exists && !status.Unspent,
			Error:    status.Error,
		})
	}
	c.JSON(http.StatusOK, &rpcwire.BitcoinOutspendsResp{BaseResp: evidenceOK(), Data: data})
}

func (s *Service) broadcastBitcoinTx(c *gin.Context) {
	var req rpcwire.BitcoinBroadcastReq
	if err := c.ShouldBindJSON(&req); err != nil {
		evidenceError(c, err)
		return
	}
	if err := requireBitcoinEvidenceBackend(); err != nil {
		evidenceError(c, err)
		return
	}
	result := &rpcwire.BitcoinBroadcastResult{}
	txid, err := bitcoin_rpc.ShareBitconRpc.SendTx(req.RawTx)
	if err != nil {
		result.Error = err.Error()
	} else {
		result.Accepted = true
		result.TxID = strings.Trim(txid, "\"")
	}
	c.JSON(http.StatusOK, &rpcwire.BitcoinBroadcastResp{BaseResp: evidenceOK(), Data: result})
}

func (s *Service) getBitcoinTip(c *gin.Context) {
	if err := requireBitcoinEvidenceBackend(); err != nil {
		evidenceError(c, err)
		return
	}
	hash, err := bitcoin_rpc.ShareBitconRpc.GetBestBlockHash()
	if err != nil {
		evidenceError(c, err)
		return
	}
	header, err := bitcoin_rpc.ShareBitconRpc.GetBlockHeader(hash)
	if err != nil {
		evidenceError(c, err)
		return
	}
	c.JSON(http.StatusOK, &rpcwire.BitcoinTipResp{
		BaseResp: evidenceOK(),
		Data:     &rpcwire.BitcoinTip{Height: header.Height, BlockHash: hash, Chainwork: header.Chainwork},
	})
}

func (s *Service) getBitcoinBlockHeader(c *gin.Context) {
	if err := requireBitcoinEvidenceBackend(); err != nil {
		evidenceError(c, err)
		return
	}
	height, err := strconv.ParseUint(c.Param("height"), 10, 64)
	if err != nil {
		evidenceError(c, err)
		return
	}
	hash, err := bitcoin_rpc.ShareBitconRpc.GetBlockHash(height)
	if err != nil {
		evidenceError(c, err)
		return
	}
	header, err := bitcoin_rpc.ShareBitconRpc.GetBlockHeader(hash)
	if err != nil {
		evidenceError(c, err)
		return
	}
	c.JSON(http.StatusOK, &rpcwire.BitcoinBlockHeaderResp{
		BaseResp: evidenceOK(),
		Data: &rpcwire.BitcoinBlockHeader{
			Height:            header.Height,
			Hash:              header.Hash,
			PreviousBlockHash: header.Previousblockhash,
			MerkleRoot:        header.Merkleroot,
			Time:              header.Time,
			MedianTime:        header.Mediantime,
			Confirmations:     header.Confirmations,
			Chainwork:         header.Chainwork,
		},
	})
}

func estimateBitcoinFeeRate(blocks int, mode string) (float64, error) {
	if err := requireBitcoinEvidenceBackend(); err != nil {
		return 0, err
	}
	result, err := bitcoin_rpc.ShareBitconRpc.EstimateSmartFeeWithMode(blocks, mode)
	if err != nil {
		return 0, err
	}
	if result == nil || result.FeeRate <= 0 {
		return 0, fmt.Errorf("fee estimator returned no rate for %d blocks", blocks)
	}
	return result.FeeRate * 100000, nil
}

func (s *Service) getBitcoinFeeRate(c *gin.Context) {
	slow, err := estimateBitcoinFeeRate(6, "ECONOMICAL")
	if err != nil {
		evidenceError(c, err)
		return
	}
	normal, err := estimateBitcoinFeeRate(3, "ECONOMICAL")
	if err != nil {
		evidenceError(c, err)
		return
	}
	fast, err := estimateBitcoinFeeRate(1, "CONSERVATIVE")
	if err != nil {
		evidenceError(c, err)
		return
	}
	c.JSON(http.StatusOK, &rpcwire.BitcoinFeeRateResp{
		BaseResp: evidenceOK(),
		Data:     &rpcwire.BitcoinFeeRate{Slow: slow, Normal: normal, Fast: fast, Unit: "sat/vB"},
	})
}
