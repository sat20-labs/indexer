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
	for i, tx := range transactions {
		inputs := []*common.TxInput{}
		outputs := []*common.TxOutputV2{}

		msgTx := tx.MsgTx()
		for _, txIn := range msgTx.TxIn {
			input := &common.TxInput{
				TxOutputV2: common.TxOutputV2{
					TxOutput: common.TxOutput{
						UtxoId:        common.INVALID_ID,
						OutPointStr:   txIn.PreviousOutPoint.String(),
						Offsets:       make(map[common.AssetName]common.AssetOffsets),
						SatBindingMap: make(map[int64]*common.AssetInfo),
						Invalids:      make(map[common.AssetName]bool),
					},
					Vout:    int(txIn.PreviousOutPoint.Index),
				},
				Witness: txIn.Witness,
				TxId:    txIn.PreviousOutPoint.Hash.String(),
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

			output := common.GenerateTxOutput(msgTx, j)
			output.UtxoId = common.ToUtxoId(height, i, j)
			outputs = append(outputs, &common.TxOutputV2{
				TxOutput: *output,
				TxIndex:  i,
				Vout:     j,
				Height:   height,
				AddressType: int(scyptClass),
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
