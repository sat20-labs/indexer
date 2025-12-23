package indexer

import "github.com/sat20-labs/indexer/common"

// 另外一套更精确执行区块数据编译的接口，按区块中交易逐个模块回调执行。如果某个模块的结果对下一个模块有影响，直接将编译结果放在tx中
func (b *IndexerMgr) PrepareUpdateTransfer(block *common.Block, coinbase []*common.Range) {
	b.brc20Indexer.PrepareUpdateTransfer(block, coinbase)
}

func  (b *IndexerMgr) TxInputProcess(txIndex int, tx *common.Transaction, 
block *common.Block, coinbase []*common.Range) *common.TxOutput {
	return b.brc20Indexer.TxInputProcess(txIndex, tx, block, coinbase)
}

func  (b *IndexerMgr) UpdateTransferFinished(block *common.Block) {
	b.brc20Indexer.UpdateTransferFinished(block)
}


