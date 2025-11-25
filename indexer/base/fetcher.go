package base

import (
	"encoding/hex"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/txscript"
	"github.com/sat20-labs/indexer/common"
)

// 不要panic，可能会影响写数据库
func (b *BaseIndexer) fetchBlock(height int) *common.Block {
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
	for i, tx := range transactions {
		inputs := []*common.TxInput{}
		outputs := []*common.TxOutputV2{}

		msgTx := tx.MsgTx()
		for _, txIn := range msgTx.TxIn {
			input := &common.TxInput{
				TxOutput: common.TxOutput{
					UtxoId:      common.INVALID_ID,
					OutPointStr: txIn.PreviousOutPoint.String(),
					Offsets:     make(map[common.AssetName]common.AssetOffsets),
					SatBindingMap: make(map[int64]*common.AssetInfo),
					Invalids: make(map[common.AssetName]bool),
				},
				Witness: txIn.Witness,
				Vout: int(txIn.PreviousOutPoint.Index),
				Txid: txIn.PreviousOutPoint.Hash.String(),
			}
			inputs = append(inputs, input)
		}

		// parse the raw tx values
		for j, v := range msgTx.TxOut {
			//Determine the type of the script and extract the address
			scyptClass, addrs, reqSig, err := txscript.ExtractPkScriptAddrs(v.PkScript, b.chaincfgParam)
			if err != nil {
				common.Log.Errorf("ExtractPkScriptAddrs %d failed. %v", height, err)
				return nil
				//common.Log.Panicf("BaseIndexer.fetchBlock-> Failed to extract address: %v", err)
			}

			addrsString := make([]string, len(addrs))
			for i, x := range addrs {
				if scyptClass == txscript.MultiSigTy {
					addrsString[i] = hex.EncodeToString(x.ScriptAddress()) // pubkey
				} else {
					addrsString[i] = x.EncodeAddress()
				}
			}

			var receiver common.ScriptPubKey

			if len(addrs) == 0 {
				address := "UNKNOWN"
				if scyptClass == txscript.NullDataTy {
					address = "OP_RETURN"
				}
				receiver = common.ScriptPubKey{
					Addresses: []string{address},
					Type:      int(scyptClass),
					PkScript: v.PkScript,
					ReqSig:   reqSig,
				}
			} else {
				receiver = common.ScriptPubKey{
					Addresses: addrsString,
					Type:      int(scyptClass),
					PkScript: v.PkScript,
					ReqSig:   reqSig,
				}
			}

			output := common.GenerateTxOutput(msgTx, j)
			output.UtxoId = common.ToUtxoId(height, i, j)
			outputs = append(outputs, &common.TxOutputV2{
				TxOutput: *output,
				Address: &receiver,
				TxIndex: i,
				Vout:    j,
				Height:  height,
			})
		}

		txs[i] = &common.Transaction{
			Txid:    tx.Hash().String(),
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
			block := b.fetchBlock(currentHeight)
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
