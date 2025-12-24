package common

// 1.0.0  2025.02.03   支持STP
// 1.0.1  2025.05.20   runes协议使用spacer name作为对外的索引
// 1.0.3  2025.08.21   升级数据库
// 1.0.4  2025.11.06   brc20
// 1.1.0  2025.12.24   using local range for asset binding
const ORDX_INDEXER_VERSION = "1.1.0"


// 1.1.0  2024.07.01-
// 1.2.0  2024.07.20    multi-address
// 1.3.0  2024.10.21    utxoId = channel short id
// 1.4.0  2024.12.24    support STP
// 1.5.0  2025.02.03    support n parameter
// 1.6.0  2025.08.10    change db from badger to levelDB
// 1.7.0  2025.12.01    use local Range
const BASE_DB_VERSION = "1.7.0"
