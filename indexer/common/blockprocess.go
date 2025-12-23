package common

import "github.com/sat20-labs/indexer/common"


type BlockProcCallback interface {
	PrepareUpdateTransfer(block *common.Block, coinbase []*common.Range)
	TxInputProcess(txIndex int, tx *common.Transaction, 
		block *common.Block, coinbase []*common.Range) *common.TxOutput
	UpdateTransferFinished(block *common.Block)
}