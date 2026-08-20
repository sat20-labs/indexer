package common

import "github.com/sat20-labs/indexer/common"

var STEP_RUN_MODE = false // true: 模拟正常的服务状态更新数据。仅用于测试。生产环境设置为false。
var CHECK_SELF = true // 当STEP_RUN_MODE为true时，最好设置为false。生产环境设置为true

type BlockProcCallback interface {
	PrepareUpdateTransfer(block *common.Block, coinbase []*common.Range)
	TxInputProcess(txIndex int, tx *common.Transaction, 
		block *common.Block, coinbase []*common.Range) *common.TxOutput
	UpdateTransferFinished(block *common.Block)
}