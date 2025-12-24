package base

import (
	"encoding/hex"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/sat20-labs/indexer/common"
)

// 不要panic，可能会影响写数据库
func FetchBlock(height int, chaincfgParam *chaincfg.Params) *common.Block {
	hash, err := getBlockHash(uint64(height))
	if err != nil {
		common.Log.Errorf("getBlockHash %d failed. %v", height, err)
		return nil
		//common.Log.Fatalln(err)
	}

	rawBlock, err := getRawBlock(hash)
	if err != nil {
		common.Log.Errorf("getRawBlock %d %s failed. %v", height, hash, err)
		return nil
		//common.Log.Fatalln(err)
	}
	blockData, err := hex.DecodeString(rawBlock)
	if err != nil {
		common.Log.Errorf("DecodeString %d %s failed. %v", height, rawBlock, err)
		return nil
		//common.Log.Panicf("BaseIndexer.fetchBlock-> Failed to decode block: %v", err)
	}

	// Deserialize the bytes into a btcutil.Block.
	block, err := btcutil.NewBlockFromBytes(blockData)
	if err != nil {
		common.Log.Errorf("NewBlockFromBytes %d failed. %v", height, err)
		return nil
		//common.Log.Panicf("BaseIndexer.fetchBlock-> Failed to parse block: %v", err)
	}

	transactions := block.Transactions()
	txs := make([]*common.Transaction, len(transactions))
	for txIndex, tx := range transactions {
		inputs := []*common.TxInput{}
		outputs := []*common.TxOutputV2{}

		msgTx := tx.MsgTx()
		for index, txIn := range msgTx.TxIn {
			input := &common.TxInput{
				TxOutputV2: common.TxOutputV2{
					TxOutput: common.TxOutput{
						UtxoId:        0,
						OutPointStr:   txIn.PreviousOutPoint.String(),
						Offsets:       make(map[common.AssetName]common.AssetOffsets),
						SatBindingMap: make(map[int64]*common.AssetInfo),
						Invalids:      make(map[common.AssetName]bool),
					},
					TxOutIndex: int(txIn.PreviousOutPoint.Index),
				},
				Witness:   txIn.Witness,
				TxId:      txIn.PreviousOutPoint.Hash.String(),
				InHeight:  height,
				InTxIndex: txIndex,
				TxInIndex: index,
			}
			inputs = append(inputs, input)
		}

		// parse the raw tx values
		for j, v := range msgTx.TxOut {
			//Determine the type of the script and extract the address
			scyptClass, _, _, err := txscript.ExtractPkScriptAddrs(v.PkScript, chaincfgParam)
			if err != nil {
				common.Log.Errorf("ExtractPkScriptAddrs %d failed. %v", height, err)
				return nil
				//common.Log.Panicf("BaseIndexer.fetchBlock-> Failed to extract address: %v", err)
			}
			// MultiSigTy, 例如testnet4: 21f4713326dbe56bdd553613fdf6f112086425f55c83fd54e8a2f36045e1d965 2/3签名
			// if len(addresses) > 1 {
			// 	common.Log.Infof("tx: %s has multi addresses %v", msgTx.TxID(), addresses)
			// }
			if scyptClass == txscript.NonStandardTy {
				// 很多opreturn被识别为NonStandarTy，比如testnet4 2826ead858ddb5b58d331849481189bd7b705ab0582a1d39b3b1c0bd42c864f9
				//common.Log.Infof("tx: %s has nonStandard address %v", msgTx.TxID(), addresses)
				if common.IsOpReturn(v.PkScript) {
					scyptClass = txscript.NullDataTy
				} else {
					//common.Log.Infof("tx: %s has not std address, pkscript %s", msgTx.TxID(), hex.EncodeToString(v.PkScript))
				}
				if len(v.PkScript) == 0 {
					// testnet4: 47cfff6998e67852eb8c2fe7fbef2a39c8443c9fa480c7b33a87d8dde1d8e3bd
					v.PkScript = []byte{0x51}
				}
			}

			output := common.GenerateTxOutput(msgTx, j)
			output.UtxoId = common.ToUtxoId(height, txIndex, j)
			outputs = append(outputs, &common.TxOutputV2{
				TxOutput:    *output,
				OutTxIndex:  txIndex,
				TxOutIndex:  j,
				OutHeight:   height,
				AddressType: int(scyptClass),
			})
		}

		txs[txIndex] = &common.Transaction{
			TxId:    tx.Hash().String(),
			Inputs:  inputs,
			Outputs: outputs,
		}
	}

	t := block.MsgBlock().Header.Timestamp
	bl := &common.Block{
		Timestamp:     t,
		Height:        height,
		Hash:          block.Hash().String(),
		PrevBlockHash: block.MsgBlock().Header.PrevBlock.String(),
		Transactions:  txs,
	}

	return bl
}

// Prefetches blocks from bitcoind and sends them to the blocksChan
func (b *BaseIndexer) spawnBlockFetcher(startHeigh int, endHeight int, stopChan chan struct{}) {
	currentHeight := startHeigh
	for currentHeight <= endHeight {
		select {
		case <-stopChan:
			return
		default:
			block := FetchBlock(currentHeight, b.chaincfgParam)
			b.blocksChan <- block
			currentHeight += 1
		}
	}

	<-stopChan
}

func (b *BaseIndexer) drainBlocksChan() {
	for {
		select {
		case <-b.blocksChan:
		default:
			return
		}
	}
}
